<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import StatusChip from './StatusChip.vue'
import { isCancellable, isRetryable } from '../../domain/value-objects/taskStatus.js'
import { taskProgress } from '../../domain/value-objects/progress.js'
import { fmtCost, fmtDuration, fmtTime, fmtTokens } from '../lib/format.js'
import { t } from '../lib/i18n.js'

const props = defineProps({
  detail: { type: Object, required: true },
})
const emit = defineEmits(['close', 'open', 'cancel', 'retry', 'answer'])

const note = ref('')
const expanded = ref(null)

const task = computed(() => props.detail.task)
const attempts = computed(() => props.detail.attempts ?? [])
const events = computed(() => props.detail.events ?? [])
const children = computed(() => props.detail.children ?? [])
const contextEntries = computed(() => props.detail.context ?? [])
const canCancel = computed(() => isCancellable(task.value.status))
const canRetry = computed(() => isRetryable(task.value.status))
const now = ref(Date.now())
const progress = computed(() => taskProgress(task.value, [task.value, ...children.value], now.value))
const waitingForModel = computed(
  () => ['planning', 'running', 'verifying'].includes(task.value.status) && !attempts.value.length,
)

let clockTimer = null
onMounted(() => {
  clockTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer)
})

function stageProgress(child) {
  return taskProgress(child, [child, ...children.value], now.value)
}

watch(
  () => task.value.id,
  () => {
    note.value = ''
    expanded.value = null
  },
)

function toggle(index) {
  expanded.value = expanded.value === index ? null : index
}

function submitAnswer() {
  const value = note.value.trim()
  if (!value) return
  emit('answer', value)
  note.value = ''
}
</script>

