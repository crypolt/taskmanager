<script setup>
import { computed } from 'vue'
import { statusLabelKey } from '../../domain/value-objects/taskStatus.js'
import { t } from '../lib/i18n.js'

const props = defineProps({
  status: { type: String, default: '' },
  label: { type: String, default: '' },
})

const text = computed(() => {
  if (props.label) return props.label
  const key = statusLabelKey(props.status)
  return key ? t(key) : props.status
})
</script>

<template>
  <span class="chip" :class="status">{{ text }}</span>
</template>
