<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'
import { state, scan, pickCacheDir } from '../store'
import { t } from '../i18n'
import { bytes } from '../format'

const total = computed(() => state.videos.reduce((s, v) => s + v.size, 0))
</script>

<template>
  <header class="topbar">
    <div class="logo display"><img src="../assets/logo.png" alt="" />M4S CONVERTER</div>
    <button class="chip path" :class="{ on: !!state.settings?.cacheDir }" :title="t('top.change')" @click="pickCacheDir">
      <span class="dot" />
      <span class="ellipsis">{{ state.settings?.cacheDir || t('top.noCache') }}</span>
    </button>
    <span v-if="state.videos.length" class="chip static">{{ t('top.videos', { n: state.videos.length, size: bytes(total) }) }}</span>
    <span class="sp" />
    <button class="btn ghost" :disabled="state.scanning || !state.settings?.cacheDir" @click="scan">
      <Icon name="refresh" :class="{ spin: state.scanning }" />{{ t('top.rescan') }}
    </button>
  </header>
</template>

<style scoped>
.topbar {
  display: flex; align-items: center; gap: 10px; padding: 0 14px; height: 48px;
  border-bottom: 1px solid var(--line); background: rgba(10, 14, 21, 0.72); backdrop-filter: blur(8px);
  position: relative; z-index: 2;
}
.logo { display: flex; align-items: center; gap: 9px; font-weight: 700; font-size: 13px; letter-spacing: 0.1em; margin-right: 8px; }
.logo img { width: 22px; height: 22px; filter: drop-shadow(0 0 8px rgba(56, 217, 245, 0.45)); }
.path { max-width: 46vw; min-width: 0; }
.path .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--faint); flex: none; }
.path.on .dot { background: var(--signal); box-shadow: 0 0 8px var(--signal); }
.static { cursor: default; }
.static:hover { color: var(--dim); }
.sp { flex: 1; }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
