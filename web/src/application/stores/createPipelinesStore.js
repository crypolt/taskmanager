import { computed, reactive, ref } from 'vue'

export function createPipelinesStore({ pipelineRepository }) {
  const items = ref([])
  const loading = ref(false)
  const error = ref('')

  const byId = computed(() =>
    items.value.reduce((acc, pipeline) => {
      acc[pipeline.id] = pipeline
      return acc
    }, {}),
  )

  async function load() {
    if (items.value.length) return items.value
    loading.value = true
    try {
      items.value = await pipelineRepository.list()
      error.value = ''
    } catch (err) {
      error.value = err?.message || String(err)
    } finally {
      loading.value = false
    }
    return items.value
  }

  return reactive({ items, loading, error, byId, load })
}
