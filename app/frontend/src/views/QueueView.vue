<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import gsap from 'gsap'
import Icon from '../components/Icon.vue'
import { api, coverURL } from '../api'
import { effects, jobList, playOutput, saveSettings, state, videoById } from '../store'
import { t } from '../i18n'
import { bytes, duration, placeholderBg } from '../format'
import type { Job } from '../types'

const visibleJobs = jobList
const stats = computed(() => {
  const all = jobList.value.filter((j) => j.state !== 'canceled')
  const done = all.filter((j) => j.state === 'done' || j.state === 'skipped').length
  const failed = all.filter((j) => j.state === 'failed').length
  const total = all.reduce((s, j) => s + j.size, 0)
  const finished = all.reduce((s, j) => s + j.size * j.progress, 0)
  return { done, total: all.length, failed, bytesLeft: Math.max(0, total - finished) }
})
const speed = computed(() => state.tick?.speed ?? 0)
const eta = computed(() => (speed.value > 0 ? stats.value.bytesLeft / speed.value : 0))

// 数字滚动显示
const shown = reactive({ done: 0, speed: 0 })
watch(() => [stats.value.done, speed.value], ([d, s]) => {
  const dur = effects.value === 'off' ? 0 : 0.6
  gsap.to(shown, { done: d, speed: s, duration: dur, ease: 'power2.out' })
}, { immediate: true })

function stageText(j: Job) {
  if (j.state === 'running') return t('job.' + (j.stage || 'prepare'))
  if (j.state === 'failed') return j.error
  return t('job.' + j.state)
}
const cover = (j: Job) => {
  const v = videoById.value.get(j.videoId)
  return v && (v.coverFile || v.coverUrl) ? coverURL(v.id) : ''
}
async function clear() {
  const jobs = await api.clearFinished()
  state.jobs.clear()
  for (const j of jobs) state.jobs.set(j.id, j)
}
</script>

<template>
  <div class="queue">
    <div class="toolbar">
      <h2 class="display">{{ t('queue.title') }}</h2>
      <span class="sp" />
      <label class="workers">
        <span class="dim">{{ t('queue.workers') }}</span>
        <div class="seg">
          <button v-for="n in 4" :key="n" :class="{ on: state.settings?.workers === n }" @click="saveSettings({ workers: n })">{{ n }}</button>
        </div>
      </label>
      <button class="btn ghost danger" :disabled="!jobList.some((j) => j.state === 'running' || j.state === 'queued')" @click="api.cancelAll()">
        <Icon name="x" />{{ t('queue.cancelAll') }}
      </button>
      <button class="btn ghost" :disabled="!jobList.some((j) => !['running', 'queued'].includes(j.state))" @click="clear">{{ t('queue.clear') }}</button>
      <button class="btn" @click="api.openPath(state.outputDir)"><Icon name="folder" />{{ t('queue.openOutput') }}</button>
    </div>

    <div v-if="!jobList.length" class="empty">
      <h2>{{ t('queue.empty.title') }}</h2>
      <p>{{ t('queue.empty.body') }}</p>
      <button class="btn pri" @click="state.view = 'library'"><Icon name="library" />{{ t('queue.toLibrary') }}</button>
    </div>

    <template v-else>
      <div class="stats">
        <div class="stat"><span class="v display">{{ Math.round(shown.done) }} / {{ stats.total }}</span><span class="l">{{ t('queue.done') }}</span></div>
        <div class="stat"><span class="v display">{{ shown.speed > 0 ? bytes(shown.speed) + '/s' : '—' }}</span><span class="l">{{ t('queue.speed') }}</span></div>
        <div class="stat"><span class="v display">{{ eta > 0 ? duration(eta) : '—' }}</span><span class="l">{{ t('queue.left') }}</span></div>
        <div class="stat" :class="{ bad: stats.failed }"><span class="v display">{{ stats.failed }}</span><span class="l">{{ t('queue.failed') }}</span></div>
      </div>

      <TransitionGroup tag="div" name="job" class="jobs">
        <div v-for="j in visibleJobs" :key="j.id" class="job" :class="j.state">
          <div class="mini" :style="{ background: placeholderBg(j.series) }">
            <img v-if="cover(j)" :src="cover(j)" alt="" loading="lazy" @error="($event.target as HTMLElement).remove()" />
          </div>
          <div class="txt">
            <div class="t ellipsis" :title="j.part">{{ j.part }}</div>
            <div class="sub mono ellipsis" :title="j.state === 'failed' ? j.error : j.output">
              {{ stageText(j) }}<template v-if="j.state !== 'failed'"> · {{ bytes(j.size) }} → {{ j.output }}</template>
            </div>
          </div>
          <div class="bar" :class="{ live: j.state === 'running' }"><i :style="{ width: Math.round(j.progress * 100) + '%' }" /></div>
          <div class="pct mono">{{ j.state === 'queued' ? '—' : Math.round(j.progress * 100) + '%' }}</div>
          <div class="acts">
            <button v-if="j.state === 'running' || j.state === 'queued'" class="btn icon ghost" :title="t('job.cancel')" @click="api.cancelJob(j.id)"><Icon name="x" /></button>
            <button v-if="j.state === 'failed' || j.state === 'canceled'" class="btn icon ghost" :title="t('job.retry')" @click="api.retryJob(j.id)"><Icon name="retry" /></button>
            <template v-if="j.state === 'done' || j.state === 'skipped'">
              <button class="btn icon ghost" :title="t('job.play')" @click="playOutput(j.output)"><Icon name="play" /></button>
              <button class="btn icon ghost" :title="t('job.reveal')" @click="api.revealPath(j.output)"><Icon name="reveal" /></button>
            </template>
          </div>
        </div>
      </TransitionGroup>
    </template>
  </div>
