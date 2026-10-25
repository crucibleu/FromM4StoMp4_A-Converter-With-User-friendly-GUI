package engine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"m4s-converter/conver"
	"m4s-converter/internal"
)

// ConvertOptions 单个视频的转换参数
type ConvertOptions struct {
	Output          OutputOptions
	MP4Box          string // MP4Box 可执行文件路径
	Danmaku         bool   // 生成 .xml 与 .ass 弹幕文件
	DownloadDanmaku bool   // 本地没有弹幕时从网络下载
	Overwrite       bool   // 覆盖已存在的输出文件
	TempDir         string // 去除文件头的临时文件存放位置，为空时使用输出目录
}

// Progress 转换进度
type Progress struct {
	Stage    string  `json:"stage"`    // prepare / merge / danmaku
	Fraction float64 `json:"fraction"` // 0..1
}

// ErrExists 输出文件已存在且未选择覆盖
var ErrExists = errors.New("output file already exists")

// Convert 把一个缓存视频合成为 mp4，返回输出文件路径
func Convert(ctx context.Context, v Video, opt ConvertOptions, onProgress func(Progress)) (string, error) {
	if onProgress == nil {
		onProgress = func(Progress) {}
	}
	if v.VideoFile == "" || v.AudioFile == "" {
		return "", errors.New("missing audio or video stream")
	}
	out := v.Output
	if out == "" {
		out = OutputPath(v, opt.Output)
	}
	if !opt.Overwrite && fileExists(out) {
		return out, ErrExists
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	tmpDir := opt.TempDir
	if tmpDir == "" {
		tmpDir = filepath.Join(opt.Output.Dir, ".m4s-converter-tmp")
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}

	// 1. 去掉 B 站 9 字节文件头（没有文件头的直接使用原文件）
	total := float64(fileSize(v.VideoFile) + fileSize(v.AudioFile))
	var copied float64
	report := func(n int64) {
		copied += float64(n)
		if total > 0 {
			onProgress(Progress{"prepare", copied / total})
		}
	}
	var cleanup []string
	defer func() {
		for _, p := range cleanup {
			_ = os.Remove(p)
		}
		_ = os.Remove(tmpDir) // 目录为空时才会删除
	}()
	videoIn, err := stripPrefix(ctx, v.VideoFile, filepath.Join(tmpDir, v.ID+"-video.mp4"), report)
	if err != nil {
		return "", err
	}
	if videoIn != v.VideoFile {
		cleanup = append(cleanup, videoIn)
	}
	audioIn, err := stripPrefix(ctx, v.AudioFile, filepath.Join(tmpDir, v.ID+"-audio.mp4"), report)
	if err != nil {
		return "", err
	}
	if audioIn != v.AudioFile {
		cleanup = append(cleanup, audioIn)
	}

	// 2. MP4Box 合成到临时文件，成功后再改名，避免留下半个文件
	partial := strings.TrimSuffix(out, ".mp4") + ".partial.mp4"
	cleanup = append(cleanup, partial)
	if err := mux(ctx, opt.MP4Box, videoIn, audioIn, partial, v, func(f float64) {
		onProgress(Progress{"merge", f})
	}); err != nil {
		return "", err
	}
	if opt.Overwrite {
		_ = os.Remove(out)
	}
	if err := os.Rename(partial, out); err != nil {
		return "", err
	}
	// 3. 弹幕：保存原始 xml（内置播放器使用）并生成 ass（其他播放器使用）
	if opt.Danmaku {
		onProgress(Progress{"danmaku", 0})
		_ = writeDanmaku(ctx, v, out, opt.DownloadDanmaku)
	}
	onProgress(Progress{"danmaku", 1})
	return out, nil
}

// stripPrefix 如果文件带有 B 站前缀，复制一份去掉前缀的文件
func stripPrefix(ctx context.Context, src, dst string, report func(int64)) (string, error) {
	off, err := DataOffset(src)
	if err != nil {
		return "", err
	}
	if off == 0 {
		report(fileSize(src))
		return src, nil
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if _, err := in.Seek(off, io.SeekStart); err != nil {
		return "", err
	}
	outF, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 4<<20)
	for {
		if err := ctx.Err(); err != nil {
			outF.Close()
			_ = os.Remove(dst)
			return "", err
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := outF.Write(buf[:n]); werr != nil {
				outF.Close()
				_ = os.Remove(dst)
				return "", werr
			}
			report(int64(n))
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			outF.Close()
			_ = os.Remove(dst)
			return "", rerr
		}
	}
	if err := outF.Close(); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	return dst, nil
}

var progressRe = regexp.MustCompile(`^(.*?)[:|].*\((\d+)/(\d+)\)`)

// mux 调用 MP4Box 合成音视频并解析其进度输出
func mux(ctx context.Context, mp4box, video, audio, out string, v Video, onProgress func(float64)) error {
	tag := func(s string) string { return strings.NewReplacer(":", "：", "=", "＝").Replace(s) }
	tags := []string{"title=" + tag(v.Part), "artist=" + tag(v.Uploader), "album=" + tag(v.Series)}
	if v.BVID != "" {
		tags = append(tags, "comment="+tag(v.BVID))
	}
	args := []string{
		"-charset", "utf8",
		"-add", video + "#video",
		"-add", audio + "#audio",
		"-itags", strings.Join(tags, ":"),
		"-new", out,
	}
	cmd := internal.HideWindow(exec.CommandContext(ctx, mp4box, args...))
	// stdout 由 exec 的协程写入，stderr 由下面的循环读取，两者使用各自的缓冲区
	var stdout, logBuf bytes.Buffer
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start MP4Box: %w", err)
	}

	// MP4Box 依次输出：导入视频、导入音频、写文件，每段都有 (x/100)
	const phases = 3
	phase, lastLabel := 0, ""
	sc := bufio.NewScanner(stderr)
	sc.Split(splitCRLF)
	for sc.Scan() {
		line := sc.Text()
		m := progressRe.FindStringSubmatch(line)
		if m == nil {
			if strings.TrimSpace(line) != "" && logBuf.Len() < 64<<10 {
				logBuf.WriteString(line + "\n")
			}
			continue
		}
		label := strings.TrimSpace(m[1])
		if lastLabel != "" && label != lastLabel && phase < phases-1 {
			phase++
		}
		lastLabel = label
		var done, all float64
		fmt.Sscan(m[2], &done)
		fmt.Sscan(m[3], &all)
		if all > 0 {
			onProgress((float64(phase) + done/all) / phases)
		}
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg := strings.TrimSpace(logBuf.String() + stdout.String())
		if len(msg) > 600 {
			msg = msg[len(msg)-600:]
		}
		return fmt.Errorf("MP4Box failed: %v: %s", err, msg)
	}
	onProgress(1)
	return nil
}