<template>
  <aside class="drawer">
    <div class="drawer-head">
      <div class="drawer-top">
        <StatusChip :status="task.status" />
        <span v-if="task.pipeline" class="chip">{{ task.pipeline }}</span>
        <span v-if="task.stage" class="chip">{{ task.stage }}</span>
        <button class="close" :title="t('action.close')" @click="$emit('close')">✕</button>
      </div>
      <h2 class="drawer-title">{{ task.title }}</h2>
      <div class="progress">
        <div class="progress-head">
          <span>{{ progress.percent }}%</span>
          <span v-if="progress.total" class="hint">{{ progress.finished }}/{{ progress.total }}</span>
          <span v-if="progress.active && progress.etaMs > 0" class="hint">
            {{ t('board.eta', { time: fmtDuration(progress.etaMs) }) }}
          </span>
          <span v-else-if="progress.elapsed > 0" class="hint">{{ fmtDuration(progress.elapsed) }}</span>
        </div>
        <div class="progress-bar"><i :style="{ width: progress.percent + '%' }"></i></div>
      </div>
      <div class="drawer-actions">
        <button v-if="canRetry" class="btn small" @click="$emit('retry', task.id)">
          {{ t('action.retry') }}
        </button>
        <button v-if="canCancel" class="btn small danger" @click="$emit('cancel', task.id)">
          {{ t('action.cancel') }}
        </button>
      </div>
    </div>

    <div class="drawer-body">
      <section>
        <div class="section-title">{{ t('drawer.task') }}</div>
        <dl class="kv">
          <dt>ID</dt><dd>{{ task.id }}</dd>
          <dt>{{ t('drawer.role') }}</dt><dd>{{ task.role }}</dd>
          <dt>{{ t('drawer.priority') }}</dt><dd>{{ task.priority }}</dd>
          <dt>{{ t('drawer.attempts') }}</dt><dd>{{ task.attemptCount }} / {{ task.maxAttempts }}</dd>
          <dt>{{ t('drawer.tokens') }}</dt>
          <dd>
            {{ fmtTokens(task.spentTokens) }}<span v-if="task.budgetTokens"> / {{ fmtTokens(task.budgetTokens) }}</span>
          </dd>
          <dt>{{ t('drawer.cost') }}</dt><dd>{{ fmtCost(task.costUsd) }}</dd>
          <dt>{{ t('drawer.created') }}</dt><dd>{{ fmtTime(task.createdAt) }}</dd>
          <dt>{{ t('drawer.started') }}</dt><dd>{{ fmtTime(task.startedAt) }}</dd>
          <dt>{{ t('drawer.finished') }}</dt><dd>{{ fmtTime(task.finishedAt) }}</dd>
          <template v-if="task.modelHint">
            <dt>{{ t('drawer.model') }}</dt><dd>{{ task.modelHint }}</dd>
          </template>
        </dl>
        <p v-if="task.error" class="hint drawer-error">{{ task.error }}</p>
        <p v-if="task.description" class="hint drawer-description">{{ task.description }}</p>
      </section>

      <section v-if="children.length">
        <div class="section-title">{{ t('drawer.stages') }}</div>
        <div class="stage-list">
          <div v-for="child in children" :key="child.id" class="stage" @click="$emit('open', child.id)">
            <StatusChip :status="child.status" />
            <span class="stage-name">{{ child.title }}</span>
            <span class="hint">{{ stageProgress(child).percent }}%</span>
            <span v-if="stageProgress(child).active" class="hint">
              {{ t('board.eta', { time: fmtDuration(stageProgress(child).etaMs) }) }}
            </span>
            <span class="hint">{{ fmtCost(child.costUsd) }}</span>
            <div class="progress-bar thin"><i :style="{ width: stageProgress(child).percent + '%' }"></i></div>
          </div>
        </div>
      </section>

      <section v-if="task.status === 'needs_human'">
        <div class="section-title">{{ t('drawer.human') }}</div>
        <textarea v-model="note" class="field full" :placeholder="t('drawer.humanPlaceholder')"></textarea>
        <button class="btn primary small" :disabled="!note.trim()" @click="submitAnswer">
          {{ t('action.continue') }}
        </button>
      </section>

      <section v-if="contextEntries.length">
        <div class="section-title">{{ t('drawer.contextTitle') }}</div>
        <div class="context-list">
          <div v-for="entry in contextEntries" :key="entry.id" class="context-entry">
            <div class="context-head">
              <span class="chip">{{ t('drawer.contextKind.' + entry.kind) }}</span>
              <span class="context-name">{{ entry.title }}</span>
              <span class="hint">{{ fmtTokens(entry.tokens) }}</span>
            </div>
            <pre class="out short">{{ entry.body }}</pre>
          </div>
        </div>
      </section>
      <section v-else>
        <div class="section-title">{{ t('drawer.contextTitle') }}</div>
        <div class="empty">{{ t('drawer.contextEmpty') }}</div>
      </section>

      <section v-if="task.result">
        <div class="section-title">{{ t('drawer.result') }}</div>
        <pre class="out">{{ task.result }}</pre>
      </section>

      <section>
        <div class="section-title">{{ t('drawer.attemptsTitle', { count: attempts.length }) }}</div>
        <div v-if="waitingForModel" class="hint model-waiting">{{ t('drawer.modelWorking') }}</div>
        <table v-if="attempts.length" class="table">
          <thead>
            <tr>
              <th>#</th>
              <th>{{ t('drawer.agent') }}</th>
              <th>{{ t('drawer.model') }}</th>
              <th>{{ t('drawer.tokens') }}</th>
              <th>{{ t('drawer.price') }}</th>
              <th>{{ t('drawer.time') }}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <template v-for="(attempt, index) in attempts" :key="attempt.id">
              <tr class="clickable" @click="toggle(index)">
                <td>{{ attempt.attemptNo }}</td>
                <td>{{ attempt.agent }}</td>
                <td>{{ attempt.model }}</td>
                <td>
                  {{ fmtTokens(attempt.inputTokens) }} / {{ fmtTokens(attempt.outputTokens) }}
                  <span v-if="attempt.cacheReadTokens" class="hint">
                    (+{{ fmtTokens(attempt.cacheReadTokens) }} {{ t('drawer.cache') }})
                  </span>
                </td>
                <td>{{ fmtCost(attempt.costUsd) }}</td>
                <td>{{ fmtDuration(attempt.latencyMs) }}</td>
                <td>
                  <span
                    class="chip"
                    :class="attempt.status === 'ok' ? 'done' : attempt.status === 'running' ? 'running' : 'failed'"
                  >
                    {{ attempt.status === 'running' ? t('drawer.running') : attempt.status }}
                  </span>
                </td>
              </tr>
              <tr class="attempt-log">
                <td colspan="7">
                  <div v-if="attempt.status === 'running'" class="hint">{{ t('drawer.attemptRunning') }}</div>
                  <div v-else-if="attempt.error" class="form-error">{{ attempt.error }}</div>
                  <pre v-else-if="attempt.output" class="out short">{{ attempt.output }}</pre>
                  <div v-else class="hint">{{ t('drawer.noOutput') }}</div>
                  <template v-if="expanded === index">
                    <div class="section-title spaced">{{ t('drawer.prompt') }}</div>
                    <pre class="out short">{{ attempt.prompt }}</pre>
                  </template>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
        <div v-else class="empty">{{ t('drawer.noAttempts') }}</div>
      </section>

      <section>
        <div class="section-title">{{ t('drawer.log') }}</div>
        <div class="timeline">
          <div v-for="event in events" :key="event.id" class="tl-item">
            <span class="tl-time">{{ fmtTime(event.createdAt) }}</span>
            <span class="tl-msg">
              <StatusChip v-if="event.toStatus" :status="event.toStatus" class="tl-chip" />
              {{ event.message || event.type }}
            </span>
          </div>
          <div v-if="!events.length" class="empty">{{ t('drawer.noEvents') }}</div>
        </div>
      </section>
    </div>
  </aside>
</template>
