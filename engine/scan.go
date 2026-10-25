package engine

import (
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Status 缓存视频的状态
type Status string

const (
	StatusReady      Status = "ready"      // 可以转换
	StatusConverted  Status = "converted"  // 输出文件已存在
	StatusIncomplete Status = "incomplete" // 客户端还没有缓存完成
	StatusBroken     Status = "broken"     // 缺少音频或视频流
)

// Video 缓存目录中的一个视频（一个分P）
type Video struct {
	ID       string `json:"id"`
	Dir      string `json:"dir"`      // 视频所在的缓存目录
	Source   string `json:"source"`   // pc / android
	Series   string `json:"series"`   // 合集或整个视频的标题
	Part     string `json:"part"`     // 分P标题
	Page     int    `json:"page"`     // 分P序号，未知时为 0
	Uploader string `json:"uploader"` // UP主
	BVID     string `json:"bvid"`
	AID      string `json:"aid"`
	CID      string `json:"cid"`

	VideoFile string    `json:"videoFile"`
	AudioFile string    `json:"audioFile"`
	Size      int64     `json:"size"`     // 音视频总大小
	Duration  float64   `json:"duration"` // 秒
	Video     MediaInfo `json:"video"`
	Audio     MediaInfo `json:"audio"`

	CoverFile   string `json:"coverFile"`   // 本地封面
	CoverURL    string `json:"coverUrl"`    // 网络封面
	DanmakuFile string `json:"danmakuFile"` // 本地弹幕 xml
	Progress    int    `json:"progress"`    // 未缓存完成时的进度百分比，未知为 -1

	Status Status `json:"status"`
	Output string `json:"output"` // 按当前输出设置计算出的输出文件
	Note   string `json:"note"`   // 状态说明（broken 时的原因等）
}

// ScanOptions 扫描参数
type ScanOptions struct {
	Output OutputOptions // 用于计算输出路径并判断是否已转换
}

// Scan 递归查找 root 下所有缓存视频，不修改缓存目录中的任何文件
func Scan(root string, opt ScanOptions) ([]Video, error) {
	root = filepath.Clean(root)
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	outDir := ""
	if opt.Output.Dir != "" {
		outDir = filepath.Clean(opt.Output.Dir)
	}

	var videos []Video
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 无权限的子目录跳过，不中断整个扫描
			if path != root && d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && (strings.HasPrefix(d.Name(), ".") || (outDir != "" && path == outDir)) {
			return filepath.SkipDir
		}
		if v, ok := scanDir(path); ok {
			videos = append(videos, v)
			if v.Source == "android" {
				return filepath.SkipDir // 子目录已经处理
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for i := range videos {
		v := &videos[i]
		if v.Status == StatusReady || v.Status == StatusConverted {
			v.Output = OutputPath(*v, opt.Output)
			if _, err := os.Stat(v.Output); err == nil {
				v.Status = StatusConverted
			} else {
				v.Status = StatusReady
			}
		}
	}
	sort.SliceStable(videos, func(i, j int) bool {
		a, b := videos[i], videos[j]
		if a.Series != b.Series {
			return a.Series < b.Series
		}
		if a.Page != b.Page {
			return a.Page < b.Page
		}
		return a.Part < b.Part
	})
	return videos, nil
}

// scanDir 判断目录是否是一个视频的缓存目录
func scanDir(dir string) (Video, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Video{}, false
	}
	var m4s []string
	var infoFile, entryFile string
	for _, e := range entries {
		name := e.Name()
		switch {
		case e.IsDir():
		case strings.HasSuffix(name, ".m4s"):
			m4s = append(m4s, filepath.Join(dir, name))
		case name == "videoInfo.json" || strings.HasSuffix(name, ".videoInfo"):
			infoFile = filepath.Join(dir, name)
		case name == "entry.json":
			entryFile = filepath.Join(dir, name)
		}
	}
	switch {
	case entryFile != "":
		return scanAndroid(dir, entryFile, entries)
	case len(m4s) > 0:
		return scanPC(dir, infoFile, m4s), true
	}
	return Video{}, false
}

func newVideo(dir, source string) Video {
	sum := sha1.Sum([]byte(dir))
	return Video{ID: hex.EncodeToString(sum[:8]), Dir: dir, Source: source, Progress: -1}
}

// scanPC PC 客户端：<cid>/{*.m4s, videoInfo.json|.videoInfo, .playurl, <cid>.xml}
func scanPC(dir, infoFile string, m4s []string) Video {
	v := newVideo(dir, "pc")
	var js gjson.Result
	if infoFile != "" {
		if b, err := os.ReadFile(infoFile); err == nil {
			js = gjson.ParseBytes(b)
		}
	}
	v.Series = firstNonEmpty(js.Get("groupTitle").String(), js.Get("title").String())
	v.Part = firstNonEmpty(js.Get("page_data.part").String(), js.Get("title").String())
	v.Uploader = firstNonEmpty(js.Get("uname").String(), js.Get("owner_name").String())
	v.BVID = js.Get("bvid").String()
	v.AID = firstNonEmpty(js.Get("aid").String(), js.Get("itemId").String())
	v.CID = firstNonEmpty(js.Get("cid").String(), filepath.Base(dir))
	v.Page = int(js.Get("p").Int())
	v.Duration = js.Get("duration").Float()
	v.CoverURL = js.Get("coverUrl").String()
	if p := js.Get("coverPath").String(); p != "" && fileExists(p) {
		v.CoverFile = p
	}

	pickStreams(&v, m4s, dir)
	v.DanmakuFile = findDanmaku(dir, filepath.Join(dir, filepath.Base(dir)+".xml"))
	if v.CoverFile == "" {
		v.CoverFile = findImage(dir)
	}

	status := firstNonEmpty(js.Get("status").String(), js.Get("page_data.download_title").String())
	if progress := js.Get("progress"); progress.Exists() {
		v.Progress = int(progress.Int())
	}
	if status != "" && status != "completed" && status != "视频已缓存完成" {
		v.Status = StatusIncomplete
	}
	finish(&v)
	return v
}

// scanAndroid 安卓客户端：<avid>/c_<cid>/{entry.json, danmaku.xml, <quality>/{video.m4s, audio.m4s, index.json}}
func scanAndroid(dir, entryFile string, entries []os.DirEntry) (Video, bool) {
	b, err := os.ReadFile(entryFile)
	if err != nil {
		return Video{}, false
	}
	js := gjson.ParseBytes(b)
	var m4s []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		files, _ := os.ReadDir(sub)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".m4s") {
				m4s = append(m4s, filepath.Join(sub, f.Name()))
			}
		}
	}
	if len(m4s) == 0 {
		return Video{}, false // 旧版 blv 格式等暂不支持
	}

	v := newVideo(dir, "android")
	v.Series = js.Get("title").String()
	v.Part = firstNonEmpty(js.Get("page_data.part").String(), js.Get("page_data.download_subtitle").String(), v.Series)
	v.Uploader = js.Get("owner_name").String()
	v.BVID = js.Get("bvid").String()
	v.AID = js.Get("avid").String()
	v.CID = js.Get("page_data.cid").String()
	v.Page = int(js.Get("page_data.page").Int())
	v.Duration = js.Get("total_time_milli").Float() / 1000
	v.CoverURL = js.Get("cover").String()

	pickStreams(&v, m4s, dir)
	v.DanmakuFile = findDanmaku(dir, filepath.Join(dir, "danmaku.xml"))
	v.CoverFile = findImage(dir)

	if total := js.Get("total_bytes").Int(); total > 0 {
		v.Progress = int(js.Get("downloaded_bytes").Int() * 100 / total)
	}
	if c := js.Get("is_completed"); c.Exists() && !c.Bool() {
		v.Status = StatusIncomplete
	} else if t := js.Get("page_data.download_title").String(); t != "" && t != "completed" && t != "视频已缓存完成" {
		v.Status = StatusIncomplete
	}
	finish(&v)
	return v, true
}

