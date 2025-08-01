<template>
  <div class="alerts-container">
    <div class="alerts-header">
      <h1>🚨 {{ pt('title') }}</h1>
      <p>{{ pt('subtitle') }}</p>
    </div>
    
    <div v-if="alerts.length === 0" class="no-alerts">
      <div class="empty-icon">✅</div>
      <p>{{ pt('noAlerts') }}</p>
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
            <span class="alert-level">{{ getSeverityText(alert.level) }}</span>
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
    // Helper pour les traductions du plugin
    pt(key) {
      // Plugin Translation: traduit une clé en utilisant le namespace du plugin
      return this.$t(`plugins.alerts.${key}`)
    },
    
    // Helper pour vérifier si une clé de traduction existe
    pte(key) {
      return this.$te(`plugins.alerts.${key}`)
    },
    
    // Mapper les niveaux de l'API vers les niveaux de traduction normalisés
    normalizeLevel(level) {
      const levelMapping = {
        'critical': 'high',
        'high': 'high',
        'warning': 'medium',
        'medium': 'medium',
        'info': 'low',
        'low': 'low'
      }
      return levelMapping[level] || 'medium'
    },
    
    // Helper pour obtenir le texte du niveau de sévérité
    getSeverityText(level) {
      const mappedLevel = this.normalizeLevel(level)
      return this.pt(`severity.${mappedLevel}`)
    },
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
      const mappedLevel = this.normalizeLevel(severity)
      
      switch (mappedLevel) {
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

<style scoped src="../styles/alerts.scss" lang="scss"></style>
