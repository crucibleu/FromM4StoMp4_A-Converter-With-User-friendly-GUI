<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { WindowFullscreen, WindowUnfullscreen } from '../../wailsjs/runtime/runtime'
import Icon from '../components/Icon.vue'
import DanmakuLayer from '../components/DanmakuLayer.vue'
import { api, fileURL, thumbURL } from '../api'
import { loadLibrary, saveSettings, state } from '../store'
import { t } from '../i18n'
import { bytes, canPlay, codecLabel, duration, placeholderBg, resLabel } from '../format'
import type { Comment, LibraryItem } from '../types'

const videoEl = ref<HTMLVideoElement | null>(null)
const stageEl = ref<HTMLElement | null>(null)
const query = ref('')
const comments = ref<Comment[]>([])
const ui = reactive({ playing: false, time: 0, dur: 0, buffered: 0, failed: false, full: false, idle: false, rate: 1, muted: false, showDm: false })
const thumbs = reactive<Record<string, string>>({})
let idleTimer = 0
let saveTimer = 0

const items = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? state.library.filter((i) => `${i.folder} ${i.name}`.toLowerCase().includes(q)) : state.library
})
const groups = computed(() => {
  const m = new Map<string, LibraryItem[]>()
  for (const i of items.value) {
    if (!m.has(i.folder)) m.set(i.folder, [])
    m.get(i.folder)!.push(i)
  }
  return [...m.entries()]
})
const current = computed(() => state.library.find((i) => i.path === state.nowPlaying) ?? null)
const s = computed(() => state.settings!)

async function refresh() {
  await loadLibrary(true)
  if (!current.value && state.library.length) {
    state.nowPlaying = (state.library.find((i) => !i.play.watched) ?? state.library[0]).path
  }
  queueThumbs()
}

watch(() => state.nowPlaying, async () => {
  ui.failed = false
  comments.value = []
  const c = current.value
  if (!c) return
  if (c.info.codec && !canPlay(c.info.codec)) ui.failed = true
  if (c.hasDanmaku) comments.value = await api.danmaku(c.path)
})

function onLoaded() {
  const v = videoEl.value, c = current.value
  if (!v || !c) return
  ui.dur = v.duration
  v.volume = s.value.volume / 100
  v.playbackRate = ui.rate
  const pos = c.play.position
  if (!c.play.watched && pos > 5 && pos < v.duration - 10) v.currentTime = pos
  v.play().catch(() => {})
}
function onTime() {
  const v = videoEl.value
  if (!v) return
  ui.time = v.currentTime
  if (v.buffered.length) ui.buffered = v.buffered.end(v.buffered.length - 1)
  if (Date.now() - saveTimer > 5000) persist()
}
function persist() {
  const v = videoEl.value, c = current.value
  if (!v || !c || !isFinite(v.duration)) return
  saveTimer = Date.now()
  const watched = v.currentTime > v.duration * 0.95
  c.play = { position: v.currentTime, watched: c.play.watched || watched }
  api.savePlayState(c.path, v.currentTime, c.play.watched)
}
function onPlayState() {
  const v = videoEl.value
  ui.playing = !!v && !v.paused
  state.playerActive = ui.playing
  if (!ui.playing) persist()
  poke()
}
function next() {
  const list = items.value
  const i = list.findIndex((x) => x.path === state.nowPlaying)
  if (i >= 0 && i < list.length - 1) state.nowPlaying = list[i + 1].path
}
function onEnded() {
  persist()
  next()
}

