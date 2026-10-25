<script setup lang="ts">
import { Quit } from '../../wailsjs/runtime/runtime'
import { saveSettings } from '../store'
import { t } from '../i18n'
</script>

<template>
  <div class="scrim">
    <div class="dlg panel" role="dialog" aria-modal="true" aria-labelledby="terms-title">
      <h2 id="terms-title" class="display">{{ t('terms.title') }}</h2>
      <p>{{ t('terms.body') }}</p>
      <div class="row">
        <button class="btn ghost" @click="Quit()">{{ t('terms.quit') }}</button>
        <button class="btn pri" @click="saveSettings({ acceptedTerms: true })">{{ t('terms.agree') }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scrim { position: fixed; inset: 0; background: rgba(4, 7, 12, 0.7); backdrop-filter: blur(4px); display: grid; place-items: center; z-index: 60; }
.dlg { width: min(480px, 90vw); padding: 24px; display: grid; gap: 14px; background: var(--panel); box-shadow: 0 0 60px -20px rgba(56, 217, 245, 0.5); }
h2 { margin: 0; font-size: 20px; }
p { margin: 0; color: var(--dim); line-height: 1.7; }
.row { display: flex; justify-content: flex-end; gap: 10px; margin-top: 6px; }
</style>
