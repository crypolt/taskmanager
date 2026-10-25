import { inject } from 'vue'
import { createTasksStore } from '../application/stores/createTasksStore.js'
import { createModelsStore } from '../application/stores/createModelsStore.js'
import { createMcpStore } from '../application/stores/createMcpStore.js'
import { createPipelinesStore } from '../application/stores/createPipelinesStore.js'
import { createHttpClient } from '../infrastructure/http/httpClient.js'
import { createHttpTaskRepository } from '../infrastructure/repositories/httpTaskRepository.js'
import { createHttpModelRepository } from '../infrastructure/repositories/httpModelRepository.js'
import { createHttpMcpRepository } from '../infrastructure/repositories/httpMcpRepository.js'
import { createHttpPipelineRepository } from '../infrastructure/repositories/httpPipelineRepository.js'
import { createRealtime } from '../infrastructure/realtime/sseClient.js'
import { getLocale, onLocaleChange } from '../locales/index.js'

export const CONTAINER_KEY = Symbol('app-container')

// Composition root: the only place where infrastructure is wired to the
// application layer. Presentation never constructs adapters itself.
export function createContainer({ baseUrl = '/api/v1' } = {}) {
  const http = createHttpClient({ baseUrl })
  const realtime = createRealtime({ url: `${baseUrl}/events`, getLocale })
  onLocaleChange(() => realtime.reconnect())

  const tasks = createTasksStore({
    taskRepository: createHttpTaskRepository(http),
    realtime,
  })
  const models = createModelsStore({ modelRepository: createHttpModelRepository(http) })
  const mcp = createMcpStore({ mcpRepository: createHttpMcpRepository(http) })
  const pipelines = createPipelinesStore({ pipelineRepository: createHttpPipelineRepository(http) })

  return { http, realtime, tasks, models, mcp, pipelines }
}

export function useContainer() {
  const container = inject(CONTAINER_KEY)
  if (!container) throw new Error('App container is not provided')
  return container
}
