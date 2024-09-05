import App from './App.vue'
import router from './router'
import { createApp } from 'vue'
import mitt from 'mitt';

import './assets/styles.scss'
import 'bootstrap-icons/font/bootstrap-icons.scss'
import 'bootstrap'
import 'vue-css-donut-chart/src/styles/main.css'

const app = createApp(App)

// Global event bus used to subscribe to websocket events
// such as message deletes, updates & truncation.
const eventBus = mitt()
app.provide('eventBus', eventBus)

app.use(router)
app.mount('#app')
