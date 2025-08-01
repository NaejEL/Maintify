import { createRouter, createWebHistory } from 'vue-router'
// Import différé du store pour éviter la dépendance circulaire
// import store from '../store'
import Home from '../components/Home.vue'

// Routes statiques de base (non-plugins)
const staticRoutes = [
  {
    path: '/test',
    name: 'SimpleTest',
    component: () => import('../components/SimpleTest.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../components/Login.vue'),
    meta: {
      requiresGuest: true // Accessible seulement si non connecté
    }
  },
  {
    path: '/',
    name: 'Home',
    component: Home,
    meta: {
      requiresAuth: true // Nécessite d'être connecté
    }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('../components/Profile.vue'),
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/plugin-test',
    name: 'PluginTest',
    component: () => import('../components/PluginTest.vue'),
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/admin/users',
    name: 'AdminUsers',
    component: () => import('../components/UserManagement.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true // Accessible seulement aux admins
    }
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

// Créer le router avec les routes statiques
const router = createRouter({
  history: createWebHistory(),
  routes: staticRoutes
})

// Fonction pour ajouter les routes plugins de manière asynchrone
export async function loadPluginRoutes() {
  try {
    console.log('🔄 Chargement des routes plugins...')
    const { pluginManager } = await import('../plugins/plugin-manager.js')
    
    if (pluginManager && typeof pluginManager.getRoutes === 'function') {
      const pluginRoutes = pluginManager.getRoutes()
      console.log(`📍 Ajout de ${pluginRoutes.length} routes plugins`)
      
      // Supprimer la route de redirection par défaut temporairement
      const defaultRoute = router.getRoutes().find(route => route.path === '/:pathMatch(.*)*')
      if (defaultRoute && defaultRoute.name) {
        router.removeRoute(defaultRoute.name)
      }
      
      // Ajouter les routes plugins
      pluginRoutes.forEach(route => {
        router.addRoute(route)
      })
      
      // Re-ajouter la route de redirection par défaut à la fin
      router.addRoute({
        path: '/:pathMatch(.*)*',
        redirect: '/'
      })
      
      console.log('✅ Routes plugins ajoutées avec succès')
    }
  } catch (error) {
    console.warn('⚠️ Erreur lors du chargement des routes plugins:', error)
  }
}

// Navigation Guard pour l'authentification - Version avec import dynamique
router.beforeEach(async (to, from, next) => {
  console.log('🔍 Navigation vers:', to.path, 'Meta:', to.meta)
  
  // Pour debug, on laisse passer toutes les routes sans authentification
  if (to.path === '/login' || to.path === '/test') {
    console.log('✅ Route autorisée sans auth:', to.path)
    next()
    return
  }
  
  try {
    // Import dynamique du store pour éviter la dépendance circulaire
    const { default: store } = await import('../store')
    
    // Initialiser l'authentification si pas encore fait
    if (!store.state.auth.isAuthenticated && localStorage.getItem('auth_token')) {
      console.log('🔄 Initialisation de l\'authentification...')
      await store.dispatch('auth/initAuth')
    }

    const isAuthenticated = store.getters['auth/isAuthenticated']
    console.log('🔐 Authentifié:', isAuthenticated)
    
    // Route nécessitant d'être connecté
    if (to.meta.requiresAuth && !isAuthenticated) {
      console.log('❌ Redirection vers login - non authentifié')
      next('/login')
      return
    }
    
    // Route nécessitant d'être déconnecté (page de login)
    if (to.meta.requiresGuest && isAuthenticated) {
      console.log('❌ Redirection vers home - déjà authentifié')
      next('/')
      return
    }
    
    console.log('✅ Navigation autorisée vers:', to.path)
    next()
  } catch (error) {
    console.error('❌ Erreur dans le guard de navigation:', error)
    next('/login')
  }
})

export default router
