// Localization regression suite for the SPA catalogue
// (web/src/locales/*.json + the in-house t()/tn() loader).
//
// Runs on the built-in node:test runner; .vue files are compiled by the
// project's own Vite config through ssrLoadModule, so the tests exercise the
// real SFC pipeline and the real locale loader — no mocks of the i18n layer.
//
//   cd web && npm test

import { after, before, test } from 'node:test'
import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
import { createServer } from 'vite'
import { createSSRApp, nextTick } from 'vue'
import { renderToString } from '@vue/server-renderer'

const webRoot = fileURLToPath(new URL('..', import.meta.url))
const localeDir = fileURLToPath(new URL('../src/locales', import.meta.url))

const PLACEHOLDER = /\{([A-Za-z0-9_]+)\}/g

function readCatalogues() {
  return readdirSync(localeDir)
    .filter((file) => file.endsWith('.json'))
    .sort()
    .map((file) => {
      const data = JSON.parse(readFileSync(new URL(`../src/locales/${file}`, import.meta.url), 'utf8'))
      return { file, meta: data._meta, messages: flatten(data) }
    })
}

function flatten(node, prefix = '', out = new Map()) {
  for (const [name, value] of Object.entries(node)) {
    if (name === '_meta') continue
    const key = prefix ? `${prefix}.${name}` : name
    if (typeof value === 'string') out.set(key, value)
    else if (value && typeof value === 'object') flatten(value, key, out)
  }
  return out
}

function placeholders(template) {
  return [...template.matchAll(PLACEHOLDER)].map((match) => match[1])
}

function fill(template, params) {
  return template.replace(PLACEHOLDER, (match, name) => (params[name] === undefined ? match : String(params[name])))
}

let server
let i18n
let StatusChip

before(async () => {
  server = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'silent',
    server: { middlewareMode: true },
  })
  i18n = await server.ssrLoadModule('/src/locales/index.js')
  StatusChip = (await server.ssrLoadModule('/src/presentation/components/StatusChip.vue')).default
})

after(async () => {
  await server?.close()
})

function render(component, props) {
  return renderToString(createSSRApp(component, props))
}

test('default-locale key set is present in every other locale file (drift gate)', () => {
  const catalogues = readCatalogues()
  const base = catalogues.find((entry) => entry.meta?.base === true)
  assert.ok(base, 'exactly one locale file must declare "_meta": { "base": true }')
  assert.equal(catalogues.filter((entry) => entry.meta?.base === true).length, 1)

  for (const locale of catalogues) {
    if (locale === base) continue
    const missing = [...base.messages.keys()].filter((key) => !locale.messages.has(key))
    const extra = [...locale.messages.keys()].filter((key) => !base.messages.has(key))
    assert.deepEqual(missing, [], `${locale.file}: keys present in the base ${base.file} but missing here`)
    assert.deepEqual(extra, [], `${locale.file}: keys absent from the base ${base.file}`)
    for (const [key, template] of locale.messages) {
      assert.deepEqual(
        placeholders(template).sort(),
        placeholders(base.messages.get(key) ?? '').sort(),
        `${locale.file}: placeholder set for "${key}" differs from ${base.file}`,
      )
    }
  }
})

test('a sample component renders the translated label of the active locale', async () => {
  i18n.setLocale('ru')
  const ru = await render(StatusChip, { status: 'pending' })
  assert.match(ru, /в очереди/, 'ru render must use the ru catalogue value')

  i18n.setLocale('en')
  await nextTick()
  const en = await render(StatusChip, { status: 'pending' })
  assert.match(en, /queued/, 'en render must use the en catalogue value')
  assert.doesNotMatch(en, /в очереди/, 'the previous locale must not leak into the new render')
  assert.doesNotMatch(en, /status\.pending/, 'the raw key must never reach the DOM')

  i18n.setLocale('ru')
  await nextTick()
})

test('switching language changes every message with no key omissions or empty strings', () => {
  const catalogues = readCatalogues()
  const base = catalogues.find((entry) => entry.meta?.base === true)
  assert.ok(base)

  for (const locale of catalogues) {
    i18n.setLocale(locale.meta.code)
    assert.equal(i18n.getLocale(), locale.meta.code, `setLocale must activate ${locale.meta.code}`)

    for (const [key, template] of base.messages) {
      const expected = locale.messages.get(key)
      assert.equal(typeof expected, 'string', `${locale.file}: "${key}" must be a translated string`)
      assert.ok(expected.trim().length > 0, `${locale.file}: "${key}" must not be empty`)

      const params = Object.fromEntries(placeholders(expected).map((name) => [name, name === 'count' ? 2 : 'X']))
      const rendered = i18n.t(key, params)
      assert.notEqual(rendered, key, `${locale.file}: t("${key}") returned the raw key`)
      assert.doesNotMatch(rendered, /\{[A-Za-z0-9_]+\}/, `${locale.file}: t("${key}") left a placeholder`)
      assert.equal(rendered, fill(expected, params), `${locale.file}: t("${key}") must render its own value`)
    }
  }
  i18n.setLocale('ru')
})

