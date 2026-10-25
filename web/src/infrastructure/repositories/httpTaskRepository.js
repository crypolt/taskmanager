import {
  normalizeStats,
  normalizeTask,
  normalizeTaskDetail,
} from '../../domain/entities/task.js'
import { normalizePlan } from '../../domain/entities/plan.js'

function asArray(payload) {
  if (Array.isArray(payload)) return payload
  if (Array.isArray(payload?.items)) return payload.items
  if (Array.isArray(payload?.tasks)) return payload.tasks
  return []
}

const encode = (id) => encodeURIComponent(id)

export function createHttpTaskRepository(http) {
  return {
    async list() {
      return asArray(await http.get('/tasks')).map(normalizeTask)
    },
    async detail(id) {
      return normalizeTaskDetail(await http.get(`/tasks/${encode(id)}`))
    },
    async create(payload) {
      return normalizeTask(await http.post('/tasks', payload))
    },
    async cancel(id) {
      await http.post(`/tasks/${encode(id)}/cancel`)
    },
    async retry(id) {
      await http.post(`/tasks/${encode(id)}/retry`)
    },
    async move(id, status) {
      await http.post(`/tasks/${encode(id)}/move`, { status })
    },
    async answer(id, note) {
      await http.post(`/tasks/${encode(id)}/answer`, { note })
    },
    async previewPlan(payload) {
      return normalizePlan(await http.post('/plans/preview', payload, { timeout: 0 }))
    },
    async createFromPlan(payload) {
      return normalizeTask(await http.post('/plans', payload))
    },
    async stats() {
      return normalizeStats(await http.get('/stats'))
    },
  }
}
