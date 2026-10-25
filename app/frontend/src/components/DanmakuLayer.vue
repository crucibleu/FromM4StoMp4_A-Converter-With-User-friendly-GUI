<script setup lang="ts">
// 弹幕层：位置完全由视频的 currentTime 计算，暂停、拖动进度后保持同步
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Comment } from '../types'

const props = withDefaults(defineProps<{
  media: HTMLMediaElement | null
  comments: Comment[]
  enabled?: boolean
  opacity?: number // 0-1
  density?: number // 0.1-1：可用轨道比例
  scale?: number // 字号倍数
}>(), { enabled: true, opacity: 0.85, density: 0.7, scale: 1 })

const canvas = ref<HTMLCanvasElement | null>(null)
const SCROLL_SECS = 8
const FIXED_SECS = 4

interface Active { c: Comment; lane: number; w: number; start: number; color: string }
let active: Active[] = []
let idx = 0
let lastT = -1
let W = 0, H = 0, dpr = 1
let raf = 0
let dirty = true
let ro: ResizeObserver | null = null

function fontPx() {
  return Math.max(14, Math.round((H / 720) * 25 * props.scale))
}

function lowerBound(t: number) {
  let lo = 0, hi = props.comments.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (props.comments[mid].t < t) lo = mid + 1
    else hi = mid
  }
  return lo
}

function reset(t: number) {
  active = []
  idx = lowerBound(Math.max(0, t - 0.05))
}

function resize() {
  const el = canvas.value
  if (!el) return
  const r = el.getBoundingClientRect()
  dpr = Math.min(2, window.devicePixelRatio || 1)
  W = r.width
  H = r.height
  el.width = Math.max(1, Math.round(W * dpr))
  el.height = Math.max(1, Math.round(H * dpr))
  dirty = true
}

function spawn(ctx: CanvasRenderingContext2D, c: Comment, t: number) {
  const px = fontPx() * (c.s >= 30 ? 1.15 : c.s && c.s < 20 ? 0.85 : 1)
  ctx.font = `600 ${px}px ${getComputedStyle(document.body).fontFamily}`
  const w = ctx.measureText(c.text).width
  const laneH = fontPx() * 1.3
  const total = Math.max(1, Math.floor((H * 0.92) / laneH))
  const usable = Math.max(1, Math.round(total * props.density))
  const color = '#' + (c.c & 0xffffff).toString(16).padStart(6, '0')

  if (c.m === 4 || c.m === 5) {
    for (let lane = 0; lane < Math.min(usable, Math.ceil(total / 2)); lane++) {
      const busy = active.some((a) => a.c.m === c.m && a.lane === lane && t - a.start < FIXED_SECS)
      if (!busy) {
        active.push({ c, lane, w, start: t, color })
        return
      }
    }
    return // 没有空轨道时丢弃，避免重叠
  }
  for (let lane = 0; lane < usable; lane++) {
    let ok = true
    for (const a of active) {
      if (a.lane !== lane || a.c.m === 4 || a.c.m === 5) continue
      const x = W - ((t - a.start) / SCROLL_SECS) * (W + a.w)
      if (x + a.w + 24 > W) { ok = false; break }
    }
    if (ok) {
      active.push({ c, lane, w, start: t, color })
      return
    }
  }
}

function frame() {
  raf = requestAnimationFrame(frame)
  const el = canvas.value
  const media = props.media
  if (!el || !media) return
  const ctx = el.getContext('2d')
  if (!ctx) return
  const t = media.currentTime
  if (t === lastT && !dirty) return
  if (lastT < 0 || t < lastT - 0.3 || t > lastT + 1.5) reset(t)
  lastT = t
  dirty = false

  while (idx < props.comments.length && props.comments[idx].t <= t) {
    spawn(ctx, props.comments[idx], props.comments[idx].t)
    idx++
  }
  active = active.filter((a) => t - a.start < (a.c.m === 4 || a.c.m === 5 ? FIXED_SECS : SCROLL_SECS) && t >= a.start - 0.05)

  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, W, H)
  ctx.globalAlpha = props.opacity
  ctx.textBaseline = 'top'
  ctx.lineJoin = 'round'
  const laneH = fontPx() * 1.3
  const family = getComputedStyle(document.body).fontFamily
  for (const a of active) {
    const px = fontPx() * (a.c.s >= 30 ? 1.15 : a.c.s && a.c.s < 20 ? 0.85 : 1)
    ctx.font = `600 ${px}px ${family}`
    let x: number, y: number
    if (a.c.m === 5) {
      x = (W - a.w) / 2
      y = 6 + a.lane * laneH
    } else if (a.c.m === 4) {
      x = (W - a.w) / 2
      y = H - laneH - 10 - a.lane * laneH
    } else {
      x = W - ((t - a.start) / SCROLL_SECS) * (W + a.w)
      y = 6 + a.lane * laneH
    }
    ctx.lineWidth = Math.max(2, px / 9)
    ctx.strokeStyle = 'rgba(0,0,0,0.75)'
    ctx.strokeText(a.c.text, x, y)
    ctx.fillStyle = a.color
    ctx.fillText(a.c.text, x, y)
  }
}

function start() {
  cancelAnimationFrame(raf)
  if (props.enabled) {
    lastT = -1
    raf = requestAnimationFrame(frame)
  } else {
    const ctx = canvas.value?.getContext('2d')
    ctx?.clearRect(0, 0, canvas.value!.width, canvas.value!.height)
  }
}

onMounted(() => {
  resize()
  ro = new ResizeObserver(resize)
  if (canvas.value) ro.observe(canvas.value)
  start()
})
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  ro?.disconnect()
})
watch(() => [props.enabled, props.media], start)
watch(() => props.comments, () => { lastT = -1; dirty = true })
watch(() => [props.opacity, props.density, props.scale], () => { lastT = -1; dirty = true })
</script>

<template>
  <canvas ref="canvas" class="danmaku" aria-hidden="true" />
</template>

<style scoped>
.danmaku { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
</style>
