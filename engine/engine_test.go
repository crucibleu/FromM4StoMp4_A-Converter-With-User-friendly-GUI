package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"m4s-converter/internal"
)

const danmakuXML = `<?xml version="1.0" encoding="UTF-8"?><i><chatserver>chat.bilibili.com</chatserver>` +
	`<d p="0.5,1,25,16777215,1700000000,0,abc,1">第一条弹幕</d>` +
	`<d p="1.5,5,25,16711680,1700000001,0,abd,2">top danmaku</d>` +
	`<d p="0.2,7,25,16711680,1700000001,0,abd,3">[advanced]</d></i>`

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T, name string, prefix bool) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if prefix {
		return append([]byte("000000000"), b...)
	}
	return b
}

// makePCVideo 模拟 PC 客户端的缓存目录
func makePCVideo(t *testing.T, root, cid, series, part, status string, withAudio bool) string {
	dir := filepath.Join(root, cid)
	writeFile(t, filepath.Join(dir, cid+"-1-100048.m4s"), fixture(t, "video_avc.m4s", true))
	if withAudio {
		writeFile(t, filepath.Join(dir, cid+"-1-30280.m4s"), fixture(t, "audio.m4s", true))
	}
	writeFile(t, filepath.Join(dir, ".playurl"), []byte(`{"data":{"dash":{"video":[{"id":100048}],"audio":[{"id":30280}]}}}`))
	info := fmt.Sprintf(`{"groupTitle":%q,"title":%q,"uname":"测试UP主","status":%q,"bvid":"BV1xx411c7mD","aid":170001,"cid":%s,"p":1,"coverUrl":"https://i0.hdslb.com/x.jpg"}`,
		series, part, status, cid)
	writeFile(t, filepath.Join(dir, "videoInfo.json"), []byte(info))
	return dir
}

func mp4box(t *testing.T) string {
	t.Helper()
	p, err := internal.GetMP4Box()
	if err != nil {
		t.Skip("MP4Box not available: ", err)
	}
	return p
}

func TestProbe(t *testing.T) {
	cases := []struct {
		file          string
		kind, codec   string
		width, height int
	}{
		{"video_avc.m4s", "video", "avc", 128, 72},
		{"video_hevc.m4s", "video", "hevc", 128, 72},
		{"audio.m4s", "audio", "aac", 0, 0},
	}
	for _, c := range cases {
		for _, prefix := range []bool{false, true} {
			p := filepath.Join(t.TempDir(), c.file)
			writeFile(t, p, fixture(t, c.file, prefix))
			info, err := Probe(p)
			if err != nil {
				t.Fatalf("%s prefix=%v: %v", c.file, prefix, err)
			}
			if info.Kind != c.kind || info.Codec != c.codec || info.Width != c.width || info.Height != c.height {
				t.Errorf("%s prefix=%v: got %+v", c.file, prefix, info)
			}
			if math.Abs(info.Duration-2) > 0.2 {
				t.Errorf("%s: duration %.2f, want ~2", c.file, info.Duration)
			}
		}
	}
}

