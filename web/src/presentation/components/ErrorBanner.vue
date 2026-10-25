<script setup>
import { t } from '../lib/i18n.js'

defineProps({
  message: { type: String, default: '' },
  detail: { type: String, default: '' },
})
defineEmits(['dismiss'])
</script>

<template>
  <div v-if="message" class="form-error banner">
    <div class="banner-text">
      <span>{{ message }}</span>
      <code v-if="detail" class="error-detail">{{ detail }}</code>
    </div>
    <button class="btn small ghost" @click="$emit('dismiss')">{{ t('action.hide') }}</button>
  </div>
</template>
