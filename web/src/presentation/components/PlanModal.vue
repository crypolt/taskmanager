<script setup>
import { ref } from 'vue'
import { emptyPlanStep, PLAN_ROLES } from '../../domain/entities/plan.js'
import { t } from '../lib/i18n.js'

const props = defineProps({
  preview: { type: Function, required: true },
  create: { type: Function, required: true },
})
const emit = defineEmits(['close', 'created'])

const stage = ref('form')
const title = ref('')
const context = ref('')
const plan = ref({ model: '', provider: '', subtasks: [] })
const generating = ref(false)
const creating = ref(false)
const error = ref('')
const errorDetail = ref('')

function clearError() {
  error.value = ''
  errorDetail.value = ''
}

function fail(err) {
  error.value = err?.message || String(err)
  errorDetail.value = err?.detail ?? ''
}

function scopeText(step) {
  return (step.scope ?? []).join(', ')
}

function setScope(step, value) {
  step.scope = value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

async function generate() {
  if (!context.value.trim()) {
    clearError()
    error.value = t('plan.contextRequired')
    return
  }
  clearError()
  generating.value = true
  try {
    plan.value = await props.preview({ title: title.value.trim(), context: context.value.trim() })
    if (!title.value.trim()) title.value = plan.value.title
    stage.value = 'plan'
  } catch (err) {
    fail(err)
  } finally {
    generating.value = false
  }
}

function addStep() {
  plan.value.subtasks.push(emptyPlanStep())
}

function removeStep(index) {
  plan.value.subtasks.splice(index, 1)
}

async function submit() {
  const steps = plan.value.subtasks.filter((step) => step.title.trim())
  if (!steps.length) {
    clearError()
    error.value = t('plan.empty')
    return
  }
  clearError()
  creating.value = true
  try {
    await props.create({
      title: title.value.trim(),
      context: context.value.trim(),
      subtasks: steps.map((step) => ({
        title: step.title.trim(),
        description: step.description,
        role: step.role,
        scope: step.scope ?? [],
        depends_on: [],
      })),
    })
    emit('created')
  } catch (err) {
    fail(err)
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="modal-backdrop" @click.self="$emit('close')">
    <div class="modal wide">
      <h2>{{ t('plan.title') }}</h2>
      <div v-if="error" class="form-error">
        <span>{{ error }}</span>
        <code v-if="errorDetail" class="error-detail">{{ errorDetail }}</code>
      </div>

      <template v-if="stage === 'form'">
        <div class="field">
          <label>{{ t('plan.context') }}</label>
          <textarea
            v-model="context"
            class="tall"
            :placeholder="t('plan.contextPlaceholder')"
            autofocus
          ></textarea>
        </div>
        <div class="field">
          <label>{{ t('plan.name') }}</label>
          <input v-model="title" :placeholder="t('plan.namePlaceholder')" />
        </div>
        <p class="hint">{{ t('plan.hint') }}</p>
        <div class="modal-actions">
          <button class="btn ghost" @click="$emit('close')">{{ t('action.cancel') }}</button>
          <button class="btn primary" :disabled="generating" @click="generate">
            {{ generating ? t('plan.generating') : t('plan.generate') }}
          </button>
        </div>
      </template>

      <template v-else>
        <p class="hint">{{ t('plan.generated', { model: plan.model || '—' }) }}</p>
        <div class="plan-steps">
          <div v-for="(step, index) in plan.subtasks" :key="index" class="plan-step">
            <div class="plan-step-head">
              <span class="idx">{{ index + 1 }}</span>
              <select v-model="step.role">
                <option v-for="role in PLAN_ROLES" :key="role" :value="role">{{ t('role.' + role) }}</option>
              </select>
              <input v-model="step.title" class="grow" :placeholder="t('plan.subtaskTitle')" />
              <button class="btn small ghost" :title="t('plan.remove')" @click="removeStep(index)">✕</button>
            </div>
            <textarea
              v-model="step.description"
              :placeholder="t('plan.subtaskDescription')"
            ></textarea>
            <input
              :value="scopeText(step)"
              :placeholder="t('plan.scopePlaceholder')"
              @input="setScope(step, $event.target.value)"
            />
          </div>
        </div>
        <button class="btn small ghost plan-add" @click="addStep">{{ t('plan.addSubtask') }}</button>
        <div class="modal-actions">
          <button class="btn ghost" @click="stage = 'form'">{{ t('action.back') }}</button>
          <button class="btn primary" :disabled="creating" @click="submit">
            {{ creating ? t('plan.creating') : t('plan.createTasks', { count: plan.subtasks.length }) }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>
