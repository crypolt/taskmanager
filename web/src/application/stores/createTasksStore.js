import { computed, reactive, ref } from 'vue'
import { groupByColumns, isRootTask, normalizeStats, normalizeTask } from '../../domain/entities/task.js'

const EMPTY_STATS = normalizeStats()

export function createTasksStore({ taskRepository, realtime }) {
  const items = ref([])
  const stats = ref({ ...EMPTY_STATS })
  const detail = ref(null)
  const selectedId = ref(null)
  const loading = ref(false)
  const error = ref('')
  const errorDetail = ref('')
  const showStages = ref(false)
  const connected = ref(false)
  const query = ref('')
  const now = ref(Date.now())

  let detailTimer = null
  let statsTimer = null
  let unsubscribeRealtime = null
  let unsubscribeConnection = null
  let clockTimer = null
  let started = false

  const searchedItems = computed(() => {
    const needle = query.value.trim().toLowerCase()
    if (!needle) return items.value
    return items.value.filter((task) =>
      [task.title, task.description, task.id, task.role, task.stage]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(needle)),
    )
  })
  const visibleItems = computed(() =>
    showStages.value ? searchedItems.value : searchedItems.value.filter(isRootTask),
  )
  const columns = computed(() => groupByColumns(visibleItems.value))
  const selectedTask = computed(() => detail.value?.task ?? null)
  const hasError = computed(() => error.value !== '')

  function fail(err) {
    error.value = err?.message || String(err)
    errorDetail.value = err?.detail ?? ''
  }

  function dismissError() {
    error.value = ''
    errorDetail.value = ''
  }

  function upsert(task) {
    const index = items.value.findIndex((item) => item.id === task.id)
    if (index >= 0) items.value[index] = task
    else items.value.push(task)
  }

  async function load() {
    loading.value = true
    try {
      const [list, nextStats] = await Promise.all([taskRepository.list(), taskRepository.stats()])
      items.value = list
      stats.value = nextStats
      if (!selectedId.value && list.length) selectedId.value = list[0].id
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      fail(err)
    } finally {
      loading.value = false
    }
  }

  async function loadStats() {
    try {
      stats.value = await taskRepository.stats()
    } catch {
      /* stats are advisory; keep the last value */
    }
  }

  async function open(id) {
    if (!id) return
    selectedId.value = id
    try {
      detail.value = await taskRepository.detail(id)
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      fail(err)
    }
  }

  function close() {
    selectedId.value = null
    detail.value = null
  }

  function scheduleStats() {
    if (statsTimer) return
    statsTimer = setTimeout(() => {
      statsTimer = null
      loadStats()
    }, 700)
  }

  function scheduleDetail(id) {
    if (detailTimer) clearTimeout(detailTimer)
    detailTimer = setTimeout(() => {
      detailTimer = null
      if (selectedId.value === id) open(id)
    }, 400)
  }

  async function create(payload) {
    error.value = ''
    errorDetail.value = ''
    try {
      const task = await taskRepository.create(payload)
      await Promise.all([load(), open(task.id)])
      return task
    } catch (err) {
      fail(err)
      throw err
    }
  }

  async function previewPlan(payload) {
    error.value = ''
    errorDetail.value = ''
    try {
      return await taskRepository.previewPlan(payload)
    } catch (err) {
      fail(err)
      throw err
    }
  }

  async function createPlan(payload) {
    error.value = ''
    errorDetail.value = ''
    try {
      const task = await taskRepository.createFromPlan(payload)
      await Promise.all([load(), open(task.id)])
      return task
    } catch (err) {
      fail(err)
      throw err
    }
  }

  async function cancel(id) {
    try {
      await taskRepository.cancel(id)
      await open(id)
    } catch (err) {
      fail(err)
    }
  }

  async function retry(id) {
    try {
      await taskRepository.retry(id)
      await open(id)
    } catch (err) {
      fail(err)
    }
  }

  async function move(id, status) {
    try {
      await taskRepository.move(id, status)
      await Promise.all([load(), id === selectedId.value ? open(id) : Promise.resolve()])
    } catch (err) {
      fail(err)
    }
  }

  async function answer(note) {
    if (!selectedId.value) return
    try {
      await taskRepository.answer(selectedId.value, note)
      await open(selectedId.value)
    } catch (err) {
      fail(err)
    }
  }

  function toggleStages() {
    showStages.value = !showStages.value
  }

  function onRealtimeEvent(event) {
    if (event.type === 'task') {
      const task = normalizeTask(event.data)
      upsert(task)
      if (task.id === selectedId.value) scheduleDetail(task.id)
      scheduleStats()
      return
    }
    if ((event.type === 'attempt' || event.type === 'event') && event.data?.task_id === selectedId.value) {
      scheduleDetail(event.data.task_id)
    }
  }

  async function start() {
    if (started) return
    started = true
    unsubscribeRealtime = realtime.subscribe(onRealtimeEvent)
    clockTimer = setInterval(() => {
      now.value = Date.now()
    }, 1000)
    unsubscribeConnection = realtime.onConnection((value) => {
      connected.value = value
    })
    await load()
    if (selectedId.value) await open(selectedId.value)
  }

  function stop() {
    started = false
    if (unsubscribeRealtime) unsubscribeRealtime()
    if (unsubscribeConnection) unsubscribeConnection()
    if (detailTimer) clearTimeout(detailTimer)
    if (statsTimer) clearTimeout(statsTimer)
    if (clockTimer) clearInterval(clockTimer)
    unsubscribeRealtime = null
    unsubscribeConnection = null
    detailTimer = null
    statsTimer = null
    clockTimer = null
  }

  return reactive({
    items,
    stats,
    detail,
    selectedId,
    loading,
    error,
    errorDetail,
    showStages,
    connected,
    query,
    now,
    visibleItems,
    columns,
    selectedTask,
    hasError,
    load,
    loadStats,
    open,
    close,
    create,
    previewPlan,
    createPlan,
    cancel,
    retry,
    move,
    answer,
    toggleStages,
    dismissError,
    start,
    stop,
  })
}
