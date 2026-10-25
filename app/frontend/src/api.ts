// 对 Wails 生成的绑定做一层类型封装
import * as Go from '../wailsjs/go/main/App'
import { EventsOn, OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime'
import type {
  Comment, InitData, Job, LibraryItem, QueueStats, QueueTick, ScanResult, Settings,
} from './types'

let mediaBase = ''
let mediaToken = ''

export const api = {
  init: async (): Promise<InitData> => {
    const d = (await Go.Init()) as unknown as InitData
    mediaBase = d.mediaBase
    mediaToken = d.mediaToken
    return d
  },
  saveSettings: (s: Settings) => Go.SaveSettings(s as any) as Promise<string>,
  pickFolder: (title: string, start = '') => Go.PickFolder(title, start),
  pickFile: (title: string) => Go.PickFile(title),
  scan: () => Go.Scan() as unknown as Promise<ScanResult>,
  outputExample: (videoId: string, dir: string, template: string) => Go.OutputExample(videoId, dir, template),
  enqueue: (ids: string[]) => Go.Enqueue(ids) as unknown as Promise<Job[]>,
  jobs: () => Go.Jobs() as unknown as Promise<Job[]>,
  cancelJob: (id: string) => Go.CancelJob(id),
  cancelAll: () => Go.CancelAll(),
  retryJob: (id: string) => Go.RetryJob(id),
  clearFinished: () => Go.ClearFinished() as unknown as Promise<Job[]>,
  library: () => Go.Library() as unknown as Promise<LibraryItem[]>,
  danmaku: (mp4: string) => Go.Danmaku(mp4) as unknown as Promise<Comment[]>,
  cacheDanmaku: (id: string) => Go.CacheDanmaku(id) as unknown as Promise<Comment[]>,
  savePlayState: (path: string, position: number, watched: boolean) => Go.SavePlayState(path, position, watched),
  saveThumb: (mp4: string, dataUrl: string) => Go.SaveThumb(mp4, dataUrl),
  diskFree: (p: string) => Go.DiskFree(p),
  openPath: (p: string) => Go.OpenPath(p),
  revealPath: (p: string) => Go.RevealPath(p),
  isDir: (p: string) => Go.IsDir(p),
}

export const events = {
  onJob: (fn: (j: Job) => void) => EventsOn('job', fn),
  onTick: (fn: (t: QueueTick) => void) => EventsOn('queue:tick', fn),
  onIdle: (fn: (s: QueueStats) => void) => EventsOn('queue:idle', fn),
  onDrop: (fn: (paths: string[]) => void) => {
    OnFileDrop((_x: number, _y: number, paths: string[]) => fn(paths), false)
    return () => OnFileDropOff()
  },
}

function url(path: string, params: Record<string, string>) {
  const q = new URLSearchParams({ ...params, t: mediaToken })
  return `${mediaBase}${path}?${q.toString()}`
}

/** 播放本地文件（m4s 会自动跳过 B 站文件头） */
export const fileURL = (p: string, kind: 'video' | 'audio' = 'video') => url('/file', { p, kind })
export const coverURL = (id: string) => url('/cover', { id })
export const thumbURL = (mp4: string) => url('/thumb', { p: mp4 })