// pickStreams 通过读取文件头区分音频和视频，同时兼容 .playurl 中的编号
func pickStreams(v *Video, m4s []string, dir string) {
	for _, p := range m4s {
		info, err := Probe(p)
		if err != nil {
			continue
		}
		size := fileSize(p)
		switch info.Kind {
		case "video":
			if v.VideoFile == "" || size > fileSize(v.VideoFile) {
				v.VideoFile, v.Video = p, info
			}
		case "audio":
			if v.AudioFile == "" || size > fileSize(v.AudioFile) {
				v.AudioFile, v.Audio = p, info
			}
		}
	}
	// 文件头无法读取时（例如还在下载），回退到 .playurl 中的音频编号
	if v.AudioFile == "" || v.VideoFile == "" {
		audioID := gjsonFile(filepath.Join(dir, ".playurl"), "data.dash.audio.0.id", "result.dash.audio.0.id")
		for _, p := range m4s {
			isAudio := audioID != "" && strings.HasSuffix(strings.TrimSuffix(filepath.Base(p), ".m4s"), "-"+audioID) ||
				filepath.Base(p) == "audio.m4s"
			if isAudio && v.AudioFile == "" {
				v.AudioFile = p
			} else if !isAudio && v.VideoFile == "" {
				v.VideoFile = p
			}
		}
	}
	v.Size = fileSize(v.VideoFile) + fileSize(v.AudioFile)
}

func finish(v *Video) {
	if v.Duration == 0 {
		v.Duration = maxf(v.Video.Duration, v.Audio.Duration)
	}
	if v.Series == "" {
		v.Series = v.Part
	}
	if v.Part == "" {
		v.Part = firstNonEmpty(v.Series, v.CID, filepath.Base(v.Dir))
		if v.Series == "" {
			v.Series = v.Part
		}
	}
	if v.Status == StatusIncomplete {
		return
	}
	switch {
	case v.VideoFile == "" && v.AudioFile == "":
		v.Status, v.Note = StatusBroken, "missing-streams"
	case v.VideoFile == "":
		v.Status, v.Note = StatusBroken, "missing-video"
	case v.AudioFile == "":
		v.Status, v.Note = StatusBroken, "missing-audio"
	case v.Video.Codec == "encrypted" || v.Audio.Codec == "encrypted":
		v.Status, v.Note = StatusBroken, "encrypted"
	default:
		v.Status = StatusReady
	}
}

func findDanmaku(dir, preferred string) string {
	if fileSize(preferred) > 0 {
		return preferred
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.xml"))
	for _, m := range matches {
		if fileSize(m) > 0 {
			return m
		}
	}
	return ""
}

func findImage(dir string) string {
	for _, pattern := range []string{"*.jpg", "*.jpeg", "*.png", "*.webp"} {
		if matches, _ := filepath.Glob(filepath.Join(dir, pattern)); len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func gjsonFile(path string, keys ...string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, k := range keys {
		if r := gjson.GetBytes(b, k); r.Exists() {
			return r.String()
		}
	}
	return ""
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func fileSize(p string) int64 {
	if p == "" {
		return 0
	}
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func itoa(i int) string { return strconv.Itoa(i) }
