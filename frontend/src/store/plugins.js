import { pluginManager } from '@/plugins/plugin-manager.js'

export default {
  namespaced: true,
  
  state: {
    plugins: {},
    routes: [],
    menuItems: [],
    isInitialized: false,
    loading: false,
    error: null
  },
  
  getters: {
    plugins: state => state.plugins,
    routes: state => state.routes,
    menuItems: state => state.menuItems,
    isInitialized: state => state.isInitialized,
    loading: state => state.loading,
    error: state => state.error,
    
    // Retourne les plugins par catégorie
    pluginsByCategory: state => {
      const categories = {}
      Object.values(state.plugins).forEach(plugin => {
        const category = plugin.category || 'general'
        if (!categories[category]) {
          categories[category] = []
        }
        categories[category].push(plugin)
      })
      return categories
    },
    
    // Retourne un plugin spécifique
    getPlugin: state => pluginName => {
      return state.plugins[pluginName]
    },
    
    // Retourne les éléments de menu triés par ordre
    sortedMenuItems: state => {
      return [...state.menuItems].sort((a, b) => (a.order || 999) - (b.order || 999))
    }
  },
  
  mutations: {
    SET_LOADING(state, loading) {
      state.loading = loading
    },
    
    SET_ERROR(state, error) {
      state.error = error
    },
    
    SET_PLUGINS(state, plugins) {
      state.plugins = plugins
    },
    
    SET_ROUTES(state, routes) {
      state.routes = routes
    },
    
    SET_MENU_ITEMS(state, menuItems) {
      state.menuItems = menuItems
    },
    
    SET_INITIALIZED(state, initialized) {
      state.isInitialized = initialized
    },
    
    ADD_PLUGIN(state, plugin) {
      state.plugins[plugin.name] = plugin
    },
    
    REMOVE_PLUGIN(state, pluginName) {
      delete state.plugins[pluginName]
      state.routes = state.routes.filter(route => route.plugin !== pluginName)
      state.menuItems = state.menuItems.filter(item => item.plugin !== pluginName)
    },
    
    UPDATE_PLUGIN(state, { pluginName, updates }) {
      if (state.plugins[pluginName]) {
        state.plugins[pluginName] = { ...state.plugins[pluginName], ...updates }
      }
    }
  },
  
  actions: {
    // Initialiser les plugins (utilise les données déjà chargées)
    async initialize({ commit, dispatch }) {
      commit('SET_LOADING', true)
      commit('SET_ERROR', null)
      
      try {
        console.log('🔌 Synchronisation du store plugins...')
        
        // Utiliser les données déjà chargées par le plugin manager
        // (déjà initialisé dans main.js avec i18n)
        
        // Mettre à jour le store avec les données existantes
        commit('SET_PLUGINS', pluginManager.getAllPlugins())
        commit('SET_ROUTES', pluginManager.getRoutes())
        commit('SET_MENU_ITEMS', pluginManager.getMenuItems())
        commit('SET_INITIALIZED', true)
        
        const result = {
          plugins: pluginManager.getAllPlugins().size || Object.keys(pluginManager.getAllPlugins()).length,
          routes: pluginManager.getRoutes().length,
          menuItems: pluginManager.getMenuItems().length
        }
        
        console.log('✅ Store plugins synchronisé:', result)
        
        return result
      } catch (error) {
        console.error('❌ Erreur lors de la synchronisation du store plugins:', error)
        commit('SET_ERROR', error.message)
        throw error
      } finally {
        commit('SET_LOADING', false)
      }
    },
    
    // Recharger les plugins
    async reload({ dispatch }) {
      console.log('🔄 Rechargement des plugins...')
      return await dispatch('initialize')
    },
    
    // Charger un plugin spécifique
    async loadPlugin({ commit }, pluginName) {
      try {
        console.log(`🔌 Chargement du plugin: ${pluginName}`)
        
        const pluginInfo = pluginManager.getPluginInfo(pluginName)
        if (pluginInfo) {
          commit('ADD_PLUGIN', pluginInfo)
          console.log(`✅ Plugin ${pluginName} chargé`)
          return pluginInfo
        } else {
          throw new Error(`Plugin ${pluginName} non trouvé`)
        }
      } catch (error) {
        console.error(`❌ Erreur lors du chargement du plugin ${pluginName}:`, error)
        throw error
      }
    },
    
    // Décharger un plugin
    unloadPlugin({ commit }, pluginName) {
      console.log(`🗑️ Déchargement du plugin: ${pluginName}`)
      commit('REMOVE_PLUGIN', pluginName)
    },
    
    // Mettre à jour un plugin
    updatePlugin({ commit }, { pluginName, updates }) {
      commit('UPDATE_PLUGIN', { pluginName, updates })
    }
  }
}
