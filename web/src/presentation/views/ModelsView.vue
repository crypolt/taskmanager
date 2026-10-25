<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useContainer } from '../../app/container.js'
import ErrorBanner from '../components/ErrorBanner.vue'
import { t } from '../lib/i18n.js'
import { fmtCost, fmtDuration, fmtTokens } from '../lib/format.js'

const { models: store } = useContainer()
const draft = reactive({})
const newRows = reactive({})
const saving = ref('')

const keyOf = (route) => `${route.role}:${route.position}`

watch(
  () => store.models,
  (models) => {
    Object.values(models.routes ?? {}).forEach((chain) => {
      chain.forEach((route) => {
        draft[keyOf(route)] = { ...route }
      })
    })
  },
  { immediate: true, deep: true },
)

onMounted(() => {
  store.load()
  store.loadDaily()
  store.loadHealth()
})

const dailyByModel = computed(() => {
  const map = new Map()
  store.daily.items.forEach((item) => map.set(`${item.provider}:${item.model}`, item))
  return map
})

function chainOf(role) {
  const chain = store.models.routes?.[role] ?? []
  const rows = chain.map((route) => draft[keyOf(route)] ?? route)
  if (newRows[role]) rows.push(newRows[role])
  return rows
}

function roleLabel(role) {
  const key = `role.${role}`
  const value = t(key)
  return value && value !== key ? value : role
}

function healthOf(route) {
  return store.health[`${route.provider}:${route.model}`]
}

function quotaOf(route) {
  const item = dailyByModel.value.get(`${route.provider}:${route.model}`)
  const today = Number(item?.todayTokens ?? 0)
  const limit = Number(route.dailyTokens ?? 0)
  const ratio = limit > 0 ? today / limit : null
  return { today, limit, ratio, calls: item?.calls ?? 0 }
}

async function save(route) {
  const key = keyOf(route)
  saving.value = key
  try {
    const ok = await store.save(route)
    if (ok && newRows[route.role] && route.position === newRows[route.role].position) {
      delete newRows[route.role]
    }
    if (ok) store.loadDaily()
  } finally {
    saving.value = ''
  }
}

async function move(route, dir) {
  const to = route.position + dir
  if (to < 0 || !chainOf(route.role).some((item) => item.position === to)) return
  await store.moveRoute({ role: route.role, from: route.position, to })
}

async function remove(route) {
  if (newRows[route.role] && route.position === newRows[route.role].position) {
    delete newRows[route.role]
    return
  }
  await store.removeRoute({ role: route.role, position: route.position })
}

function add(role) {
  const chain = store.models.routes?.[role] ?? []
  const next = chain.reduce((max, route) => Math.max(max, route.position ?? 0), -1) + 1
  newRows[role] = { role, position: next, provider: 'opencode', model: '', priceIn: 0, priceOut: 0, dailyTokens: 0 }
}

function refresh() {
  store.load()
  store.loadDaily()
  store.loadHealth()
}
</script>

