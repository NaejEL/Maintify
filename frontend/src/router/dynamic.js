import { createRouter, createWebHistory } from 'vue-router'
import store from '../store'

/**
 * Routeur dynamique pour Maintify
 * Charge les routes des plugins de manière dynamique depuis l'API backend
 */
class DynamicRouter {
  constructor() {
    this.router = null
    this.pluginManager = null
    this.staticRoutes = [
      // Routes fixes de l'application
      {
        path: '/',
        name: 'Home',
        component: () => import('../components/Home.vue'),
        meta: { requiresAuth: true }
      },
      {
        path: '/login',
        name: 'Login',
        component: () => import('../components/Login.vue'),
        meta: { requiresGuest: true }
      },
      {
        path: '/profile',
        name: 'Profile',
        component: () => import('../components/Profile.vue'),
        meta: { requiresAuth: true }
      },
      {
        path: '/admin/users',
        name: 'UserManagement',
        component: () => import('../components/UserManagement.vue'),
        meta: { requiresAuth: true, requiresAdmin: true }
      },
      {
        path: '/plugin-test',
        name: 'PluginTest',
        component: () => import('../components/PluginTest.vue'),
        meta: { requiresAuth: true }
      }
    ]
  }

  /**
   * Initialise le routeur avec les routes statiques
   */
  async initialize() {
    console.log('🚦 Initialisation du routeur dynamique...')
    
    // Créer le routeur avec les routes statiques seulement
    this.router = createRouter({
      history: createWebHistory(),
      routes: [...this.staticRoutes]
    })

    // Ajouter les guards d'authentification
    this.setupAuthGuards()
    
    console.log(`✅ Routeur initialisé avec ${this.staticRoutes.length} routes statiques`)
    return this.router
  }

  /**
   * Charge et ajoute les routes des plugins de manière dynamique
   */
  async loadPluginRoutes() {
    try {
      console.log('🔄 Chargement dynamique des routes plugins...')
      
      // Import dynamique du plugin manager (chemin corrigé)
      const { pluginManager } = await import('../plugins/plugin-manager.js')
      this.pluginManager = pluginManager
      
      if (!pluginManager || typeof pluginManager.getRoutes !== 'function') {
        console.warn('⚠️ Plugin manager non disponible ou méthode getRoutes manquante')
        return []
      }

      // Récupérer les routes depuis le plugin manager
      const pluginRoutes = pluginManager.getRoutes()
      console.log(`📍 ${pluginRoutes.length} routes plugins découvertes`)

      // Ajouter chaque route au routeur
      for (const route of pluginRoutes) {
        try {
          this.router.addRoute(this.createPluginRoute(route))
          console.log(`✅ Route ajoutée: ${route.path} -> ${route.plugin}/${route.component}`)
        } catch (error) {
          console.error(`❌ Erreur lors de l'ajout de la route ${route.path}:`, error)
        }
      }

      console.log(`🎯 ${pluginRoutes.length} routes plugins ajoutées avec succès`)
      
      // Ajouter la route catch-all après avoir chargé tous les plugins
      this.addCatchAllRoute()
      
      return pluginRoutes

    } catch (error) {
      console.error('❌ Erreur lors du chargement des routes plugins:', error)
      return []
    }
  }

  /**
   * Ajoute la route catch-all après le chargement des plugins
   */
  addCatchAllRoute() {
    this.router.addRoute({
      path: '/:pathMatch(.*)*',
      name: 'NotFound',
      redirect: '/'
    })
    console.log('🔄 Route catch-all ajoutée après les plugins')
  }

  /**
   * Crée une route Vue Router à partir d'une configuration de route plugin
   */
  createPluginRoute(pluginRoute) {
    return {
      path: pluginRoute.path,
      name: pluginRoute.name,
      component: async () => {
        const componentLoader = await this.loadPluginComponent(pluginRoute.plugin, pluginRoute.component)
        return await componentLoader()
      },
      meta: {
        ...pluginRoute.meta,
        plugin: pluginRoute.plugin
        // Garde les métadonnées originales du plugin (incluant requiresAuth)
      }
    }
  }

