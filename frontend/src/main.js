import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import i18n from './i18n'

const app = createApp(App)

app.use(store)
app.use(router)
app.use(i18n)

// Initialiser l'authentification au démarrage de l'app
store.dispatch('auth/initAuth')

app.mount('#app')
