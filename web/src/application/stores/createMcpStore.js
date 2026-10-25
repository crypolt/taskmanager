import { computed, reactive, ref } from 'vue'

export function createMcpStore({ mcpRepository }) {
  const path = ref('')
  const servers = ref([])
  const loading = ref(false)
  const error = ref('')
  const errorDetail = ref('')

  const count = computed(() => servers.value.length)
  const enabledCount = computed(() => servers.value.filter((server) => server.enabled).length)

  async function load() {
    loading.value = true
    try {
      apply(await mcpRepository.list())
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      error.value = err?.message || String(err)
      errorDetail.value = err?.detail ?? ''
    } finally {
      loading.value = false
    }
  }

  async function save(server, isNew = false) {
    try {
      const payload = server
      apply(isNew ? await mcpRepository.create(payload) : await mcpRepository.update(server.name, payload))
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      error.value = err?.message || String(err)
      errorDetail.value = err?.detail ?? ''
      throw err
    }
  }

  async function remove(name) {
    try {
      apply(await mcpRepository.remove(name))
      error.value = ''
      errorDetail.value = ''
    } catch (err) {
      error.value = err?.message || String(err)
      errorDetail.value = err?.detail ?? ''
      throw err
    }
  }

  function apply(config) {
    path.value = config.path
    servers.value = config.servers
  }

  function dismissError() {
    error.value = ''
    errorDetail.value = ''
  }

  return reactive({ path, servers, loading, error, errorDetail, count, enabledCount, load, save, remove, dismissError })
}