function toggle() {
  const v = videoEl.value
  if (!v) return
  v.paused ? v.play().catch(() => {}) : v.pause()
}
function seek(ev: MouseEvent) {
  const v = videoEl.value
  const el = ev.currentTarget as HTMLElement
  if (!v || !ui.dur) return
  const r = el.getBoundingClientRect()
  v.currentTime = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width)) * ui.dur
}
function setVolume(val: number) {
  const v = videoEl.value
  if (v) { v.volume = val / 100; v.muted = false; ui.muted = false }
  saveSettings({ volume: val })
}
function toggleMute() {
  const v = videoEl.value
  if (v) { v.muted = !v.muted; ui.muted = v.muted }
}
const rates = [0.5, 0.75, 1, 1.25, 1.5, 2]
function cycleRate() {
  ui.rate = rates[(rates.indexOf(ui.rate) + 1) % rates.length]
  if (videoEl.value) videoEl.value.playbackRate = ui.rate
}
function toggleFull() {
  ui.full = !ui.full
  ui.full ? WindowFullscreen() : WindowUnfullscreen()
}
const pipOk = 'pictureInPictureEnabled' in document && (document as any).pictureInPictureEnabled
function pip() {
  videoEl.value?.requestPictureInPicture?.().catch(() => {})
}
function poke() {
  ui.idle = false
  clearTimeout(idleTimer)
  idleTimer = window.setTimeout(() => (ui.idle = ui.playing), 2500)
}

function onKey(e: KeyboardEvent) {
  if (state.view !== 'player' || (e.target as HTMLElement)?.tagName === 'INPUT') return
  const v = videoEl.value
  switch (e.key) {
    case ' ': e.preventDefault(); toggle(); break
    case 'ArrowLeft': if (v) v.currentTime = Math.max(0, v.currentTime - 5); break
    case 'ArrowRight': if (v) v.currentTime = Math.min(ui.dur, v.currentTime + 5); break
    case 'ArrowUp': e.preventDefault(); setVolume(Math.min(100, s.value.volume + 5)); break
    case 'ArrowDown': e.preventDefault(); setVolume(Math.max(0, s.value.volume - 5)); break
    case 'f': case 'F': toggleFull(); break
    case 'd': case 'D': saveSettings({ danmakuOverlay: !s.value.danmakuOverlay }); break
    case 'n': case 'N': next(); break
    case 'Escape': if (ui.full) toggleFull(); break
  }
  poke()
}

// 缩略图：先用已保存的；没有则截取视频 10% 处的画面并保存
let thumbBusy = false
async function queueThumbs() {
  if (thumbBusy) return
  thumbBusy = true
  for (const it of [...state.library]) {
    if (thumbs[it.path]) continue
    const url = thumbURL(it.path)
    if (await exists(url)) { thumbs[it.path] = url; continue }
    if (!canPlay(it.info.codec)) continue
    const data = await capture(it)
    if (data) {
      await api.saveThumb(it.path, data)
      thumbs[it.path] = url + '&v=' + Date.now()
    }
  }
  thumbBusy = false
}
function exists(url: string) {
  return new Promise<boolean>((res) => {
    const img = new Image()
    img.onload = () => res(true)
    img.onerror = () => res(false)
    img.src = url
  })
}
function capture(it: LibraryItem) {
  return new Promise<string>((res) => {
    const v = document.createElement('video')
    v.crossOrigin = 'anonymous'
    v.muted = true
    v.preload = 'auto'
    const done = (d: string) => { v.removeAttribute('src'); v.load(); res(d) }
    const timer = setTimeout(() => done(''), 8000)
    v.onloadedmetadata = () => (v.currentTime = Math.min(v.duration * 0.1, 30))
    v.onseeked = () => {
      try {
        const c = document.createElement('canvas')
        c.width = 320; c.height = 180
        c.getContext('2d')!.drawImage(v, 0, 0, 320, 180)
        clearTimeout(timer)
        done(c.toDataURL('image/jpeg', 0.72))
      } catch { clearTimeout(timer); done('') }
    }
    v.onerror = () => { clearTimeout(timer); done('') }
    v.src = fileURL(it.path)
  })
}

onMounted(() => {
  refresh()
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  persist()
  state.playerActive = false
  window.removeEventListener('keydown', onKey)
  if (ui.full) WindowUnfullscreen()
})
watch(() => state.view, (v) => { if (v !== 'player') videoEl.value?.pause() })
watch(() => state.libraryLoaded, (l) => { if (!l && state.view === 'player') nextTick(refresh) })
</script>

