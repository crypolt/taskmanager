// Machine-readable ownership contract for the two locale catalogues
// (docs/localization-design.md §3.7). check-locales.mjs imports this file;
// the checker never hardcodes the boundary itself, so changing it always
// happens in the same commit as the keys it governs.
//
//   Backend catalogue: utils/i18n/locales  — rendered by i18n.T(ctx, …)
//   Frontend catalogue: web/src/locales    — rendered by t()/tn() in the SPA
//
// No key may exist on both sides. Stories beyond these prefixes need a
// declaration here before the checker will pass.

export const BACKEND_DIR = 'utils/i18n/locales'
export const FRONTEND_DIR = 'web/src/locales'

export const FRONTEND_PREFIXES = [
  'app', 'nav', 'board', 'column', 'status', 'role', 'drawer', 'modal',
  'plan', 'models', 'mcp', 'action', 'field', 'stats', 'hint', 'format',
  'sse', 'validation', 'clientError',
]
export const BACKEND_PREFIXES = ['error', 'task', 'pipeline', 'stage', 'context', 'static']

// Enum-driven key sets whose leaves are composed at runtime, never passed as
// a literal t('…') call. The checker validates them against the frontend
// catalogue explicitly.
export const DYNAMIC_KEYS = {
  'status.': ['pending', 'planning', 'blocked', 'running', 'verifying', 'retrying', 'needsHuman', 'done', 'failed', 'cancelled'],
  'column.': ['todo', 'running', 'verifying', 'blocked', 'human', 'done', 'failed'],
  'drawer.contextKind.': ['plan', 'result', 'note'],
  'role.': ['analyze', 'design', 'implement', 'test', 'review', 'docs', 'generic'],
}
