import { createRouter, createWebHistory } from 'vue-router'
import store from '../store'
import Home from '../components/Home.vue'

const routes = [
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
    path: '/dashboard',
    name: 'Dashboard',
    component: Home, // Pour l'instant, utilise le même composant
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/alerts',
    name: 'Alerts',
    component: () => import('../plugins/alerts/AlertsView.vue'),
    meta: {
      requiresAuth: true
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
    path: '/admin/users',
    name: 'AdminUsers',
    component: () => import('../components/UserManagement.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true // Accessible seulement aux admins
    }
  },
  // Redirection par défaut
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation Guard pour l'authentification
router.beforeEach(async (to, from, next) => {
  // Initialiser l'authentification si pas encore fait
  if (!store.state.auth.isAuthenticated && localStorage.getItem('auth_token')) {
    await store.dispatch('auth/initAuth')
  }

  const isAuthenticated = store.getters['auth/isAuthenticated']
  const isAdmin = store.getters['auth/isAdmin']
  
  // Route nécessitant d'être connecté
  if (to.meta.requiresAuth && !isAuthenticated) {
    next('/login')
    return
  }
  
  // Route nécessitant d'être déconnecté (page de login)
  if (to.meta.requiresGuest && isAuthenticated) {
    next('/')
    return
  }
  
  // Route nécessitant des droits admin
  if (to.meta.requiresAdmin && !isAdmin) {
    store.dispatch('showNotification', {
      type: 'error',
      message: 'Accès non autorisé. Droits administrateur requis.'
    })
    next('/')
    return
  }
  
  next()
})

export default router
