<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'
import { saveSettings, scan, state } from '../store'
import { t } from '../i18n'

const emit = defineEmits<{ (e: 'close'): void }>()
const s = state.settings!
const form = reactive({
  outputDir: s.outputDir, template: s.template, danmaku: s.danmaku,
  downloadDanmaku: s.downloadDanmaku, overwrite: s.overwrite,
})
const presets = [
  { key: 'default', tpl: '{series}-{uploader}/{part}' },
  { key: 'series', tpl: '{series}/{page}-{part}' },
  { key: 'flat', tpl: '{part}' },
  { key: 'bvid', tpl: '{series}/{part}-{bvid}' },
]
const example = ref('')
const defaultDir = () => (s.cacheDir ? s.cacheDir.replace(/[\\/]+$/, '') + (s.cacheDir.includes('\\') ? '\\' : '/') + 'output' : '')

async function updateExample() {
  const id = state.focusedId || state.videos.find((v) => v.status === 'ready')?.id || ''
  example.value = await api.outputExample(id, form.outputDir || defaultDir(), form.template)
}
watch(() => [form.outputDir, form.template], updateExample)
onMounted(updateExample)

async function browse() {
  const p = await api.pickFolder(t('out.folder'), form.outputDir || defaultDir())
  if (p) form.outputDir = p
}

async function save() {
  const changed = form.outputDir !== s.outputDir || form.template !== s.template
  await saveSettings({ ...form })
  emit('close')
  if (changed) scan() // 重新判断哪些视频已经转换
}
</script>

<template>
  <div class="scrim" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="dlg panel" role="dialog" aria-modal="true" aria-labelledby="out-title">
      <h2 id="out-title" class="display">{{ t('out.title') }}</h2>

      <label class="field">
        <span class="lbl">{{ t('out.folder') }}</span>
        <div class="row">
          <input v-model="form.outputDir" class="input mono" :placeholder="t('out.default')" spellcheck="false" />
          <button class="btn" @click="browse">{{ t('out.browse') }}</button>
        </div>
      </label>

      <div class="field">
        <span class="lbl">{{ t('out.template') }}</span>
        <div class="presets">
          <button v-for="p in presets" :key="p.key" class="chip" :class="{ on: form.template === p.tpl }" @click="form.template = p.tpl">
            {{ t('out.preset.' + p.key) }}
          </button>
        </div>
        <input v-model="form.template" class="input mono" spellcheck="false" />
        <span class="hint">{{ t('out.tokens', { tokens: '{series} {part} {uploader} {page} {bvid} {cid}' }) }}</span>
        <div class="example mono selectable"><span class="dim">{{ t('out.example') }} →</span> {{ example }}</div>
      </div>

      <label class="check"><input v-model="form.danmaku" type="checkbox" /><span>{{ t('out.danmaku') }}</span></label>
      <label class="check" :class="{ disabled: !form.danmaku }"><input v-model="form.downloadDanmaku" type="checkbox" :disabled="!form.danmaku" /><span>{{ t('out.download') }}</span></label>
      <label class="check"><input v-model="form.overwrite" type="checkbox" /><span>{{ t('out.overwrite') }}</span></label>

      <div class="row end">
        <button class="btn ghost" @click="emit('close')">{{ t('out.cancel') }}</button>
        <button class="btn pri" @click="save">{{ t('out.save') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scrim { position: fixed; inset: 0; background: rgba(4, 7, 12, 0.65); backdrop-filter: blur(4px); display: grid; place-items: center; z-index: 40; }
.dlg { width: min(620px, 92vw); padding: 22px; display: grid; gap: 16px; background: var(--panel); box-shadow: 0 0 60px -24px rgba(56, 217, 245, 0.55); }
h2 { margin: 0; font-size: 18px; }
.field { display: grid; gap: 8px; }
.lbl { font: 600 11px var(--font-mono); text-transform: uppercase; letter-spacing: 0.1em; color: var(--dim); }
.row { display: flex; gap: 8px; align-items: center; }
.row .input { flex: 1; }
.row.end { justify-content: flex-end; margin-top: 4px; }
.presets { display: flex; flex-wrap: wrap; gap: 6px; }
.hint { font-size: 11.5px; color: var(--faint); }
.example { font-size: 11.5px; padding: 8px 10px; border-radius: 8px; background: var(--ink-2); border: 1px dashed var(--line-2); word-break: break-all; }
.disabled { opacity: 0.5; }
</style>
