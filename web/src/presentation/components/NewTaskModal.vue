<script setup>
import { computed, reactive, ref } from 'vue'
import { t } from '../lib/i18n.js'

const props = defineProps({
  pipelines: { type: Array, default: () => [] },
  busy: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'create'])

const form = reactive({
  title: '',
  description: '',
  pipeline: props.pipelines[0]?.id ?? 'free',
  model: '',
  priority: 0,
  max_attempts: 3,
  budget_tokens: 0,
})
const error = ref('')

const selectedPipeline = computed(() => props.pipelines.find((pipeline) => pipeline.id === form.pipeline))

function submit() {
  if (!form.title.trim()) {
    error.value = t('validation.titleRequired')
    return
  }
  error.value = ''
  emit('create', {
    title: form.title.trim(),
    description: form.description,
    pipeline: form.pipeline,
    model: form.model.trim(),
    priority: Number(form.priority) || 0,
    max_attempts: Number(form.max_attempts) || 3,
    budget_tokens: Number(form.budget_tokens) || 0,
  })
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal">
      <h2>{{ t('modal.title') }}</h2>
      <div v-if="error" class="form-error">{{ error }}</div>
      <div class="field">
        <label>{{ t('field.title') }}</label>
        <input v-model="form.title" :placeholder="t('field.titlePlaceholder')" autofocus />
      </div>
      <div class="field">
        <label>{{ t('field.description') }}</label>
        <textarea v-model="form.description" :placeholder="t('field.descriptionPlaceholder')"></textarea>
      </div>
      <div class="row">
        <div class="field">
          <label>{{ t('field.pipeline') }}</label>
          <select v-model="form.pipeline">
            <option v-for="pipeline in pipelines" :key="pipeline.id" :value="pipeline.id">{{ pipeline.name }}</option>
          </select>
        </div>
        <div class="field">
          <label>{{ t('field.model') }}</label>
          <input v-model="form.model" :placeholder="t('field.modelPlaceholder')" />
        </div>
      </div>
      <div class="row">
        <div class="field">
          <label>{{ t('field.priority') }}</label>
          <input v-model.number="form.priority" type="number" min="0" max="10" />
        </div>
        <div class="field">
          <label>{{ t('field.attempts') }}</label>
          <input v-model.number="form.max_attempts" type="number" min="1" max="5" />
        </div>
        <div class="field">
          <label>{{ t('field.budget') }}</label>
          <input v-model.number="form.budget_tokens" type="number" min="0" step="10000" />
        </div>
      </div>
      <p class="hint">
        {{ selectedPipeline?.about || t('hint.pipelineFallback') }}
        {{ t('hint.budget') }}
      </p>
      <div class="modal-actions">
        <button class="btn ghost" @click="$emit('close')">{{ t('action.cancel') }}</button>
        <button class="btn primary" :disabled="busy" @click="submit">{{ t('action.create') }}</button>
      </div>
    </div>
  </div>
</template>
