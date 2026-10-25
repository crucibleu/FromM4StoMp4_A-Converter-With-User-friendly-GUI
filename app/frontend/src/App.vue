<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import TopBar from './components/TopBar.vue'
import Rail from './components/Rail.vue'
import StatusStrip from './components/StatusStrip.vue'
import ParticleField from './components/ParticleField.vue'
import Toasts from './components/Toasts.vue'
import TermsDialog from './components/TermsDialog.vue'
import LibraryView from './views/LibraryView.vue'
import QueueView from './views/QueueView.vue'
import PlayerView from './views/PlayerView.vue'
import SettingsView from './views/SettingsView.vue'
import { boot, effects, state, toast } from './store'
import { errText, t } from './i18n'

const views = { library: LibraryView, queue: QueueView, player: PlayerView, settings: SettingsView }
const current = computed(() => views[state.view])

// 帧率看门狗：完整动画下连续两个 3 秒窗口平均低于 45fps 时自动切换为精简
let raf = 0
let frames = 0
let windowStart = 0
let slowWindows = 0
function watchdog(now: number) {
  raf = requestAnimationFrame(watchdog)
  if (effects.value !== 'full' || document.hidden || state.playerActive) {
    frames = 0
    windowStart = now
    return
  }
  frames++
  if (now - windowStart >= 3000) {
    const fps = (frames * 1000) / (now - windowStart)
    slowWindows = fps < 45 ? slowWindows + 1 : 0
    frames = 0
    windowStart = now
    if (slowWindows >= 2) {
      state.fxAutoReduced = true
      toast(t('strip.fxAuto'))
    }
  }
}

onMounted(async () => {
  try {
    await boot()
  } catch (e) {
    toast(errText(e), 'err')
  }
  setTimeout(() => { windowStart = performance.now(); raf = requestAnimationFrame(watchdog) }, 5000)
})
onBeforeUnmount(() => cancelAnimationFrame(raf))
watch(effects, (lv) => {
  document.documentElement.classList.remove('fx-full', 'fx-reduced', 'fx-off')
  document.documentElement.classList.add('fx-' + lv)
}, { immediate: true })
</script>

<template>
  <div class="shell">
    <ParticleField />
    <TopBar />
    <Rail />
    <main class="content">
      <Transition name="view" mode="out-in">
        <KeepAlive :include="['PlayerView']">
          <component :is="current" :key="state.view" />
        </KeepAlive>
      </Transition>
    </main>
    <StatusStrip />
    <Toasts />
    <TermsDialog v-if="state.ready && !state.settings?.acceptedTerms" />
  </div>
</template>

<style scoped>
.shell {
  height: 100%; display: grid; grid-template-columns: 74px minmax(0, 1fr); grid-template-rows: 48px minmax(0, 1fr) 28px;
  position: relative; overflow: hidden;
  background:
    radial-gradient(900px 420px at 88% -12%, rgba(56, 217, 245, 0.09), transparent 60%),
    radial-gradient(700px 400px at -10% 110%, rgba(251, 114, 153, 0.05), transparent 60%),
    linear-gradient(rgba(56, 217, 245, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(56, 217, 245, 0.03) 1px, transparent 1px),
    var(--ink);
  background-size: auto, auto, 28px 28px, 28px 28px, auto;
}
.shell > :deep(.topbar) { grid-column: 1 / -1; }
.content { position: relative; z-index: 1; min-width: 0; min-height: 0; overflow: hidden; }
.content > * { height: 100%; }
</style>
