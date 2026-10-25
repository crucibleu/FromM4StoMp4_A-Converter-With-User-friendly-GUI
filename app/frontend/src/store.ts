import { computed, reactive, ref, watch } from 'vue'
import { api, events } from './api'
import { errText, langSetting, t } from './i18n'
import type { EffectsLevel, InitData, Job, LibraryItem, QueueTick, Settings, Video } from './types'

export type View = 'library' | 'queue' | 'player' | 'settings'

export const state = reactive({
  ready: false,
  init: null as InitData | null,
  settings: null as Settings | null,
  outputDir: '',
  view: 'library' as View,

  scanning: false,
  scanError: '',
  videos: [] as Video[],
  scannedDir: '',
  selected: new Set<string>(),
  focusedId: '',

  jobs: new Map<string, Job>(),
  tick: null as QueueTick | null,

  library: [] as LibraryItem[],
  libraryLoaded: false,
  nowPlaying: '' as string, // 输出视频路径

  freeBytes: 0,
  playerActive: false, // 正在播放视频时暂停背景动画
  fxAutoReduced: false,
  showOutputDialog: false,
})

export const toasts = ref<{ id: number; text: string; kind: 'info' | 'ok' | 'err' }[]>([])
let toastSeq = 0
export function toast(text: string, kind: 'info' | 'ok' | 'err' = 'info') {
  const id = ++toastSeq
  toasts.value.push({ id, text, kind })
  setTimeout(() => (toasts.value = toasts.value.filter((x) => x.id !== id)), 4200)
}

// 动画级别：设置值 + 系统“减少动态效果” + 自动降级
const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)')
const reducedMotion = ref(prefersReduced.matches)
prefersReduced.addEventListener?.('change', (e) => (reducedMotion.value = e.matches))
export const effects = computed<EffectsLevel>(() => {
  const s = state.settings?.effects ?? 'full'
  if (s === 'off' || reducedMotion.value) return 'off'
  if (s === 'reduced' || state.fxAutoReduced) return 'reduced'
  return 'full'
})

export const videoById = computed(() => new Map(state.videos.map((v) => [v.id, v])))
export const focused = computed(() => videoById.value.get(state.focusedId) ?? null)
export const jobList = computed(() => [...state.jobs.values()])
export const activeJobs = computed(() => jobList.value.filter((j) => j.state === 'running' || j.state === 'queued'))
export const selectedVideos = computed(() => state.videos.filter((v) => state.selected.has(v.id)))

export async function boot() {
  const d = await api.init()
  state.init = d
  state.settings = d.settings
  state.outputDir = d.outputDir
  langSetting.value = d.settings.language
  for (const j of await api.jobs()) state.jobs.set(j.id, j)

  events.onJob((j) => {
    const prev = state.jobs.get(j.id)
    state.jobs.set(j.id, j)
    if ((j.state === 'done' || j.state === 'skipped') && prev?.state !== j.state) markConverted(j)
  })
  events.onTick((tk) => (state.tick = tk))
  events.onIdle((s) => {
    state.tick = null
    if (s.total === 0) return
    window.dispatchEvent(new CustomEvent('m4s:queue-finished', { detail: s }))
    toast(s.failed ? t('queue.finishedFailed', { done: s.done, failed: s.failed }) : t('queue.finished', { done: s.done }), s.failed ? 'err' : 'ok')
    state.libraryLoaded = false
    refreshFree()
  })
  events.onDrop(async (paths) => {
    const p = paths[0]
    if (!p) return
    if (!(await api.isDir(p))) return toast(t('toast.notDir'), 'err')
    await saveSettings({ cacheDir: p })
    toast(t('toast.dropped', { dir: p }))
    state.view = 'library'
    scan()
  })
  state.ready = true
  if (d.settings.cacheDir) scan()
  refreshFree()
}

function markConverted(j: Job) {
  const v = videoById.value.get(j.videoId)
  if (v) {
    v.status = 'converted'
    v.output = j.output
  }
}

export async function saveSettings(patch: Partial<Settings>) {
  if (!state.settings) return
  const next = { ...state.settings, ...patch }
  state.settings = next
  langSetting.value = next.language
  try {
    state.outputDir = await api.saveSettings(next)
  } catch (e) {
    toast(errText(e), 'err')
  }
  if ('outputDir' in patch || 'cacheDir' in patch) {
    state.libraryLoaded = false
    refreshFree()
  }
}

export async function scan() {
  if (state.scanning) return
  state.scanning = true
  state.scanError = ''
  try {
    const r = await api.scan()
    state.videos = r.videos
    state.scannedDir = r.cacheDir
    state.outputDir = r.outputDir
    const ids = new Set(r.videos.map((v) => v.id))
    for (const id of [...state.selected]) if (!ids.has(id)) state.selected.delete(id)
    if (!ids.has(state.focusedId)) state.focusedId = ''
  } catch (e) {
    state.videos = []
    state.scanError = String((e as any)?.message ?? e)
  } finally {
    state.scanning = false
  }
}

export async function pickCacheDir() {
  const p = await api.pickFolder(t('lib.empty.pick'), state.settings?.cacheDir ?? '')
  if (!p) return
  await saveSettings({ cacheDir: p })
  scan()
}

export async function enqueueSelected(ids?: string[]) {
  const list = ids ?? [...state.selected]
  if (!list.length) return
  try {
    const jobs = await api.enqueue(list)
    for (const j of jobs) state.jobs.set(j.id, j)
    if (!ids) state.selected.clear()
    toast(t('toast.added', { n: jobs.length }), 'ok')
    state.view = 'queue'
  } catch (e) {
    toast(errText(e), 'err')
  }
}

export async function loadLibrary(force = false) {
  if (state.libraryLoaded && !force) return
  state.library = await api.library()
  state.libraryLoaded = true
}

export async function refreshFree() {
  const dir = state.outputDir || state.settings?.cacheDir || ''
  state.freeBytes = dir ? await api.diskFree(dir) : 0
}

export function playOutput(path: string) {
  state.nowPlaying = path
  state.view = 'player'
}

// 设置语言改变时更新页面语言属性
watch(langSetting, (l) => (document.documentElement.lang = l === 'en' ? 'en' : 'zh-CN'))
