import { createStore } from 'vuex'
import auth from './auth'
import plugins from './plugins'

export default createStore({
  state: {
    // États globaux de l'application
    appLoading: false,
    notifications: []
  },
  
  getters: {
    appLoading: state => state.appLoading,
    notifications: state => state.notifications,
    notification: state => state.notifications.length > 0 ? state.notifications[0] : null
  },
  
  mutations: {
    SET_APP_LOADING(state, loading) {
      state.appLoading = loading
    },
    
    ADD_NOTIFICATION(state, notification) {
      const id = Date.now()
      state.notifications.push({
        id,
        type: notification.type || 'info',
        message: notification.message,
        duration: notification.duration || 5000
      })
      
      // Auto-remove notification after duration
      setTimeout(() => {
        state.notifications = state.notifications.filter(n => n.id !== id)
      }, notification.duration || 5000)
    },
    
    REMOVE_NOTIFICATION(state, id) {
      state.notifications = state.notifications.filter(n => n.id !== id)
    }
  },
  
  actions: {
    showNotification({ commit }, notification) {
      commit('ADD_NOTIFICATION', notification)
    },
    
    removeNotification({ commit }, id) {
      commit('REMOVE_NOTIFICATION', id)
    },
    
    clearNotification({ commit, state }) {
      if (state.notifications.length > 0) {
        const firstNotification = state.notifications[0]
        commit('REMOVE_NOTIFICATION', firstNotification.id)
      }
    }
  },
  
  modules: {
    auth,
    plugins
  }
})