func splitCRLF(data []byte, atEOF bool) (int, []byte, error) {
	for i, b := range data {
		if b == '\r' || b == '\n' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// DanmakuPathFor 输出视频对应的弹幕 xml 路径
func DanmakuPathFor(mp4 string) string {
	return strings.TrimSuffix(mp4, filepath.Ext(mp4)) + ".xml"
}

func writeDanmaku(ctx context.Context, v Video, mp4 string, download bool) error {
	xmlOut := DanmakuPathFor(mp4)
	assOut := strings.TrimSuffix(mp4, filepath.Ext(mp4)) + ".ass"
	switch {
	case v.DanmakuFile != "":
		if err := copyFile(v.DanmakuFile, xmlOut); err != nil {
			return err
		}
	case download && v.CID != "":
		if err := downloadDanmaku(ctx, v.CID, xmlOut); err != nil {
			return err
		}
	default:
		return nil
	}
	setting := conver.DefaultSetting
	if v.Video.Width > 0 && v.Video.Height > 0 {
		setting.Width, setting.Height = v.Video.Width, v.Video.Height
	}
	return conver.XmlToAssFile(xmlOut, assOut, setting)
}

func downloadDanmaku(ctx context.Context, cid, dst string) error {
	urls := []string{
		"https://comment.bilibili.com/" + cid + ".xml",
		"https://api.bilibili.com/x/v1/dm/list.so?oid=" + cid,
	}
	client := &http.Client{Timeout: 8 * time.Second}
	var lastErr error
	for _, u := range urls {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := readMaybeDeflate(resp)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("<i>")) && !bytes.Contains(body, []byte("<i ")) {
			lastErr = fmt.Errorf("danmaku download failed: %s", resp.Status)
			continue
		}
		return os.WriteFile(dst, body, 0o644)
	}
	return lastErr
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
