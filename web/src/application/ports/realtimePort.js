/**
 * Realtime port: a push channel of domain events from the server.
 * Implemented by infrastructure/realtime/sseClient.js.
 *
 * @typedef {Object} RealtimePort
 * @property {(handler: (event: {type: string, data: any}) => void) => () => void} subscribe
 *   Subscribes and returns an unsubscribe function.
 * @property {(handler: (connected: boolean) => void) => () => void} onConnection
 */

export {}