<template>
  <div class="player" :class="{ full: ui.full }">
    <div v-if="state.libraryLoaded && !state.library.length" class="empty">
      <h2>{{ t('player.empty.title') }}</h2>
      <p class="mono">{{ t('player.empty.body', { dir: state.outputDir }) }}</p>
      <button class="btn pri" @click="state.view = 'library'"><Icon name="library" />{{ t('queue.toLibrary') }}</button>
    </div>

    <template v-else>
      <section class="stage-wrap">
        <div ref="stageEl" class="stage" :class="{ idle: ui.idle }" @mousemove="poke" @dblclick="toggleFull">
          <video
            v-if="current && !ui.failed" ref="videoEl" :key="current.path" :src="fileURL(current.path)" playsinline
            @loadedmetadata="onLoaded" @timeupdate="onTime" @play="onPlayState" @pause="onPlayState" @ended="onEnded"
            @error="ui.failed = true" @click="toggle"
          />
          <DanmakuLayer
            v-if="current && !ui.failed && comments.length" :media="videoEl" :comments="comments"
            :enabled="s.danmakuOverlay" :opacity="s.danmakuOpacity / 100" :density="s.danmakuDensity / 100" :scale="s.danmakuSize / 100"
          />
          <div v-if="current && ui.failed" class="fallback">
            <Icon name="warn" />
            <h3>{{ t('player.cantPlay') }}</h3>
            <p>{{ t('player.cantPlayBody', { codec: codecLabel(current.info.codec) }) }}</p>
            <button class="btn pri" @click="api.openPath(current.path)"><Icon name="external" />{{ t('player.openSystem') }}</button>
          </div>

          <div v-if="current && !ui.failed" class="controls">
            <div class="seek" @click="seek">
              <i class="buf" :style="{ width: (ui.dur ? (ui.buffered / ui.dur) * 100 : 0) + '%' }" />
              <i class="pos" :style="{ width: (ui.dur ? (ui.time / ui.dur) * 100 : 0) + '%' }" />
            </div>
            <div class="row mono">
              <button class="cbtn" :aria-label="ui.playing ? 'pause' : 'play'" @click="toggle"><Icon :name="ui.playing ? 'pause' : 'play'" /></button>
              <button class="cbtn" :title="t('player.next')" @click="next"><Icon name="next" /></button>
              <span class="time">{{ duration(ui.time) }} / {{ duration(ui.dur) }}</span>
              <span class="sp" />
              <button class="cbtn tg" :class="{ on: s.danmakuOverlay }" :disabled="!comments.length" @click="saveSettings({ danmakuOverlay: !s.danmakuOverlay })">
                <Icon name="danmaku" />{{ t('player.danmaku') }} {{ s.danmakuOverlay ? t('player.on') : t('player.off') }}
              </button>
              <div class="pop-wrap">
                <button class="cbtn" :title="t('player.danmakuSettings')" :disabled="!comments.length" @click="ui.showDm = !ui.showDm"><Icon name="sliders" /></button>
                <div v-if="ui.showDm" class="pop panel">
                  <label>{{ t('player.opacity') }}<input type="range" min="10" max="100" :value="s.danmakuOpacity" @change="saveSettings({ danmakuOpacity: +($event.target as HTMLInputElement).value })" /></label>
                  <label>{{ t('player.density') }}<input type="range" min="10" max="100" :value="s.danmakuDensity" @change="saveSettings({ danmakuDensity: +($event.target as HTMLInputElement).value })" /></label>
                  <label>{{ t('player.size') }}<input type="range" min="50" max="150" :value="s.danmakuSize" @change="saveSettings({ danmakuSize: +($event.target as HTMLInputElement).value })" /></label>
                </div>
              </div>
              <button class="cbtn tg" :title="t('player.speed')" @click="cycleRate">{{ ui.rate }}×</button>
              <button class="cbtn" @click="toggleMute"><Icon :name="ui.muted || s.volume === 0 ? 'mute' : 'volume'" /></button>
              <input class="vol" type="range" min="0" max="100" :value="s.volume" :aria-label="'volume'" @input="setVolume(+($event.target as HTMLInputElement).value)" />
              <button v-if="pipOk" class="cbtn" :title="t('player.pip')" @click="pip"><Icon name="pip" /></button>
              <button class="cbtn" :title="t('player.fullscreen')" @click="toggleFull"><Icon name="fullscreen" /></button>
            </div>
          </div>
        </div>
        <div v-if="current" class="nowt">
          <h3 class="ellipsis" :title="current.name">{{ current.name }}</h3>
          <span class="mono dim">
            {{ codecLabel(current.info.codec) }} · {{ current.info.width }}×{{ current.info.height }} {{ resLabel(current.info.height, current.info.width) }} · {{ bytes(current.size) }}
          </span>
          <span class="sp" />
          <button class="btn ghost" @click="api.revealPath(current.path)"><Icon name="reveal" />{{ t('job.reveal') }}</button>
          <button class="btn ghost" @click="api.openPath(current.path)"><Icon name="external" />{{ t('player.openSystem') }}</button>
        </div>
        <p class="keys mono">{{ t('player.shortcuts') }}</p>
      </section>

      <aside class="plist">
        <label class="search">
          <Icon name="search" />
          <input v-model="query" class="input" :placeholder="t('player.search')" spellcheck="false" />
        </label>
        <div class="scroll">
          <section v-for="[folder, list] in groups" :key="folder" class="pg">
            <div class="h mono ellipsis" :title="folder">{{ folder || '/' }}</div>
            <button
              v-for="it in list" :key="it.path" class="pi" :class="{ on: it.path === state.nowPlaying }"
              @click="state.nowPlaying = it.path"
            >
              <span class="mini" :style="{ background: placeholderBg(it.folder || it.name) }">
                <img v-if="thumbs[it.path]" :src="thumbs[it.path]" alt="" />
                <span v-if="it.play.position > 0 && !it.play.watched && it.info.duration" class="prog"><i :style="{ width: (it.play.position / it.info.duration) * 100 + '%' }" /></span>
              </span>
              <span class="txt">
                <span class="t ellipsis">{{ it.name }}</span>
                <span class="d mono">
                  {{ duration(it.info.duration) }}
                  <template v-if="it.path === state.nowPlaying"> · {{ t('player.playing') }}</template>
                  <template v-else-if="it.play.watched"> · {{ t('player.watched') }}</template>
                  <template v-else-if="it.play.position > 0"> · {{ t('player.left', { t: duration(it.info.duration - it.play.position) }) }}</template>
                </span>
              </span>
            </button>
          </section>
        </div>
      </aside>
    </template>
  </div>
