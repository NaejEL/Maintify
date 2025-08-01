import axios from 'axios'

const API_BASE_URL = process.env.VUE_APP_API_URL || 'http://localhost:5000'

// Configuration Axios pour les requêtes API
const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Intercepteur pour ajouter automatiquement le token JWT
api.interceptors.request.use(config => {
  const token = localStorage.getItem('auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
}, error => {
  return Promise.reject(error)
})

// Intercepteur pour gérer les erreurs d'authentification
api.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      // Token expiré ou invalide
      localStorage.removeItem('auth_token')
      localStorage.removeItem('user_data')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default {
  namespaced: true,
  
  state: {
    user: null,
    token: localStorage.getItem('auth_token') || null,
    isAuthenticated: false,
    loading: false
  },
  
  getters: {
    isAuthenticated: state => !!state.token && !!state.user,
    user: state => state.user,
    userRole: state => state.user?.role?.replace('UserRole.', '') || null,
    isAdmin: state => state.user?.role === 'UserRole.ADMIN',
    isTechnician: state => ['UserRole.ADMIN', 'UserRole.TECHNICIAN'].includes(state.user?.role),
    fullName: state => state.user?.full_name || '',
    loading: state => state.loading
  },
  
  mutations: {
    SET_LOADING(state, loading) {
      state.loading = loading
    },
    
    SET_AUTH_DATA(state, { user, token }) {
      state.user = user
      state.token = token
      state.isAuthenticated = true
      
      // Sauvegarder dans localStorage
      localStorage.setItem('auth_token', token)
      localStorage.setItem('user_data', JSON.stringify(user))
    },
    
    CLEAR_AUTH_DATA(state) {
      state.user = null
      state.token = null
      state.isAuthenticated = false
      
      // Nettoyer localStorage
      localStorage.removeItem('auth_token')
      localStorage.removeItem('user_data')
    },
    
    UPDATE_USER(state, userData) {
      state.user = { ...state.user, ...userData }
      localStorage.setItem('user_data', JSON.stringify(state.user))
    }
  },
  
  actions: {
    // Initialiser l'authentification au chargement de l'app
    async initAuth({ commit, dispatch }) {
      try {
        const token = localStorage.getItem('auth_token')
        const userData = localStorage.getItem('user_data')
        
        if (token && userData) {
          commit('SET_AUTH_DATA', {
            token,
            user: JSON.parse(userData)
          })
          
          // Vérifier que le token est toujours valide
          try {
            await dispatch('fetchProfile')
          } catch (error) {
            console.warn('Token invalide ou expiré:', error)
            commit('CLEAR_AUTH_DATA')
          }
        }
      } catch (error) {
        console.error('Erreur lors de l\'initialisation de l\'authentification:', error)
        commit('CLEAR_AUTH_DATA')
      }
    },
    
    // Connexion
    async login({ commit }, credentials) {
      commit('SET_LOADING', true)
      
      try {
        const response = await api.post('/api/auth/login', credentials)
        const { access_token, user } = response.data
        
        commit('SET_AUTH_DATA', {
          token: access_token,
          user: user
        })
        
        return { success: true, user }
      } catch (error) {
        const message = error.response?.data?.message || 'Erreur de connexion'
        throw new Error(message)
      } finally {
        commit('SET_LOADING', false)
      }
    },
    
    // Déconnexion
    logout({ commit }) {
      commit('CLEAR_AUTH_DATA')
      // Rediriger vers la page de connexion
      window.location.href = '/login'
    },
    
    // Récupérer le profil utilisateur
    async fetchProfile({ commit }) {
      try {
        const response = await api.get('/api/auth/profile')
        commit('UPDATE_USER', response.data)
        return response.data
      } catch (error) {
        throw new Error('Erreur lors de la récupération du profil')
      }
    },
    
    // Mettre à jour le profil
    async updateProfile({ commit }, profileData) {
      try {
        const response = await api.put('/api/auth/profile', profileData)
        commit('UPDATE_USER', response.data)
        return response.data
      } catch (error) {
        const message = error.response?.data?.message || 'Erreur lors de la mise à jour'
        throw new Error(message)
      }
    },
    
    // Changer le mot de passe
    async changePassword({ }, passwordData) {
      try {
        await api.put('/api/auth/change-password', passwordData)
        return { success: true }
      } catch (error) {
        const message = error.response?.data?.message || 'Erreur lors du changement de mot de passe'
        throw new Error(message)
      }
    },
    
    // Récupérer tous les utilisateurs (admin seulement)
    async fetchUsers() {
      try {
        const response = await api.get('/api/auth/users')
        return response.data
      } catch (error) {
        const message = error.response?.data?.message || 'Erreur lors de la récupération des utilisateurs'
        throw new Error(message)
      }
    },
    
    // Créer un utilisateur (admin seulement)
    async createUser({ }, userData) {
      try {
        const response = await api.post('/api/auth/register', userData)
        return response.data
      } catch (error) {
        const message = error.response?.data?.message || 'Erreur lors de la création de l\'utilisateur'
        throw new Error(message)
      }
    }
  }
}

// Export de l'instance API pour l'utiliser dans d'autres modules
export { api }
