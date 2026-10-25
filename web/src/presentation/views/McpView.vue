<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useContainer } from '../../app/container.js'
import ErrorBanner from '../components/ErrorBanner.vue'
import { MCP_MASK, emptyMCPServer } from '../../domain/entities/mcp.js'
import { t } from '../lib/i18n.js'

const { mcp: store } = useContainer()
const drafts = reactive({})
const saving = ref('')
const creating = ref(false)
const created = reactive(emptyMCPServer())

onMounted(() => store.load())

function draftOf(server) {
  if (!drafts[server.name]) {
    drafts[server.name] = {
      name: server.name,
      type: server.type,
      enabled: server.enabled,
      command: server.command.join('\n'),
      url: server.url,
      env: Object.entries(server.environment).map(([key, value]) => ({ key, value })),
      masked: server.masked,
    }
  }
  return drafts[server.name]
}

function payload(draft) {
  const environment = {}
  draft.env.forEach((row) => {
    if (row.key.trim()) environment[row.key.trim()] = row.value
  })
  return {
    name: draft.name,
    type: draft.type,
    enabled: draft.enabled,
    command: draft.command
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean),
    environment,
    url: draft.url,
  }
}

async function save(server) {
  const draft = draftOf(server)
  saving.value = server.name
  try {
    await store.save(payload(draft))
    delete drafts[server.name]
  } catch {
    /* error is surfaced by the store */
  } finally {
    saving.value = ''
  }
}

async function remove(name) {
  try {
    await store.remove(name)
    delete drafts[name]
  } catch {
    /* error is surfaced by the store */
  }
}

function addEnv(draft) {
  draft.env.push({ key: '', value: '' })
}

function removeEnv(draft, index) {
  draft.env.splice(index, 1)
}

async function create() {
  if (!created.name.trim()) return
  creating.value = true
  try {
    const command = created.command.split('\n').map((line) => line.trim()).filter(Boolean)
    await store.save(
      {
        name: created.name.trim(),
        type: created.type,
        enabled: created.enabled,
        command,
        environment: {},
        url: created.url,
      },
      true,
    )
    Object.assign(created, emptyMCPServer())
  } catch {
    /* error is surfaced by the store */
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="mcp">
    <ErrorBanner :message="store.error" :detail="store.errorDetail" @dismiss="store.dismissError" />
    <p class="hint intro">{{ t('mcp.about') }}</p>
    <p class="hint" v-if="store.path">{{ t('mcp.path', { path: store.path }) }}</p>

    <div v-if="store.servers.length" class="mcp-grid">
      <div v-for="server in store.servers" :key="server.name" class="mcp-card">
        <div class="mcp-head">
          <label class="switch">
            <input v-model="draftOf(server).enabled" type="checkbox" />
            <span>{{ t('mcp.enabled') }}</span>
          </label>
          <span class="chip">{{ server.type }}</span>
          <span class="mcp-name">{{ server.name }}</span>
        </div>

        <div class="field">
          <label>{{ t('mcp.type') }}</label>
          <select v-model="draftOf(server).type">
            <option value="local">{{ t('mcp.typeLocal') }}</option>
            <option value="remote">{{ t('mcp.typeRemote') }}</option>
          </select>
        </div>

        <div class="field" v-if="draftOf(server).type === 'local'">
          <label>{{ t('mcp.command') }}</label>
          <textarea v-model="draftOf(server).command" class="mono"></textarea>
        </div>
        <div class="field" v-else>
          <label>{{ t('mcp.url') }}</label>
          <input v-model="draftOf(server).url" />
        </div>

        <div class="field">
          <label>{{ t('mcp.env') }}</label>
          <div class="env-rows">
            <div v-for="(row, index) in draftOf(server).env" :key="index" class="env-row">
              <input v-model="row.key" :placeholder="t('mcp.envKey')" class="env-key" />
              <input
                v-model="row.value"
                :type="server.masked.includes(row.key) ? 'password' : 'text'"
                :placeholder="t('mcp.envValue')"
              />
              <button class="btn small ghost" @click="removeEnv(draftOf(server), index)">✕</button>
            </div>
            <button class="btn small ghost" @click="addEnv(draftOf(server))">{{ t('mcp.addEnv') }}</button>
          </div>
          <p v-if="server.masked.length" class="hint">{{ t('mcp.maskedHint', { mask: MCP_MASK }) }}</p>
        </div>

        <div class="mcp-actions">
          <button class="btn small danger" @click="remove(server.name)">{{ t('mcp.delete') }}</button>
          <button class="btn small primary" :disabled="saving === server.name" @click="save(server)">
            {{ saving === server.name ? t('action.saving') : t('mcp.save') }}
          </button>
        </div>
      </div>
    </div>
    <div v-else-if="!store.loading" class="empty">{{ t('mcp.empty') }}</div>

    <div class="model-card">
      <h3>{{ t('mcp.add') }}</h3>
      <div class="mcp-form">
        <div class="field">
          <label>{{ t('mcp.name') }}</label>
          <input v-model="created.name" :placeholder="t('mcp.namePlaceholder')" />
        </div>
        <div class="field">
          <label>{{ t('mcp.type') }}</label>
          <select v-model="created.type">
            <option value="local">{{ t('mcp.typeLocal') }}</option>
            <option value="remote">{{ t('mcp.typeRemote') }}</option>
          </select>
        </div>
        <div class="field" v-if="created.type === 'local'">
          <label>{{ t('mcp.command') }}</label>
          <textarea v-model="created.command" class="mono" :placeholder="t('mcp.commandPlaceholder')"></textarea>
        </div>
        <div class="field" v-else>
          <label>{{ t('mcp.url') }}</label>
          <input v-model="created.url" />
        </div>
        <label class="switch">
          <input v-model="created.enabled" type="checkbox" />
          <span>{{ t('mcp.enabled') }}</span>
        </label>
        <button class="btn primary" :disabled="creating || !created.name.trim()" @click="create">
          {{ creating ? t('action.saving') : t('mcp.create') }}
        </button>
      </div>
    </div>
  </div>
</template>
