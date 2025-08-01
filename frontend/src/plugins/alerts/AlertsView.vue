<template>
  <div class="alerts-container">
    <div class="alerts-header">
      <h1>🚨 {{ $t('alerts.title') }}</h1>
      <p>{{ $t('alerts.subtitle') }}</p>
    </div>
    
    <div v-if="alerts.length === 0" class="no-alerts">
      <div class="empty-icon">✅</div>
      <p>{{ $t('alerts.noAlerts') }}</p>
    </div>
    
    <div v-else class="alerts-list">
      <div 
        v-for="alert in alerts" 
        :key="alert.id" 
        class="alert-item"
        :class="alert.severity"
      >
        <div class="alert-icon">
          {{ getAlertIcon(alert.severity) }}
        </div>
        <div class="alert-content">
          <div class="alert-title">{{ alert.message }}</div>
          <div class="alert-meta">
            <span class="alert-level">{{ $t(`alerts.severity.${alert.level}`) }}</span>
            <span class="alert-time">{{ formatDate(alert.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script>
import axios from 'axios'

export default {
  name: 'AlertsView',
  
  data() {
    return {
      alerts: []
    }
  },
  
  mounted() {
    this.loadAlerts()
  },
  
  methods: {
    async loadAlerts() {
      try {
        const response = await axios.get('/api/alerts')
        this.alerts = response.data
      } catch (error) {
        console.error('Erreur lors du chargement des alertes:', error)
        // Données fictives pour la démonstration
        this.alerts = [
          {
            id: 1,
            message: 'Température élevée détectée',
            level: 'high',
            severity: 'high',
            created_at: new Date()
          },
          {
            id: 2,
            message: 'Maintenance programmée',
            level: 'medium',
            severity: 'medium',
            created_at: new Date()
          }
        ]
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
    
    formatDate(date) {
      return new Date(date).toLocaleString()
    }
  }
}
</script>

<style scoped src="@/css/components/alerts-view.scss" lang="scss"></style>
