package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"m4s-converter/common"
	"m4s-converter/engine"
)

// Settings 用户设置，保存在系统配置目录 m4s-converter/settings.json
type Settings struct {
	CacheDir        string `json:"cacheDir"`
	OutputDir       string `json:"outputDir"` // 为空时使用 <缓存目录>/output，与命令行版本一致
	Template        string `json:"template"`
	Danmaku         bool   `json:"danmaku"`
	DownloadDanmaku bool   `json:"downloadDanmaku"`
	Overwrite       bool   `json:"overwrite"`
	Workers         int    `json:"workers"`
	Effects         string `json:"effects"`  // full / reduced / off
	Language        string `json:"language"` // auto / zh / en
	MP4Box          string `json:"mp4box"`   // 自定义 MP4Box，为空时使用内置版本
	AcceptedTerms   bool   `json:"acceptedTerms"`
	DanmakuOverlay  bool   `json:"danmakuOverlay"` // 播放器默认显示弹幕
	DanmakuOpacity  int    `json:"danmakuOpacity"` // 10-100
	DanmakuDensity  int    `json:"danmakuDensity"` // 10-100
	DanmakuSize     int    `json:"danmakuSize"`    // 50-150（百分比）
	Volume          int    `json:"volume"`         // 0-100
}

// PlayState 播放进度
type PlayState struct {
	Position float64 `json:"position"`
	Watched  bool    `json:"watched"`
}

type store struct {
	mu       sync.Mutex
	dir      string
	settings Settings
	play     map[string]PlayState
}

func defaultSettings() Settings {
	dirs := detectCacheDirs()
	cache := common.DefaultCachePath()
	if len(dirs) > 0 {
		cache = dirs[0]
	}
	return Settings{
		CacheDir: cache, Template: engine.DefaultTemplate, Danmaku: true, DownloadDanmaku: true,
		Workers: 2, Effects: "full", Language: "auto", DanmakuOverlay: true,
		DanmakuOpacity: 85, DanmakuDensity: 70, DanmakuSize: 100, Volume: 80,
	}
}

func openStore() *store {
	s := &store{settings: defaultSettings(), play: map[string]PlayState{}}
	if dir, err := os.UserConfigDir(); err == nil {
		s.dir = filepath.Join(dir, "m4s-converter")
		if b, err := os.ReadFile(filepath.Join(s.dir, "settings.json")); err == nil {
			_ = json.Unmarshal(b, &s.settings)
		}
		if b, err := os.ReadFile(filepath.Join(s.dir, "playback.json")); err == nil {
			_ = json.Unmarshal(b, &s.play)
		}
	}
	if s.settings.Template == "" {
		s.settings.Template = engine.DefaultTemplate
	}
	return s
}

func (s *store) get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *store) set(v Settings) error {
	s.mu.Lock()
	s.settings = v
	s.mu.Unlock()
	return s.save("settings.json", v)
}

func (s *store) playState(path string) PlayState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.play[path]
}

func (s *store) setPlay(path string, st PlayState) error {
	s.mu.Lock()
	s.play[path] = st
	cp := make(map[string]PlayState, len(s.play))
	for k, v := range s.play {
		cp[k] = v
	}
	s.mu.Unlock()
	return s.save("playback.json", cp)
}

func (s *store) save(name string, v any) error {
	if s.dir == "" {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, name+".tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, name))
}

// outputDir 实际使用的输出目录
func (st Settings) outputDir() string {
	if strings.TrimSpace(st.OutputDir) != "" {
		return st.OutputDir
	}
	if st.CacheDir == "" {
		return ""
	}
	return filepath.Join(st.CacheDir, "output")
}

// detectCacheDirs 返回本机存在的常见 B 站缓存目录
func detectCacheDirs() []string {
	home, _ := os.UserHomeDir()
	var cands []string
	if home != "" {
		cands = append(cands, filepath.Join(home, "Videos", "bilibili"))
		if runtime.GOOS == "linux" {
			if out, err := exec.Command("xdg-user-dir", "VIDEOS").Output(); err == nil {
				if p := strings.TrimSpace(string(out)); p != "" && p != home {
					cands = append(cands, filepath.Join(p, "bilibili"))
				}
			}
			cands = append(cands, filepath.Join(home, "视频", "bilibili"))
		}
	}
	if runtime.GOOS == "windows" {
		for _, d := range []string{"D:", "E:"} {
			cands = append(cands, filepath.Join(d+`\`, "bilibili"), filepath.Join(d+`\`, "Videos", "bilibili"))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.IsDir() && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}
