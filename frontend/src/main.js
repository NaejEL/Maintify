import { createApp } from 'vue'
import App from './App.vue'
import store from './store'
import dynamicRouter from './router/dynamic.js'
import i18n from './i18n'

console.log('🚀 Application Vue avec routeur dynamique et plugins')

async function initializeApp() {
  try {
    console.log('📱 Création de l\'app Vue...')
    const app = createApp(App)
    
    console.log('🗃️ Ajout du store Vuex...')
    app.use(store)
    
    console.log('🔐 Initialisation de l\'authentification...')
    await store.dispatch('auth/initAuth')
    
    console.log('🌐 Ajout de i18n...')
    app.use(i18n)
    
    console.log('🚦 Initialisation du routeur dynamique...')
    const router = await dynamicRouter.initialize()
    
    console.log('🔧 Création du gestionnaire i18n des plugins...')
    const { createPluginI18nManager } = await import('./plugins/i18n-manager.js')
    const pluginI18nManager = createPluginI18nManager(i18n)
    
    console.log('🔌 Initialisation du plugin manager...')
    const { pluginManager } = await import('./plugins/plugin-manager.js')
    await pluginManager.initialize(i18n)
    
    console.log('📍 Chargement dynamique des routes plugins...')
    await dynamicRouter.loadPluginRoutes()
    
    console.log('🔗 Ajout du router à l\'application...')
    app.use(router)
    
    // Réessayer le chargement des messages en cas de problème de timing
    setTimeout(() => {
      console.log('🔄 Vérification des messages i18n en attente...')
      pluginI18nManager.retryPendingMessages()
    }, 100)
    
    console.log('🎯 Montage de l\'application Vue...')
    app.mount('#app')
    
    // Exposer le routeur dynamique globalement pour le debug
    if (process.env.NODE_ENV === 'development') {
      window.dynamicRouter = dynamicRouter
      window.debugRoutes = () => dynamicRouter.debugRoutes()
      console.log('🔧 Routeur dynamique exposé globalement pour debug (window.dynamicRouter)')
    }
    
    console.log('✅ Application Vue montée avec succès !')
    console.log('📊 Résumé:')
    console.log(`   - Plugins chargés: ${pluginManager.plugins.size}`)
    console.log(`   - Routes plugins: ${pluginManager.routes.length}`)
    console.log(`   - Éléments de menu: ${pluginManager.menuItems.length}`)
    
  } catch (error) {
    console.error('❌ Erreur lors de l\'initialisation:', error)
    
    // Affichage d'erreur pour l'utilisateur
    document.body.innerHTML = `
      <div style="padding: 20px; background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; border-radius: 4px; margin: 20px;">
        <h3>❌ Erreur d'initialisation</h3>
        <p>L'application n'a pas pu démarrer correctement.</p>
        <details>
          <summary>Détails de l'erreur</summary>
          <pre>${error.stack || error.message}</pre>
        </details>
        <p><button onclick="location.reload()">🔄 Recharger la page</button></p>
      </div>
    `
  }
}

// Initialiser l'application
initializeApp()
