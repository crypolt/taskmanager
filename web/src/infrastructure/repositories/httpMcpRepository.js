import { normalizeMCPConfig } from '../../domain/entities/mcp.js'

export function createHttpMcpRepository(http) {
  return {
    async list() {
      return normalizeMCPConfig(await http.get('/mcp'))
    },
    async create(payload) {
      return normalizeMCPConfig(await http.post('/mcp', payload))
    },
    async update(name, payload) {
      return normalizeMCPConfig(await http.put(`/mcp/${encodeURIComponent(name)}`, payload))
    },
    async remove(name) {
      return normalizeMCPConfig(await http.delete(`/mcp/${encodeURIComponent(name)}`))
    },
  }
}
