<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import gsap from 'gsap'
import Icon from '../components/Icon.vue'
import VideoCard from '../components/VideoCard.vue'
import DetailPanel from '../components/DetailPanel.vue'
import OutputDialog from '../components/OutputDialog.vue'
import { api } from '../api'
import { effects, enqueueSelected, pickCacheDir, saveSettings, scan, selectedVideos, state } from '../store'
import { errText, t } from '../i18n'
import { bytes } from '../format'
import type { Video } from '../types'

type Filter = 'all' | 'ready' | 'converted' | 'incomplete' | 'broken' | 'danmaku' | 'hd'
const filter = ref<Filter>('all')
const query = ref('')
const grouped = ref(true)
const sortBy = ref<'series' | 'size' | 'duration'>('series')
const outputPath = ref('')
const gridEl = ref<HTMLElement | null>(null)
const sweepEl = ref<HTMLElement | null>(null)
let lastIndex = -1

const isSelectable = (v: Video) => v.status === 'ready' || v.status === 'converted'
const matchers: Record<Filter, (v: Video) => boolean> = {
  all: () => true,
  ready: (v) => v.status === 'ready',
  converted: (v) => v.status === 'converted',
  incomplete: (v) => v.status === 'incomplete',
  broken: (v) => v.status === 'broken',
  danmaku: (v) => !!v.danmakuFile,
  hd: (v) => Math.min(v.video.height, v.video.width || v.video.height) >= 1000,
}
const counts = computed(() => {
  const c = {} as Record<Filter, number>
  for (const f of Object.keys(matchers) as Filter[]) c[f] = state.videos.filter(matchers[f]).length
  return c
})
const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  let list = state.videos.filter(matchers[filter.value])
  if (q) list = list.filter((v) => `${v.series} ${v.part} ${v.uploader} ${v.bvid}`.toLowerCase().includes(q))
  if (sortBy.value === 'size') list = [...list].sort((a, b) => b.size - a.size)
  if (sortBy.value === 'duration') list = [...list].sort((a, b) => b.duration - a.duration)
  return list
})
const groups = computed(() => {
  if (!grouped.value) return [{ name: '', videos: visible.value }]
  const m = new Map<string, Video[]>()
  for (const v of visible.value) {
    const k = v.series + (v.uploader ? ` · ${v.uploader}` : '')
    if (!m.has(k)) m.set(k, [])
    m.get(k)!.push(v)
  }
  return [...m.entries()].map(([name, videos]) => ({ name, videos }))
})
const selSize = computed(() => selectedVideos.value.reduce((s, v) => s + v.size, 0))
const lowSpace = computed(() => state.freeBytes > 0 && selSize.value * 1.05 > state.freeBytes)
const readyCount = computed(() => state.videos.filter((v) => v.status === 'ready').length)

function pick(v: Video, ev: MouseEvent) {
  const i = visible.value.indexOf(v)
  state.focusedId = v.id
  if (ev.shiftKey && lastIndex >= 0) {
    const [a, b] = [Math.min(lastIndex, i), Math.max(lastIndex, i)]
    for (const x of visible.value.slice(a, b + 1)) if (isSelectable(x)) state.selected.add(x.id)
  } else if (ev.ctrlKey || ev.metaKey || ev.detail === 2) {
    toggle(v)
  }
  lastIndex = i
}
function toggle(v: Video) {
  if (!isSelectable(v)) return
  state.focusedId = v.id
  if (state.selected.has(v.id)) state.selected.delete(v.id)
  else state.selected.add(v.id)
  lastIndex = visible.value.indexOf(v)
}
function groupAllSelected(list: Video[]) {
  const sel = list.filter(isSelectable)
  return sel.length > 0 && sel.every((v) => state.selected.has(v.id))
}
function toggleGroup(list: Video[]) {
  const on = !groupAllSelected(list)
  for (const v of list) {
    if (!isSelectable(v)) continue
    if (on && v.status === 'ready') state.selected.add(v.id)
    else if (!on) state.selected.delete(v.id)
  }
}
function selectAllReady() {
  for (const v of visible.value) if (v.status === 'ready') state.selected.add(v.id)
}

async function updateOutputPath() {
  const s = state.settings
  if (!s) return
  outputPath.value = await api.outputExample(state.focusedId || selectedVideos.value[0]?.id || '', state.outputDir, s.template)
}
watch(() => [state.outputDir, state.settings?.template, state.focusedId, state.selected.size], updateOutputPath, { immediate: true })

async function useDetected(dir: string) {
  await saveSettings({ cacheDir: dir })
  scan()
}

