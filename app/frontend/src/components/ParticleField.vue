<script setup lang="ts">
// 背景粒子网络 + 队列完成时的粒子爆发（tsParticles）
// 播放视频、窗口隐藏时完全暂停；转换进行中时减少粒子；动画关闭时不加载
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { tsParticles, type Container, type ISourceOptions } from '@tsparticles/engine'
import { loadSlim } from '@tsparticles/slim'
import { activeJobs, effects, state } from '../store'

const bgEl = ref<HTMLElement | null>(null)
const burstEl = ref<HTMLElement | null>(null)
let bg: Container | undefined
let burst: Container | undefined
let ready: Promise<void> | null = null
const hidden = ref(document.hidden)

const SIGNAL = '#38d9f5', PINK = '#fb7299', OK = '#3ddc97'

const busy = computed(() => activeJobs.value.some((j) => j.state === 'running'))
const paused = computed(() => hidden.value || state.playerActive || state.view === 'player')

function bgOptions(level: 'full' | 'reduced', thin: boolean): ISourceOptions {
  const full = level === 'full'
  const n = full ? (thin ? 40 : 70) : 22
  return {
    fullScreen: { enable: false },
    fpsLimit: full ? 60 : 30,
    detectRetina: true,
    background: { color: 'transparent' },
    particles: {
      number: { value: n, density: { enable: true, width: 1400, height: 900 } },
      paint: { fill: { enable: true, color: { value: SIGNAL } } },
      opacity: { value: { min: 0.15, max: 0.55 } },
      size: { value: { min: 0.8, max: 2 } },
      move: { enable: true, speed: full ? 0.35 : 0.15, direction: 'none', outModes: { default: 'out' } },
      links: { enable: full, distance: 130, color: SIGNAL, opacity: 0.14, width: 0.8 },
    },
    interactivity: {
      detectsOn: 'window',
      events: { onHover: { enable: full, mode: 'grab' } },
      modes: { grab: { distance: 150, links: { opacity: 0.35 } } },
    },
  }
}

const burstOptions: ISourceOptions = {
  fullScreen: { enable: false },
  fpsLimit: 60,
  detectRetina: true,
  background: { color: 'transparent' },
  particles: {
    number: { value: 0 },
    paint: { fill: { enable: true, color: { value: [SIGNAL, PINK, OK] } } },
    shape: { type: 'circle' },
    opacity: { value: { min: 0.6, max: 1 }, animation: { enable: true, speed: 0.9, startValue: 'max', destroy: 'min' } },
    size: { value: { min: 1.2, max: 3.2 } },
    life: { count: 1, duration: { value: { min: 1.1, max: 1.9 } } },
    move: { enable: true, speed: { min: 6, max: 22 }, decay: 0.05, direction: 'none', random: true, straight: false, outModes: { default: 'destroy' }, gravity: { enable: true, acceleration: 3 } },
  },
}

async function ensureEngine() {
  if (!ready) ready = loadSlim(tsParticles)
  await ready
}

async function load() {
  const level = effects.value
  bg?.destroy()
  bg = undefined
  if (level === 'off' || !bgEl.value) return
  await ensureEngine()
  bg = await tsParticles.load({ id: 'bg-particles', element: bgEl.value, options: bgOptions(level, busy.value) })
  applyPause()
}

function applyPause() {
  if (!bg) return
  if (paused.value) bg.pause()
  else bg.play()
}

async function fireBurst() {
  if (effects.value === 'off' || !burstEl.value) return
  await ensureEngine()
  if (!burst) burst = await tsParticles.load({ id: 'burst-particles', element: burstEl.value, options: burstOptions })
  if (!burst) return
  const w = burstEl.value.clientWidth, h = burstEl.value.clientHeight
  burst.particles.push(effects.value === 'full' ? 160 : 40, { x: w / 2, y: h * 0.42 })
  burst.play()
}

function onVisibility() { hidden.value = document.hidden }
function onFinished() { fireBurst() }

onMounted(() => {
  load()
  document.addEventListener('visibilitychange', onVisibility)
  window.addEventListener('m4s:queue-finished', onFinished)
})
onBeforeUnmount(() => {
  bg?.destroy()
  burst?.destroy()
  document.removeEventListener('visibilitychange', onVisibility)
  window.removeEventListener('m4s:queue-finished', onFinished)
})
watch(effects, load)
watch(busy, () => effects.value === 'full' && load())
watch(paused, applyPause)
</script>

<template>
  <div ref="bgEl" class="bg" aria-hidden="true" />
  <div ref="burstEl" class="burst" aria-hidden="true" />
</template>

<style scoped>
.bg, .burst { position: absolute; inset: 0; pointer-events: none; }
.bg { z-index: 0; }
.burst { z-index: 30; }
.bg :deep(canvas), .burst :deep(canvas) { position: absolute; inset: 0; width: 100% !important; height: 100% !important; }
</style>
