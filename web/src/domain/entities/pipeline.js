export function normalizePipeline(raw = {}) {
  return {
    id: raw.id ?? '',
    name: raw.name ?? '',
    about: raw.about ?? '',
    stages: Array.isArray(raw.stages)
      ? raw.stages.map((stage) => ({
          key: stage.key ?? '',
          role: stage.role ?? '',
          title: stage.title ?? '',
          brief: stage.brief ?? '',
        }))
      : [],
  }
}
