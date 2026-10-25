export function normalizeUsage(raw = {}) {
  const items = Array.isArray(raw.items) ? raw.items : []
  return {
    items: items.map((item) => ({
      provider: item.provider ?? '',
      model: item.model ?? '',
      calls: Number(item.calls ?? 0),
      inputTokens: Number(item.input_tokens ?? 0),
      outputTokens: Number(item.output_tokens ?? 0),
      tokens: Number(item.tokens ?? 0),
      cacheReadTokens: Number(item.cache_read_tokens ?? 0),
      todayTokens: Number(item.today_tokens ?? 0),
      todayCalls: Number(item.today_calls ?? 0),
      costUsd: Number(item.cost_usd ?? 0),
      avgLatencyMs: Number(item.avg_latency_ms ?? 0),
      share: Number(item.share ?? 0),
    })),
    totalTokens: Number(raw.total_tokens ?? 0),
    totalCost: Number(raw.total_cost ?? 0),
  }
}
