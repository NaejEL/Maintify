<template>
  <div id="app">
    <!-- Barre de navigation -->
    <nav v-if="isAuthenticated" class="navbar">
      <div class="nav-container">
        <div class="nav-brand">
          <router-link to="/" class="brand-link">
            🔧 <span>Maintify</span>
          </router-link>
        </div>
        
        <div class="nav-links">
          <router-link to="/" class="nav-link">
            🏠 {{ $t('navigation.home') }}
          </router-link>
          
          <!-- Navigation dynamique des plugins -->
          <PluginNavigation />
          
          <router-link to="/plugin-test" class="nav-link" style="background: #28a745;">
            🧪 Test Plugins
          </router-link>
          
          <router-link to="/profile" class="nav-link">
            👤 {{ $t('navigation.profile') }}
          </router-link>
          
          <router-link 
            v-if="isAdmin" 
            to="/admin/users" 
            class="nav-link admin-link"
          >
            👥 {{ $t('navigation.management') }}
          </router-link>
          
          <div class="nav-user">
            <LanguageSelector />
            <span class="user-name">{{ userDisplayName }}</span>
            <button @click="handleLogout" class="logout-btn">
              🚪 {{ $t('navigation.logout') }}
            </button>
          </div>
        </div>
      </div>
    </nav>

    <!-- Contenu principal -->
    <main class="main-content" :class="{ 'with-nav': isAuthenticated }">
      <router-view />
    </main>

    <!-- Notifications -->
    <div v-if="notification" class="notification" :class="notification.type">
      <div class="notification-content">
        <span class="notification-icon">
          {{ notificationIcon }}
        </span>
        <span class="notification-message">{{ notification.message }}</span>
        <button @click="clearNotification" class="notification-close">✕</button>
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters, mapActions } from 'vuex'
import LanguageSelector from './components/LanguageSelector.vue'
import PluginNavigation from './components/PluginNavigation.vue'

export default {
  name: 'App',
  
  components: {
    LanguageSelector,
    PluginNavigation
  },
  
  computed: {
    ...mapGetters('auth', ['isAuthenticated', 'user', 'isAdmin', 'fullName']),
    ...mapGetters(['notification']),
    
    userDisplayName() {
      return this.fullName || this.user?.email?.split('@')[0] || 'Utilisateur'
    },
    
    notificationIcon() {
      switch (this.notification?.type) {
        case 'success': return '✅'
        case 'error': return '❌'
        case 'warning': return '⚠️'
        case 'info': return 'ℹ️'
        default: return 'ℹ️'
      }
    }
  },
  
  mounted() {
    // Initialiser l'authentification au chargement de l'app
    this.$store.dispatch('auth/initAuth')
  },
  
  methods: {
    ...mapActions(['clearNotification']),
    
    async handleLogout() {
      try {
        await this.$store.dispatch('auth/logout')
        this.$router.push('/login')
      } catch (error) {
        console.error('Erreur lors de la déconnexion:', error)
      }
    }
  }
}
</script>

<style src="@/css/app.scss" lang="scss"></style>
