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

<style scoped>
.home {
  max-width: 1400px;
  margin: 0 auto;
  padding: 30px;
  min-height: 100vh;
}

/* En-tête de bienvenue */
.welcome-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 20px;
  padding: 40px;
  margin-bottom: 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: white;
  box-shadow: 0 10px 30px rgba(102, 126, 234, 0.3);
}

.welcome-content {
  flex: 1;
}

.welcome-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 10px;
  line-height: 1.2;
}

.welcome-subtitle {
  font-size: 1.2rem;
  opacity: 0.9;
  font-weight: 300;
}

.welcome-icon {
  font-size: 4rem;
  opacity: 0.8;
}

/* Grille du tableau de bord */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 30px;
}

.section-title {
  font-size: 1.4rem;
  font-weight: 600;
  margin-bottom: 20px;
  color: #2c3e50;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Section statistiques */
.stats-section {
  grid-column: 1 / -1;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.stat-card {
  background: white;
  padding: 25px;
  border-radius: 15px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
  display: flex;
  align-items: center;
  gap: 20px;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.12);
}

.stat-icon {
  font-size: 2.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  width: 70px;
  height: 70px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-number {
  font-size: 2rem;
  font-weight: 700;
  color: #2c3e50;
  line-height: 1;
}

.stat-label {
  color: #7f8c8d;
  font-size: 0.9rem;
  font-weight: 500;
}

/* Sections alertes et activité */
.alerts-section,
.activity-section {
  background: white;
  padding: 30px;
  border-radius: 15px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
}

.alerts-list,
.activity-list {
  max-height: 400px;
  overflow-y: auto;
}

.alert-item,
.activity-item {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 15px;
  border-radius: 10px;
  margin-bottom: 10px;
  transition: background-color 0.3s ease;
}

.alert-item:hover,
.activity-item:hover {
  background-color: #f8f9fa;
}

.alert-item.high {
  border-left: 4px solid #e74c3c;
}

.alert-item.medium {
  border-left: 4px solid #f39c12;
}

.alert-item.low {
  border-left: 4px solid #27ae60;
}

.alert-icon,
.activity-icon {
  font-size: 1.5rem;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8f9fa;
  border-radius: 50%;
}

.alert-title,
.activity-text {
  font-weight: 500;
  color: #2c3e50;
  margin-bottom: 5px;
}

.alert-meta,
.activity-time {
  font-size: 0.8rem;
  color: #7f8c8d;
}

/* Actions rapides */
.quick-actions {
  grid-column: 1 / -1;
  background: white;
  padding: 30px;
  border-radius: 15px;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
}

.action-buttons {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 12px;
  text-decoration: none;
  color: inherit;
  transition: all 0.3s ease;
}

.action-btn:hover {
  background: #e9ecef;
  transform: translateY(-2px);
}

.action-btn.admin {
  background: linear-gradient(135deg, #ff7675 0%, #fd79a8 100%);
  color: white;
}

.action-btn.admin:hover {
  background: linear-gradient(135deg, #e84393 0%, #fd79a8 100%);
}

.action-icon {
  font-size: 2rem;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  border-radius: 50%;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.action-btn.admin .action-icon {
  background: rgba(255, 255, 255, 0.2);
}

.action-title {
  font-weight: 600;
  font-size: 1.1rem;
  margin-bottom: 5px;
}

.action-desc {
  font-size: 0.9rem;
  opacity: 0.8;
}

/* État vide */
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #7f8c8d;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 15px;
}

/* Responsive */
@media (max-width: 1200px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
  
  .stats-section {
    grid-column: 1;
  }
  
  .quick-actions {
    grid-column: 1;
  }
}

@media (max-width: 768px) {
  .home {
    padding: 20px;
  }
  
  .welcome-header {
    padding: 30px 20px;
    flex-direction: column;
    text-align: center;
    gap: 20px;
  }
  
  .welcome-title {
    font-size: 2rem;
  }
  
  .welcome-subtitle {
    font-size: 1rem;
  }
  
  .stats-cards {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 15px;
  }
  
  .stat-card {
    padding: 20px;
    flex-direction: column;
    text-align: center;
    gap: 15px;
  }
  
  .stat-icon {
    width: 60px;
    height: 60px;
    font-size: 2rem;
  }
  
  .stat-number {
    font-size: 1.5rem;
  }
  
  .action-buttons {
    grid-template-columns: 1fr;
  }
  
  .action-btn {
    padding: 15px;
  }
  
  .alerts-section,
  .activity-section {
    padding: 20px;
  }
}

@media (max-width: 480px) {
  .stats-cards {
    grid-template-columns: 1fr;
  }
  
  .welcome-title {
    font-size: 1.6rem;
  }
  
  .section-title {
    font-size: 1.2rem;
  }
}
</style>
