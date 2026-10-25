#!/usr/bin/env node
// Validates BOTH locale catalogues and the boundary between them
// (docs/localization-design.md §4.3 invariants 1–8):
//
//   backend  catalogue: utils/i18n/locales/*.json — rendered by i18n.T() in Go
//   frontend catalogue: web/src/locales/*.json    — rendered by t()/tn() in the SPA
//
// Checks:
//   - every file is valid JSON with a valid _meta block (_meta.code == file
//     name, dir ∈ {ltr, rtl}, label/dateLocale/order present);
//   - exactly one file per directory carries "_meta": { "base": true };
//   - every non-base file has exactly the base file's key set (missing AND
//     extra keys both fail);
//   - no value is empty; identical {placeholder} sets per key in a directory;
//   - depth ≤ 3, no dot inside a segment, every plural node defines "other";
//   - every key matches an owned namespace (locale-ownership.mjs);
//   - no flattened key exists in both catalogues (disjointness);
//   - every DYNAMIC_KEYS leaf exists in the frontend catalogue;
//   - every literal t('…')/tn('…') key used in web/src resolves in the
//     frontend catalogue;
//   - every t('prefix.' + …) concatenation prefix used in web/src is
//     declared in DYNAMIC_KEYS.
//
// Exit code 1 on any violation, so it can gate CI (npm run check:locales).

import { readdirSync, readFileSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join, resolve } from 'node:path'
import {
  BACKEND_DIR,
  FRONTEND_DIR,
  FRONTEND_PREFIXES,
  BACKEND_PREFIXES,
  DYNAMIC_KEYS,
} from './locale-ownership.mjs'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..')

const PLURAL_CATEGORIES = new Set(['one', 'few', 'many', 'other'])
const META_FIELDS = ['code', 'label', 'dateLocale', 'dir', 'order']

function placeholders(template) {
  return [...template.matchAll(/\{([A-Za-z0-9_]+)\}/g)].map((match) => match[1]).sort()
}

/** Flattens a catalogue file, validating depth, segments, plurals and empties. */
function flatten(node, file, prefix = '', out = new Map(), depth = 0) {
  const problems = []
  for (const [name, value] of Object.entries(node)) {
    if (name === '_meta') continue
    if (name.includes('.')) {
      problems.push(`${file}: segment "${name}" contains a dot`)
      continue
    }
    const key = prefix ? `${prefix}.${name}` : name
    if (typeof value === 'string') {
      if (!value.trim()) problems.push(`${file}: key "${key}" has an empty text`)
      if (key.split('.').length > 3) problems.push(`${file}: key "${key}" is deeper than 3 segments`)
      out.set(key, value)
    } else if (value && typeof value === 'object') {
      const children = Object.keys(value)
      const isPlural = children.length > 0 && children.every((item) => PLURAL_CATEGORIES.has(item))
      if (isPlural) {
        if (!('other' in value)) {
          problems.push(`${file}: plural node "${key}" must define "other"`)
        }
        for (const [category, template] of Object.entries(value)) {
          if (typeof template !== 'string') {
            problems.push(`${file}: plural node "${key}" has a non-string "${category}"`)
          }
        }
        const flat = children.map((category) => `${key}.${category}`)
        for (const item of flat) out.set(item, value[item])
      } else {
        problems.push(...flatten(value, file, key, out, depth + 1))
      }
    } else {
      problems.push(`${file}: key "${key}" is neither a string nor an object`)
    }
  }
  return problems
}

function loadCatalogue(label, dir, ownedPrefixes) {
  const files = readdirSync(dir).filter((file) => file.endsWith('.json')).sort()
  if (!files.length) return { label, dir, error: `no locale files in ${dir}` }

  const locales = []
  const errors = []
  for (const file of files) {
    const name = file.replace(/\.json$/, '')
    let data
    try {
      data = JSON.parse(readFileSync(join(dir, file), 'utf8'))
    } catch (err) {
      errors.push(`${file}: invalid JSON — ${err.message}`)
      continue
    }
    const meta = data?._meta
    if (!meta) {
      errors.push(`${file}: missing _meta block`)
      continue
    }
    for (const field of META_FIELDS) {
      if (meta[field] === undefined) errors.push(`${file}: _meta.${field} is required`)
    }
    if (meta.code !== name) errors.push(`${file}: _meta.code "${meta.code}" does not match the file name "${name}"`)
    if (!['ltr', 'rtl'].includes(meta.dir)) errors.push(`${file}: _meta.dir must be ltr or rtl`)

    const messages = new Map()
    errors.push(...flatten(data, file, '', messages, 1))
    for (const [key] of messages) {
      const owner = key.split('.')[0]
      if (!ownedPrefixes.includes(owner)) {
        errors.push(`${file}: key "${key}" does not match an owned namespace (${ownedPrefixes.join(', ')})`)
      }
    }
    locales.push({ name, meta, messages })
  }

  const bases = locales.filter((locale) => locale.meta?.base === true)
  if (bases.length !== 1) {
    errors.push(`${label}: exactly one file must carry "_meta": { "base": true }, found ${bases.length}`)
  }
  const base = bases[0]
  if (base) {
    for (const locale of locales) {
      if (locale === base) continue
      for (const key of base.messages.keys()) {
        if (!locale.messages.has(key)) errors.push(`${label}/${locale.name}: missing key "${key}"`)
      }
      for (const key of locale.messages.keys()) {
        if (!base.messages.has(key)) errors.push(`${label}/${locale.name}: extra key "${key}"`)
      }
      for (const [key, value] of locale.messages) {
        const expected = base.messages.get(key)
        if (expected === undefined) continue
        const left = placeholders(expected).join(',')
        const right = placeholders(value).join(',')
        if (left !== right) {
          errors.push(`${label}/${locale.name}: placeholder mismatch for "${key}" ({${left}} != {${right}})`)
        }
      }
    }
  }
  return { label, dir, locales, base, errors }
}

