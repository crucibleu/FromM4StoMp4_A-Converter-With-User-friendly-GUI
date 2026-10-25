package engine

import (
	"compress/flate"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Comment 一条弹幕，供内置播放器渲染
type Comment struct {
	Time  float64 `json:"t"` // 出现时间（秒）
	Mode  int     `json:"m"` // 1 滚动 4 底部 5 顶部 6 逆向
	Size  int     `json:"s"` // 字号，25 为标准
	Color int     `json:"c"` // RGB
	Text  string  `json:"text"`
}

// LoadDanmaku 解析 B 站 xml 弹幕文件，按时间排序；跳过高级弹幕和代码弹幕
func LoadDanmaku(path string) ([]Comment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	dec.Strict = false
	var out []Comment
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if len(out) > 0 {
				break // 文件结尾损坏时保留已解析的部分
			}
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "d" {
			continue
		}
		var p string
		for _, a := range se.Attr {
			if a.Name.Local == "p" {
				p = a.Value
			}
		}
		var text string
		if err := dec.DecodeElement(&text, &se); err != nil {
			continue
		}
		fields := strings.Split(p, ",")
		if len(fields) < 4 || strings.TrimSpace(text) == "" {
			continue
		}
		t, _ := strconv.ParseFloat(fields[0], 64)
		mode, _ := strconv.Atoi(fields[1])
		size, _ := strconv.Atoi(fields[2])
		color, _ := strconv.Atoi(fields[3])
		switch mode {
		case 1, 2, 3:
			mode = 1
		case 4, 5, 6:
		default:
			continue
		}
		out = append(out, Comment{Time: t, Mode: mode, Size: size, Color: color, Text: text})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out, nil
}

func readMaybeDeflate(resp *http.Response) ([]byte, error) {
	var r io.Reader = io.LimitReader(resp.Body, 32<<20)
	if resp.Header.Get("Content-Encoding") == "deflate" {
		fr := flate.NewReader(r)
		defer fr.Close()
		r = fr
	}
	return io.ReadAll(r)
}
