export const MCP_MASK = '••••••'

export function normalizeMCPConfig(raw = {}) {
  return {
    path: raw.path ?? '',
    servers: Array.isArray(raw.servers) ? raw.servers.map(normalizeMCPServer) : [],
  }
}

export function normalizeMCPServer(raw = {}) {
  return {
    name: raw.name ?? '',
    type: raw.type ?? 'local',
    enabled: raw.enabled !== false,
    command: Array.isArray(raw.command) ? raw.command : [],
    environment: raw.environment && typeof raw.environment === 'object' ? { ...raw.environment } : {},
    url: raw.url ?? '',
    masked: Array.isArray(raw.masked) ? raw.masked : [],
  }
}

export function emptyMCPServer() {
  return { name: '', type: 'local', enabled: true, command: [], environment: {}, url: '', masked: [] }
}

export function toMCPPayload(server) {
  return {
    name: server.name.trim(),
    type: server.type,
    enabled: Boolean(server.enabled),
    command: server.command,
    environment: server.environment,
    url: server.url,
  }
}
