export const TaskStatus = Object.freeze({
  PENDING: 'pending',
  PLANNING: 'planning',
  BLOCKED: 'blocked',
  RUNNING: 'running',
  VERIFYING: 'verifying',
  RETRYING: 'retrying',
  NEEDS_HUMAN: 'needs_human',
  DONE: 'done',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
})

const LABEL_KEYS = Object.freeze({
  [TaskStatus.PENDING]: 'status.pending',
  [TaskStatus.PLANNING]: 'status.planning',
  [TaskStatus.BLOCKED]: 'status.blocked',
  [TaskStatus.RUNNING]: 'status.running',
  [TaskStatus.VERIFYING]: 'status.verifying',
  [TaskStatus.RETRYING]: 'status.retrying',
  [TaskStatus.NEEDS_HUMAN]: 'status.needsHuman',
  [TaskStatus.DONE]: 'status.done',
  [TaskStatus.FAILED]: 'status.failed',
  [TaskStatus.CANCELLED]: 'status.cancelled',
})

const TERMINAL = new Set([TaskStatus.DONE, TaskStatus.FAILED, TaskStatus.CANCELLED])
const ACTIVE = new Set([TaskStatus.PLANNING, TaskStatus.RUNNING, TaskStatus.VERIFYING])

export const BoardColumn = Object.freeze([
  { key: 'todo', titleKey: 'column.todo', statuses: [TaskStatus.PENDING, TaskStatus.RETRYING], drop: TaskStatus.PENDING },
  { key: 'running', titleKey: 'column.running', statuses: [TaskStatus.PLANNING, TaskStatus.RUNNING] },
  { key: 'verifying', titleKey: 'column.verifying', statuses: [TaskStatus.VERIFYING] },
  { key: 'blocked', titleKey: 'column.blocked', statuses: [TaskStatus.BLOCKED] },
  { key: 'human', titleKey: 'column.human', statuses: [TaskStatus.NEEDS_HUMAN], drop: TaskStatus.NEEDS_HUMAN },
  { key: 'done', titleKey: 'column.done', statuses: [TaskStatus.DONE], drop: TaskStatus.DONE },
  {
    key: 'failed',
    titleKey: 'column.failed',
    statuses: [TaskStatus.FAILED, TaskStatus.CANCELLED],
    drop: TaskStatus.CANCELLED,
  },
])

/** Status applied when a card is dropped on a column; null = not droppable. */
export function dropStatus(columnKey) {
  const column = BoardColumn.find((item) => item.key === columnKey)
  return column?.drop ?? null
}

/** Returns the i18n key of a status label; the presentation layer translates it. */
export function statusLabelKey(status) {
  return LABEL_KEYS[status] ?? ''
}

export function isTerminal(status) {
  return TERMINAL.has(status)
}

export function isActive(status) {
  return ACTIVE.has(status)
}

export function isCancellable(status) {
  return !TERMINAL.has(status)
}

export function isRetryable(status) {
  return isTerminal(status) || status === TaskStatus.NEEDS_HUMAN
}
