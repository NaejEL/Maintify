import axios from 'axios'
import { getPluginI18nManager } from './i18n-manager.js'

class FrontendPluginManager {
  constructor() {
    this.plugins = new Map()
    this.routes = []
    this.menuItems = []
    this.loadedComponents = new Map()
    this.i18nManager = null
  }

  /**
   * Initialise le gestionnaire de plugins en récupérant les infos depuis l'API
   */
  async initialize(i18nInstance = null) {
    try {
      console.log('🔌 Initialisation du gestionnaire de plugins...')
      
      const response = await axios.get('/api/plugins')
      const { plugins, frontend_routes, menu_items, locales } = response.data

      // Stockage des plugins
      Object.entries(plugins).forEach(([name, config]) => {
        this.plugins.set(name, config)
      })

      // Stockage des routes et menus
      this.routes = frontend_routes || []
      this.menuItems = menu_items || []

      // Chargement des locales des plugins
      if (i18nInstance && locales) {
        this.i18nManager = getPluginI18nManager()
        await this.loadPluginsLocales(locales)
      }

      console.log(`✅ ${this.plugins.size} plugins chargés:`, Array.from(this.plugins.keys()))
      console.log(`📍 ${this.routes.length} routes découvertes`)
      console.log(`📋 ${this.menuItems.length} éléments de menu`)
      console.log(`🌐 ${Object.keys(locales || {}).length} plugins avec locales`)

      return {
        plugins: this.plugins.size,
        routes: this.routes.length,
        menuItems: this.menuItems.length,
        locales: Object.keys(locales || {}).length
      }
    } catch (error) {
      console.error('❌ Erreur lors de l\'initialisation des plugins:', error)
      throw error
    }
  }

  /**
   * Charge les locales de tous les plugins
   */
  async loadPluginsLocales(pluginsLocales) {
    if (!this.i18nManager) {
      console.warn('⚠️ I18n Manager non disponible, locales des plugins ignorées')
      return
    }

    for (const [pluginName, locales] of Object.entries(pluginsLocales)) {
      try {
        await this.i18nManager.loadPluginMessages(pluginName, locales)
        console.log(`🌐 Locales chargées pour le plugin: ${pluginName}`)
      } catch (error) {
        console.error(`❌ Erreur lors du chargement des locales pour ${pluginName}:`, error)
      }
    }
  }

  /**
   * Retourne toutes les routes des plugins directement utilisables par Vue Router
   */
  getRoutes() {
    return this.routes
  }

  /**
   * Retourne les éléments de menu triés
   */
  getMenuItems() {
    return this.menuItems.sort((a, b) => (a.order || 999) - (b.order || 999))
  }

  /**
   * Charge dynamiquement un composant de plugin
   */
  async loadComponent(pluginName, componentName) {
    const cacheKey = `${pluginName}:${componentName}`
    
    if (this.loadedComponents.has(cacheKey)) {
      return this.loadedComponents.get(cacheKey)
    }

    try {
      // Système générique de chargement des composants des plugins
      console.log(`🔍 Tentative de chargement du composant: ${pluginName}/${componentName}`)
      
      let component

      // Essayer de charger depuis la nouvelle structure modulaire (montée via Docker volume)
      try {
        const pluginComponentPath = `@plugins/${pluginName}/frontend/components/${componentName}.vue`
        console.log(`📂 Chargement depuis: ${pluginComponentPath}`)
        component = () => import(
          /* webpackChunkName: "plugin-[request]" */
          `@plugins/${pluginName}/frontend/components/${componentName}.vue`
        )
      } catch (pluginError) {
        console.warn(`⚠️ Composant plugin non trouvé: ${pluginName}/${componentName}`)
        
        // Fallback vers les composants globaux si nécessaire
        if (pluginName === 'dashboard' && componentName === 'Home') {
          console.log(`📂 Fallback vers composant global: Home.vue`)
          component = () => import('@/components/Home.vue')
        } else {
          console.log(`📂 Fallback vers PlaceholderView.vue`)
          component = () => import('@/components/PlaceholderView.vue')
        }
      }

      this.loadedComponents.set(cacheKey, component)
      console.log(`✅ Composant chargé: ${pluginName}/${componentName}`)
      return component
    } catch (error) {
      console.error(`❌ Erreur lors du chargement du composant ${componentName} du plugin ${pluginName}:`, error)
      // Fallback d'urgence vers un composant d'erreur
      return () => import('@/components/PlaceholderView.vue')
    }
  }

  /**
   * Charge les styles d'un plugin
   */
  async loadPluginStyles(pluginName) {
    const plugin = this.plugins.get(pluginName)
    if (!plugin || !plugin.frontend || !plugin.frontend.styles) {
      return
    }

    for (const stylePath of plugin.frontend.styles) {
      try {
        // Pour l'instant, les styles sont déjà compilés avec SCSS
        // Plus tard, on pourra implémenter un chargement dynamique
        console.log(`Styles du plugin ${pluginName} déjà chargés: ${stylePath}`)
      } catch (error) {
        console.error(`Erreur lors du chargement du style ${stylePath} pour ${pluginName}:`, error)
      }
    }
  }

  /**
   * Vérifie si un plugin est chargé
   */
  isPluginLoaded(pluginName) {
    return this.plugins.has(pluginName)
  }

  /**
   * Retourne les informations d'un plugin
   */
  getPluginInfo(pluginName) {
    return this.plugins.get(pluginName)
  }

  /**
   * Retourne tous les plugins chargés
   */
  getAllPlugins() {
    return Object.fromEntries(this.plugins)
  }
}

// Instance globale
export const pluginManager = new FrontendPluginManager()
export default pluginManager
