import { createRouter, createWebHistory } from 'vue-router'
import BoardView from '../presentation/views/BoardView.vue'
import ModelsView from '../presentation/views/ModelsView.vue'
import McpView from '../presentation/views/McpView.vue'

export function createAppRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', redirect: '/board' },
      { path: '/board', name: 'board', component: BoardView },
      { path: '/models', name: 'models', component: ModelsView },
      { path: '/mcp', name: 'mcp', component: McpView },
      { path: '/:pathMatch(.*)*', redirect: '/board' },
    ],
  })
}
