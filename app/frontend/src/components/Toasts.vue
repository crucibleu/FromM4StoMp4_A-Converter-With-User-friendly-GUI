<script setup lang="ts">
import { toasts } from '../store'
</script>

<template>
  <div class="toasts" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="x in toasts" :key="x.id" class="toast" :class="x.kind">{{ x.text }}</div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toasts { position: fixed; right: 18px; top: 60px; display: grid; gap: 8px; z-index: 50; pointer-events: none; }
.toast {
  max-width: 380px; padding: 10px 14px; border-radius: 9px; font-size: 12.5px;
  background: rgba(23, 32, 51, 0.95); border: 1px solid var(--line-2); box-shadow: 0 12px 30px -12px rgba(0, 0, 0, 0.6);
  border-left: 3px solid var(--signal);
}
.toast.ok { border-left-color: var(--ok); }
.toast.err { border-left-color: var(--err); }
.toast-enter-active, .toast-leave-active { transition: all 0.25s var(--ease); }
.toast-enter-from { opacity: 0; transform: translateX(20px); }
.toast-leave-to { opacity: 0; transform: translateY(-6px); }
</style>
