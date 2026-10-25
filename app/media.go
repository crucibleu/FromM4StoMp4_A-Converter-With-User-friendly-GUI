package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"m4s-converter/engine"
)

// mediaServer 在 127.0.0.1 上为播放器提供文件（支持 Range 拖动进度），
// 只允许访问缓存目录、输出目录和程序自己的缓存目录，并要求随机令牌。
type mediaServer struct {
	base  string // http://127.0.0.1:port
	token string
	cache string // 程序缓存目录（封面、缩略图）

	mu     sync.RWMutex
	roots  []string
	covers map[string]coverSrc // 视频 ID -> 封面
}

type coverSrc struct{ file, url string }

func startMediaServer(cacheDir string) (*mediaServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	tok := make([]byte, 16)
	_, _ = rand.Read(tok)
	m := &mediaServer{
		base:   "http://" + ln.Addr().String(),
		token:  hex.EncodeToString(tok),
		cache:  cacheDir,
		covers: map[string]coverSrc{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/file", m.handleFile)
	mux.HandleFunc("/cover", m.handleCover)
	mux.HandleFunc("/thumb", m.handleThumb)
	srv := &http.Server{Handler: m.guard(mux), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return m, nil
}

func (m *mediaServer) setRoots(roots ...string) {
	var clean []string
	for _, r := range roots {
		if r != "" {
			clean = append(clean, filepath.Clean(r))
		}
	}
	m.mu.Lock()
	m.roots = append(clean, m.cache)
	m.mu.Unlock()
}

func (m *mediaServer) setCovers(videos []engine.Video) {
	covers := make(map[string]coverSrc, len(videos))
	for _, v := range videos {
		covers[v.ID] = coverSrc{v.CoverFile, v.CoverURL}
	}
	m.mu.Lock()
	m.covers = covers
	m.mu.Unlock()
}

func (m *mediaServer) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // 允许页面截取视频帧生成缩略图
		if r.URL.Query().Get("t") != m.token {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *mediaServer) allowed(p string) bool {
	p = filepath.Clean(p)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, root := range m.roots {
		rel, err := filepath.Rel(root, p)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// handleFile 播放文件；m4s 会跳过 B 站的 9 字节文件头，浏览器可以直接预览未转换的视频
func (m *mediaServer) handleFile(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("p")
	if p == "" || !m.allowed(p) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	name := filepath.Base(p)
	var content io.ReadSeeker = f
	switch strings.ToLower(filepath.Ext(p)) {
	case ".m4s":
		off, _ := engine.DataOffset(p)
		content = io.NewSectionReader(f, off, st.Size()-off)
		ct := "video/mp4"
		if r.URL.Query().Get("kind") == "audio" {
			ct = "audio/mp4"
		}
		w.Header().Set("Content-Type", ct)
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	}
	http.ServeContent(w, r, name, st.ModTime(), content)
}

// handleCover 封面：优先使用缓存目录里的图片，否则下载一次后保存在程序缓存目录
func (m *mediaServer) handleCover(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	m.mu.RLock()
	src, ok := m.covers[id]
	m.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	if src.file != "" {
		w.Header().Set("Cache-Control", "max-age=86400")
		http.ServeFile(w, r, src.file)
		return
	}
	if src.url == "" {
		http.NotFound(w, r)
		return
	}
	sum := sha1.Sum([]byte(src.url))
	local := filepath.Join(m.cache, "covers", hex.EncodeToString(sum[:])+".jpg")
	if _, err := os.Stat(local); err != nil {
		if err := download(src.url, local); err != nil {
			http.NotFound(w, r)
			return
		}
	}
	w.Header().Set("Cache-Control", "max-age=86400")
	http.ServeFile(w, r, local)
}

func download(url, dst string) error {
	url = strings.Replace(url, "http://", "https://", 1)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 10<<20)); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, dst)
}

// thumbPath 输出视频的缩略图（由页面截取视频帧后保存）
func (m *mediaServer) thumbPath(mp4 string) string {
	sum := sha1.Sum([]byte(mp4))
	return filepath.Join(m.cache, "thumbs", hex.EncodeToString(sum[:10])+".jpg")
}

func (m *mediaServer) handleThumb(w http.ResponseWriter, r *http.Request) {
	p := m.thumbPath(r.URL.Query().Get("p"))
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, p)
}
