<script setup lang="ts">
import { ref } from 'vue'
import type { Video } from '../types'
import { coverURL } from '../api'
import { bytes, codecLabel, duration, placeholderBg, resLabel } from '../format'
import { t } from '../i18n'

const props = defineProps<{ video: Video; selected: boolean; focused: boolean }>()
const emit = defineEmits<{ (e: 'pick', ev: MouseEvent): void; (e: 'toggle', ev: MouseEvent): void }>()
const coverOk = ref(true)
const selectable = () => props.video.status === 'ready' || props.video.status === 'converted'
</script>

<template>
  <article
    class="card" :class="{ sel: selected, focus: focused, off: !selectable() }"
    tabindex="0" :aria-selected="selected"
    @click="emit('pick', $event)" @keydown.space.prevent="emit('toggle', $event as any)" @keydown.enter="emit('pick', $event as any)"
  >
    <div class="thumb" :style="{ background: placeholderBg(video.series) }">
      <span class="initial display">{{ (video.series || video.part).slice(0, 1) }}</span>
      <img v-if="coverOk && (video.coverFile || video.coverUrl)" :src="coverURL(video.id)" alt="" loading="lazy" @error="coverOk = false" />
      <button
        v-if="selectable()" class="ck" :class="{ on: selected }" :aria-label="video.part"
        @click.stop="emit('toggle', $event)"
      />
      <span v-if="video.video.height" class="res mono">{{ resLabel(video.video.height, video.video.width) }}</span>
      <span v-if="video.video.codec && video.video.codec !== 'avc'" class="codec mono">{{ codecLabel(video.video.codec) }}</span>
      <span class="dur mono">{{ duration(video.duration) }}</span>
      <span v-if="video.status === 'incomplete' && video.progress >= 0" class="dl"><i :style="{ width: video.progress + '%' }" /></span>
    </div>
    <div class="info">
      <div class="t ellipsis" :title="video.part">{{ video.page > 0 ? `P${video.page} ` : '' }}{{ video.part }}</div>
      <div class="s mono">
        <span>{{ video.status === 'incomplete' && video.progress >= 0 ? video.progress + '%' : bytes(video.size) }}</span>
        <span class="st" :class="video.status">{{ t('status.' + video.status) }}</span>
      </div>
    </div>
  </article>
</template>

<style scoped>
.card {
  background: var(--panel); border: 1px solid var(--line); border-radius: 10px; overflow: hidden; cursor: pointer;
  display: grid; transition: border-color 0.18s, box-shadow 0.25s var(--ease), transform 0.25s var(--ease); outline: none;
}
.card:hover { border-color: var(--line-2); transform: translateY(-2px); box-shadow: 0 10px 24px -14px rgba(0, 0, 0, 0.8); }
.fx-off .card:hover { transform: none; }
.card:focus-visible { border-color: var(--signal); }
.card.focus { border-color: var(--line-2); box-shadow: 0 0 0 1px var(--line-2); }
.card.sel { border-color: rgba(56, 217, 245, 0.7); box-shadow: 0 0 0 1px rgba(56, 217, 245, 0.35), 0 0 24px -8px rgba(56, 217, 245, 0.65); }
.card.off .thumb { filter: saturate(0.4) brightness(0.75); }
.thumb { aspect-ratio: 16 / 9; position: relative; display: grid; place-items: center; overflow: hidden; }
.thumb img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
.initial { font-size: 34px; font-weight: 700; color: rgba(255, 255, 255, 0.35); }
.ck {
  position: absolute; left: 7px; top: 7px; width: 18px; height: 18px; border-radius: 5px; padding: 0;
  border: 1.5px solid rgba(255, 255, 255, 0.75); background: rgba(0, 0, 0, 0.35); cursor: pointer; z-index: 1;
  display: grid; place-items: center; transition: background 0.15s, border-color 0.15s, transform 0.15s;
}
.ck:hover { transform: scale(1.1); }
.ck.on { background: var(--signal); border-color: var(--signal); box-shadow: 0 0 10px var(--signal); }
.ck.on::after { content: ''; width: 5px; height: 9px; border: solid #04121a; border-width: 0 2px 2px 0; transform: translateY(-1px) rotate(45deg); }
.res, .codec, .dur { position: absolute; bottom: 6px; font-size: 10px; border-radius: 4px; padding: 0 5px; line-height: 16px; }
.res { left: 6px; color: var(--signal); border: 1px solid rgba(56, 217, 245, 0.55); background: rgba(0, 0, 0, 0.5); font-weight: 600; }
.codec { left: 52px; color: var(--pink); border: 1px solid rgba(251, 114, 153, 0.5); background: rgba(0, 0, 0, 0.5); font-weight: 600; }
.dur { right: 6px; background: rgba(0, 0, 0, 0.62); }
.dl { position: absolute; left: 0; right: 0; bottom: 0; height: 3px; background: rgba(0, 0, 0, 0.5); }
.dl i { display: block; height: 100%; background: var(--warn); }
.info { padding: 8px 10px 10px; display: grid; gap: 5px; min-width: 0; }
.t { font-size: 12.5px; font-weight: 600; }
.s { display: flex; justify-content: space-between; align-items: center; gap: 6px; font-size: 10.5px; color: var(--dim); }
</style>
