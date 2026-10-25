package engine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// JobState 任务状态
type JobState string

const (
	JobQueued   JobState = "queued"
	JobRunning  JobState = "running"
	JobDone     JobState = "done"
	JobSkipped  JobState = "skipped" // 输出已存在
	JobFailed   JobState = "failed"
	JobCanceled JobState = "canceled"
)

// Job 队列中的一个转换任务
type Job struct {
	ID       string   `json:"id"`
	VideoID  string   `json:"videoId"`
	Series   string   `json:"series"`
	Part     string   `json:"part"`
	Size     int64    `json:"size"`
	Output   string   `json:"output"`
	State    JobState `json:"state"`
	Stage    string   `json:"stage"`
	Progress float64  `json:"progress"` // 0..1
	Error    string   `json:"error"`
	Started  int64    `json:"started"`  // unix ms
	Finished int64    `json:"finished"` // unix ms

	video  Video
	opt    ConvertOptions
	cancel context.CancelFunc
}

// QueueStats 队列汇总
type QueueStats struct {
	Total    int   `json:"total"`
	Queued   int   `json:"queued"`
	Running  int   `json:"running"`
	Done     int   `json:"done"`
	Failed   int   `json:"failed"`
	Bytes    int64 `json:"bytes"`    // 所有任务的大小
	Finished int64 `json:"finished"` // 已处理的字节数（按进度估算）
}

// Queue 并行执行转换任务
type Queue struct {
	mu       sync.Mutex
	jobs     []*Job
	workers  int
	running  int
	seq      int
	reserved map[string]string // 输出路径 -> 任务 ID，避免同一批任务写同一个文件

	// OnUpdate 任务状态变化时调用（在工作协程中调用，需自行切换线程）
	OnUpdate func(Job)
	// OnIdle 所有任务结束时调用
	OnIdle func(QueueStats)
	// convert 便于测试替换
	convert func(ctx context.Context, v Video, opt ConvertOptions, p func(Progress)) (string, error)
}

// NewQueue 创建队列，workers 为同时进行的任务数
func NewQueue(workers int) *Queue {
	q := &Queue{reserved: map[string]string{}, convert: Convert}
	q.SetWorkers(workers)
	return q
}

// SetWorkers 设置并行数（1-4）
func (q *Queue) SetWorkers(n int) {
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	q.mu.Lock()
	q.workers = n
	q.mu.Unlock()
	q.schedule()
}

// Add 加入任务，返回新任务
func (q *Queue) Add(videos []Video, opt ConvertOptions) []Job {
	q.mu.Lock()
	var added []Job
	for _, v := range videos {
		q.seq++
		v.Output = q.uniqueOutput(OutputPath(v, opt.Output), opt.Overwrite)
		j := &Job{
			ID: "j" + strconv.Itoa(q.seq), VideoID: v.ID, Series: v.Series, Part: v.Part,
			Size: v.Size, Output: v.Output, State: JobQueued, video: v, opt: opt,
		}
		q.reserved[v.Output] = j.ID
		q.jobs = append(q.jobs, j)
		added = append(added, j.snapshot())
	}
	q.mu.Unlock()
	for _, j := range added {
		q.emit(j)
	}
	q.schedule()
	return added
}

// uniqueOutput 同一批中出现重名时追加 (2)、(3)…；调用者持有锁
func (q *Queue) uniqueOutput(p string, overwrite bool) string {
	base := strings.TrimSuffix(p, ".mp4")
	for i := 1; ; i++ {
		cand := p
		if i > 1 {
			cand = fmt.Sprintf("%s (%d).mp4", base, i)
		}
		if _, taken := q.reserved[cand]; taken {
			continue
		}
		if i > 1 && !overwrite && fileExists(cand) {
			continue
		}
		return cand
	}
}

// Cancel 取消一个排队中或运行中的任务
func (q *Queue) Cancel(id string) {
	q.mu.Lock()
	j := q.find(id)
	if j == nil {
		q.mu.Unlock()
		return
	}
	var snap *Job
	switch j.State {
	case JobQueued:
		j.State = JobCanceled
		delete(q.reserved, j.Output)
		s := j.snapshot()
		snap = &s
	case JobRunning:
		if j.cancel != nil {
			j.cancel()
		}
	}
	q.mu.Unlock()
	if snap != nil {
		q.emit(*snap)
		q.checkIdle()
	}
}

// CancelAll 取消所有未完成的任务
func (q *Queue) CancelAll() {
	q.mu.Lock()
	var ids []string
	for _, j := range q.jobs {
		if j.State == JobQueued || j.State == JobRunning {
			ids = append(ids, j.ID)
		}
	}
	q.mu.Unlock()
	for _, id := range ids {
		q.Cancel(id)
	}
}

