// 与 Go 端结构体对应的类型

export interface MediaInfo {
  kind: string
  codec: string
  width: number
  height: number
  duration: number
  hasVideo: boolean
  hasAudio: boolean
}

export type VideoStatus = 'ready' | 'converted' | 'incomplete' | 'broken'

export interface Video {
  id: string
  dir: string
  source: string
  series: string
  part: string
  page: number
  uploader: string
  bvid: string
  aid: string
  cid: string
  videoFile: string
  audioFile: string
  size: number
  duration: number
  video: MediaInfo
  audio: MediaInfo
  coverFile: string
  coverUrl: string
  danmakuFile: string
  progress: number
  status: VideoStatus
  output: string
  note: string
}

export interface Settings {
  cacheDir: string
  outputDir: string
  template: string
  danmaku: boolean
  downloadDanmaku: boolean
  overwrite: boolean
  workers: number
  effects: EffectsLevel
  language: 'auto' | 'zh' | 'en'
  mp4box: string
  acceptedTerms: boolean
  danmakuOverlay: boolean
  danmakuOpacity: number
  danmakuDensity: number
  danmakuSize: number
  volume: number
}

export type EffectsLevel = 'full' | 'reduced' | 'off'

export interface InitData {
  settings: Settings
  outputDir: string
  version: string
  platform: string
  mediaBase: string
  mediaToken: string
  detectedDirs: string[]
  defaultTemplate: string
}

export interface ScanResult {
  cacheDir: string
  outputDir: string
  videos: Video[]
  elapsed: number
}

export type JobState = 'queued' | 'running' | 'done' | 'skipped' | 'failed' | 'canceled'

export interface Job {
  id: string
  videoId: string
  series: string
  part: string
  size: number
  output: string
  state: JobState
  stage: string
  progress: number
  error: string
  started: number
  finished: number
}

export interface QueueStats {
  total: number
  queued: number
  running: number
  done: number
  failed: number
  bytes: number
  finished: number
}

export interface QueueTick {
  stats: QueueStats
  speed: number
}

export interface PlayState {
  position: number
  watched: boolean
}

export interface LibraryItem {
  path: string
  name: string
  folder: string
  size: number
  modified: number
  info: MediaInfo
  hasDanmaku: boolean
  play: PlayState
}

export interface Comment {
  t: number
  m: number
  s: number
  c: number
  text: string
}
