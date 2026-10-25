package engine

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// bilibiliPrefix PC 客户端缓存的 m4s 文件开头会多出 9 个字符 '0'
var bilibiliPrefix = []byte("000000000")

// MediaInfo 从 MP4/m4s 文件头读取的流信息，不需要解码
type MediaInfo struct {
	Kind     string  `json:"kind"`     // video / audio / ""
	Codec    string  `json:"codec"`    // avc / hevc / av1 / aac / ec3 / flac ...
	Width    int     `json:"width"`    // 仅视频
	Height   int     `json:"height"`   // 仅视频
	Duration float64 `json:"duration"` // 秒
	HasVideo bool    `json:"hasVideo"`
	HasAudio bool    `json:"hasAudio"`
}

// DataOffset 返回媒体数据的起始位置：带 B 站前缀的 m4s 为 9，否则为 0
func DataOffset(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	head := make([]byte, len(bilibiliPrefix))
	if _, err := io.ReadFull(f, head); err != nil {
		return 0, nil
	}
	if string(head) == string(bilibiliPrefix) {
		return int64(len(bilibiliPrefix)), nil
	}
	return 0, nil
}

// Probe 读取文件的 moov/sidx 盒子，获得编码、分辨率和时长
func Probe(path string) (MediaInfo, error) {
	var info MediaInfo
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return info, err
	}
	off, err := DataOffset(path)
	if err != nil {
		return info, err
	}

	var mvhdDur, sidxDur float64
	var tracks []trackInfo
	pos, end := off, st.Size()
	for pos+8 <= end {
		typ, hdr, size, err := readBoxHeader(f, pos, end)
		if err != nil {
			break
		}
		switch typ {
		case "moov":
			if size > 64<<20 {
				return info, errors.New("moov box too large")
			}
			buf := make([]byte, size-hdr)
			if _, err := f.ReadAt(buf, pos+hdr); err != nil {
				return info, err
			}
			mvhdDur, tracks = parseMoov(buf)
		case "sidx":
			buf := make([]byte, min64(size-hdr, 1<<20))
			if _, err := f.ReadAt(buf, pos+hdr); err == nil {
				sidxDur = parseSidx(buf)
			}
		case "moof":
			// DASH 分片开始，后面不会再有 moov/sidx
			if len(tracks) > 0 {
				pos = end
				continue
			}
		}
		pos += size
	}
	if len(tracks) == 0 {
		return info, errors.New("no media tracks found")
	}

	for _, t := range tracks {
		switch t.handler {
		case "vide":
			info.HasVideo = true
			if info.Kind == "" || info.Kind == "audio" {
				info.Kind, info.Codec, info.Width, info.Height = "video", t.codec, t.width, t.height
			}
		case "soun":
			info.HasAudio = true
			if info.Kind == "" {
				info.Kind, info.Codec = "audio", t.codec
			}
		}
		if t.duration > info.Duration {
			info.Duration = t.duration
		}
	}
	if sidxDur > 0 {
		info.Duration = sidxDur
	} else if info.Duration == 0 {
		info.Duration = mvhdDur
	}
	return info, nil
}

