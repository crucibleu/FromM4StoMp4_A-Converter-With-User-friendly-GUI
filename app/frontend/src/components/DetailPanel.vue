<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Icon from './Icon.vue'
import DanmakuLayer from './DanmakuLayer.vue'
import { api, coverURL, fileURL } from '../api'
import { focused, state, enqueueSelected, playOutput } from '../store'
import { t } from '../i18n'
import { bytes, canPlay, codecLabel, duration, placeholderBg, resLabel } from '../format'
import type { Comment } from '../types'

const v = focused
const previewing = ref(false)
const failed = ref(false)
const videoEl = ref<HTMLVideoElement | null>(null)
const audioEl = ref<HTMLAudioElement | null>(null)
const comments = ref<Comment[]>([])
const coverOk = ref(true)
let syncTimer = 0

const playable = computed(() => !!v.value && canPlay(v.value.video.codec))
const canConvert = computed(() => v.value && (v.value.status === 'ready' || v.value.status === 'converted'))

watch(() => v.value?.id, async (id) => {
  stop()
  coverOk.value = true
  comments.value = []
  if (id && v.value?.danmakuFile) comments.value = await api.cacheDanmaku(id)
})

function startPreview() {
  if (!v.value) return
  failed.value = !playable.value
  previewing.value = true
}

function stop() {
  previewing.value = false
  failed.value = false
  state.playerActive = false
  clearInterval(syncTimer)
}

// 视频与音频是两个独立的文件，用视频元素的时间驱动音频
function bind() {
  const vv = videoEl.value, a = audioEl.value
  if (!vv || !a) return
  a.volume = (state.settings?.volume ?? 80) / 100
  vv.onplay = () => { state.playerActive = true; a.currentTime = vv.currentTime; a.play().catch(() => {}) }
  vv.onpause = () => { state.playerActive = false; a.pause() }
  vv.onwaiting = () => a.pause()
  vv.onplaying = () => { a.currentTime = vv.currentTime; a.play().catch(() => {}) }
  vv.onseeking = () => (a.currentTime = vv.currentTime)
  vv.onratechange = () => (a.playbackRate = vv.playbackRate)
  vv.onvolumechange = () => { a.volume = vv.volume; a.muted = vv.muted }
  vv.onended = () => { a.pause(); state.playerActive = false }
  vv.onerror = () => (failed.value = true)
  vv.volume = a.volume
  vv.muted = false
  clearInterval(syncTimer)
  syncTimer = window.setInterval(() => {
    if (!vv.paused && Math.abs(a.currentTime - vv.currentTime) > 0.25) a.currentTime = vv.currentTime
  }, 1000)
  vv.play().catch(() => {})
}

onBeforeUnmount(stop)
</script>

