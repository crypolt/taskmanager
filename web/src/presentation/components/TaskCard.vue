<script setup>
import { computed } from 'vue'
import StatusChip from './StatusChip.vue'
import { taskProgress } from '../../domain/value-objects/progress.js'
import { fmtCost, fmtDuration, fmtTime } from '../lib/format.js'
import { t } from '../lib/i18n.js'

const props = defineProps({
  task: { type: Object, required: true },
  selected: { type: Boolean, default: false },
  all: { type: Array, default: () => [] },
  now: { type: Number, default: 0 },
})
defineEmits(['open', 'dragstart'])

const progress = computed(() =>
  taskProgress(props.task, props.all.length ? props.all : [props.task], props.now || Date.now()),
)
</script>

<template>
  <div
    class="card"
    :class="{ selected }"
    draggable="true"
    @click="$emit('open', task.id)"
    @dragstart="$emit('dragstart', task.id, $event)"
  >
    <div class="card-title">{{ task.title }}</div>
    <div class="card-meta">
      <StatusChip :status="task.status" />
      <span v-if="task.pipeline" class="chip">{{ task.pipeline }}</span>
      <span v-if="task.stage" class="chip">{{ task.stage }}</span>
      <span class="dot"></span>
      <span>{{ fmtCost(task.costUsd) }}</span>
      <span class="dot"></span>
      <span>{{ fmtTime(task.createdAt) }}</span>
    </div>
    <div v-if="progress.active || progress.percent > 0" class="progress">
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
    <div v-if="task.error" class="hint card-error">
      {{ task.error.length > 120 ? task.error.slice(0, 120) + '…' : task.error }}
    </div>
  </div>
</template>
