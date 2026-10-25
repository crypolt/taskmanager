<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useContainer } from './container.js'
import AppHeader from '../presentation/components/AppHeader.vue'
import ErrorBanner from '../presentation/components/ErrorBanner.vue'
import TaskDrawer from '../presentation/components/TaskDrawer.vue'
import NewTaskModal from '../presentation/components/NewTaskModal.vue'
import PlanModal from '../presentation/components/PlanModal.vue'

const { tasks, pipelines } = useContainer()
const router = useRouter()
const showNew = ref(false)
const showPlan = ref(false)
const creating = ref(false)

onMounted(async () => {
  await Promise.all([tasks.start(), pipelines.load()])
})

onBeforeUnmount(() => tasks.stop())

async function createTask(payload) {
  creating.value = true
  try {
    await tasks.create(payload)
    showNew.value = false
    if (router.currentRoute.value.name !== 'board') await router.push('/board')
  } catch {
    /* the store exposes the error */
  } finally {
    creating.value = false
  }
}

async function onPlanCreated() {
  showPlan.value = false
  if (router.currentRoute.value.name !== 'board') await router.push('/board')
}
</script>

<template>
  <div class="app">
    <AppHeader
      :stats="tasks.stats"
      :connected="tasks.connected"
      @new-task="showNew = true"
      @new-plan="showPlan = true"
    />
    <ErrorBanner :message="tasks.error" :detail="tasks.errorDetail" @dismiss="tasks.dismissError" />
    <div class="layout">
      <router-view />
      <TaskDrawer
        v-if="tasks.detail"
        :detail="tasks.detail"
        @close="tasks.close()"
        @open="tasks.open"
        @cancel="tasks.cancel"
        @retry="tasks.retry"
        @answer="tasks.answer"
      />
    </div>
    <NewTaskModal
      v-if="showNew"
      :pipelines="pipelines.items"
      :busy="creating"
      @close="showNew = false"
      @create="createTask"
    />
    <PlanModal
      v-if="showPlan"
      :preview="tasks.previewPlan"
      :create="tasks.createPlan"
      @close="showPlan = false"
      @created="onPlanCreated"
    />
  </div>
</template>