// 扫描时一道扫描线划过网格；扫描完成后卡片依次出现
let sweep: gsap.core.Tween | null = null
watch(() => state.scanning, async (on) => {
  sweep?.kill()
  if (on && effects.value !== 'off') {
    await nextTick()
    if (sweepEl.value) {
      sweep = gsap.fromTo(sweepEl.value, { xPercent: -100, opacity: 1 }, {
        xPercent: 100, duration: effects.value === 'full' ? 1.4 : 2.4, ease: 'none', repeat: -1,
      })
    }
  }
  if (!on) {
    if (sweepEl.value) gsap.to(sweepEl.value, { opacity: 0, duration: 0.3 })
    await nextTick()
    animateCards()
  }
})
function animateCards() {
  if (effects.value === 'off' || !gridEl.value) return
  const cards = [...gridEl.value.querySelectorAll('.card')].slice(0, 48)
  gsap.fromTo(cards, { opacity: 0, y: 14, scale: 0.98 }, {
    opacity: 1, y: 0, scale: 1, duration: effects.value === 'full' ? 0.45 : 0.25,
    stagger: effects.value === 'full' ? 0.018 : 0.006, ease: 'power3.out', clearProps: 'all',
  })
}
onBeforeUnmount(() => sweep?.kill())
</script>

<template>
  <div class="lib">
    <section class="main">
      <!-- 还没有选择缓存目录，或目录不存在 -->
      <div v-if="!state.settings?.cacheDir || state.scanError" class="empty hero">
        <div class="ring"><Icon name="drop" /></div>
        <h2>{{ t('lib.empty.title') }}</h2>
        <p>{{ state.scanError ? errText(state.scanError) : t('lib.empty.body') }}</p>
        <button class="btn pri" @click="pickCacheDir"><Icon name="folder" />{{ t('lib.empty.pick') }}</button>
        <div v-if="state.init?.detectedDirs.length" class="detected">
          <span class="dim">{{ t('lib.empty.detected') }}</span>
          <button v-for="d in state.init.detectedDirs" :key="d" class="chip" @click="useDetected(d)">{{ d }}</button>
        </div>
      </div>

      <template v-else>
        <div class="toolbar">
          <label class="search">
            <Icon name="search" />
            <input v-model="query" class="input" :placeholder="t('lib.search')" spellcheck="false" />
          </label>
          <div class="seg">
            <button :class="{ on: grouped }" @click="grouped = true">{{ t('lib.group.series') }}</button>
            <button :class="{ on: !grouped }" @click="grouped = false">{{ t('lib.group.none') }}</button>
          </div>
          <select v-model="sortBy" class="input sort" :aria-label="t('lib.sort.series')">
            <option value="series">{{ t('lib.sort.series') }}</option>
            <option value="size">{{ t('lib.sort.size') }}</option>
            <option value="duration">{{ t('lib.sort.duration') }}</option>
          </select>
        </div>
        <div class="filters">
          <button
            v-for="f in (['all', 'ready', 'converted', 'incomplete', 'broken', 'danmaku', 'hd'] as Filter[])" :key="f"
            v-show="f === 'all' || counts[f] > 0" class="chip" :class="{ on: filter === f }" @click="filter = f"
          >
            {{ t('lib.filter.' + f) }} <span class="n">{{ counts[f] }}</span>
          </button>
        </div>

        <div ref="gridEl" class="scroll">
          <div ref="sweepEl" class="sweep" aria-hidden="true" />
          <div v-if="state.scanning && !state.videos.length" class="empty"><p>{{ t('lib.scanning') }}</p></div>
          <div v-else-if="!state.videos.length" class="empty">
            <h2>{{ t('lib.none.title') }}</h2>
            <p>{{ t('lib.none.body') }}</p>
            <button class="btn" @click="pickCacheDir"><Icon name="folder" />{{ t('top.change') }}</button>
          </div>
          <div v-else-if="!visible.length" class="empty"><p>{{ t('lib.noMatch') }}</p></div>

          <section v-for="g in groups" :key="g.name" class="group">
            <header v-if="g.name" class="group-h">
              <h4 class="ellipsis" :title="g.name">{{ g.name }}</h4>
              <span class="mono dim">{{ t('lib.parts', { n: g.videos.length }) }} · {{ bytes(g.videos.reduce((s, v) => s + v.size, 0)) }}</span>
              <button v-if="g.videos.some(isSelectable)" class="link" @click="toggleGroup(g.videos)">
                {{ groupAllSelected(g.videos) ? t('lib.unselectGroup') : t('lib.selectGroup') }}
              </button>
            </header>
            <div class="grid">
              <VideoCard
                v-for="v in g.videos" :key="v.id" :video="v"
                :selected="state.selected.has(v.id)" :focused="state.focusedId === v.id"
                @pick="(e) => pick(v, e)" @toggle="toggle(v)"
              />
            </div>
          </section>
        </div>

        <footer class="selbar" :class="{ active: state.selected.size > 0 }">
          <template v-if="state.selected.size">
            <span class="n mono">{{ t('lib.selected', { n: state.selected.size }) }}</span>
            <span class="mono dim">{{ bytes(selSize) }}</span>
            <button class="link" @click="state.selected.clear()">{{ t('lib.clearSel') }}</button>
          </template>
          <button v-else-if="readyCount" class="link" @click="selectAllReady">{{ t('lib.selectAllReady') }} ({{ readyCount }})</button>
          <span v-if="lowSpace" class="warn"><Icon name="warn" />{{ t('lib.noSpace', { need: bytes(selSize), free: bytes(state.freeBytes) }) }}</span>
          <span class="sp" />
          <span class="out mono ellipsis" :title="outputPath">{{ t('lib.outputTo') }} → {{ outputPath }}</span>
          <button class="btn ghost" @click="state.showOutputDialog = true">{{ t('lib.change') }}</button>
          <button class="btn pri" :disabled="!state.selected.size" @click="enqueueSelected()">
            <Icon name="convert" />{{ t('lib.convertSelected') }}
          </button>
        </footer>
      </template>
    </section>
    <DetailPanel />
    <OutputDialog v-if="state.showOutputDialog" @close="state.showOutputDialog = false" />
  </div>
