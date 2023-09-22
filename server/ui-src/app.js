import App from './App.vue'
import router from './router'
import { createApp } from 'vue'

import './assets/styles.scss'
import 'bootstrap-icons/font/bootstrap-icons.scss'
import 'bootstrap'

const app = createApp(App)
app.use(router)
app.mount('#app')
