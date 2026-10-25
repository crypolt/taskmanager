<script setup>
import KanbanBoard from '../components/KanbanBoard.vue'
import { useContainer } from '../../app/container.js'
import { t, tn } from '../lib/i18n.js'

const { tasks } = useContainer()
</script>

<template>
  <div class="board-view">
    <div class="board-toolbar">
      <button class="btn small" @click="tasks.toggleStages()">
        {{ tasks.showStages ? t('board.onlyRoots') : t('board.showStages') }}
      </button>
      <input
        v-model="tasks.query"
        class="search"
        type="search"
        :placeholder="t('board.searchPlaceholder')"
      />
      <span v-if="tasks.loading" class="hint">{{ t('board.loading') }}</span>
      <span class="hint">{{ tn('board.tasks', tasks.visibleItems.length) }}</span>
    </div>
    <KanbanBoard
      :columns="tasks.columns"
      :selected-id="tasks.selectedId"
      :all="tasks.items"
      :now="tasks.now"
      @open="tasks.open"
      @move="tasks.move"
    />
  </div>
</template>