func TestScanPC(t *testing.T) {
	root := t.TempDir()
	makePCVideo(t, root, "1332097557", "【4K】世界美景", "P1 中国", "completed", true)
	makePCVideo(t, root, "1332097558", "【4K】世界美景", "P2 冰岛", "downloading", true)
	makePCVideo(t, root, "1332097559", "Other", "No audio", "completed", false)
	writeFile(t, filepath.Join(root, "1332097557", "1332097557.xml"), []byte(danmakuXML))
	out := filepath.Join(root, "output")

	videos, err := Scan(root, ScanOptions{Output: OutputOptions{Dir: out}})
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 3 {
		t.Fatalf("got %d videos, want 3", len(videos))
	}
	byCID := map[string]Video{}
	for _, v := range videos {
		byCID[v.CID] = v
	}
	v := byCID["1332097557"]
	if v.Status != StatusReady || v.Series != "【4K】世界美景" || v.Part != "P1 中国" || v.Uploader != "测试UP主" || v.BVID != "BV1xx411c7mD" {
		t.Errorf("unexpected video: %+v", v)
	}
	if v.Video.Codec != "avc" || v.Audio.Codec != "aac" || v.Video.Width != 128 || v.DanmakuFile == "" {
		t.Errorf("streams not detected: %+v", v)
	}
	if want := filepath.Join(out, "【4K】世界美景-测试UP主", "P1 中国.mp4"); v.Output != want {
		t.Errorf("output %q, want %q", v.Output, want)
	}
	if s := byCID["1332097558"].Status; s != StatusIncomplete {
		t.Errorf("downloading video status %q", s)
	}
	if v := byCID["1332097559"]; v.Status != StatusBroken || v.Note != "missing-audio" {
		t.Errorf("no-audio video: %s %s", v.Status, v.Note)
	}
}