</template>

<style scoped>
.lib { display: grid; grid-template-columns: minmax(0, 1fr) 320px; height: 100%; min-height: 0; }
.main { display: grid; grid-template-rows: auto auto 1fr auto; gap: 12px; padding: 14px 16px 12px; min-width: 0; min-height: 0; }
.main > .hero { grid-row: 1 / -1; }
.toolbar { display: flex; gap: 8px; align-items: center; }
.search { flex: 1; position: relative; display: flex; }
.search svg { position: absolute; left: 10px; top: 50%; width: 15px; height: 15px; transform: translateY(-50%); color: var(--faint); }
.search .input { flex: 1; padding-left: 32px; }
.sort { padding: 6px 8px; }
.filters { display: flex; gap: 6px; flex-wrap: wrap; }
.scroll { overflow-y: auto; min-height: 0; position: relative; padding-right: 4px; display: grid; align-content: start; gap: 18px; }
.sweep {
  position: absolute; inset: 0; pointer-events: none; opacity: 0; z-index: 3;
  background: linear-gradient(90deg, transparent 0%, rgba(56, 217, 245, 0) 40%, rgba(56, 217, 245, 0.16) 49%, rgba(56, 217, 245, 0.9) 50%, transparent 50.3%);
}
.group { display: grid; gap: 10px; }
.group-h { display: flex; align-items: baseline; gap: 12px; min-width: 0; }
.group-h h4 { margin: 0; font-size: 13px; font-weight: 600; min-width: 0; }
.group-h .mono { font-size: 11px; white-space: nowrap; }
.link { border: 0; background: none; color: var(--signal); cursor: pointer; font-size: 12px; padding: 0; white-space: nowrap; }
.link:hover { text-decoration: underline; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(178px, 1fr)); gap: 12px; }
.selbar {
  display: flex; align-items: center; gap: 12px; border-radius: 11px; padding: 8px 10px 8px 14px;
  background: var(--panel); border: 1px solid var(--line); min-width: 0; transition: border-color 0.2s, box-shadow 0.3s;
}
.selbar.active { border-color: var(--signal-line); box-shadow: 0 0 26px -14px var(--signal); background: var(--panel-2); }
.selbar .n { color: var(--signal); font-weight: 600; }
.selbar .sp { flex: 1; }
.selbar .out { font-size: 11px; color: var(--dim); max-width: 38%; }
.warn { display: inline-flex; gap: 6px; align-items: center; color: var(--warn); font-size: 12px; }
.warn svg { width: 14px; height: 14px; }
.hero { align-content: center; }
.ring {
  width: 74px; height: 74px; border-radius: 50%; display: grid; place-items: center; color: var(--signal);
  border: 1px solid var(--signal-line); box-shadow: 0 0 40px -10px var(--signal), inset 0 0 22px -10px var(--signal);
}
.ring svg { width: 30px; height: 30px; }
.detected { display: flex; flex-wrap: wrap; gap: 8px; justify-content: center; align-items: center; max-width: 640px; }
</style>