// Retry 重新执行失败或取消的任务
func (q *Queue) Retry(id string) {
	q.mu.Lock()
	j := q.find(id)
	if j == nil || (j.State != JobFailed && j.State != JobCanceled) {
		q.mu.Unlock()
		return
	}
	j.State, j.Error, j.Progress, j.Stage, j.Started, j.Finished = JobQueued, "", 0, "", 0, 0
	q.reserved[j.Output] = j.ID
	s := j.snapshot()
	q.mu.Unlock()
	q.emit(s)
	q.schedule()
}

// ClearFinished 从列表中移除已结束的任务
func (q *Queue) ClearFinished() {
	q.mu.Lock()
	keep := q.jobs[:0]
	for _, j := range q.jobs {
		if j.State == JobQueued || j.State == JobRunning {
			keep = append(keep, j)
		}
	}
	q.jobs = keep
	q.mu.Unlock()
}

// Jobs 返回所有任务的快照
func (q *Queue) Jobs() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		out = append(out, j.snapshot())
	}
	return out
}

// Stats 返回汇总
func (q *Queue) Stats() QueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.statsLocked()
}

func (q *Queue) statsLocked() QueueStats {
	var s QueueStats
	for _, j := range q.jobs {
		if j.State == JobCanceled {
			continue
		}
		s.Total++
		s.Bytes += j.Size
		s.Finished += int64(float64(j.Size) * j.Progress)
		switch j.State {
		case JobQueued:
			s.Queued++
		case JobRunning:
			s.Running++
		case JobDone, JobSkipped:
			s.Done++
		case JobFailed:
			s.Failed++
		}
	}
	return s
}

// Busy 是否还有未完成的任务
func (q *Queue) Busy() bool {
	s := q.Stats()
	return s.Queued+s.Running > 0
}

func (q *Queue) find(id string) *Job {
	for _, j := range q.jobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}

func (q *Queue) schedule() {
	for {
		q.mu.Lock()
		if q.running >= q.workers {
			q.mu.Unlock()
			return
		}
		var next *Job
		for _, j := range q.jobs {
			if j.State == JobQueued {
				next = j
				break
			}
		}
		if next == nil {
			q.mu.Unlock()
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		next.cancel = cancel
		next.State, next.Started, next.Stage = JobRunning, time.Now().UnixMilli(), "prepare"
		q.running++
		s := next.snapshot()
		q.mu.Unlock()
		q.emit(s)
		go q.run(ctx, next)
	}
}

func (q *Queue) run(ctx context.Context, j *Job) {
	var lastEmit time.Time
	out, err := q.convert(ctx, j.video, j.opt, func(p Progress) {
		q.mu.Lock()
		j.Stage = p.Stage
		// prepare 0-35%，merge 35-95%，danmaku 95-100%
		switch p.Stage {
		case "prepare":
			j.Progress = 0.35 * p.Fraction
		case "merge":
			j.Progress = 0.35 + 0.6*p.Fraction
		case "danmaku":
			j.Progress = 0.95 + 0.05*p.Fraction
		}
		s := j.snapshot()
		q.mu.Unlock()
		if time.Since(lastEmit) > 150*time.Millisecond {
			lastEmit = time.Now()
			q.emit(s)
		}
	})

	q.mu.Lock()
	j.cancel = nil
	j.Finished = time.Now().UnixMilli()
	switch {
	case err == nil:
		j.State, j.Progress, j.Output = JobDone, 1, out
	case errors.Is(err, ErrExists):
		j.State, j.Progress = JobSkipped, 1
	case errors.Is(err, context.Canceled):
		j.State, j.Progress = JobCanceled, 0
	default:
		j.State, j.Error = JobFailed, err.Error()
	}
	delete(q.reserved, j.Output)
	q.running--
	s := j.snapshot()
	q.mu.Unlock()
	q.emit(s)
	q.schedule()
	q.checkIdle()
}

func (q *Queue) checkIdle() {
	q.mu.Lock()
	stats := q.statsLocked()
	idle := q.running == 0 && stats.Queued == 0
	q.mu.Unlock()
	if idle && q.OnIdle != nil {
		q.OnIdle(stats)
	}
}

func (q *Queue) emit(j Job) {
	if q.OnUpdate != nil {
		q.OnUpdate(j)
	}
}

func (j *Job) snapshot() Job {
	c := *j
	c.cancel = nil
	return c
}