func TestScanAndroid(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "download", "170001", "c_2001")
	writeFile(t, filepath.Join(entry, "entry.json"), []byte(`{"title":"安卓视频","owner_name":"UP","bvid":"BV1A","avid":170001,
		"total_time_milli":2000,"is_completed":true,"total_bytes":100,"downloaded_bytes":100,
		"page_data":{"cid":2001,"page":2,"part":"第二集"}}`))
	writeFile(t, filepath.Join(entry, "danmaku.xml"), []byte(danmakuXML))
	writeFile(t, filepath.Join(entry, "80", "video.m4s"), fixture(t, "video_hevc.m4s", false))
	writeFile(t, filepath.Join(entry, "80", "audio.m4s"), fixture(t, "audio.m4s", false))

	videos, err := Scan(root, ScanOptions{Output: OutputOptions{Dir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 1 {
		t.Fatalf("got %d videos, want 1", len(videos))
	}
	v := videos[0]
	if v.Source != "android" || v.Series != "安卓视频" || v.Part != "第二集" || v.Page != 2 || v.CID != "2001" ||
		v.Status != StatusReady || v.Video.Codec != "hevc" || v.DanmakuFile == "" {
		t.Errorf("unexpected video: %+v", v)
	}
}

func TestOutputPath(t *testing.T) {
	v := Video{Series: `A/B: "C"`, Part: "第1集？", Uploader: "", Page: 3, BVID: "BV1"}
	got := OutputPath(v, OutputOptions{Dir: "/out", Template: "{series}-{uploader}/{page}-{part}"})
	want := filepath.Join("/out", `A／B： ＂C＂`, "3-第1集？.mp4")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := OutputPath(v, OutputOptions{Dir: "/out", Template: "{bvid}"}); got != filepath.Join("/out", "BV1.mp4") {
		t.Errorf("flat template: %q", got)
	}
}

func TestLoadDanmaku(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.xml")
	writeFile(t, p, []byte(danmakuXML))
	c, err := LoadDanmaku(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 2 || c[0].Text != "第一条弹幕" || c[0].Mode != 1 || c[1].Mode != 5 || c[1].Color != 0xff0000 {
		t.Errorf("unexpected comments: %+v", c)
	}
}

func TestConvert(t *testing.T) {
	box := mp4box(t)
	root := t.TempDir()
	dir := makePCVideo(t, root, "1332097557", "世界美景", "P1: 中国", "completed", true)
	writeFile(t, filepath.Join(dir, "1332097557.xml"), []byte(danmakuXML))
	outDir := filepath.Join(t.TempDir(), "out")
	opt := ConvertOptions{Output: OutputOptions{Dir: outDir}, MP4Box: box, Danmaku: true}

	videos, err := Scan(root, ScanOptions{Output: opt.Output})
	if err != nil || len(videos) != 1 {
		t.Fatalf("scan: %v %d", err, len(videos))
	}
	var stages []string
	var last float64
	out, err := Convert(context.Background(), videos[0], opt, func(p Progress) {
		if len(stages) == 0 || stages[len(stages)-1] != p.Stage {
			stages = append(stages, p.Stage)
		}
		last = p.Fraction
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(stages, ",") != "prepare,merge,danmaku" || last != 1 {
		t.Errorf("stages %v last %.2f", stages, last)
	}
	info, err := Probe(out)
	if err != nil || !info.HasVideo || !info.HasAudio || info.Codec != "avc" || math.Abs(info.Duration-2) > 0.2 {
		t.Fatalf("output probe: %+v %v", info, err)
	}
	for _, ext := range []string{".xml", ".ass"} {
		if fileSize(strings.TrimSuffix(out, ".mp4")+ext) == 0 {
			t.Errorf("missing %s next to output", ext)
		}
	}
	if fileExists(filepath.Join(outDir, ".m4s-converter-tmp")) {
		t.Error("temp dir not cleaned up")
	}
	if matches, _ := filepath.Glob(filepath.Join(filepath.Dir(out), "*.partial.mp4")); len(matches) > 0 {
		t.Error("partial file left behind")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 5 {
		t.Errorf("cache dir was modified: %d entries", len(entries))
	}

	videos, _ = Scan(root, ScanOptions{Output: opt.Output})
	if videos[0].Status != StatusConverted {
		t.Errorf("status after convert: %s", videos[0].Status)
	}
	if _, err := Convert(context.Background(), videos[0], opt, nil); !errors.Is(err, ErrExists) {
		t.Errorf("second convert: %v, want ErrExists", err)
	}
	opt.Overwrite = true
	if _, err := Convert(context.Background(), videos[0], opt, nil); err != nil {
		t.Errorf("overwrite: %v", err)
	}
}

func TestQueue(t *testing.T) {
	q := NewQueue(2)
	var mu sync.Mutex
	maxRunning, running := 0, 0
	release := make(chan struct{})
	q.convert = func(ctx context.Context, v Video, opt ConvertOptions, p func(Progress)) (string, error) {
		mu.Lock()
		running++
		if running > maxRunning {
			maxRunning = running
		}
		mu.Unlock()
		defer func() { mu.Lock(); running--; mu.Unlock() }()
		p(Progress{"merge", 0.5})
		select {
		case <-release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		if v.Part == "fail" {
			return "", errors.New("boom")
		}
		return v.Output, nil
	}
	idle := make(chan QueueStats, 4)
	q.OnIdle = func(s QueueStats) { idle <- s }

	opt := ConvertOptions{Output: OutputOptions{Dir: "/out", Template: "{part}"}}
	jobs := q.Add([]Video{
		{ID: "a", Part: "same", Size: 10}, {ID: "b", Part: "same", Size: 10},
		{ID: "c", Part: "fail", Size: 10}, {ID: "d", Part: "cancel me", Size: 10},
	}, opt)
	if jobs[0].Output == jobs[1].Output || !strings.HasSuffix(jobs[1].Output, "same (2).mp4") {
		t.Errorf("duplicate outputs not resolved: %q %q", jobs[0].Output, jobs[1].Output)
	}
	q.Cancel(jobs[3].ID)
	close(release)
	select {
	case s := <-idle:
		if s.Done != 2 || s.Failed != 1 || s.Total != 3 {
			t.Errorf("stats %+v", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("queue did not finish")
	}
	if maxRunning > 2 {
		t.Errorf("ran %d jobs at once, want <= 2", maxRunning)
	}
	states := map[string]JobState{}
	for _, j := range q.Jobs() {
		states[j.VideoID] = j.State
	}
	if states["a"] != JobDone || states["c"] != JobFailed || states["d"] != JobCanceled {
		t.Errorf("states %v", states)
	}
	q.Retry(jobs[2].ID)
	<-idle
	if j := q.Jobs()[2]; j.State != JobFailed || j.Error != "boom" {
		t.Errorf("retry: %+v", j)
	}
}
