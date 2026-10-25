import { createApp } from 'vue'
import App from './app/App.vue'
import { CONTAINER_KEY, createContainer } from './app/container.js'
import { createAppRouter } from './app/router.js'
import './style.css'

const app = createApp(App)
app.provide(CONTAINER_KEY, createContainer())
app.use(createAppRouter())
app.mount('#app')
