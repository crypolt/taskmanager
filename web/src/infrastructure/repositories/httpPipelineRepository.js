import { normalizePipeline } from '../../domain/entities/pipeline.js'

export function createHttpPipelineRepository(http) {
  return {
    async list() {
      const payload = await http.get('/pipelines')
      const items = Array.isArray(payload) ? payload : (payload?.items ?? payload?.pipelines ?? [])
      return items.map(normalizePipeline)
    },
  }
}
