import { getLocale, t } from '../../locales/index.js'

export class HttpError extends Error {
  constructor(message, { status = 0, code = '', payload = null, detail = '' } = {}) {
    super(message)
    this.name = 'HttpError'
    this.status = status
    this.code = code
    this.payload = payload
    this.detail = detail
  }
}

function parseBody(text) {
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function errorMessage(status, payload) {
  if (payload && typeof payload === 'object') {
    if (typeof payload.message === 'string' && payload.message) return payload.message
    if (payload.error && typeof payload.error === 'string') return payload.error
    if (payload.error && typeof payload.error.message === 'string') return payload.error.message
  }
  if (typeof payload === 'string' && payload) return payload
  return t('clientError.httpStatus', { status })
}

export function createHttpClient({ baseUrl = '/api/v1', fetchImpl = globalThis.fetch, timeoutMs = 30000 } = {}) {
  async function request(path, { method = 'GET', body, timeout = timeoutMs } = {}) {
    const controller = new AbortController()
    const timer = timeout > 0 ? setTimeout(() => controller.abort(), timeout) : null
    try {
      const response = await fetchImpl(`${baseUrl}${path}`, {
        method,
        headers: {
          Accept: 'application/json',
          'X-Locale': getLocale(),
          ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: controller.signal,
      })
      const payload = parseBody(await response.text())
      if (!response.ok) {
        throw new HttpError(errorMessage(response.status, payload), {
          status: response.status,
          code: payload?.code ?? '',
          payload,
          detail: typeof payload === 'object' && payload !== null ? payload?.detail ?? '' : '',
        })
      }
      return payload
    } catch (err) {
      if (err instanceof HttpError) throw err
      if (err?.name === 'AbortError') throw new HttpError(t('clientError.timeout'), { status: 0, code: 'timeout' })
      throw new HttpError(err?.message || t('clientError.network'), { status: 0, code: 'network' })
    } finally {
      if (timer) clearTimeout(timer)
    }
  }

  return {
    request,
    get: (path, options) => request(path, { ...options, method: 'GET' }),
    post: (path, body, options) => request(path, { ...options, method: 'POST', body }),
    put: (path, body, options) => request(path, { ...options, method: 'PUT', body }),
    delete: (path, options) => request(path, { ...options, method: 'DELETE' }),
  }
}