/** Collects the catalogues declared in locale-ownership.mjs. */
function walkSource(dir, exts, out = []) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) walkSource(path, exts, out)
    else if (exts.some((ext) => name.endsWith(ext))) out.push(path)
  }
  return out
}

// Literal t('…') / tn('…') calls, quoted key captured.
const CALL_RE = /(?:^|[^\w$.])(t|tn)\(\s*(['"])((?:\\.|(?!\2).)*)\2/g
// Dynamic t('prefix.' + …) / tn('prefix.' + …) concatenations, prefix captured.
const DYN_RE = /\bt(?:n)?\(\s*(['"])([a-z][A-Za-z0-9.]*)\1\s*\+/g

function scanSource(base, sourceFiles) {
  const errors = []
  for (const path of sourceFiles) {
    const short = path.slice(repoRoot.length + 1)
    const text = readFileSync(path, 'utf8')
    // Calls matched as dynamic concatenations must not also be validated as
    // plain keys: their leading string literal is only a prefix.
    const dynamicStarts = new Set()
    for (const match of text.matchAll(DYN_RE)) {
      dynamicStarts.add(match.index)
    }
    for (const match of text.matchAll(CALL_RE)) {
      const start = match.index + 1
      if (dynamicStarts.has(start)) continue
      const [, kind, , key] = match
      const forward = kind === 'tn'
      const covered =
        base.messages.has(key) ||
        (forward && [...base.messages.keys()].some((item) => item.startsWith(`${key}.`)))
      if (!covered) {
        const where = text.slice(0, match.index).split('\n').length
        errors.push(`${short}:${where}: key "${key}" used by ${kind}() is not in the frontend catalogue`)
      }
    }
    for (const match of text.matchAll(DYN_RE)) {
      const prefix = match[2]
      if (!Object.keys(DYNAMIC_KEYS).includes(prefix)) {
        const where = text.slice(0, match.index).split('\n').length
        errors.push(`${short}:${where}: dynamic key prefix "${prefix}" is not declared in DYNAMIC_KEYS`)
      }
    }
  }
  return errors
}

const backend = loadCatalogue('backend', join(repoRoot, ...BACKEND_DIR.split('/')), BACKEND_PREFIXES)
const frontend = loadCatalogue('frontend', join(repoRoot, ...FRONTEND_DIR.split('/')), FRONTEND_PREFIXES)

const catalogues = [backend, frontend]
const errors = catalogues.flatMap((catalogue) =>
  catalogue.error ? [`${catalogue.label}: ${catalogue.error}`] : catalogue.errors,
)

const keySets = new Map()
for (const catalogue of catalogues) {
  if (!catalogue.base) continue
  keySets.set(catalogue.label, catalogue.base.messages)
}

if (keySets.get('backend') && keySets.get('frontend')) {
  const backendKeys = keySets.get('backend')
  const frontendKeys = keySets.get('frontend')
  for (const key of backendKeys.keys()) {
    if (frontendKeys.has(key)) errors.push(`key "${key}" exists in BOTH catalogues`)
  }
}

if (frontend.base) {
  for (const [prefix, leaves] of Object.entries(DYNAMIC_KEYS)) {
    for (const leaf of leaves) {
      const key = `${prefix}${leaf}`
      if (!frontend.base.messages.has(key)) {
        errors.push(`frontend: DYNAMIC_KEYS entry "${key}" is missing from the catalogue`)
      }
    }
  }
  const sourceFiles = walkSource(join(repoRoot, 'web', 'src'), ['.js', '.vue'])
  errors.push(...scanSource(frontend.base, sourceFiles))
}

if (errors.length) {
  console.error(`locale check failed (${errors.length}):`)
  for (const error of errors) console.error(`  - ${error}`)
  process.exit(1)
}

const summary = catalogues
  .map((catalogue) =>
    catalogue.base
      ? `${catalogue.label}(${catalogue.locales.map((locale) => locale.name).join(', ')}: ${catalogue.base.messages.size} keys)`
      : catalogue.label,
  )
  .join(', ')
console.log(`locale check ok: ${summary}, disjoint`)
