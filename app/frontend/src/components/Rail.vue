<script setup lang="ts">
import Icon from './Icon.vue'
import { state, activeJobs, type View } from '../store'
import { t } from '../i18n'

const items: { view: View; icon: string }[] = [
  { view: 'library', icon: 'library' },
  { view: 'queue', icon: 'queue' },
  { view: 'player', icon: 'player' },
  { view: 'settings', icon: 'settings' },
]
</script>

<template>
  <nav class="rail" :aria-label="'navigation'">
    <button
      v-for="it in items" :key="it.view" class="it" :class="{ on: state.view === it.view }"
      :title="t('nav.' + it.view)" :aria-label="t('nav.' + it.view)" :aria-current="state.view === it.view ? 'page' : undefined"
      @click="state.view = it.view"
    >
      <Icon :name="it.icon" />
      <span class="lbl">{{ t('nav.' + it.view) }}</span>
      <span v-if="it.view === 'queue' && activeJobs.length" class="badge mono">{{ activeJobs.length }}</span>
    </button>
  </nav>
</template>

<style scoped>
.rail {
  display: flex; flex-direction: column; gap: 6px; padding: 12px 8px; align-items: center;
  border-right: 1px solid var(--line); background: rgba(10, 14, 21, 0.6); position: relative; z-index: 2;
}
.it {
  width: 58px; padding: 9px 0 7px; border-radius: 11px; border: 0; background: transparent; color: var(--dim);
  display: grid; justify-items: center; gap: 4px; cursor: pointer; position: relative;
  transition: color 0.15s, background 0.15s, box-shadow 0.2s;
}
.it svg { width: 20px; height: 20px; }
.it .lbl { font-size: 10.5px; line-height: 1; }
.it:hover { color: var(--text); background: rgba(255, 255, 255, 0.03); }
.it.on { color: var(--signal); background: var(--signal-soft); box-shadow: inset 0 0 0 1px var(--signal-line), 0 0 18px -8px var(--signal); }
.badge {
  position: absolute; top: 4px; right: 8px; font-size: 9.5px; font-weight: 600; background: var(--pink); color: #fff;
  border-radius: 6px; padding: 0 4px; line-height: 15px; box-shadow: 0 0 10px -2px var(--pink);
}
</style>
