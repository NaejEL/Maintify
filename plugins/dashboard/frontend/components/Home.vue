<template>
  <div class="dashboard-container">
    <div class="dashboard-header">
      <h1>📊 {{ pt('title') }}</h1>
      <p>{{ pt('subtitle') }}</p>
    </div>

    <div class="dashboard-content">
      <div class="dashboard-stats">
        <div class="stat-card">
          <div class="stat-icon">⚙️</div>
          <div class="stat-content">
            <h3>{{ equipmentCount }}</h3>
            <p>{{ pt('stats.equipment') }}</p>
          </div>
        </div>
        
        <div class="stat-card">
          <div class="stat-icon">🚨</div>
          <div class="stat-content">
            <h3>{{ alertsCount }}</h3>
            <p>{{ pt('stats.alerts') }}</p>
          </div>
        </div>
        
        <div class="stat-card">
          <div class="stat-icon">🔧</div>
          <div class="stat-content">
            <h3>{{ maintenanceCount }}</h3>
            <p>{{ pt('stats.maintenance') }}</p>
          </div>
        </div>
      </div>

      <div class="dashboard-overview">
        <h2>{{ pt('overview') }}</h2>
        <p>{{ pt('welcome') }}</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'DashboardView',
  
  data() {
    return {
      equipmentCount: 0,
      alertsCount: 0,
      maintenanceCount: 0
    }
  },
  
  mounted() {
    this.loadDashboardData()
  },
  
  methods: {
    // Helper pour les traductions du plugin
    pt(key) {
      return this.$t(`plugins.dashboard.${key}`)
    },
    
    // Helper pour vérifier si une clé de traduction existe
    pte(key) {
      return this.$te(`plugins.dashboard.${key}`)
    },
    
    async loadDashboardData() {
      try {
        // Simuler le chargement de données
        this.equipmentCount = 42
        this.alertsCount = 3
        this.maintenanceCount = 7
      } catch (error) {
        console.error('Erreur lors du chargement des données dashboard:', error)
      }
    }
  }
}
</script>

<style scoped src="../styles/dashboard.scss" lang="scss"></style>
