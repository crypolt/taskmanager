const EVENT_NAMES = ['task', 'attempt', 'event']

/**
 * EventSource adapter. The server sends an SSE frame whose data is
 * {"type":"task|attempt|event","data":{...}} (unnamed event), so a plain
 * onmessage handler is enough; named frames are also supported.
 */
export function createRealtime({
  url = '/api/v1/events',
  EventSourceImpl = globalThis.EventSource,
  getLocale = () => '',
} = {}) {
  const handlers = new Set()
  const connectionHandlers = new Set()
  let source = null
  let closed = false
  let connected = false

  function emitConnection(value) {
    if (connected === value) return
    connected = value
    connectionHandlers.forEach((handler) => handler(value))
  }

  function dispatch(type, raw) {
    let data = raw
    try {
      data = JSON.parse(raw)
    } catch {
      /* keep raw string */
    }
    if (data && typeof data === 'object' && typeof data.type === 'string' && 'data' in data) {
      handlers.forEach((handler) => handler({ type: data.type, data: data.data }))
      return
    }
    handlers.forEach((handler) => handler({ type, data }))
  }

  function targetUrl() {
    const locale = getLocale()
    if (!locale) return url
    return `${url}${url.includes('?') ? '&' : '?'}lang=${encodeURIComponent(locale)}`
  }

  function connect() {
    if (closed || source) return
    if (typeof EventSourceImpl !== 'function') return
    source = new EventSourceImpl(targetUrl())
    source.onopen = () => emitConnection(true)
    source.onerror = () => emitConnection(false)
    source.onmessage = (message) => dispatch('message', message.data)
    EVENT_NAMES.forEach((name) => {
      source.addEventListener(name, (message) => dispatch(name, message.data))
    })
  }

  function subscribe(handler) {
    handlers.add(handler)
    connect()
    return () => handlers.delete(handler)
  }

  function onConnection(handler) {
    connectionHandlers.add(handler)
    handler(connected)
    return () => connectionHandlers.delete(handler)
  }

  function reconnect() {
    if (closed) return
    if (source) {
      source.close()
      source = null
    }
    emitConnection(false)
    connect()
  }

  function close() {
    closed = true
    if (source) {
      source.close()
      source = null
    }
    emitConnection(false)
  }

  return { subscribe, onConnection, reconnect, close }
}
