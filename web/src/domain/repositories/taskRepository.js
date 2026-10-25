/**
 * Task repository port. Implementations live in infrastructure/.
 * All methods return domain entities, never raw transport payloads.
 *
 * @typedef {Object} TaskRepository
 * @property {() => Promise<import('../entities/task.js').Task[]>} list
 * @property {(id: string) => Promise<object>} detail
 * @property {(payload: object) => Promise<object>} create
 * @property {(id: string) => Promise<void>} cancel
 * @property {(id: string) => Promise<void>} retry
 * @property {(id: string, note: string) => Promise<void>} answer
 * @property {(payload: object) => Promise<object>} previewPlan
 * @property {(payload: object) => Promise<object>} createFromPlan
 * @property {() => Promise<object>} stats
 */

export {}