<template>
  <div class="models">
    <ErrorBanner :message="store.error" :detail="store.errorDetail" @dismiss="store.dismissError" />
    <div class="intro-row">
      <p class="hint intro">
        {{ t('models.about') }}
        <b>{{ t('models.providers', { providers: store.providers.join(', ') || '—' }) }}</b>
      </p>
      <button class="btn small" @click="refresh">{{ t('models.refresh') }}</button>
    </div>

    <div class="grid">
      <div v-for="role in store.roles" :key="role" class="model-card">
        <div class="card-head">
          <h3>{{ roleLabel(role) }}</h3>
          <span class="hint">{{ t('models.chainCount', { count: chainOf(role).length }) }}</span>
        </div>

        <div class="chain">
          <template v-for="(route, index) in chainOf(role)" :key="route.position">
            <span v-if="index > 0" class="arrow">→</span>
            <span class="chip" :class="{ bad: healthOf(route)?.status === 'error' }">{{ route.provider }} · {{ route.model }}</span>
          </template>
        </div>

        <div class="entries">
          <div v-for="route in chainOf(role)" :key="route.position" class="entry">
            <div class="entry-head">
              <span class="position">{{ t('models.position', { position: route.position }) }}</span>
              <span v-if="healthOf(route)?.status === 'error'" class="hint bad">
                {{ t('models.statusError') }}
                <template v-if="healthOf(route).lastErrorAt">
                  · {{ t('models.refusedAt', { time: String(healthOf(route).lastErrorAt).slice(0, 16) }) }}
                </template>
              </span>
              <span v-else class="hint ok">{{ t('models.statusOk') }}</span>
              <span class="spacer"></span>
              <button class="btn small" :disabled="route.position === 0" @click="move(route, -1)">↑</button>
              <button class="btn small" :disabled="route.position >= chainOf(role).length - 1" @click="move(route, 1)">↓</button>
              <button class="btn small danger" :disabled="store.busy === keyOf(route)" @click="remove(route)">{{ t('models.removeRoute') }}</button>
            </div>

            <div class="entry-fields">
              <label>
                <span class="hint">{{ t('models.provider') }}</span>
                <select v-model="route.provider" class="inline-input">
                  <option v-for="provider in store.providers" :key="provider" :value="provider">{{ provider }}</option>
                </select>
              </label>
              <label class="grow">
                <span class="hint">{{ t('models.model') }}</span>
                <input v-model="route.model" class="inline-input wide" />
              </label>
              <label>
                <span class="hint">{{ t('models.priceIn') }}</span>
                <input v-model.number="route.priceIn" type="number" step="0.01" class="inline-input price" />
              </label>
              <label>
                <span class="hint">{{ t('models.priceOut') }}</span>
                <input v-model.number="route.priceOut" type="number" step="0.01" class="inline-input price" />
              </label>
              <label>
                <span class="hint">{{ t('models.dailyTokens') }}</span>
                <input v-model.number="route.dailyTokens" type="number" min="0" step="1000" class="inline-input price" />
              </label>
              <button class="btn small" :disabled="store.busy === keyOf(route)" @click="save(route)">
                {{ saving === keyOf(route) ? t('action.saving') : t('action.save') }}
              </button>
            </div>

            <div class="quota">
              <div class="quota-bar">
                <span :style="{ width: Math.min(100, (quotaOf(route).ratio ?? 0) * 100) + '%' }"></span>
              </div>
              <span class="hint">
                <template v-if="quotaOf(route).limit > 0">
                  {{ t('models.quota', { used: fmtTokens(quotaOf(route).today), limit: fmtTokens(quotaOf(route).limit) }) }}
                </template>
                <template v-else>
                  {{ t('models.quotaUnlimited', { used: fmtTokens(quotaOf(route).today) }) }}
                </template>
              </span>
            </div>
          </div>
        </div>

        <button class="btn small" @click="add(role)">{{ t('models.addEntry') }}</button>
      </div>
    </div>

    <div class="model-card">
      <h3>{{ t('models.usage') }}</h3>
      <table v-if="store.daily.items.length" class="table">
        <thead>
          <tr>
            <th>{{ t('models.provider') }}</th>
            <th>{{ t('models.model') }}</th>
            <th>{{ t('models.calls') }}</th>
            <th>{{ t('models.tokensIn') }}</th>
            <th>{{ t('models.tokensOut') }}</th>
            <th>{{ t('models.cache') }}</th>
            <th>{{ t('models.tokensTotal') }}</th>
            <th>{{ t('models.share') }}</th>
            <th>{{ t('models.latency') }}</th>
            <th>{{ t('models.price') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in store.daily.items" :key="item.provider + item.model">
            <td>{{ item.provider }}</td>
            <td>{{ item.model }}</td>
            <td>{{ item.calls }}</td>
            <td>{{ fmtTokens(item.inputTokens) }}</td>
            <td>{{ fmtTokens(item.outputTokens) }}</td>
            <td>{{ fmtTokens(item.cacheReadTokens) }}</td>
            <td>{{ fmtTokens(item.tokens) }}</td>
            <td class="nowrap">
              <span class="usage-bar"><span :style="{ width: item.share + '%' }"></span></span>
              <span class="hint share-value">{{ item.share.toFixed(1) }}%</span>
            </td>
            <td>{{ fmtDuration(item.avgLatencyMs) }}</td>
            <td>{{ fmtCost(item.costUsd) }}</td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">{{ t('models.usageEmpty') }}</div>
      <p v-if="store.daily.items.length" class="hint">
        {{ t('models.total', { tokens: fmtTokens(store.daily.totalTokens), cost: fmtCost(store.daily.totalCost) }) }}
      </p>
      <p v-if="store.daily.items.length" class="hint">{{ t('models.cacheNote') }}</p>
    </div>
  </div>
</template>

<style scoped>
.models {
  display: grid;
  gap: 16px;
}
.intro-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
}
.model-card {
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 12px;
  padding: 16px;
  display: grid;
  gap: 12px;
  align-content: start;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.chain {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.arrow {
  color: var(--muted, #6b7280);
}
.chip.bad {
  border-color: #dc2626;
  color: #dc2626;
}
.entries {
  display: grid;
  gap: 12px;
}
.entry {
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 10px;
  padding: 10px;
  display: grid;
  gap: 8px;
}
.entry-head {
  display: flex;
  gap: 8px;
  align-items: center;
}
.position {
  font-weight: 600;
  min-width: 30px;
}
.spacer {
  flex: 1;
}
.entry-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-end;
}
.entry-fields label {
  display: grid;
  gap: 2px;
}
.entry-fields label.grow {
  flex: 1;
  min-width: 200px;
}
.inline-input {
  padding: 6px 8px;
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 8px;
  background: #fff;
  width: 100%;
}
.inline-input.wide {
  min-width: 180px;
}
.inline-input.price {
  width: 90px;
}
.quota {
  display: flex;
  align-items: center;
  gap: 8px;
}
.quota-bar {
  width: 140px;
  height: 8px;
  border-radius: 4px;
  background: var(--border, #e5e7eb);
  overflow: hidden;
}
.quota-bar span {
  display: block;
  height: 100%;
  background: var(--primary, #4f46e5);
}
.hint.bad {
  color: #dc2626;
}
.hint.ok {
  color: #16a34a;
}
.btn.danger {
  color: #dc2626;
}
</style>