</template>

<style scoped>
.player { display: grid; grid-template-columns: minmax(0, 1fr) 300px; height: 100%; min-height: 0; }
.player > .empty { grid-column: 1 / -1; align-content: center; }
.stage-wrap { padding: 14px 16px; display: grid; grid-template-rows: minmax(0, 1fr) auto auto; gap: 10px; min-width: 0; min-height: 0; }
.stage { position: relative; background: #000; border-radius: 12px; overflow: hidden; min-height: 0; border: 1px solid #1c2638; }
.stage video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; background: #000; }
.full .stage { position: fixed; inset: 0; z-index: 100; border-radius: 0; border: 0; }
.stage.idle { cursor: none; }
.controls {
  position: absolute; left: 0; right: 0; bottom: 0; padding: 26px 14px 10px; display: grid; gap: 8px; z-index: 2;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.78)); transition: opacity 0.3s;
}
.stage.idle .controls { opacity: 0; }
.seek { height: 14px; display: flex; align-items: center; position: relative; cursor: pointer; }
.seek::before { content: ''; position: absolute; left: 0; right: 0; height: 4px; border-radius: 2px; background: rgba(255, 255, 255, 0.18); }
.seek i { position: absolute; left: 0; height: 4px; border-radius: 2px; }
.seek .buf { background: rgba(255, 255, 255, 0.28); }
.seek .pos { background: linear-gradient(90deg, var(--signal-2), var(--signal)); box-shadow: 0 0 10px var(--signal); }
.seek:hover::before, .seek:hover i { height: 6px; }
.row { display: flex; align-items: center; gap: 6px; font-size: 12px; }
.cbtn { display: inline-flex; align-items: center; gap: 6px; border: 0; background: transparent; color: #fff; padding: 5px 7px; border-radius: 6px; cursor: pointer; }
.cbtn:hover { background: rgba(255, 255, 255, 0.1); }
.cbtn:disabled { opacity: 0.4; pointer-events: none; }
.cbtn svg { width: 17px; height: 17px; }
.cbtn.tg { border: 1px solid rgba(255, 255, 255, 0.22); font-family: var(--font-mono); font-size: 11.5px; }
.cbtn.tg.on { color: var(--pink); border-color: rgba(251, 114, 153, 0.6); }
.time { color: rgba(255, 255, 255, 0.85); margin-left: 4px; }
.sp { flex: 1; }
.vol { width: 80px; }
.pop-wrap { position: relative; }
.pop { position: absolute; bottom: 38px; right: 0; padding: 12px 14px; display: grid; gap: 10px; width: 220px; background: rgba(17, 24, 39, 0.96); font-family: var(--font-body); }
.pop label { display: grid; gap: 4px; font-size: 12px; color: var(--dim); }
.fallback { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: 10px; text-align: center; padding: 20px; color: var(--dim); }
.fallback svg { width: 34px; height: 34px; color: var(--warn); }
.fallback h3 { margin: 0; color: var(--text); }
.fallback p { margin: 0; max-width: 46ch; }
.nowt { display: flex; align-items: center; gap: 12px; min-width: 0; }
.nowt h3 { margin: 0; font-size: 15px; min-width: 0; }
.nowt .mono { font-size: 11px; white-space: nowrap; }
.keys { margin: 0; font-size: 10.5px; color: var(--faint); }
.plist { border-left: 1px solid var(--line); background: rgba(13, 18, 28, 0.82); padding: 12px 10px; display: grid; grid-template-rows: auto 1fr; gap: 10px; min-height: 0; }
.search { position: relative; display: flex; }
.search svg { position: absolute; left: 10px; top: 50%; width: 15px; height: 15px; transform: translateY(-50%); color: var(--faint); }
.search .input { flex: 1; padding-left: 32px; }
.scroll { overflow-y: auto; min-height: 0; display: grid; gap: 14px; align-content: start; }
.pg { display: grid; gap: 4px; }
.pg .h { font-size: 10.5px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--dim); padding: 0 6px 2px; }
.pi { display: grid; grid-template-columns: 76px minmax(0, 1fr); gap: 9px; align-items: center; padding: 6px; border-radius: 8px; border: 0; background: transparent; color: inherit; text-align: left; cursor: pointer; }
.pi:hover { background: rgba(255, 255, 255, 0.04); }
.pi.on { background: var(--signal-soft); box-shadow: inset 0 0 0 1px var(--signal-line); }
.mini { aspect-ratio: 16 / 9; border-radius: 5px; overflow: hidden; position: relative; display: block; }
.mini img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.prog { position: absolute; left: 0; right: 0; bottom: 0; height: 3px; background: rgba(0, 0, 0, 0.5); }
.prog i { display: block; height: 100%; background: var(--pink); }
.txt { display: grid; gap: 2px; min-width: 0; }
.t { font-size: 12px; font-weight: 600; }
.d { font-size: 10.5px; color: var(--dim); }
</style>
