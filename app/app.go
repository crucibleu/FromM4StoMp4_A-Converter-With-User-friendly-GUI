package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"m4s-converter/engine"
	"m4s-converter/internal"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version 程序版本，构建时可通过 -ldflags "-X main.Version=..." 覆盖
var Version = "1.0.0-dev"

// App 暴露给前端的方法
type App struct {
	ctx   context.Context
	store *store
	media *mediaServer
	queue *engine.Queue

	mu     sync.Mutex
	videos map[string]engine.Video // 最近一次扫描结果

	probeMu    sync.Mutex
	probeCache map[string]probeEntry
}

type probeEntry struct {
	mod  time.Time
	size int64
	info engine.MediaInfo
}

// InitData 启动时前端需要的数据
type InitData struct {
	Settings        Settings `json:"settings"`
	OutputDir       string   `json:"outputDir"`
	Version         string   `json:"version"`
	Platform        string   `json:"platform"`
	MediaBase       string   `json:"mediaBase"`
	MediaToken      string   `json:"mediaToken"`
	DetectedDirs    []string `json:"detectedDirs"`
	DefaultTemplate string   `json:"defaultTemplate"`
}

// ScanResult 扫描结果
type ScanResult struct {
	CacheDir  string         `json:"cacheDir"`
	OutputDir string         `json:"outputDir"`
	Videos    []engine.Video `json:"videos"`
	Elapsed   int64          `json:"elapsed"` // 毫秒
}

// LibraryItem 输出目录中的一个视频
type LibraryItem struct {
	Path       string           `json:"path"`
	Name       string           `json:"name"`
	Folder     string           `json:"folder"` // 相对输出目录的文件夹，"" 表示根目录
	Size       int64            `json:"size"`
	Modified   int64            `json:"modified"` // unix ms
	Info       engine.MediaInfo `json:"info"`
	HasDanmaku bool             `json:"hasDanmaku"`
	Play       PlayState        `json:"play"`
}

// QueueTick 队列进度汇总，转换过程中每 0.5 秒发送一次
type QueueTick struct {
	Stats engine.QueueStats `json:"stats"`
	Speed float64           `json:"speed"` // 字节/秒
}

func newApp() *App {
	return &App{store: openStore(), videos: map[string]engine.Video{}, probeCache: map[string]probeEntry{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	cacheDir = filepath.Join(cacheDir, "m4s-converter")
	_ = os.MkdirAll(cacheDir, 0o755)
	if a.media, err = startMediaServer(cacheDir); err != nil {
		runtime.LogError(ctx, "media server: "+err.Error())
	}
	st := a.store.get()
	if a.media != nil {
		a.media.setRoots(st.CacheDir, st.outputDir())
	}

	a.queue = engine.NewQueue(st.Workers)
	a.queue.OnUpdate = func(j engine.Job) { runtime.EventsEmit(ctx, "job", j) }
	a.queue.OnIdle = func(s engine.QueueStats) { runtime.EventsEmit(ctx, "queue:idle", s) }
	go a.tick()
}

// beforeClose 转换进行中时确认是否退出
func (a *App) beforeClose(ctx context.Context) bool {
	if a.queue == nil || !a.queue.Busy() {
		return false
	}
	zh := a.isChinese()
	title, msg, yes, no := "Conversion in progress", "Videos are still being converted. Stop and quit?", "Quit", "Keep converting"
	if zh {
		title, msg, yes, no = "正在转换", "还有视频正在转换，是否停止并退出？", "退出", "继续转换"
	}
	res, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type: runtime.QuestionDialog, Title: title, Message: msg,
		Buttons: []string{yes, no}, DefaultButton: no, CancelButton: no,
	})
	if err != nil || res != yes && res != "Yes" {
		return true
	}
	a.queue.CancelAll()
	return false
}

func (a *App) isChinese() bool {
	switch a.store.get().Language {
	case "zh":
		return true
	case "en":
		return false
	}
	for _, k := range []string{"LANGUAGE", "LC_ALL", "LANG"} {
		if strings.HasPrefix(strings.ToLower(os.Getenv(k)), "zh") {
			return true
		}
	}
	return goruntime.GOOS == "windows" // Windows 上无法可靠读取，按中文用户为主
}

// tick 转换进行中时定期发送进度汇总
func (a *App) tick() {
	var lastBytes int64
	var lastTime time.Time
	var speed float64
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case now := <-t.C:
			s := a.queue.Stats()
			if s.Running == 0 {
				lastTime, speed = time.Time{}, 0
				continue
			}
			if !lastTime.IsZero() {
				inst := float64(s.Finished-lastBytes) / now.Sub(lastTime).Seconds()
				if inst < 0 {
					inst = 0
				}
				speed = speed*0.7 + inst*0.3
			}
			lastBytes, lastTime = s.Finished, now
			runtime.EventsEmit(a.ctx, "queue:tick", QueueTick{Stats: s, Speed: speed})
		}
	}
}

