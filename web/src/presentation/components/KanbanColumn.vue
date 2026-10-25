<script setup>
import { computed, ref } from 'vue'
import TaskCard from './TaskCard.vue'
import { dropStatus } from '../../domain/value-objects/taskStatus.js'
import { t } from '../lib/i18n.js'

const props = defineProps({
  column: { type: Object, required: true },
  selectedId: { type: String, default: null },
  all: { type: Array, default: () => [] },
  now: { type: Number, default: 0 },
  dragging: { type: Boolean, default: false },
})
const emit = defineEmits(['open', 'dragstart', 'move'])

const over = ref(false)
const droppable = computed(() => Boolean(dropStatus(props.column.key)))

function onOver() {
  if (droppable.value) over.value = true
}

function onLeave() {
  over.value = false
}

function onDrop() {
  over.value = false
  if (!droppable.value) return
  emit('move', props.column.key)
}
</script>

<template>
  <div
    class="column"
    :class="{ 'drop-target': over && droppable, 'drop-available': dragging && droppable }"
    @dragover.prevent="onOver"
    @dragleave="onLeave"
    @drop.prevent="onDrop"
  >
    <div class="column-head">
      {{ t(column.titleKey) }}
      <span class="count">{{ column.tasks.length }}</span>
    </div>
    <div class="column-body">
      <TaskCard
        v-for="task in column.tasks"
        :key="task.id"
        :task="task"
        :selected="task.id === selectedId"
        :all="all"
        :now="now"
        @open="$emit('open', $event)"
        @dragstart="$emit('dragstart', $event)"
      />
      <div v-if="!column.tasks.length" class="empty">{{ t('column.empty') }}</div>
    </div>
  </div>
</template>
