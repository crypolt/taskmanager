import { computed, reactive, ref } from 'vue'
import { flattenRoutes, toRoutePayload } from '../../domain/entities/route.js'

export function createModelsStore({ modelRepository }) {
  const models = ref({ routes: {}, providers: [] })
  const usage = ref({ items: [], totalTokens: 0, totalCost: 0 })
  const daily = ref({ items: [], totalTokens: 0, totalCost: 0 })
  const health = ref({})
  const loading = ref(false)
  const error = ref('')
  const errorDetail = ref('')
  const busy = ref('')

  const roles = computed(() => Object.keys(models.value.routes ?? {}).sort())
  const rows = computed(() => flattenRoutes(models.value))
  const providers = computed(() => models.value.providers ?? [])
  const hasError = computed(() => error.value !== '')

  function fail(err) {
    error.value = err?.message || String(err)
    errorDetail.value = err?.detail ?? ''
  }

  async function load() {
    loading.value = true
    try {
      const [list, usageData] = await Promise.all([modelRepository.list(), modelRepository.usage()])
      models.value = list
      usage.value = usageData
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      fail(err)
    } finally {
      loading.value = false
    }
  }

  async function loadDaily() {
    try {
      daily.value = await modelRepository.usageDaily()
    } catch (err) {
      fail(err)
    }
  }

  async function loadHealth() {
    try {
      health.value = (await modelRepository.health()) ?? {}
    } catch {
      /* health is advisory: keep the previous snapshot */
    }
  }

  async function save(route) {
    busy.value = `${route.role}:${route.position}`
    try {
      await modelRepository.saveRoute(toRoutePayload(route))
      await load()
      error.value = ''
      errorDetail.value = ''
      return true
    } catch (err) {
      fail(err)
      return false
    } finally {
      busy.value = ''
    }
  }

  async function removeRoute({ role, position }) {
    busy.value = `${role}:${position}`
    try {
      await modelRepository.deleteRoute({ role, position })
      await load()
      return true
    } catch (err) {
      fail(err)
      return false
    } finally {
      busy.value = ''
    }
  }

  async function moveRoute({ role, from, to }) {
    busy.value = `${role}:${from}`
    try {
      await modelRepository.moveRoute({ role, from, to })
      await load()
      return true
    } catch (err) {
      fail(err)
      return false
    } finally {
      busy.value = ''
    }
  }

  function dismissError() {
    error.value = ''
    errorDetail.value = ''
  }

  return reactive({
    models,
    usage,
    daily,
    health,
    loading,
    error,
    errorDetail,
    busy,
    roles,
    rows,
    providers,
    hasError,
    load,
    loadDaily,
    loadHealth,
    save,
    removeRoute,
    moveRoute,
    dismissError,
  })
}
