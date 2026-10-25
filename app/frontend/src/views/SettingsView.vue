<script setup lang="ts">
import { computed } from 'vue'
import Icon from '../components/Icon.vue'
import { api } from '../api'
import { saveSettings, scan, state, pickCacheDir } from '../store'
import { t } from '../i18n'
import type { EffectsLevel } from '../types'

const s = computed(() => state.settings!)

async function pickOutput() {
  const p = await api.pickFolder(t('set.outputDir'), state.outputDir)
  if (p) {
    await saveSettings({ outputDir: p })
    scan()
  }
}
async function resetOutput() {
  await saveSettings({ outputDir: '' })
  scan()
}
async function pickMP4Box() {
  const p = await api.pickFile(t('set.mp4boxCustom'))
  if (p) saveSettings({ mp4box: p })
}
</script>

<template>
  <div v-if="state.settings" class="settings">
    <h2 class="display">{{ t('set.title') }}</h2>

    <section class="panel sec">
      <h3>{{ t('set.folders') }}</h3>
      <div class="field">
        <span class="lbl">{{ t('set.cacheDir') }}</span>
        <div class="row">
          <span class="path mono selectable">{{ s.cacheDir || '—' }}</span>
          <button class="btn" @click="pickCacheDir"><Icon name="folder" />{{ t('out.browse') }}</button>
        </div>
      </div>
      <div class="field">
        <span class="lbl">{{ t('set.outputDir') }}</span>
        <div class="row">
          <span class="path mono selectable">{{ state.outputDir || '—' }}</span>
          <button v-if="s.outputDir" class="btn ghost" @click="resetOutput">{{ t('out.default') }}</button>
          <button class="btn" @click="pickOutput"><Icon name="folder" />{{ t('out.browse') }}</button>
        </div>
      </div>
      <button class="btn ghost fit" @click="state.view = 'library'; state.showOutputDialog = true">{{ t('out.title') }} · {{ t('out.template') }}</button>
    </section>

    <section class="panel sec">
      <h3>{{ t('set.conversion') }}</h3>
      <div class="field inline">
        <span>{{ t('set.workers') }}</span>
        <div class="seg">
          <button v-for="n in 4" :key="n" :class="{ on: s.workers === n }" @click="saveSettings({ workers: n })">{{ n }}</button>
        </div>
      </div>
      <label class="check"><input type="checkbox" :checked="s.danmaku" @change="saveSettings({ danmaku: !s.danmaku })" /><span>{{ t('out.danmaku') }}</span></label>
      <label class="check"><input type="checkbox" :checked="s.downloadDanmaku" :disabled="!s.danmaku" @change="saveSettings({ downloadDanmaku: !s.downloadDanmaku })" /><span>{{ t('out.download') }}</span></label>
      <label class="check"><input type="checkbox" :checked="s.overwrite" @change="saveSettings({ overwrite: !s.overwrite })" /><span>{{ t('out.overwrite') }}</span></label>
      <div class="field">
        <span class="lbl">{{ t('set.mp4box') }}</span>
        <div class="row">
          <div class="seg">
            <button :class="{ on: !s.mp4box }" @click="saveSettings({ mp4box: '' })">{{ t('set.mp4boxBuiltin') }}</button>
            <button :class="{ on: !!s.mp4box }" @click="pickMP4Box">{{ t('set.mp4boxCustom') }}</button>
          </div>
          <span v-if="s.mp4box" class="path mono selectable">{{ s.mp4box }}</span>
        </div>
      </div>
    </section>

    <section class="panel sec">
      <h3>{{ t('set.appearance') }}</h3>
      <div class="field inline">
        <span>{{ t('set.effects') }}</span>
        <div class="seg">
          <button v-for="lv in (['full', 'reduced', 'off'] as EffectsLevel[])" :key="lv" :class="{ on: s.effects === lv }"
            @click="state.fxAutoReduced = false; saveSettings({ effects: lv })">{{ t('set.effects.' + lv) }}</button>
        </div>
      </div>
      <p class="hint">{{ t('set.effectsHint') }}</p>
      <div class="field inline">
        <span>{{ t('set.language') }}</span>
        <div class="seg">
          <button :class="{ on: s.language === 'auto' }" @click="saveSettings({ language: 'auto' })">{{ t('set.lang.auto') }}</button>
          <button :class="{ on: s.language === 'zh' }" @click="saveSettings({ language: 'zh' })">中文</button>
          <button :class="{ on: s.language === 'en' }" @click="saveSettings({ language: 'en' })">English</button>
        </div>
      </div>
    </section>

    <section class="panel sec">
      <h3>{{ t('set.player') }}</h3>
      <label class="check"><input type="checkbox" :checked="s.danmakuOverlay" @change="saveSettings({ danmakuOverlay: !s.danmakuOverlay })" /><span>{{ t('player.danmaku') }}</span></label>
      <div class="sliders">
        <label>{{ t('player.opacity') }} <span class="mono dim">{{ s.danmakuOpacity }}%</span><input type="range" min="10" max="100" :value="s.danmakuOpacity" @change="saveSettings({ danmakuOpacity: +($event.target as HTMLInputElement).value })" /></label>
        <label>{{ t('player.density') }} <span class="mono dim">{{ s.danmakuDensity }}%</span><input type="range" min="10" max="100" :value="s.danmakuDensity" @change="saveSettings({ danmakuDensity: +($event.target as HTMLInputElement).value })" /></label>
        <label>{{ t('player.size') }} <span class="mono dim">{{ s.danmakuSize }}%</span><input type="range" min="50" max="150" :value="s.danmakuSize" @change="saveSettings({ danmakuSize: +($event.target as HTMLInputElement).value })" /></label>
      </div>
    </section>

    <section class="panel sec">
      <h3>{{ t('set.about') }}</h3>
      <p class="mono dim">M4S Converter · {{ t('set.version', { v: state.init?.version ?? '' }) }} · {{ state.init?.platform }}</p>
      <p class="hint">{{ t('set.terms') }}</p>
      <p class="hint mono">github.com/crucibleu/FromM4StoMp4_A-Converter-With-User-friendly-GUI · MIT</p>
    </section>
  </div>
</template>

<style scoped>
.settings { padding: 18px 22px 30px; display: grid; gap: 14px; align-content: start; overflow-y: auto; height: 100%; max-width: 860px; }
h2 { margin: 0 0 4px; font-size: 20px; font-weight: 600; }
.sec { padding: 16px 18px; display: grid; gap: 12px; }
.sec h3 { margin: 0; font: 600 11.5px var(--font-mono); text-transform: uppercase; letter-spacing: 0.12em; color: var(--signal); }
.field { display: grid; gap: 6px; }
.field.inline { display: flex; align-items: center; justify-content: space-between; gap: 14px; }
.lbl { font-size: 12px; color: var(--dim); }
.row { display: flex; align-items: center; gap: 8px; min-width: 0; flex-wrap: wrap; }
.path { flex: 1; min-width: 0; font-size: 12px; padding: 7px 10px; background: var(--ink-2); border: 1px solid var(--line); border-radius: 8px; word-break: break-all; }
.fit { justify-self: start; }
.hint { margin: 0; font-size: 12px; color: var(--faint); }
.sliders { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.sliders label { display: grid; gap: 4px; font-size: 12px; }
p { margin: 0; }
</style>
