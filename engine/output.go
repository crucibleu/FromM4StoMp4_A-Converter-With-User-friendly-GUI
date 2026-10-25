package engine

import (
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DefaultTemplate 与命令行版本的输出结构一致：<合集>-<UP主>/<分P>.mp4
const DefaultTemplate = "{series}-{uploader}/{part}"

// OutputOptions 输出位置与命名
type OutputOptions struct {
	Dir      string `json:"dir"`
	Template string `json:"template"` // 支持 {series} {part} {uploader} {page} {bvid} {cid} {title}，"/" 表示子目录
}

// OutputPath 计算视频的输出文件路径（不含冲突处理）
func OutputPath(v Video, opt OutputOptions) string {
	tpl := strings.TrimSpace(opt.Template)
	if tpl == "" {
		tpl = DefaultTemplate
	}
	page := ""
	if v.Page > 0 {
		page = itoa(v.Page)
	}
	values := map[string]string{
		"series":   v.Series,
		"part":     v.Part,
		"title":    v.Part,
		"uploader": v.Uploader,
		"page":     page,
		"bvid":     v.BVID,
		"cid":      v.CID,
	}
	tpl = strings.ReplaceAll(tpl, "\\", "/")
	var parts []string
	for _, seg := range strings.Split(tpl, "/") {
		for k, val := range values {
			seg = strings.ReplaceAll(seg, "{"+k+"}", SanitizeName(val))
		}
		seg = cleanSegment(seg)
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	if len(parts) == 0 {
		parts = []string{SanitizeName(firstNonEmpty(v.Part, v.CID, v.ID))}
	}
	parts[len(parts)-1] += ".mp4"
	return filepath.Join(append([]string{opt.Dir}, parts...)...)
}

// SanitizeName 将文件名中 Windows/Linux 不允许的字符替换为外观相近的全角字符
func SanitizeName(s string) string {
	r := strings.NewReplacer(
		"/", "／", "\\", "＼", ":", "：", "*", "＊", "?", "？",
		"\"", "＂", "<", "＜", ">", "＞", "|", "｜",
	)
	s = r.Replace(s)
	s = strings.Map(func(c rune) rune {
		if c < 0x20 || c == 0x7f {
			return -1
		}
		return c
	}, s)
	s = strings.TrimSpace(s)
	// 单个文件名控制在 120 个字符以内，避免超出路径长度限制
	if utf8.RuneCountInString(s) > 120 {
		s = string([]rune(s)[:120])
	}
	return s
}

// cleanSegment 去掉占位符为空时留下的多余连接符，以及 Windows 不允许的结尾点和空格
func cleanSegment(s string) string {
	for _, pair := range []string{"--", "__", "  "} {
		for strings.Contains(s, pair) {
			s = strings.ReplaceAll(s, pair, pair[:1])
		}
	}
	s = strings.Trim(s, " -_.")
	if s == "." || s == ".." {
		return ""
	}
	return s
}