  /**
   * Charge un composant de plugin avec gestion d'erreur
   */
  async loadPluginComponent(pluginName, componentName) {
    try {
      if (!this.pluginManager) {
        throw new Error('Plugin manager non initialisé')
      }
      
      return await this.pluginManager.loadComponent(pluginName, componentName)
    } catch (error) {
      console.error(`❌ Erreur lors du chargement du composant ${pluginName}/${componentName}:`, error)
      
      // Composant de fallback en cas d'erreur
      return () => import('../components/PlaceholderView.vue')
    }
  }

  /**
   * Ajoute une nouvelle route plugin à chaud
   */
  async addPluginRoute(pluginRoute) {
    try {
      const route = this.createPluginRoute(pluginRoute)
      this.router.addRoute(route)
      console.log(`🔥 Route ajoutée à chaud: ${pluginRoute.path}`)
      return true
    } catch (error) {
      console.error(`❌ Erreur lors de l'ajout à chaud de la route ${pluginRoute.path}:`, error)
      return false
    }
  }

  /**
   * Supprime une route plugin
   */
  removePluginRoute(routeName) {
    try {
      this.router.removeRoute(routeName)
      console.log(`🗑️ Route supprimée: ${routeName}`)
      return true
    } catch (error) {
      console.error(`❌ Erreur lors de la suppression de la route ${routeName}:`, error)
      return false
    }
  }

  /**
   * Met en place les guards d'authentification
   */
  setupAuthGuards() {
    this.router.beforeEach(async (to, from, next) => {
      const isAuthenticated = store.getters['auth/isAuthenticated']
      const isAdmin = store.getters['auth/isAdmin']
      
      // Debug des routes et de l'authentification
      console.log(`🔍 Navigation de ${from.path} vers ${to.path}`)
      console.log(`🔐 Authentifié: ${isAuthenticated}`)
      console.log(`👑 Admin: ${isAdmin}`)
      console.log(`📋 Meta de la route:`, to.meta)
      
      // Vérifier l'authentification
      if (to.meta.requiresAuth && !isAuthenticated) {
        console.log('🔒 Redirection vers login - authentification requise')
        next('/login')
        return
      }
      
      // Vérifier les droits admin
      if (to.meta.requiresAdmin && !isAdmin) {
        console.log('🚫 Accès refusé - droits admin requis')
        next('/')
        return
      }
      
      // Éviter d'aller sur login si déjà connecté
      if (to.meta.requiresGuest && isAuthenticated) {
        console.log('🏠 Redirection vers home - déjà connecté')
        next('/')
        return
      }
      
      console.log('✅ Navigation autorisée')
      next()
    })
  }

  /**
   * Retourne l'instance du routeur
   */
  getRouter() {
    return this.router
  }

  /**
   * Recharge toutes les routes plugins
   */
  async reloadPluginRoutes() {
    console.log('🔄 Rechargement de toutes les routes plugins...')
    
    // Supprimer toutes les routes plugins existantes
    const allRoutes = this.router.getRoutes()
    for (const route of allRoutes) {
      if (route.meta?.plugin) {
        this.router.removeRoute(route.name)
      }
    }
    
    // Recharger les routes
    return await this.loadPluginRoutes()
  }

  /**
   * Debug: affiche toutes les routes actuelles
   */
  debugRoutes() {
    console.log('🔍 Debug des routes actuelles:')
    const routes = this.router.getRoutes()
    
    console.table(routes.map(route => ({
      name: route.name,
      path: route.path,
      plugin: route.meta?.plugin || 'core',
      requiresAuth: route.meta?.requiresAuth || false,
      requiresAdmin: route.meta?.requiresAdmin || false
    })))
    
    return routes
  }

  /**
   * Debug: teste le chargement d'un composant plugin
   */
  async debugLoadComponent(pluginName, componentName) {
    console.log(`🧪 Test de chargement: ${pluginName}/${componentName}`)
    try {
      const component = await this.loadPluginComponent(pluginName, componentName)
      console.log('✅ Composant chargé avec succès:', component)
      return component
    } catch (error) {
      console.error('❌ Erreur de chargement:', error)
      throw error
    }
  }
}

// Instance singleton du routeur dynamique
const dynamicRouter = new DynamicRouter()

export default dynamicRouter
export { DynamicRouter }