</template>

<style scoped>
.queue { padding: 16px 18px; display: grid; grid-template-rows: auto auto 1fr; gap: 14px; height: 100%; min-height: 0; }
.toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.toolbar h2 { margin: 0; font-size: 18px; font-weight: 600; }
.sp { flex: 1; }
.workers { display: inline-flex; align-items: center; gap: 8px; font-size: 12px; }
.empty { grid-row: 2 / -1; align-content: center; }
.stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
.stat { background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: 12px 14px; display: grid; gap: 2px; }
.stat .v { font-size: 22px; font-weight: 600; font-variant-numeric: tabular-nums; }
.stat .l { font: 10.5px var(--font-mono); color: var(--dim); text-transform: uppercase; letter-spacing: 0.08em; }
.stat.bad .v { color: var(--err); }
.jobs { display: grid; gap: 8px; align-content: start; overflow-y: auto; min-height: 0; padding-right: 4px; position: relative; }
.job {
  display: grid; grid-template-columns: 72px minmax(0, 1fr) minmax(120px, 200px) 46px auto; gap: 14px; align-items: center;
  background: var(--panel); border: 1px solid var(--line); border-radius: 10px; padding: 8px 10px 8px 8px;
}
.job.running { border-color: rgba(56, 217, 245, 0.35); }
.job.failed { border-color: rgba(255, 93, 108, 0.4); }
.job.failed .pct, .job.failed .sub { color: var(--err); }
.job.done .bar > i, .job.skipped .bar > i { background: var(--ok); box-shadow: none; }
.job.done .pct { color: var(--ok); }
.job.canceled { opacity: 0.55; }
.mini { aspect-ratio: 16 / 9; border-radius: 6px; overflow: hidden; position: relative; }
.mini img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.txt { min-width: 0; display: grid; gap: 3px; }
.t { font-weight: 600; font-size: 13px; }
.sub { font-size: 11px; color: var(--dim); }
.pct { text-align: right; font-weight: 600; font-size: 12px; }
.acts { display: flex; gap: 2px; min-width: 72px; justify-content: flex-end; }
.job-enter-active, .job-leave-active, .job-move { transition: all 0.35s var(--ease); }
.job-enter-from { opacity: 0; transform: translateX(-12px); }
.job-leave-to { opacity: 0; }
.job-leave-active { position: absolute; width: 100%; }
.fx-off .job-enter-active, .fx-off .job-leave-active, .fx-off .job-move { transition: none; }
</style>
