import { ref } from 'vue'

// Frontend catalogue — one <code>.json file per language next to this loader
// (docs/localization-design.md §3.6). The file whose _meta carries
// "base": true is the base locale: the last link of every fallback chain and
// the reference full translations are validated against. Adding a language is
// one file here; nothing else in the SPA changes. The backend keeps its own
// catalogue in utils/i18n/locales — the two key sets never overlap.
const modules = import.meta.glob('./*.json', { eager: true, import: 'default' })

const STORAGE_KEY = 'taskmanager.locale'
const COOKIE_NAME = 'lang'
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365

function normalizeTag(tag) {
  return String(tag ?? '')
    .trim()
    .toLowerCase()
    .replace(/_/g, '-')
}

function baseTag(tag) {
  const index = tag.indexOf('-')
  return index >= 0 ? tag.slice(0, index) : tag
}

function flatten(node, prefix = '', out = {}) {
  for (const [name, value] of Object.entries(node)) {
    if (name.includes('.')) continue
    const key = prefix ? `${prefix}.${name}` : name
    if (typeof value === 'string') out[key] = value
    else if (value && typeof value === 'object') flatten(value, key, out)
  }
  return out
}

const catalogue = new Map()
let baseLocale = ''
for (const data of Object.values(modules)) {
  const meta = data?._meta
  if (!meta?.code) continue
  const { _meta: ignored, ...messages } = data
  catalogue.set(meta.code, { meta, messages: flatten(messages) })
  if (meta.base) baseLocale = meta.code
}

if (!baseLocale) {
  throw new Error('web/src/locales: exactly one language file must carry "_meta": { "base": true }')
}

export const BASE_LOCALE = baseLocale

export const LOCALES = [...catalogue.values()]
  .map((entry) => entry.meta)
  .sort((left, right) => (left.order ?? 0) - (right.order ?? 0) || left.code.localeCompare(right.code))
  .map((meta) => ({
    code: meta.code,
    label: meta.label ?? meta.code.toUpperCase(),
    dateLocale: meta.dateLocale ?? meta.code,
    dir: meta.dir ?? 'ltr',
  }))

const SUPPORTED = new Set(LOCALES.map((item) => item.code))

function readStored() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) return saved
  } catch {
    /* storage is unavailable (private mode) */
  }
  if (typeof document !== 'undefined') {
    const match = document.cookie.match(/(?:^|;\s*)lang=([^;]+)/)
    if (match) {
      try {
        return decodeURIComponent(match[1])
      } catch {
        return match[1]
      }
    }
  }
  return ''
}

function detect() {
  const stored = normalizeTag(readStored())
  if (SUPPORTED.has(stored)) return stored
  if (SUPPORTED.has(baseTag(stored))) return baseTag(stored)
  const preferred = typeof navigator !== 'undefined' ? normalizeTag(navigator.language ?? '') : ''
  if (SUPPORTED.has(preferred)) return preferred
  if (SUPPORTED.has(baseTag(preferred))) return baseTag(preferred)
  return BASE_LOCALE
}

function chain(code) {
  const result = []
  const add = (candidate) => {
    if (candidate && !result.includes(candidate)) result.push(candidate)
  }
  add(code)
  add(baseTag(normalizeTag(code)))
  add(BASE_LOCALE)
  return result
}

function lookup(code, key) {
  for (const candidate of chain(code)) {
    const template = catalogue.get(candidate)?.messages[key]
    if (template !== undefined) return template
  }
  return undefined
}

function pluralCategory(code, count) {
  const meta = catalogue.get(code)?.meta
  try {
    return new Intl.PluralRules(meta?.dateLocale ?? code).select(count)
  } catch {
    return count === 1 ? 'one' : 'other'
  }
}

function interpolate(template, params) {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (match, name) =>
    params[name] === undefined || params[name] === null ? match : String(params[name]),
  )
}

const locale = ref(detect())
const listeners = new Set()

function persist(code) {
  try {
    localStorage.setItem(STORAGE_KEY, code)
  } catch {
    /* ignore unavailable storage */
  }
  if (typeof document !== 'undefined') {
    document.cookie = `${COOKIE_NAME}=${encodeURIComponent(code)};path=/;max-age=${COOKIE_MAX_AGE};SameSite=Lax`
  }
}

function applyDocument() {
  if (typeof document === 'undefined') return
  const code = locale.value
  const meta = catalogue.get(code)?.meta
  document.documentElement.lang = code
  document.documentElement.dir = meta?.dir ?? 'ltr'
  const title = lookup(code, 'app.title')
  if (title) document.title = title
}

export function getMeta(code = locale.value) {
  return catalogue.get(code)?.meta ?? null
}

export function dateLocale(code = locale.value) {
  return getMeta(code)?.dateLocale ?? code
}

export function getLocale() {
  return locale.value
}

export function setLocale(code) {
  const normalized = normalizeTag(code)
  const next = SUPPORTED.has(normalized) ? normalized : SUPPORTED.has(baseTag(normalized)) ? baseTag(normalized) : ''
  if (!next || next === locale.value) return
  locale.value = next
  persist(next)
  applyDocument()
  listeners.forEach((handler) => handler(next))
}

/** Subscribe to language changes (used to reconnect the SSE stream). */
export function onLocaleChange(handler) {
  listeners.add(handler)
  return () => listeners.delete(handler)
}

/**
 * Translates a key in the active locale. Missing keys fall back to the base
 * language and finally to the key itself — never to an empty string.
 */
export function t(key, params) {
  let template = lookup(locale.value, key)
  if (template === undefined) template = lookup(locale.value, `${key}.other`)
  if (template === undefined) return key
  return interpolate(template, params)
}

/** Translates a plural node, choosing the CLDR category for count. */
export function tn(key, count, params) {
  const merged = { ...(params ?? {}), count }
  const category = pluralCategory(locale.value, count)
  for (const candidate of [`${key}.${category}`, `${key}.other`, key]) {
    const template = lookup(locale.value, candidate)
    if (template !== undefined) return interpolate(template, merged)
  }
  return key
}

export function useI18n() {
  return { locale, t, tn, getLocale, setLocale, locales: LOCALES, dateLocale, getMeta, onLocaleChange }
}

if (typeof document !== 'undefined') applyDocument()
