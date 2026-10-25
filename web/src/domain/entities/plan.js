export const PLAN_ROLES = ['analyze', 'design', 'implement', 'test', 'review', 'docs', 'generic']

export function normalizePlanStep(raw = {}) {
  return {
    title: raw.title ?? '',
    description: raw.description ?? '',
    role: PLAN_ROLES.includes(raw.role) ? raw.role : 'generic',
    scope: Array.isArray(raw.scope) ? raw.scope : [],
    dependsOn: Array.isArray(raw.depends_on) ? raw.depends_on : [],
  }
}

export function normalizePlan(raw = {}) {
  return {
    title: raw.title ?? '',
    context: raw.context ?? '',
    provider: raw.provider ?? '',
    model: raw.model ?? '',
    subtasks: Array.isArray(raw.subtasks) ? raw.subtasks.map(normalizePlanStep) : [],
  }
}

export function emptyPlanStep() {
  return { title: '', description: '', role: 'implement', scope: [], dependsOn: [] }
}
