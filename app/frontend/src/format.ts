export function bytes(n: number): string {
  if (!n || n < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n >= 100 || i === 0 ? n.toFixed(0) : n.toFixed(1)} ${units[i]}`
}

export function duration(sec: number): string {
  if (!isFinite(sec) || sec <= 0) return '--:--'
  sec = Math.round(sec)
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  const mm = String(m).padStart(h ? 2 : 2, '0')
  const ss = String(s).padStart(2, '0')
  return h ? `${h}:${mm}:${ss}` : `${mm}:${ss}`
}

export function resLabel(height: number, width = 0): string {
  const p = Math.min(height || 0, width || height || 0) || height
  if (!p) return ''
  if (p >= 2000) return '4K'
  if (p >= 1400) return '2K'
  if (p >= 1000) return '1080P'
  if (p >= 700) return '720P'
  if (p >= 460) return '480P'
  return '360P'
}

export function codecLabel(c: string): string {
  const map: Record<string, string> = { avc: 'AVC', hevc: 'HEVC', av1: 'AV1', aac: 'AAC', ec3: 'E-AC-3', flac: 'FLAC', opus: 'Opus' }
  return map[c] ?? (c ? c.toUpperCase() : '?')
}

export function mime(codec: string): string {
  switch (codec) {
    case 'hevc': return 'video/mp4; codecs="hvc1.1.6.L120.90"'
    case 'av1': return 'video/mp4; codecs="av01.0.08M.08"'
    default: return 'video/mp4; codecs="avc1.640028"'
  }
}

/** 浏览器内核能否解码这种视频编码 */
export function canPlay(codec: string): boolean {
  const v = document.createElement('video')
  return v.canPlayType(mime(codec)) !== ''
}

/** 根据文字生成稳定的封面渐变色 */
export function hueOf(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return h % 360
}

export function placeholderBg(seed: string): string {
  const h = hueOf(seed)
  return `linear-gradient(135deg, hsl(${h} 45% 22%), hsl(${(h + 50) % 360} 55% 38%) 60%, hsl(${(h + 90) % 360} 60% 60%))`
}

export function basename(p: string): string {
  const parts = p.split(/[\\/]/)
  return parts[parts.length - 1] || p
}
