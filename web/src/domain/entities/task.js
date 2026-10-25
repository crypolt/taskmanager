import { BoardColumn } from '../value-objects/taskStatus.js'

export function normalizeTask(raw = {}) {
  return {
    id: String(raw.id ?? ''),
    parentId: raw.parent_id ?? null,
    pipeline: raw.pipeline ?? '',
    stage: raw.stage ?? '',
    role: raw.role ?? '',
    title: raw.title ?? '',
    description: raw.description ?? '',
    status: raw.status ?? 'pending',
    priority: Number(raw.priority ?? 0),
    dependsOn: Array.isArray(raw.depends_on) ? raw.depends_on : [],
    modelHint: raw.model_hint ?? '',
    result: raw.result ?? '',
    error: raw.error ?? '',
    budgetTokens: Number(raw.budget_tokens ?? 0),
    spentTokens: Number(raw.spent_tokens ?? 0),
    costUsd: Number(raw.cost_usd ?? 0),
    attemptCount: Number(raw.attempt_count ?? 0),
    maxAttempts: Number(raw.max_attempts ?? 0),
    humanNote: raw.human_note ?? '',
    createdAt: raw.created_at ?? null,
    updatedAt: raw.updated_at ?? null,
    startedAt: raw.started_at ?? null,
    finishedAt: raw.finished_at ?? null,
  }
}

export function isRootTask(task) {
  return !task.parentId
}

export function byPriorityThenAge(a, b) {
  if (b.priority !== a.priority) return b.priority - a.priority
  return String(a.createdAt ?? '').localeCompare(String(b.createdAt ?? ''))
}

export function groupByColumns(tasks) {
  const sorted = [...tasks].sort(byPriorityThenAge)
  return BoardColumn.map((column) => ({
    ...column,
    tasks: sorted.filter((task) => column.statuses.includes(task.status)),
  }))
}

export function normalizeTaskDetail(raw = {}) {
  return {
    task: normalizeTask(raw.task),
    children: Array.isArray(raw.children) ? raw.children.map(normalizeTask) : [],
    attempts: Array.isArray(raw.attempts) ? raw.attempts.map(normalizeAttempt) : [],
    events: Array.isArray(raw.events) ? raw.events.map(normalizeEvent) : [],
    context: Array.isArray(raw.context) ? raw.context.map(normalizeContextEntry) : [],
  }
}

function normalizeContextEntry(raw = {}) {
  return {
    id: String(raw.id ?? ''),
    kind: raw.kind ?? 'note',
    title: raw.title ?? '',
    body: raw.body ?? '',
    tokens: Number(raw.tokens ?? 0),
    createdAt: raw.created_at ?? null,
  }
}

function normalizeAttempt(raw = {}) {
  return {
    id: String(raw.id ?? ''),
    taskId: raw.task_id ?? '',
    attemptNo: Number(raw.attempt_no ?? 0),
    role: raw.role ?? '',
    agent: raw.agent ?? '',
    provider: raw.provider ?? '',
    model: raw.model ?? '',
    status: raw.status ?? '',
    verdict: raw.verdict ?? '',
    score: Number(raw.score ?? 0),
    inputTokens: Number(raw.input_tokens ?? 0),
    outputTokens: Number(raw.output_tokens ?? 0),
    costUsd: Number(raw.cost_usd ?? 0),
    latencyMs: Number(raw.latency_ms ?? 0),
    prompt: raw.prompt ?? '',
    output: raw.output ?? '',
    issues: raw.issues ?? '',
    error: raw.error ?? '',
    createdAt: raw.created_at ?? null,
  }
}

function normalizeEvent(raw = {}) {
  return {
    id: Number(raw.id ?? 0),
    taskId: raw.task_id ?? '',
    type: raw.type ?? '',
    fromStatus: raw.from_status ?? '',
    toStatus: raw.to_status ?? '',
    message: raw.message ?? '',
    createdAt: raw.created_at ?? null,
  }
}

export function normalizeStats(raw = {}) {
  return {
    total: Number(raw.total ?? 0),
    running: Number(raw.running ?? 0),
    done: Number(raw.done ?? 0),
    failed: Number(raw.failed ?? 0),
    needsHuman: Number(raw.needs_human ?? 0),
    costUsd: Number(raw.cost_usd ?? 0),
    tokens: Number(raw.tokens ?? 0),
  }
}
