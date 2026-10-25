import { normalizeModels, toRoutePayload } from '../../domain/entities/route.js'
import { normalizeUsage } from '../../domain/entities/modelUsage.js'

export function createHttpModelRepository(http) {
  return {
    async list() {
      return normalizeModels(await http.get('/models'))
    },
    async usage() {
      return normalizeUsage(await http.get('/models/usage'))
    },
    async usageDaily() {
      const raw = await http.get('/models/usage/daily')
      const items = Array.isArray(raw.items) ? raw.items : []
      const mapped = items.map((item) => ({
        provider: item.provider ?? '',
        model: item.model ?? '',
        calls: Number(item.calls ?? 0),
        inputTokens: Number(item.input_tokens ?? 0),
        outputTokens: Number(item.output_tokens ?? 0),
        tokens: Number(item.tokens ?? 0),
        costUsd: Number(item.cost_usd ?? 0),
        avgLatencyMs: Number(item.avg_latency_ms ?? 0),
        share: Number(item.share ?? 0),
        todayTokens: Number(item.today_tokens ?? item.todayTokens ?? 0),
        todayCalls: Number(item.today_calls ?? item.todayCalls ?? 0),
      }))
      return {
        items: mapped,
        totalTokens: Number(raw.daily_tokens ?? raw.total_tokens ?? 0),
        totalCost: Number(raw.daily_cost ?? raw.total_cost ?? 0),
      }
    },
    async health() {
      const raw = await http.get('/models/health')
      const map = raw && typeof raw === 'object' && !Array.isArray(raw) ? raw : {}
      const health = {}
      Object.entries(map).forEach(([key, value]) => {
        health[key] = {
          status: value?.status === 'error' ? 'error' : 'ok',
          lastError: value?.last_error ?? '',
          lastErrorAt: value?.last_error_at ?? null,
        }
      })
      return health
    },
    async saveRoute(payload) {
      await http.post('/models', payload)
    },
    async deleteRoute({ role, position }) {
      await http.post('/models/entries', { role, position })
    },
    async moveRoute({ role, from, to }) {
      await http.post('/models/move', { role, from, to })
    },
  }
}
