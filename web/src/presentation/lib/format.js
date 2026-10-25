import { dateLocale, t } from './i18n.js'

export function fmtTime(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) return '—'
  return date.toLocaleString(dateLocale(), {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function fmtDuration(ms) {
  if (!ms) return '—'
  if (ms < 1000) return t('format.ms', { value: ms })
  if (ms < 60000) return t('format.sec', { value: (ms / 1000).toFixed(1) })
  return t('format.min', { min: Math.floor(ms / 60000), sec: Math.round((ms % 60000) / 1000) })
}

export function fmtCost(value) {
  const cost = Number(value) || 0
  if (cost === 0) return `${t('format.currency')}0`
  if (cost < 0.01) return `${t('format.currency')}${cost.toFixed(4)}`
  return `${t('format.currency')}${cost.toFixed(2)}`
}

export function fmtTokens(value) {
  const tokens = Number(value) || 0
  if (tokens === 0) return '0'
  if (tokens < 1000) return String(tokens)
  if (tokens < 1000000) return `${(tokens / 1000).toFixed(1)}${t('format.tokens.k')}`
  return `${(tokens / 1000000).toFixed(2)}${t('format.tokens.m')}`
}
