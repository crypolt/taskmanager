# Localization — target architecture and key scheme

Status: design (stage 2). Implements the findings of the stage-1 localization audit.
Scope: separate backend and frontend catalogues so that adding a language is "drop one JSON file per side".

---

## 1. Approach

Split the single shared `utils/i18n/locales/*.json` catalogue into two physically separate catalogues with
disjoint key sets: the backend keeps `utils/i18n/locales/*.json` (embedded with `go:embed`, rendered through
`i18n.T(ctx, …)` in the request language), the frontend gets `web/src/locales/*.json` (eagerly globbed at build
time, served by the existing in-house `t()/tn()` loader). No key is allowed to exist on both sides; the boundary
is enforced by a checked-in ownership manifest plus one checker that validates both directories, runs in
`make test`, and fails on dangling or duplicated keys. The existing public APIs of both loaders are preserved —
only their file location, the base-locale declaration (`_meta.base`) and two new backend affordances
(`i18n.Base()`, `helpers.AppError.Detail`) change. Every string that the backend *renders into persisted task
text* stays backend-owned; every string the SPA renders itself is frontend-owned. Enum labels that both sides
need (statuses) are given distinct keys per side (`task.state.*` vs `status.*`) so the catalogues never collide.

---

## 2. Components

### 2.1 File tree (target)