type trackInfo struct {
	handler       string
	codec         string
	width, height int
	duration      float64
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// readBoxHeader 返回盒子类型、头长度和总长度
func readBoxHeader(r io.ReaderAt, pos, end int64) (string, int64, int64, error) {
	var h [16]byte
	if _, err := r.ReadAt(h[:8], pos); err != nil {
		return "", 0, 0, err
	}
	size := int64(binary.BigEndian.Uint32(h[:4]))
	typ := string(h[4:8])
	hdr := int64(8)
	switch size {
	case 0:
		size = end - pos
	case 1:
		if _, err := r.ReadAt(h[8:16], pos+8); err != nil {
			return "", 0, 0, err
		}
		size = int64(binary.BigEndian.Uint64(h[8:16]))
		hdr = 16
	}
	if size < hdr || pos+size > end+8 { // 末尾盒子被截断时容忍
		return "", 0, 0, errors.New("invalid box size")
	}
	return typ, hdr, size, nil
}

// children 遍历内存中的子盒子
func children(b []byte, fn func(typ string, body []byte)) {
	for len(b) >= 8 {
		size := int(binary.BigEndian.Uint32(b[:4]))
		typ := string(b[4:8])
		hdr := 8
		if size == 1 && len(b) >= 16 {
			size = int(binary.BigEndian.Uint64(b[8:16]))
			hdr = 16
		} else if size == 0 {
			size = len(b)
		}
		if size < hdr || size > len(b) {
			return
		}
		fn(typ, b[hdr:size])
		b = b[size:]
	}
}

func parseMoov(b []byte) (mvhdDur float64, tracks []trackInfo) {
	children(b, func(typ string, body []byte) {
		switch typ {
		case "mvhd":
			mvhdDur = parseTimeHeader(body)
		case "trak":
			tracks = append(tracks, parseTrak(body))
		}
	})
	return
}

// parseTimeHeader 解析 mvhd/mdhd 的 timescale 与 duration
func parseTimeHeader(b []byte) float64 {
	if len(b) < 4 {
		return 0
	}
	var scale, dur uint64
	if b[0] == 1 {
		if len(b) < 32 {
			return 0
		}
		scale = uint64(binary.BigEndian.Uint32(b[20:24]))
		dur = binary.BigEndian.Uint64(b[24:32])
	} else {
		if len(b) < 20 {
			return 0
		}
		scale = uint64(binary.BigEndian.Uint32(b[12:16]))
		dur = uint64(binary.BigEndian.Uint32(b[16:20]))
	}
	if scale == 0 || dur == 0 || dur == 0xffffffff || dur == 0xffffffffffffffff {
		return 0
	}
	return float64(dur) / float64(scale)
}

func parseTrak(b []byte) trackInfo {
	var t trackInfo
	children(b, func(typ string, body []byte) {
		switch typ {
		case "tkhd":
			// 宽高为 16.16 定点数，位于盒子末尾
			if len(body) >= 8 {
				t.width = int(binary.BigEndian.Uint32(body[len(body)-8:]) >> 16)
				t.height = int(binary.BigEndian.Uint32(body[len(body)-4:]) >> 16)
			}
		case "mdia":
			children(body, func(typ string, body []byte) {
				switch typ {
				case "mdhd":
					t.duration = parseTimeHeader(body)
				case "hdlr":
					if len(body) >= 12 {
						t.handler = string(body[8:12])
					}
				case "minf":
					children(body, func(typ string, body []byte) {
						if typ != "stbl" {
							return
						}
						children(body, func(typ string, body []byte) {
							if typ == "stsd" && len(body) >= 16 {
								t.codec = codecName(string(body[12:16]))
								if t.codec == "encv" || t.codec == "enca" {
									t.codec = "encrypted"
								}
							}
						})
					})
				}
			})
		}
	})
	return t
}

func codecName(fourcc string) string {
	switch fourcc {
	case "avc1", "avc3":
		return "avc"
	case "hev1", "hvc1":
		return "hevc"
	case "av01":
		return "av1"
	case "mp4a":
		return "aac"
	case "ec-3":
		return "ec3"
	case "ac-3":
		return "ac3"
	case "fLaC":
		return "flac"
	case "Opus":
		return "opus"
	}
	return fourcc
}

// parseSidx 累加所有分段时长
func parseSidx(b []byte) float64 {
	if len(b) < 12 {
		return 0
	}
	version := b[0]
	scale := binary.BigEndian.Uint32(b[8:12])
	p := 12
	if version == 0 {
		p += 8
	} else {
		p += 16
	}
	if scale == 0 || len(b) < p+4 {
		return 0
	}
	count := int(binary.BigEndian.Uint16(b[p+2 : p+4]))
	p += 4
	var total uint64
	for i := 0; i < count && p+12 <= len(b); i++ {
		total += uint64(binary.BigEndian.Uint32(b[p+4 : p+8]))
		p += 12
	}
	return float64(total) / float64(scale)
}
