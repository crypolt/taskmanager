# Как контрибьютить

Проект ведётся по упрощённому git-flow: стабильная ветка отделена от интеграционной,
работа идёт в короткоживущих тематических ветках и попадает в `main` только через PR.

## Ветки

| Ветка | Назначение | Кто вливает |
| --- | --- | --- |
| `main` | стабильные релизы, только merge-коммиты из `release/*` или `hotfix/*` | мейнтейнер |
| `develop` | интеграция готовых фич, база для новых веток | мейнтейнер |
| `feature/<кратко>` | одна задача/фича, ветвится от `develop`, живёт до PR | автор |
| `fix/<кратко>` | исправление бага, ветвится от `develop` | автор |
| `release/<версия>` | стабилизация перед релизом (`release/0.2.0`) | мейнтейнер |
| `hotfix/<кратко>` | срочная правка от `main`, вливается и в `main`, и в `develop` | мейнтейнер |

Правила:

- `feature/*`, `fix/*` — от `develop`, PR обратно в `develop`.
- `release/*` — от `develop`, после проверки вливается в `main` и тегируется (`v0.2.0`),
  затем `main` вливается обратно в `develop`.
- `hotfix/*` — от `main`, вливается в `main` (с тегом) и в `develop`.
- Прямые пуши в `main` и `develop` запрещены; история — линейная (`rebase`/`squash`).
- Ветка удаляется после вливания.

```bash
git switch develop && git pull
git switch -c feature/models-quota
# ... работа ...
git push -u origin feature/models-quota
```

## Коммиты

[Conventional Commits](https://www.conventionalcommits.org/): `type(scope): subject`.

- `feat:` новая функциональность, `fix:` баг, `docs:`, `refactor:`, `test:`, `chore:`, `ci:`.
- Заголовок — до 72 символов, в императиве, без точки в конце.
- В теле — «что и зачем», а не «как»; ссылка на issue, если есть.

```
feat(models): add per-route daily token limit

The models tab now shows today's spend against a per-chain-entry budget.
The limit is display-only for now: provider quotas are not exposed by the CLI.
```

## Перед PR

```bash
make test      # go test ./... + check:locales
make vet
cd web && npm test && npm run build
```

Чек-лист:

- [ ] тесты/линтеры зелёные, сборка проходит;
- [ ] нет секретов, `.env`, локальных путей и баз в диффе;
- [ ] новые строки UI/ошибок добавлены в оба каталога локалей
      (`web/src/locales/*.json`, `utils/i18n/locales/*.json`) — иначе упадёт `check:locales`;
- [ ] схема БД меняется только через `ensureColumns` (миграций нет);
- [ ] PR описывает поведение и способ проверки.

## Версии

Теги — `vMAJOR.MINOR.PATCH` на `main`. До 1.0: MINOR — новая функциональность,
PATCH — исправления и совместимые изменения.