```
task-manager/
├── main.go                                   unchanged; fx.Invoke(initI18n) → i18n.Init(cfg.Language)
├── config/models.go                          unchanged; Language `kenv:"LANGUAGE|<base>"`
├── models/
│   ├── status.go                             unchanged (source of truth for status codes)
│   ├── pipeline.go                           unchanged (NameKey/AboutKey/TitleKey stay i18n.Key)
│   └── locale.go                             NEW: AllStatuses(), StatusKey()
├── utils/i18n/                               BACKEND package (path unchanged)
│   ├── i18n.go                               + Base(); doc comment updated
│   ├── catalog.go                            + LocaleMeta.Base; base resolution + validation
│   ├── keys.go                               + TaskState* keys (10); no key removed
│   ├── lang.go                               unchanged (Match/ParseAcceptLanguage/Resolve/plural)
│   ├── context.go                            unchanged (WithLang/FromContext)
│   ├── doc.go                                unchanged
│   ├── catalog_test.go                       extended (base flag, per-status keys)
│   └── locales/                              BACKEND CATALOGUE — only file read by go:embed
│       ├── ru.json                           base (_meta.base = true)
│       └── en.json
├── utils/helpers/
│   ├── errors.go                             + AppError.Detail, WrapDetail(); reason/cause leave Args
│   └── json.go                               + ErrorBody.Detail `json:"detail,omitempty"`
├── server/http/
│   ├── middlewares/locale.go                 unchanged (resolution order preserved)
│   ├── static.go                             title ← i18n.T(ctx, i18n.StaticBrand); drop literal "taskflow"
│   └── server.go                             unchanged
├── web/
│   ├── index.html                            `lang="%LANG%"`, `<title>%TITLE%</title>`
│   ├── vite.config.js                        remove `@locales`; add transformIndexHtml locale plugin
│   ├── package.json                          `check:locales` → `node scripts/check-locales.mjs`
│   ├── scripts/
│   │   ├── check-locales.mjs                 REWRITTEN: validates BOTH directories + disjointness
│   │   ├── locale-ownership.mjs              NEW: the machine-readable ownership contract
│   │   └── split-locales.mjs                 NEW: one-shot split migration (deleted after use)
│   └── src/
│       ├── locales/                          FRONTEND CATALOGUE — only source for the SPA
│       │   ├── ru.json                       base (_meta.base = true)
│       │   ├── en.json
│       │   └── index.js                      loader; glob `./*.json`
│       ├── presentation/lib/i18n.js          unchanged re-export of ../../locales/index.js
│       ├── presentation/lib/format.js        unchanged (Intl via _meta.dateLocale)
│       └── domain/value-objects/taskStatus.js unchanged (status.* mapping is the frontend contract)
└── docs/localization-design.md               this document
```

### 2.2 Responsibilities

| Component | Owns |
|---|---|
| `utils/i18n/catalog.go` | Loading/flattening the embedded backend catalogue, `_meta` validation, one `_meta.base` invariant, fallback chain, `Key` lookup, plural categories, warn-once on missing key |
| `utils/i18n/i18n.go` | Public API: `Init`, `Default`, `Base`, `Languages`, `Meta`, `Supported`, `T`, `Tn`, `Translate`, `Has` |
| `utils/i18n/context.go` | Carrying the resolved `Lang` on `context.Context` |
| `server/http/middlewares/locale.go` | Per-request language resolution; writes `Content-Language`/`Vary`; stores `Lang` in the Fiber user context |
| `utils/helpers/errors.go` | `AppError` = (Code, Key, Args, Detail, Status); renders in request language; never puts raw technical text into `Args` |
| `models/locale.go` | Enum → key mapping for statuses (`task.state.<code>`) and the enumeration used by tests |
| `web/src/locales/index.js` | Loading/flattening the frontend catalogue, base locale from `_meta.base`, detection/persistence, `t`/`tn`, document side-effects (`lang`, `dir`, `title`), change listeners |
| `web/scripts/check-locales.mjs` | Structural gate over both catalogues and their ownership boundary |
| `web/scripts/locale-ownership.mjs` | Declares which namespaces each side owns; imported by checker and splitter |

### 2.3 Namespace ownership table (the contract)

| Namespace | Owner | Rendered by |
|---|---|---|
| `error.*` | backend | `helpers.RespondError` → `message` field of API errors |
| `task.*` (incl. `task.state.*`) | backend | event messages / `task.Error` persisted by the engine |
| `pipeline.*`, `stage.*` | backend | `engine.go` renders `NameKey`/`AboutKey`/`TitleKey` before returning to the API |
| `context.entry.*` | backend | shared-context entry titles written to the DB |
| `static.*` (incl. `static.brand`) | backend | `server/http/static.go` placeholder page |
| `app.*` | frontend | SPA title/brand |
| `nav.*`, `board.*`, `column.*`, `status.*`, `role.*`, `drawer.*`, `modal.*`, `plan.*`, `models.*`, `mcp.*`, `action.*`, `field.*`, `stats.*`, `hint.*`, `format.*`, `sse.*` | frontend | Vue components/views |
| `validation.*` | frontend | client-side form validation (`NewTaskModal`) |
| `clientError.*` | frontend | `httpClient` fallbacks (network/timeout/HTTP status) |

Renames that remove today's collisions (mandatory, they are part of the split):

| Today (shared file) | After split |
|---|---|
| `error.network`, `error.timeout`, `error.httpStatus` | `clientError.network`, `clientError.timeout`, `clientError.httpStatus` (frontend) |
| `error.titleRequired` used by `NewTaskModal.vue` | `validation.titleRequired` (frontend); backend keeps `error.titleRequired` |
| `context.title`, `context.empty`, `context.kind.*` (missing today) | `drawer.contextTitle`, `drawer.contextEmpty`, `drawer.contextKind.{plan,result,note}` (frontend) |
| hardcoded `taskflow` in `static.go:30` | `static.brand` (backend); SPA brand moves to `app.brand` |

Rationale: a namespace belongs to exactly one side, so "may I add this key here?" has a mechanical answer.
Rejected alternative: keep one namespace on both sides and rely on key-level disjointness — a reader cannot
tell which file to edit, and the checker would only report the collision after the fact.

---

## 3. Contracts

### 3.1 Key scheme (both sides)

- Flat dot path, at most three segments: `namespace.leaf` or `namespace.group.leaf`
  (`drawer.contextKind.plan`, `task.state.needs_human`, `static.uiNotBuilt.title`).
- Segments match `[a-z][A-Za-z0-9]*`; **no dot inside a segment** (both loaders reject it).
- Leaf names are camelCase (`newTask`, `titleRequired`, `dateLocale`) — matches the existing catalogue.
- Plural nodes are objects whose children are CLDR categories (`one|few|many|other`); the leaf of a plural
  node is never a string. Every plural node must define `other`; `ru` additionally defines `one|few|many`.
- Interpolation is `{name}` with camelCase names (`{count}`, `{attempt}`, `{stage}`, `{reason}`); the same key
  must use the same placeholder set in every language.
- Values contain no HTML and no markup; technical identifiers (model ids, provider names, file paths) are
  passed as arguments, never baked into the template.
- Enum-driven keys are declared exhaustively:
  - frontend: `status.<code>` for every `LABEL_KEYS` entry, `column.<key>` for every `BoardColumn`,
    `column.empty`, `drawer.contextKind.<kind>` for the three `models/context.go` kinds;
  - backend: `task.state.<code>` for every `models.Status*` constant.
- Missing key behaviour: backend warns once and returns the key text (panics when `I18N_STRICT=1`);
  frontend falls back base → key text. Never an empty string, never `undefined`.

### 3.2 Backend package API (`utils/i18n`)

```go
type Lang string
type Key string

type LocaleMeta struct {
    Code       string `json:"code"`
    Label      string `json:"label"`
    DateLocale string `json:"dateLocale"`
    Dir        string `json:"dir"`   // "ltr" | "rtl"
    Order      int    `json:"order"`
    Base       bool   `json:"base"`  // exactly one file per catalogue
}

func Init(defaultLang string) error                 // loads embedded catalogue; unsupported value → base
func Default() Lang                                 // deployment default (config LANGUAGE)
func Base() Lang                                    // the language whose file has _meta.base = true
func Languages() []Lang                             // ordered by _meta.order
func Meta(lang Lang) (LocaleMeta, bool)
func Supported(lang Lang) bool
func Match(tag string) (Lang, bool)                 // exact tag, then base tag
func ParseAcceptLanguage(header string) []Lang
func Resolve(candidates ...string) Lang             // first supported, else Default()

func T(ctx context.Context, key Key, args map[string]any) string
func Tn(ctx context.Context, key Key, n int, args map[string]any) string
func Translate(lang Lang, key Key, args map[string]any) string
func Has(lang Lang, key Key) bool

func WithLang(ctx context.Context, lang Lang) context.Context
func FromContext(ctx context.Context) Lang
```

New in this design: `Base()`, `LocaleMeta.Base`. Everything else is the current signature — no call site of
`T`/`Tn`/`Translate` changes because of the split.

### 3.3 Backend enum mapping (`models/locale.go`)

```go
// AllStatuses returns every task status code the API can emit.
func AllStatuses() []string

// StatusKey maps a status code to its backend catalogue key ("task.state."+code).
// Unknown codes return "" so callers keep the raw value.
func StatusKey(status string) i18n.Key
```

Call sites:
`internal/services/internals/engine.go:369` (`MsgTaskMoved`), `state.go:66` and `scheduler.go:98`
(`MsgTaskStageFailed`) pass `i18n.T(ctx, models.StatusKey(status), nil)` as the `{status}` argument instead of
the raw code. Rationale: keeps the sentence informative without importing frontend keys.
Rejected alternative: drop `{status}` from those messages — loses "cancelled vs failed" precision.

### 3.4 Error contract (`utils/helpers`)

```go
type AppError struct {
    Code   Code
    Key    i18n.Key
    Args   map[string]any // user-facing, already localizable values only
    Detail string         // raw technical cause; never translated, never inside the sentence
    Status int
    Err    error
}

func New(code Code, key i18n.Key, args map[string]any) *AppError
func Wrap(cause error, code Code, key i18n.Key, args map[string]any) *AppError
func WrapDetail(cause error, code Code, key i18n.Key, args map[string]any) *AppError // sets Detail
func RespondError(c *fiber.Ctx, err error) error

type ErrorBody struct {
    Code    string         `json:"code"`
    Key     string         `json:"key,omitempty"`
    Args    map[string]any `json:"args,omitempty"`
    Message string         `json:"message"`
    Detail  string         `json:"detail,omitempty"`
}
```

Rules:
- API error templates MUST NOT contain `{reason}`/`{cause}`. `ErrVerification`, `ErrPlanGeneration`,
  `ErrMCPConfig` drop the placeholder from `Args` and set `Detail` via `WrapDetail`.
- Frontend `httpClient` keeps `payload.message` as the headline and renders `payload.detail`, when present,
  in a muted `<code>`/secondary line — never concatenated into the translated sentence.
- Persisted task text (`task.retryFailed`, `task.needsHuman`, `task.stageError`) keeps `"{reason}"` after a
  separator because it is stored as a finished string. Documented degradation, see §7.

### 3.5 Per-request language (unchanged, documented)

Priority in `middlewares.Locale(def)`:
`?lang` / `?locale` → `X-Locale` → cookie `lang` → `Accept-Language` (q-ordered, base-tag fallback) → `def`
(`cfg.Language`, default from the env tag). Response sets `Content-Language` and `Vary: Accept-Language, X-Locale`.
The value reaches handlers as `i18n.FromContext(c.UserContext())` — handlers never read headers themselves.

There is **no per-user language** today (no users table). Extension point, when a profile appears: read it in
`Locale` between the cookie and `Accept-Language`, i.e. the stored user preference outranks the browser but not
an explicit `?lang=`. No storage change is made in this design.

### 3.6 Frontend API (`web/src/locales/index.js`)

```js
export const LOCALES        // [{ code, label, dateLocale, dir }] ordered by _meta.order
export const BASE_LOCALE    // code of the file with _meta.base = true
export function t(key, params?)
export function tn(key, count, params?)
export function getLocale()
export function setLocale(code)          // normalizes, persists localStorage+cookie, applies document, emits
export function onLocaleChange(handler)  // returns unsubscribe; used by sseClient
export function dateLocale(code?)
export function getMeta(code?)
export function useI18n()                // { locale, t, tn, getLocale, setLocale, locales, dateLocale, getMeta, onLocaleChange }
```

Behaviour kept from today: detection order `localStorage('taskmanager.locale')` → cookie `lang` →
`navigator.language` → `BASE_LOCALE`; fallback chain `code → baseTag(code) → BASE_LOCALE`; `t`/`tn` never
return an empty string; `applyDocument()` sets `html.lang`, `html.dir`, `document.title` from `app.title`.

`format.js` keeps using `_meta.dateLocale` for `Intl`; number formatting for tokens/cost becomes
`new Intl.NumberFormat(dateLocale(), { notation: 'compact' })` and `Intl.NumberFormat` for currency, replacing
the hardcoded `k`/`M`/`$` suffixes.

### 3.7 `web/scripts/locale-ownership.mjs`

```js
export const BACKEND_DIR = 'utils/i18n/locales'
export const FRONTEND_DIR = 'web/src/locales'
export const FRONTEND_PREFIXES = ['app', 'nav', 'board', 'column', 'status', 'role', 'drawer', 'modal',
  'plan', 'models', 'mcp', 'action', 'field', 'stats', 'hint', 'format', 'sse', 'validation', 'clientError']
export const BACKEND_PREFIXES = ['error', 'task', 'pipeline', 'stage', 'context', 'static']
// Key sets built at runtime, not by a literal t('…') call; the checker validates them explicitly.
export const DYNAMIC_KEYS = {
  'status.':      ['pending','planning','blocked','running','verifying','retrying','needsHuman','done','failed','cancelled'],
  'column.':      ['todo','running','verifying','blocked','human','done','failed'],
  'drawer.contextKind.': ['plan','result','note'],
  'role.':        ['analyze','design','implement','test','review','docs','generic'],
}
```

---

## 4. Data and state

### 4.1 Backend catalogue file (`utils/i18n/locales/ru.json`, skeleton)

```json
{
  "_meta": { "code": "ru", "label": "RU", "dateLocale": "ru-RU", "dir": "ltr", "order": 10, "base": true },
  "task": {
    "accepted": "задача принята",
    "moved": "статус изменён вручную: {status}",
    "retryFailed": "попытка {attempt}/{total} не удалась: {reason}",
    "stageFailed": "этап «{stage}» завершился со статусом {status}",
    "stagesPending": "ожидание этапов: {done}/{total} готово",
    "state": {
      "pending": "в очереди", "planning": "планирование", "blocked": "ожидает этапы",
      "running": "выполняется", "verifying": "проверка", "retrying": "повтор",
      "needs_human": "нужен человек", "done": "готово", "failed": "ошибка", "cancelled": "отменено"
    }
  },
  "context": { "entry": { "humanNote": "Заметка человека", "plan": "План" } },
  "pipeline": { "dev": { "name": "Этапы разработки", "about": "Фиксированный конвейер: …" } },
  "stage": { "analyze": "Анализ", "design": "Дизайн", "implement": "Реализация", "test": "Тесты", "review": "Ревью", "docs": "Документация", "plan": "Планирование" },
  "static": {
    "brand": "taskflow",
    "uiNotBuilt": { "title": "UI не собрана", "body": "Соберите фронтенд:", "api": "API доступен по {path}." }
  },
  "error": {
    "internal": "внутренняя ошибка",
    "verification": "проверка не удалась",
    "planFailed": "не удалось построить план",
    "mcpConfig": "не удалось сохранить конфигурацию MCP",
    "noModelForRole": "для роли {role} не настроена модель"
  }
}
```

`utils/i18n/locales/en.json` — same shape, `"code": "en"`, `"label": "EN"`, `"dateLocale": "en-US"`,
`"order": 20`, no `base` (or `"base": false`), English values.

### 4.2 Frontend catalogue file (`web/src/locales/ru.json`, skeleton)

```json
{
  "_meta": { "code": "ru", "label": "RU", "dateLocale": "ru-RU", "dir": "ltr", "order": 10, "base": true },
  "app": { "title": "Task Manager — конвейер агентов", "brand": "taskflow" },
  "nav": { "board": "Доска", "models": "Модели", "mcp": "MCP" },
  "status": {
    "pending": "в очереди", "planning": "планирование", "blocked": "ожидает этапы",
    "running": "выполняется", "verifying": "проверка", "retrying": "повтор",
    "needsHuman": "нужен человек", "done": "готово", "failed": "ошибка", "cancelled": "отменено"
  },
  "column": { "todo": "Очередь", "running": "В работе", "verifying": "Проверка", "blocked": "Ожидание", "human": "Нужен человек", "done": "Готово", "failed": "Провал / отмена", "empty": "пусто" },
  "role": { "analyze": "Анализ", "design": "Дизайн", "implement": "Реализация", "test": "Тесты", "review": "Ревью", "docs": "Документация", "generic": "Универсальная" },
  "drawer": {
    "contextTitle": "Контекст",
    "contextEmpty": "Записей нет",
    "contextKind": { "plan": "План", "result": "Результат", "note": "Заметка" },
    "attemptsTitle": "Попытки ({count})"
  },
  "board": {
    "tasks": { "one": "{count} задача", "few": "{count} задачи", "many": "{count} задач", "other": "{count} задач" }
  },
  "validation": { "titleRequired": "укажите название задачи" },
  "clientError": { "network": "ошибка сети", "timeout": "превышено время ожидания ответа", "httpStatus": "HTTP {status}" },
  "field": { "title": "Название" },
  "plan": { "title": "План" },
  "models": { "provider": "Провайдер" },
  "mcp": { "name": "Имя" },
  "action": { "newTask": "Новая задача", "cancel": "Отмена" },
  "stats": { "total": "Всего" },
  "hint": { "budget": "Бюджет" },
  "format": { "sec": "{value} с", "ms": "{value} мс" },
  "sse": { "on": "соединение активно", "off": "нет соединения" },
  "modal": { "title": "Новая задача" }
}
```

`web/src/locales/en.json` — same shape, English values, `"order": 20`.

### 4.3 Invariants (all mechanically checked)

1. **One base per directory**: exactly one file has `_meta.base === true`; its `code` is the loader fallback.
2. **Language parity within a directory**: every non-base file has exactly the base key set (missing and extra
   both fail).
3. **Cross-directory disjointness**: no flattened key exists in both catalogues; every key matches an owned
   prefix from `locale-ownership.mjs`.
4. **Meta validity**: `_meta.code` equals the file name; `dir ∈ {ltr, rtl}`; `label`, `dateLocale`, `order` present.
5. **No empty values**; **identical placeholder sets** per key across languages.
6. **Depth ≤ 3**, no dots inside segments, leaves are strings except plural/`_meta` nodes.
7. **Enum completeness**: every `task.state.<status>` exists (backend test over `models.AllStatuses()`); every
   `status.*`, `column.*`, `drawer.contextKind.*`, `role.*` entry from `DYNAMIC_KEYS` exists (checker).
8. **Dynamic references**: every `t('prefix.' + …)` prefix in `web/src` is declared in `DYNAMIC_KEYS`.

### 4.4 Lifecycle

- Backend: `go:embed` → parsed once in `Init` → immutable `catalog` swapped through `atomic.Pointer`. No
  reload at runtime; a catalogue defect is a startup/build error, never a per-request condition. `Init` may be
  skipped (tests, package-level errors) — lazy load uses the base language.
- Frontend: the module glob is resolved at build time; the catalogue is immutable at runtime; `locale` is a
  Vue `ref`; `setLocale` is the only mutator and it persists before notifying.
- Persisted localized text (`events.message`, `task.Error`, `blockedReason`) is frozen at write time in the
  language of the creating request. Not retranslated on switch — documented, not fixed here.

---

## 5. Change plan

Each step is independently applicable and leaves the tree buildable.

1. **Contracts first.** Add `LocaleMeta.Base` + base validation to `utils/i18n/catalog.go`, add `i18n.Base()`,
   add `"base": true` to `utils/i18n/locales/ru.json`, add `"base": false` to `en.json`. Keep the shared
   loader working (frontend still reads the same files). `go test ./...` must stay green.
2. **Ownership manifest + splitter.** Add `web/scripts/locale-ownership.mjs` and `web/scripts/split-locales.mjs`.
   Run the splitter: it writes `web/src/locales/{ru,en}.json` with the frontend-owned keys (renames from §2.3
   applied), removes those keys from `utils/i18n/locales/*.json`, and prints a diff summary. Review the diff,
   then delete the splitter.
3. **Frontend loader repoint.** Change the glob in `web/src/locales/index.js` to `./*.json`, derive
   `BASE_LOCALE` from `_meta.base`, drop the literal `'ru'` from `fallbackChain`. Remove `@locales` and the
   `projectRoot` `fs.allow` entry from `web/vite.config.js`; add the `transformIndexHtml` plugin that fills
   `%LANG%`/`%TITLE%` in `web/index.html` from the base file.
4. **Backend finish.** Add `models/locale.go`, `task.state.*` keys, `StaticBrand`, `AppError.Detail` +
   `WrapDetail` + `ErrorBody.Detail`; drop `{reason}` from the three API error templates; switch
   `ErrVerification`/`ErrPlanGeneration`/`ErrMCPConfig` to `WrapDetail`; replace `taskflow` in `static.go`
   with `i18n.T(ctx, i18n.StaticBrand)`; pass `models.StatusKey(...)` at the three status call sites.
5. **Code-site fixes.** Move `NewTaskModal` to `validation.titleRequired`; move `TaskDrawer` context labels to
   `drawer.contextTitle|contextEmpty|contextKind.*`; move `httpClient` fallbacks to `clientError.*`; in
   `AppHeader.vue` render `t('app.brand')` instead of the literal; render `detail` in the error banner
   (`ErrorBanner.vue`); replace `k`/`M`/`$` suffixes in `format.js` with `Intl.NumberFormat`.
6. **Gate.** Rewrite `web/scripts/check-locales.mjs` to validate both directories, disjointness, base
   uniqueness, `DYNAMIC_KEYS` completeness and dangling references; add `locales:` target to `Makefile` and
   make `test` depend on it (`go test ./... && $(MAKE) locales`); add `web` as an npm-script alias if CI calls
   npm directly.
7. **Verification & docs.** `go test ./...`, `cd web && npm run check:locales && npm run build`,
   `make build`, then update `docs/` with a one-page "how to add a language" note (two files, `_meta`, nothing
   else) and record the frozen-persisted-text limitation.

---

## 6. Failure modes

| Failure | Detection | Behaviour / degradation |
|---|---|---|
| Key missing in a language | Backend: `warnOnce` + raw key; `I18N_STRICT=1` panics. Frontend: base-language fallback, then key text | UI shows the key path; checker blocks the merge, so this only reaches runtime for a key used but never declared |
| Key in both catalogues | `check-locales.mjs` (disjointness + prefix ownership) | Build gate fails; no runtime behaviour |
| Base flag missing/duplicated | Backend `Init` returns an error (fx fails startup); checker fails; frontend loader throws on import | Fail fast — a catalogue without a base has no defined fallback |
| Broken JSON / `_meta.code` mismatch | Backend `Init` error; checker; frontend parse throws at build | Build/startup error, never a silent empty UI |
| `go:embed` misses a new file | No new file could exist unnoticed: file name is the language code and `languages` in the UI comes from the parsed catalogue; unit test asserts `en`+`ru` resolve | New language simply absent until rebuilt — detected by the parity/language-list test |
| Vite glob misses a new frontend file | `LOCALES` list shrinks; `check:locales` asserts the on-disk set equals the globbed set via a build-time test or the checker's directory listing | Switcher loses the language; checker fails |
| Unknown/unsupported request tag | `i18n.Match` returns false | Request silently served in `Default()`; never a 400 |
| Dangling dynamic key (`status.x`) | Checker validates `DYNAMIC_KEYS` against the catalogue; status list also comes from `models/status.go` | CI failure; at runtime the chip would show `status.x` |
| Technical cause containing user-visible text | `AppError.Args` no longer carries `reason`; code review + grep for `"reason"` in `helpers` | Sentence stays translated; raw cause is a separate `detail` field |
| Language switch leaves history in the old language | Expected; documented | Past events/task errors stay in their creation language |

---

## 7. Tradeoffs

| Decision | Rationale | Rejected alternative |
|---|---|---|
| Two directories (`utils/i18n/locales`, `web/src/locales`), one file per language | Physical boundary; `go:embed` and Vite glob each see only their own set; "add a language" is unambiguous | One directory with `<code>.backend.json`/`<code>.frontend.json` — two files per language in one place, easy to edit the wrong one |
| Keep `utils/i18n` as the backend package path | No import churn across ~20 files; the package is already cohesive | Move to `internal/i18n` — cleaner layering, but a pure rename with no functional gain |
| `_meta.base` flag instead of a hardcoded `ru` | One declarative source per side; `grep "'ru'"` in the loader becomes empty; `LANGUAGE` stays the deployment default | Env var (`VITE_DEFAULT_LOCALE`) — splits the default across build and runtime, and can disagree with the backend |
| In-house frontend loader (no `vue-i18n`) | Already implemented, tested by usage, zero new dependency, same key/plural semantics as the backend | `vue-i18n` — richer API, but a new runtime dependency and two plural engines |
| Backend owns `task.state.*`, frontend owns `status.*` (duplicated short labels) | Catalogues stay disjoint; backend keeps informative sentences; each side can tune tone independently | Serving enum labels from an API endpoint — extra round trip, cache, loading state in every chip |
| Structured `detail` for API errors; inline `{reason}` only in persisted events | Fixes mixed-language UI without a data-model change | Store event args instead of rendered text (correct long-term) — schema + migration, out of scope |
| `_meta.order` drives switcher order; `_meta.label` drives the button caption | Adding a language needs no JS/Go edit | Deriving labels from `Intl.DisplayNames` — inconsistent casing, and the switcher would rename itself |
| Frontend keeps sending `X-Locale` and SSE keeps `?lang=` | Existing, tested; makes the SPA's choice authoritative per request | Rely on `Accept-Language` only — the browser header is not user-controllable in-app |

---

## 8. Handoff

Do not change: the per-request resolution order in `middlewares.Locale`, the `i18n.T/Tn/Translate/Has` signatures,
`WithLang/FromContext`, the `t`/`tn`/`setLocale`/`onLocaleChange` exports, the persisted-text behaviour, or the
`@locales` removal's counterpart — `web/src/presentation/lib/i18n.js` must keep re-exporting from
`../../locales/index.js` so no component import changes. Do not add a third shared catalogue, do not put
frontend keys into the backend file "temporarily", and do not reintroduce `{reason}` into an API error template.
The checker and `locale-ownership.mjs` are the source of truth for ownership; if a new namespace is needed,
declare it there in the same commit as the keys. The stage-1 audit remains the inventory of record for
what had to move; this document is the contract for where it went.
