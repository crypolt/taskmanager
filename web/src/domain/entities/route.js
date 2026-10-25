export function normalizeRoute(raw = {}) {
  return {
    role: raw.role ?? '',
    position: Number(raw.position ?? 0),
    provider: raw.provider ?? '',
    model: raw.model ?? '',
    priceIn: Number(raw.price_in ?? 0),
    priceOut: Number(raw.price_out ?? 0),
    dailyTokens: Number(raw.daily_tokens ?? 0),
  }
}

export function toRoutePayload(route) {
  return {
    role: route.role,
    position: Number(route.position),
    provider: route.provider,
    model: route.model,
    price_in: Number(route.priceIn) || 0,
    price_out: Number(route.priceOut) || 0,
    daily_tokens: Number(route.dailyTokens) || 0,
  }
}

export function normalizeModels(raw = {}) {
  const routes = {}
  Object.entries(raw.routes ?? {}).forEach(([role, chain]) => {
    routes[role] = [...(chain ?? [])].map(normalizeRoute).sort((a, b) => a.position - b.position)
  })
  return {
    routes,
    providers: Array.isArray(raw.providers) ? raw.providers : [],
  }
}

export function flattenRoutes(models) {
  const out = []
  Object.keys(models.routes ?? {})
    .sort()
    .forEach((role) => out.push(...models.routes[role]))
  return out
}