// Init 返回启动数据
func (a *App) Init() InitData {
	st := a.store.get()
	d := InitData{
		Settings: st, OutputDir: st.outputDir(), Version: Version, Platform: goruntime.GOOS,
		DetectedDirs: detectCacheDirs(), DefaultTemplate: engine.DefaultTemplate,
	}
	if a.media != nil {
		d.MediaBase, d.MediaToken = a.media.base, a.media.token
	}
	return d
}

// SaveSettings 保存设置，返回实际使用的输出目录
func (a *App) SaveSettings(s Settings) (string, error) {
	if s.Workers < 1 || s.Workers > 4 {
		s.Workers = 2
	}
	if strings.TrimSpace(s.Template) == "" {
		s.Template = engine.DefaultTemplate
	}
	if err := a.store.set(s); err != nil {
		return "", err
	}
	a.queue.SetWorkers(s.Workers)
	if a.media != nil {
		a.media.setRoots(s.CacheDir, s.outputDir())
	}
	return s.outputDir(), nil
}

// PickFolder 打开系统的选择文件夹对话框
func (a *App) PickFolder(title, start string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, DefaultDirectory: existingDir(start)})
}

// PickFile 打开系统的选择文件对话框
func (a *App) PickFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

func existingDir(p string) string {
	for p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return ""
}

// Scan 扫描设置中的缓存目录
func (a *App) Scan() (ScanResult, error) {
	st := a.store.get()
	start := time.Now()
	if strings.TrimSpace(st.CacheDir) == "" {
		return ScanResult{}, errors.New("no-cache-dir")
	}
	if fi, err := os.Stat(st.CacheDir); err != nil || !fi.IsDir() {
		return ScanResult{}, errors.New("cache-dir-missing")
	}
	videos, err := engine.Scan(st.CacheDir, engine.ScanOptions{Output: a.outputOptions(st)})
	if err != nil {
		return ScanResult{}, err
	}
	a.mu.Lock()
	a.videos = make(map[string]engine.Video, len(videos))
	for _, v := range videos {
		a.videos[v.ID] = v
	}
	a.mu.Unlock()
	if a.media != nil {
		a.media.setCovers(videos)
		a.media.setRoots(st.CacheDir, st.outputDir())
	}
	if videos == nil {
		videos = []engine.Video{}
	}
	return ScanResult{CacheDir: st.CacheDir, OutputDir: st.outputDir(), Videos: videos, Elapsed: time.Since(start).Milliseconds()}, nil
}

func (a *App) outputOptions(st Settings) engine.OutputOptions {
	return engine.OutputOptions{Dir: st.outputDir(), Template: st.Template}
}

// OutputExample 用某个视频演示命名模板的效果
func (a *App) OutputExample(videoID, dir, template string) string {
	a.mu.Lock()
	v, ok := a.videos[videoID]
	a.mu.Unlock()
	if !ok {
		v = engine.Video{Series: "【4K】世界各地的美景", Part: "P1 中国", Uploader: "UP主", Page: 1, BVID: "BV1xx411c7mD", CID: "1332097557"}
	}
	return engine.OutputPath(v, engine.OutputOptions{Dir: dir, Template: template})
}

// Enqueue 把选中的视频加入转换队列
func (a *App) Enqueue(ids []string) ([]engine.Job, error) {
	st := a.store.get()
	box := st.MP4Box
	if box == "" {
		p, err := internal.GetMP4Box()
		if err != nil {
			return nil, err
		}
		box = p
	} else if _, err := os.Stat(box); err != nil {
		return nil, errors.New("mp4box-missing")
	}
	a.mu.Lock()
	var videos []engine.Video
	for _, id := range ids {
		if v, ok := a.videos[id]; ok && (v.Status == engine.StatusReady || v.Status == engine.StatusConverted) {
			videos = append(videos, v)
		}
	}
	a.mu.Unlock()
	if len(videos) == 0 {
		return []engine.Job{}, nil
	}
	if err := os.MkdirAll(st.outputDir(), 0o755); err != nil {
		return nil, err
	}
	opt := engine.ConvertOptions{
		Output: a.outputOptions(st), MP4Box: box, Danmaku: st.Danmaku,
		DownloadDanmaku: st.DownloadDanmaku, Overwrite: st.Overwrite,
	}
	return a.queue.Add(videos, opt), nil
}

// Jobs 当前队列
func (a *App) Jobs() []engine.Job { return a.queue.Jobs() }

