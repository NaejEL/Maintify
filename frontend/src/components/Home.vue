<template>
  <div class="home">
    <!-- En-tête de bienvenue -->
    <div class="welcome-header">
      <div class="welcome-content">
        <h1 class="welcome-title">
          👋 {{ $t('dashboard.welcome', { name: userDisplayName }) }}
        </h1>
        <p class="welcome-subtitle">
          {{ $t('dashboard.subtitle') }}
        </p>
      </div>
      <div class="welcome-icon">
        🔧
      </div>
    </div>

    <!-- Indicateurs principaux -->
    <div class="dashboard-grid">
      <!-- Statistiques rapides -->
      <div class="stats-section">
        <h2 class="section-title">📊 {{ $t('dashboard.overview') }}</h2>
        <div class="stats-cards">
          <div class="stat-card">
            <div class="stat-icon">🏭</div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.equipment || 0 }}</div>
              <div class="stat-label">{{ $t('dashboard.stats.equipment') }}</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">🚨</div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.alerts || 0 }}</div>
              <div class="stat-label">{{ $t('dashboard.stats.alerts') }}</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">🔧</div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.maintenance || 0 }}</div>
              <div class="stat-label">{{ $t('dashboard.stats.maintenance') }}</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">📍</div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.locations || 0 }}</div>
              <div class="stat-label">Lieux</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Alertes récentes -->
      <div class="alerts-section">
        <h2 class="section-title">🚨 Alertes récentes</h2>
        <div class="alerts-list">
          <div v-if="recentAlerts.length === 0" class="empty-state">
            <div class="empty-icon">✅</div>
            <p>Aucune alerte récente</p>
          </div>
          <div 
            v-else
            v-for="alert in recentAlerts" 
            :key="alert.id" 
            class="alert-item"
            :class="alert.severity"
          >
            <div class="alert-icon">
              {{ getAlertIcon(alert.severity) }}
            </div>
            <div class="alert-content">
              <div class="alert-title">{{ alert.title }}</div>
              <div class="alert-meta">{{ formatDate(alert.created_at) }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Actions rapides -->
      <div class="quick-actions">
        <h2 class="section-title">⚡ Actions rapides</h2>
        <div class="action-buttons">
          <router-link to="/equipment" class="action-btn">
            <div class="action-icon">🏭</div>
            <div class="action-text">
              <div class="action-title">Équipements</div>
              <div class="action-desc">Gérer les équipements</div>
            </div>
          </router-link>
          
          <router-link to="/maintenance" class="action-btn">
            <div class="action-icon">🔧</div>
            <div class="action-text">
              <div class="action-title">Maintenance</div>
              <div class="action-desc">Planifier des tâches</div>
            </div>
          </router-link>
          
          <router-link to="/reports" class="action-btn">
            <div class="action-icon">📊</div>
            <div class="action-text">
              <div class="action-title">Rapports</div>
              <div class="action-desc">Voir les statistiques</div>
            </div>
          </router-link>
          
          <router-link v-if="isAdmin" to="/admin/users" class="action-btn admin">
            <div class="action-icon">👥</div>
            <div class="action-text">
              <div class="action-title">Utilisateurs</div>
              <div class="action-desc">Gestion des accès</div>
            </div>
          </router-link>
        </div>
      </div>

      <!-- Activité récente -->
      <div class="activity-section">
        <h2 class="section-title">📈 Activité récente</h2>
        <div class="activity-list">
          <div v-if="recentActivity.length === 0" class="empty-state">
            <div class="empty-icon">💤</div>
            <p>Aucune activité récente</p>
          </div>
          <div 
            v-else
            v-for="activity in recentActivity" 
            :key="activity.id" 
            class="activity-item"
          >
            <div class="activity-icon">{{ getActivityIcon(activity.type) }}</div>
            <div class="activity-content">
              <div class="activity-text">{{ activity.description }}</div>
              <div class="activity-time">{{ formatDate(activity.timestamp) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'Home',
  
  data() {
    return {
      stats: {
        equipment: 0,
        alerts: 0,
        maintenance: 0,
        locations: 0
      },
      recentAlerts: [],
      recentActivity: []
    }
  },
  
  computed: {
    ...mapGetters('auth', ['user', 'isAdmin', 'fullName']),
    
    userDisplayName() {
      return this.fullName || this.user?.email?.split('@')[0] || 'Utilisateur'
    }
  },
  
  mounted() {
    this.loadDashboardData()
  },
  
  methods: {
    async loadDashboardData() {
      try {
        // Simuler des données pour le moment
        // Plus tard, ces données viendront de l'API
        this.stats = {
          equipment: 42,
          alerts: 3,
          maintenance: 8,
          locations: 5
        }
        
        this.recentAlerts = [
          {
            id: 1,
            title: 'Température élevée - Chaudière A1',
            severity: 'high',
            created_at: new Date(Date.now() - 3600000) // 1 heure
          },
          {
            id: 2,
            title: 'Maintenance programmée - Climatisation B2',
            severity: 'medium',
            created_at: new Date(Date.now() - 7200000) // 2 heures
          },
          {
            id: 3,
            title: 'Contrôle de routine - Ascenseur C1',
            severity: 'low',
            created_at: new Date(Date.now() - 86400000) // 1 jour
          }
        ]
        
        this.recentActivity = [
          {
            id: 1,
            type: 'maintenance',
            description: 'Maintenance terminée sur équipement #001',
            timestamp: new Date(Date.now() - 1800000) // 30 min
          },
          {
            id: 2,
            type: 'alert',
            description: 'Nouvelle alerte créée pour la chaudière A1',
            timestamp: new Date(Date.now() - 3600000) // 1 heure
          },
          {
            id: 3,
            type: 'user',
            description: 'Utilisateur technicien1 s\'est connecté',
            timestamp: new Date(Date.now() - 7200000) // 2 heures
          }
        ]
      } catch (error) {
        console.error('Erreur lors du chargement des données:', error)
      }
    },
    
    getAlertIcon(severity) {
      switch (severity) {
        case 'high': return '🔴'
        case 'medium': return '🟡'
        case 'low': return '🟢'
        default: return '⚪'
      }
    },
    
    getActivityIcon(type) {
      switch (type) {
        case 'maintenance': return '🔧'
        case 'alert': return '🚨'
        case 'user': return '👤'
        case 'equipment': return '🏭'
        default: return '📝'
      }
    },
    
    formatDate(date) {
      const now = new Date()
      const diff = now - new Date(date)
      const minutes = Math.floor(diff / 60000)
      const hours = Math.floor(diff / 3600000)
      const days = Math.floor(diff / 86400000)
      
      if (minutes < 1) return 'À l\'instant'
      if (minutes < 60) return `Il y a ${minutes}min`
      if (hours < 24) return `Il y a ${hours}h`
      if (days < 7) return `Il y a ${days}j`
      
      return new Date(date).toLocaleDateString('fr-FR')
    }
  }
}
</script>

<style src="@/css/components/home.scss" lang="scss" scoped></style>
