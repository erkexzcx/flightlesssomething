import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

// Import Bootstrap CSS and JavaScript, and Font Awesome CSS
import 'bootstrap/dist/css/bootstrap.min.css'
import 'bootstrap/dist/js/bootstrap.bundle.min.js'
import '@fortawesome/fontawesome-free/css/all.min.css'

// Apply saved theme before mounting to prevent flash of unstyled content (FOUC).
const validThemes = ['light', 'dark']
const storedTheme = localStorage.getItem('theme')
document.documentElement.setAttribute('data-bs-theme', validThemes.includes(storedTheme) ? storedTheme : 'dark')

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

app.mount('#app')