<template>
  <aside class="drawer">
    <template v-if="v">
      <div class="pv" :style="{ background: placeholderBg(v.series) }">
        <template v-if="previewing && !failed">
          <video ref="videoEl" :src="fileURL(v.videoFile)" controls playsinline @loadedmetadata="bind" />
          <audio ref="audioEl" :src="fileURL(v.audioFile, 'audio')" preload="auto" />
          <DanmakuLayer
            v-if="comments.length" :media="videoEl" :comments="comments"
            :enabled="state.settings?.danmakuOverlay" :opacity="(state.settings?.danmakuOpacity ?? 85) / 100"
            :density="(state.settings?.danmakuDensity ?? 70) / 100" :scale="0.75"
          />
          <button class="close btn icon" :aria-label="t('job.cancel')" @click="stop"><Icon name="x" /></button>
        </template>
        <template v-else>
          <img v-if="coverOk && (v.coverFile || v.coverUrl)" :src="coverURL(v.id)" alt="" @error="coverOk = false" />
          <button v-if="v.videoFile && v.audioFile && !failed" class="play" :aria-label="t('detail.preview')" @click="startPreview">
            <Icon name="play" />
          </button>
          <p v-if="failed" class="fail">{{ t('detail.previewFailed', { codec: codecLabel(v.video.codec) }) }}</p>
        </template>
      </div>

      <div class="head">
        <span class="st" :class="v.status">{{ t('status.' + v.status) }}</span>
        <h3>{{ v.page > 0 ? `P${v.page} ` : '' }}{{ v.part }}</h3>
        <p v-if="v.note" class="note">{{ t('note.' + v.note) }}</p>
      </div>

      <dl class="kv">
        <dt>{{ t('detail.series') }}</dt><dd :title="v.series">{{ v.series }}</dd>
        <dt>{{ t('detail.uploader') }}</dt><dd>{{ v.uploader || '—' }}</dd>
        <dt>{{ t('detail.video') }}</dt>
        <dd class="mono">{{ v.video.codec ? `${codecLabel(v.video.codec)} · ${v.video.width}×${v.video.height} · ${resLabel(v.video.height, v.video.width)}` : '—' }}</dd>
        <dt>{{ t('detail.audio') }}</dt><dd class="mono">{{ v.audio.codec ? codecLabel(v.audio.codec) : '—' }}</dd>
        <dt>{{ t('detail.duration') }}</dt><dd class="mono">{{ duration(v.duration) }}</dd>
        <dt>{{ t('detail.size') }}</dt><dd class="mono">{{ bytes(v.size) }}</dd>
        <dt>{{ t('detail.danmaku') }}</dt>
        <dd class="mono">{{ v.danmakuFile ? t('detail.danmakuCount', { n: comments.length.toLocaleString() }) : t('detail.none') }}</dd>
        <dt>{{ t('detail.source') }}</dt><dd>{{ t('source.' + v.source) }}{{ v.bvid ? ' · ' + v.bvid : '' }}</dd>
        <dt>{{ t('detail.cache') }}</dt><dd class="mono selectable" :title="v.dir">{{ v.dir }}</dd>
        <template v-if="v.output">
          <dt>{{ t('detail.output') }}</dt><dd class="mono selectable" :title="v.output">{{ v.output }}</dd>
        </template>
      </dl>

      <div class="actions">
        <button v-if="v.status === 'converted'" class="btn pri" @click="playOutput(v.output)"><Icon name="play" />{{ t('detail.playOutput') }}</button>
        <button v-else-if="canConvert" class="btn pri" @click="enqueueSelected([v.id])"><Icon name="convert" />{{ t('detail.convertThis') }}</button>
        <button class="btn" @click="api.openPath(v.dir)"><Icon name="folder" />{{ t('detail.showCache') }}</button>
      </div>
    </template>
    <div v-else class="none">
      <Icon name="player" />
      <p>{{ t('detail.empty') }}</p>
    </div>
  </aside>
</template>

<style scoped>
.drawer {
  border-left: 1px solid var(--line); background: rgba(13, 18, 28, 0.82); backdrop-filter: blur(6px);
  padding: 14px; display: flex; flex-direction: column; gap: 14px; min-width: 0; overflow-y: auto;
}
.pv { aspect-ratio: 16 / 9; border-radius: 9px; position: relative; overflow: hidden; display: grid; place-items: center; flex: none; background: #000; }
.pv img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.pv video { position: absolute; inset: 0; width: 100%; height: 100%; background: #000; }
.play {
  width: 46px; height: 46px; border-radius: 50%; border: 1px solid rgba(255, 255, 255, 0.6); background: rgba(0, 0, 0, 0.45);
  display: grid; place-items: center; color: #fff; cursor: pointer; z-index: 1; transition: transform 0.2s var(--ease), box-shadow 0.2s;
}
.play svg { width: 18px; height: 18px; margin-left: 3px; }
.play:hover { transform: scale(1.08); box-shadow: 0 0 22px rgba(56, 217, 245, 0.7); border-color: var(--signal); }
.close { position: absolute; right: 6px; top: 6px; z-index: 2; background: rgba(0, 0, 0, 0.55); }
.fail { position: relative; z-index: 1; margin: 0 16px; padding: 8px 10px; border-radius: 8px; background: rgba(0, 0, 0, 0.7); font-size: 12px; text-align: center; }
.head { display: grid; gap: 6px; justify-items: start; }
.head h3 { margin: 0; font-size: 15px; line-height: 1.35; word-break: break-word; }
.note { margin: 0; color: var(--err); font-size: 12px; }
.kv { display: grid; grid-template-columns: auto 1fr; gap: 6px 14px; margin: 0; font-size: 12px; }
.kv dt { color: var(--dim); font-family: var(--font-mono); font-size: 11px; padding-top: 1px; white-space: nowrap; }
.kv dd { margin: 0; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: auto; }
.none { flex: 1; display: grid; place-content: center; justify-items: center; gap: 10px; color: var(--faint); text-align: center; }
.none svg { width: 34px; height: 34px; }
.none p { margin: 0; max-width: 22ch; }
</style>
