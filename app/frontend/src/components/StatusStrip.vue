<script setup lang="ts">
import { computed } from 'vue'
import { state, activeJobs } from '../store'
import { t } from '../i18n'
import { bytes } from '../format'

const live = computed(() => {
  if (state.scanning) return t('strip.scanning')
  const n = activeJobs.value.filter((j) => j.state === 'running').length
  return n ? t('strip.running', { n }) : t('strip.idle')
})
const busy = computed(() => state.scanning || activeJobs.value.length > 0)
const disk = computed(() => {
  const p = state.outputDir || state.settings?.cacheDir || ''
  const m = p.match(/^[A-Za-z]:/)
  return m ? m[0] : '/'
})
</script>

<template>
  <footer class="strip mono">
    <span class="live" :class="{ busy }">{{ live }}</span>
    <span v-if="state.tick && state.tick.speed > 0">{{ bytes(state.tick.speed) }}/s</span>
    <span v-if="state.freeBytes">{{ t('strip.free', { disk, size: bytes(state.freeBytes) }) }}</span>
    <span class="sp" />
    <span>{{ state.settings?.mp4box ? t('strip.mp4boxCustom') : t('strip.mp4boxBuiltin') }}</span>
    <span>v{{ state.init?.version }}</span>
  </footer>
</template>

<style scoped>
.strip {
  grid-column: 1 / -1; display: flex; align-items: center; gap: 18px; padding: 0 14px; height: 28px;
  border-top: 1px solid var(--line); background: rgba(10, 14, 21, 0.85); color: var(--dim); font-size: 11px;
  position: relative; z-index: 2;
}
.live::before { content: ''; display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--faint); margin-right: 7px; vertical-align: 1px; }
.live.busy { color: var(--signal); }
.live.busy::before { background: var(--signal); box-shadow: 0 0 8px var(--signal); animation: pulse 1.4s ease-in-out infinite; }
@keyframes pulse { 50% { opacity: 0.35; } }
.sp { flex: 1; }
</style>
