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

export default {
  name: 'App',
  
  components: {
    LanguageSelector
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

<style>
/* Styles globaux */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background-color: #f8f9fa;
  color: #2c3e50;
  line-height: 1.6;
}

#app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

/* Barre de navigation */
.navbar {
  background: white;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  position: sticky;
  top: 0;
  z-index: 1000;
  border-bottom: 1px solid #e1e8ed;
}

.nav-container {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 30px;
  height: 65px;
}

.nav-brand {
  font-size: 1.5rem;
  font-weight: 700;
}

.brand-link {
  text-decoration: none;
  color: #667eea;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: color 0.3s ease;
}

.brand-link:hover {
  color: #764ba2;
}

.nav-links {
  display: flex;
  align-items: center;
  gap: 25px;
}

.nav-link {
  text-decoration: none;
  color: #34495e;
  font-weight: 500;
  padding: 8px 16px;
  border-radius: 8px;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.nav-link:hover {
  background-color: #f8f9fa;
  color: #667eea;
}

.nav-link.router-link-active {
  background-color: #e3f2fd;
  color: #1565c0;
}

.admin-link {
  background-color: #fff3e0;
  color: #f57c00;
}

.admin-link:hover {
  background-color: #ffe0b2;
}

.nav-user {
  display: flex;
  align-items: center;
  gap: 15px;
  padding-left: 20px;
  border-left: 1px solid #e1e8ed;
}

.user-name {
  color: #34495e;
  font-weight: 600;
  font-size: 0.9rem;
}

.logout-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.logout-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

/* Contenu principal */
.main-content {
  flex: 1;
  min-height: 100vh;
}

.main-content.with-nav {
  min-height: calc(100vh - 65px);
}

/* Notifications */
.notification {
  position: fixed;
  top: 20px;
  right: 20px;
  min-width: 320px;
  max-width: 500px;
  padding: 0;
  border-radius: 12px;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
  z-index: 10000;
  animation: slideIn 0.3s ease-out;
}

.notification.success {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.notification.error {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.notification.warning {
  background-color: #fff3cd;
  color: #856404;
  border: 1px solid #ffeaa7;
}

.notification.info {
  background-color: #d1ecf1;
  color: #0c5460;
  border: 1px solid #b8daff;
}

.notification-content {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
}

.notification-icon {
  font-size: 1.2rem;
  flex-shrink: 0;
}

.notification-message {
  flex: 1;
  font-weight: 500;
}

.notification-close {
  background: none;
  border: none;
  font-size: 1.2rem;
  cursor: pointer;
  opacity: 0.7;
  transition: opacity 0.3s ease;
  padding: 4px;
  border-radius: 4px;
}

.notification-close:hover {
  opacity: 1;
  background-color: rgba(0, 0, 0, 0.1);
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* Responsive */
@media (max-width: 768px) {
  .nav-container {
    padding: 0 20px;
    flex-wrap: wrap;
    height: auto;
    min-height: 65px;
  }
  
  .nav-links {
    gap: 15px;
    flex-wrap: wrap;
  }
  
  .nav-user {
    padding-left: 15px;
    border-left: none;
    padding-top: 10px;
    border-top: 1px solid #e1e8ed;
    width: 100%;
    justify-content: center;
  }
  
  .user-name {
    display: none;
  }
  
  .notification {
    right: 10px;
    left: 10px;
    min-width: auto;
  }
}

@media (max-width: 480px) {
  .nav-links {
    width: 100%;
    justify-content: center;
    padding-top: 10px;
  }
  
  .nav-link {
    font-size: 0.8rem;
    padding: 6px 12px;
  }
}
</style>
