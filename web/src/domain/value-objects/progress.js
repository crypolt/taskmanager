const TERMINAL = new Set(['done', 'failed', 'cancelled'])
const ACTIVE = new Set(['planning', 'running', 'verifying', 'retrying'])
const DEFAULT_MS = 90_000
const MIN_MS = 20_000
const MAX_MS = 15 * 60_000

function parse(value) {
  if (!value) return 0
  const ms = Date.parse(value)
  return Number.isNaN(ms) ? 0 : ms
}

export function elapsedMs(task, nowMs = Date.now()) {
  const started = parse(task.startedAt)
  if (!started) return 0
  const finished = parse(task.finishedAt)
  return Math.max(0, (finished || nowMs) - started)
}

/**
 * Expected duration for a task: the average of finished tasks with the same
 * role, bounded so one slow run cannot push the estimate to hours.
 */
export function expectedMs(task, all) {
  const peers = all.filter(
    (item) => item.role === task.role && item.status === 'done' && item.startedAt && item.finishedAt,
  )
  if (!peers.length) return DEFAULT_MS
  const total = peers.reduce((sum, item) => sum + elapsedMs(item), 0)
  return Math.min(MAX_MS, Math.max(MIN_MS, total / peers.length))
}

/**
 * Progress and ETA of a task. A parent reports the share of finished stages;
 * a leaf reports elapsed time against the expected duration of its role.
 */
export function taskProgress(task, all, nowMs = Date.now()) {
  const children = all.filter((item) => item.parentId === task.id)
  if (children.length) {
    const finished = children.filter((child) => TERMINAL.has(child.status)).length
    const remaining = children.filter((child) => !TERMINAL.has(child.status))
    const etaMs = remaining.reduce(
      (sum, child) => sum + Math.max(0, expectedMs(child, all) - elapsedMs(child, nowMs)),
      0,
    )
    return {
      percent: Math.round((finished / children.length) * 100),
      etaMs,
      elapsed: elapsedMs(task, nowMs),
      finished,
      total: children.length,
      active: !TERMINAL.has(task.status) && finished < children.length,
    }
  }
  if (TERMINAL.has(task.status)) {
    return { percent: 100, etaMs: 0, elapsed: elapsedMs(task, nowMs), finished: 0, total: 0, active: false }
  }
  const expected = expectedMs(task, all)
  const elapsed = elapsedMs(task, nowMs)
  const active = ACTIVE.has(task.status)
  return {
    percent: active ? Math.min(99, Math.round((elapsed / expected) * 100)) : 0,
    etaMs: Math.max(0, expected - elapsed),
    elapsed,
    finished: 0,
    total: 0,
    active,
  }
}