// Keys spread over every user-facing namespace. Their shipped translations are
// genuinely different sentences, so an untranslated copy of the base file (or a
// namespace that was forgotten) is caught here rather than in production.
const REPRESENTATIVE_KEYS = [
  'app.title',
  'nav.board',
  'status.pending',
  'action.newTask',
  'drawer.role',
  'field.title',
  'plan.title',
  'role.test',
  'sse.on',
  'stats.total',
  'column.todo',
  'validation.titleRequired',
]

test('representative keys are translated, not copied from the base locale', () => {
  const catalogues = readCatalogues()
  const base = catalogues.find((entry) => entry.meta?.base === true)
  assert.ok(base)

  for (const key of REPRESENTATIVE_KEYS) {
    assert.ok(base.messages.has(key), `${base.file}: representative key "${key}" must exist`)
  }

  for (const locale of catalogues) {
    if (locale === base) continue
    for (const key of REPRESENTATIVE_KEYS) {
      assert.ok(locale.messages.has(key), `${locale.file}: missing representative key "${key}"`)
      assert.notEqual(
        locale.messages.get(key),
        base.messages.get(key),
        `${locale.file}: "${key}" is a verbatim copy of ${base.file} — translation is missing`,
      )
    }
  }
})

test('no namespace is left untranslated in any other locale', () => {
  const catalogues = readCatalogues()
  const base = catalogues.find((entry) => entry.meta?.base === true)
  assert.ok(base)

  const baseKeys = [...base.messages.keys()]
  const namespaces = [...new Set(baseKeys.map((key) => key.split('.')[0]))]
  assert.ok(namespaces.length >= 10, `expected the base catalogue to span many namespaces, got ${namespaces.join(', ')}`)

  for (const locale of catalogues) {
    if (locale === base) continue
    for (const namespace of namespaces) {
      const translated = baseKeys
        .filter((key) => key.split('.')[0] === namespace)
        .filter((key) => locale.messages.get(key) !== base.messages.get(key))
      assert.ok(
        translated.length > 0,
        `${locale.file}: namespace "${namespace}" is byte-identical to ${base.file} — it was never translated`,
      )
    }
  }
})

function restoreGlobal(name, descriptor) {
  if (descriptor) Object.defineProperty(globalThis, name, descriptor)
  else delete globalThis[name]
}

test('switching language applies the document, persists the choice and notifies once', () => {
  const store = new Map()
  const cookies = []
  const html = { lang: '', dir: '' }
  const documentStub = {
    documentElement: html,
    title: '',
    get cookie() {
      return ''
    },
    set cookie(value) {
      cookies.push(value)
    },
  }
  const previousDocument = Object.getOwnPropertyDescriptor(globalThis, 'document')
  const previousStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  globalThis.document = documentStub
  globalThis.localStorage = {
    getItem: (name) => (store.has(name) ? store.get(name) : null),
    setItem: (name, value) => store.set(name, String(value)),
  }

  const seen = []
  const unsubscribe = i18n.onLocaleChange((code) => seen.push(code))
  try {
    i18n.setLocale('ru')
    const ruTitle = i18n.t('app.title')
    i18n.setLocale('en')

    assert.equal(i18n.getLocale(), 'en')
    assert.equal(html.lang, 'en', 'html.lang must follow the active locale')
    assert.equal(html.dir, 'ltr', 'html.dir must come from the locale _meta')
    assert.equal(documentStub.title, i18n.t('app.title'), 'document.title must be the translated app title')
    assert.notEqual(documentStub.title, ruTitle, 'document.title must change with the language')
    assert.equal(store.get('taskmanager.locale'), 'en', 'the choice must survive a reload')
    assert.ok(
      cookies.some((cookie) => cookie.startsWith('lang=en;')),
      `the locale cookie must be written, got ${JSON.stringify(cookies)}`,
    )
    assert.deepEqual(seen, ['en'], 'a switch must notify subscribers exactly once')
  } finally {
    unsubscribe()
    i18n.setLocale('ru')
    restoreGlobal('document', previousDocument)
    restoreGlobal('localStorage', previousStorage)
  }
})

test('every subscriber is notified per switch and unsubscribe stops delivery', () => {
  const first = []
  const second = []
  const offFirst = i18n.onLocaleChange((code) => first.push(code))
  const offSecond = i18n.onLocaleChange((code) => second.push(code))
  try {
    i18n.setLocale('en')
    i18n.setLocale('ru')
    offFirst()
    i18n.setLocale('en')

    assert.deepEqual(first, ['en', 'ru'], 'an unsubscribed handler must stop receiving switches')
    assert.deepEqual(second, ['en', 'ru', 'en'], 'a subscribed handler must receive every switch once')
  } finally {
    offFirst()
    offSecond()
    i18n.setLocale('ru')
  }
})

test('an unsupported language keeps the active locale and base-tag codes resolve', () => {
  i18n.setLocale('en')
  i18n.setLocale('klingon')
  assert.equal(i18n.getLocale(), 'en', 'an unknown language must not clear the active locale')

  i18n.setLocale('ru-RU')
  assert.equal(i18n.getLocale(), 'ru', 'a regional tag must resolve to its base language')

  assert.equal(i18n.t('missing.key'), 'missing.key', 'a missing key must stay visible, never empty')
  i18n.setLocale('ru')
})
