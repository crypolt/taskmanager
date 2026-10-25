import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'

function flatten(node, prefix = '', out = {}) {
  for (const [name, value] of Object.entries(node)) {
    if (name === '_meta') continue
    const key = prefix ? `${prefix}.${name}` : name
    if (typeof value === 'string') out[key] = value
    else if (value && typeof value === 'object') flatten(value, key, out)
  }
  return out
}

// Fills the %LANG% / %TITLE% placeholders in index.html from the base locale
// file (the one whose _meta declares "base": true), so the served shell speaks
// the base language before any JavaScript runs (docs/localization-design.md
// §3.6, change-plan step 3).
function htmlLocalePlaceholders() {
  return {
    name: 'html-locale-placeholders',
    transformIndexHtml(html) {
      const dir = fileURLToPath(new URL('./src/locales', import.meta.url))
      let lang = ''
      let title = ''
      for (const name of readdirSync(dir).filter((file) => file.endsWith('.json')).sort()) {
        const data = JSON.parse(readFileSync(join(dir, name), 'utf8'))
        if (!data?._meta?.base) continue
        lang = data._meta.code
        title = flatten(data).title ?? title
        break
      }
      if (!lang) {
        throw new Error('web/src/locales: no language file declares "_meta": { "base": true }')
      }
      return html
        .replaceAll('%TITLE%', title || 'Task Manager')
        .replaceAll('%LANG%', lang)
    },
  }
}

const distDir = fileURLToPath(new URL('../assets/dist', import.meta.url))

export default defineConfig({
  plugins: [vue(), htmlLocalePlaceholders()],
  build: {
    outDir: distDir,
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
