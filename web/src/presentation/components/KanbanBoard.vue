<script setup>
import { ref } from 'vue'
import KanbanColumn from './KanbanColumn.vue'
import { dropStatus } from '../../domain/value-objects/taskStatus.js'

defineProps({
  columns: { type: Array, required: true },
  selectedId: { type: String, default: null },
  all: { type: Array, default: () => [] },
  now: { type: Number, default: 0 },
})
const emit = defineEmits(['open', 'move'])

const draggingId = ref(null)

function onDragStart(id, event) {
  draggingId.value = id
  if (event?.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', id)
  }
}

function onDrop(columnKey) {
  const id = draggingId.value
  draggingId.value = null
  const status = dropStatus(columnKey)
  if (id && status) emit('move', id, status)
}

function onDragEnd() {
  draggingId.value = null
}
</script>

<template>
  <div class="board" @dragend="onDragEnd">
    <KanbanColumn
      v-for="column in columns"
      :key="column.key"
      :column="column"
      :selected-id="selectedId"
      :all="all"
      :now="now"
      :dragging="Boolean(draggingId)"
      @open="$emit('open', $event)"
      @dragstart="onDragStart"
      @move="onDrop"
    />
  </div>
</template>