// CancelJob 取消任务
func (a *App) CancelJob(id string) { a.queue.Cancel(id) }

// CancelAll 取消所有任务
func (a *App) CancelAll() { a.queue.CancelAll() }

// RetryJob 重试任务
func (a *App) RetryJob(id string) { a.queue.Retry(id) }

// ClearFinished 清除已结束的任务
func (a *App) ClearFinished() []engine.Job {
	a.queue.ClearFinished()
	return a.queue.Jobs()
}

// Library 列出输出目录中的 mp4
func (a *App) Library() ([]LibraryItem, error) {
	root := a.store.get().outputDir()
	items := []LibraryItem{}
	if root == "" {
		return items, nil
	}
	if _, err := os.Stat(root); err != nil {
		return items, nil
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.EqualFold(filepath.Ext(name), ".mp4") || strings.HasSuffix(name, ".partial.mp4") {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		folder, _ := filepath.Rel(root, filepath.Dir(p))
		if folder == "." {
			folder = ""
		}
		items = append(items, LibraryItem{
			Path: p, Name: strings.TrimSuffix(name, filepath.Ext(name)), Folder: folder,
			Size: fi.Size(), Modified: fi.ModTime().UnixMilli(), Info: a.probe(p, fi),
			HasDanmaku: fileSize(engine.DanmakuPathFor(p)) > 0, Play: a.store.playState(p),
		})
		return nil
	})
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Folder != items[j].Folder {
			return items[i].Folder < items[j].Folder
		}
		return naturalLess(items[i].Name, items[j].Name)
	})
	return items, err
}

func (a *App) probe(p string, fi fs.FileInfo) engine.MediaInfo {
	a.probeMu.Lock()
	e, ok := a.probeCache[p]
	a.probeMu.Unlock()
	if ok && e.mod.Equal(fi.ModTime()) && e.size == fi.Size() {
		return e.info
	}
	info, _ := engine.Probe(p)
	a.probeMu.Lock()
	a.probeCache[p] = probeEntry{fi.ModTime(), fi.Size(), info}
	a.probeMu.Unlock()
	return info
}

// Danmaku 读取输出视频旁边的弹幕
func (a *App) Danmaku(mp4 string) []engine.Comment {
	c, _ := engine.LoadDanmaku(engine.DanmakuPathFor(mp4))
	if c == nil {
		c = []engine.Comment{}
	}
	return c
}

// CacheDanmaku 读取缓存视频的弹幕（预览用）
func (a *App) CacheDanmaku(videoID string) []engine.Comment {
	a.mu.Lock()
	v := a.videos[videoID]
	a.mu.Unlock()
	if v.DanmakuFile == "" {
		return []engine.Comment{}
	}
	c, _ := engine.LoadDanmaku(v.DanmakuFile)
	if c == nil {
		c = []engine.Comment{}
	}
	return c
}

// SavePlayState 保存播放进度
func (a *App) SavePlayState(path string, position float64, watched bool) error {
	return a.store.setPlay(path, PlayState{Position: position, Watched: watched})
}

// SaveThumb 保存页面截取的视频帧（data URL 或 base64 JPEG）作为缩略图
func (a *App) SaveThumb(mp4, data string) error {
	if a.media == nil {
		return nil
	}
	if i := strings.Index(data, ","); i >= 0 {
		data = data[i+1:]
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	p := a.media.thumbPath(mp4)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

// DiskFree 路径所在磁盘的可用空间（字节）
func (a *App) DiskFree(p string) uint64 {
	return diskFree(existingDir(p))
}

// OpenPath 用系统默认程序打开文件或文件夹
func (a *App) OpenPath(p string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", p)
	case "darwin":
		cmd = exec.Command("open", p)
	default:
		cmd = exec.Command("xdg-open", p)
	}
	return cmd.Start()
}

// RevealPath 在文件管理器中显示文件
func (a *App) RevealPath(p string) error {
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", p).Start()
	case "darwin":
		return exec.Command("open", "-R", p).Start()
	}
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		p = filepath.Dir(p)
	}
	return exec.Command("xdg-open", p).Start()
}

// IsDir 判断路径是否为文件夹（拖入文件夹时使用）
func (a *App) IsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}

// naturalLess 按数字大小排序文件名，P2 排在 P10 前面
func naturalLess(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		ca, cb := ra[i], rb[j]
		if isDigit(ca) && isDigit(cb) {
			si := i
			for i < len(ra) && isDigit(ra[i]) {
				i++
			}
			sj := j
			for j < len(rb) && isDigit(rb[j]) {
				j++
			}
			na := strings.TrimLeft(string(ra[si:i]), "0")
			nb := strings.TrimLeft(string(rb[sj:j]), "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ca != cb {
			return ca < cb
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }
