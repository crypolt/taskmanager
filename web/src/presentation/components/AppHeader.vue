<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { fmtCost, fmtTokens } from '../lib/format.js'
import { useI18n } from '../lib/i18n.js'

const props = defineProps({
  stats: { type: Object, required: true },
  connected: { type: Boolean, default: false },
})
defineEmits(['new-task', 'new-plan'])

const route = useRoute()
const activeTab = computed(() => route.name)
const { t, locale, setLocale, locales } = useI18n()
</script>

<template>
  <header class="topbar">
    <div class="brand">{{ t('app.brandStart') }}<span>{{ t('app.brandEnd') }}</span></div>
    <div class="tabs">
      <router-link class="tab" :class="{ active: activeTab === 'board' }" to="/board">{{ t('nav.board') }}</router-link>
      <router-link class="tab" :class="{ active: activeTab === 'models' }" to="/models">{{ t('nav.models') }}</router-link>
      <router-link class="tab" :class="{ active: activeTab === 'mcp' }" to="/mcp">{{ t('nav.mcp') }}</router-link>
    </div>
    <div class="spacer"></div>
    <div class="stats">
      <span>{{ t('stats.total') }} <b>{{ stats.total }}</b></span>
      <span>{{ t('stats.running') }} <b>{{ stats.running }}</b></span>
      <span>{{ t('stats.done') }} <b>{{ stats.done }}</b></span>
      <span v-if="stats.failed">{{ t('stats.errors') }} <b>{{ stats.failed }}</b></span>
      <span v-if="stats.needsHuman">{{ t('stats.human') }} <b>{{ stats.needsHuman }}</b></span>
      <span>{{ t('stats.tokens') }} <b>{{ fmtTokens(stats.tokens) }}</b></span>
      <span class="cost">{{ t('stats.cost') }} <b>{{ fmtCost(stats.costUsd) }}</b></span>
    </div>
    <div class="lang">
      <button
        v-for="item in locales"
        :key="item.code"
        class="btn small ghost"
        :class="{ active: locale === item.code }"
        @click="setLocale(item.code)"
      >
        {{ item.label }}
      </button>
    </div>
    <button class="btn" @click="$emit('new-plan')">{{ t('action.newPlan') }}</button>
    <button class="btn primary" @click="$emit('new-task')">{{ t('action.newTask') }}</button>
    <span
      :title="connected ? t('sse.on') : t('sse.off')"
      class="connection-dot"
      :style="{ background: connected ? 'var(--green)' : 'var(--red)' }"
    ></span>
  </header>
</template>
