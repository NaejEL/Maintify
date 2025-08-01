<template>
  <div class="reports-container">
    <div class="reports-header">
      <h1>📊 {{ pt('title') }}</h1>
      <p>{{ pt('subtitle') }}</p>
    </div>

    <div class="reports-filters">
      <div class="filter-group">
        <label>{{ pt('filters.dateRange') }}:</label>
        <select v-model="selectedPeriod" @change="loadReports">
          <option value="week">{{ pt('periods.week') }}</option>
          <option value="month">{{ pt('periods.month') }}</option>
          <option value="quarter">{{ pt('periods.quarter') }}</option>
          <option value="year">{{ pt('periods.year') }}</option>
        </select>
      </div>
      
      <div class="filter-group">
        <label>{{ pt('filters.equipment') }}:</label>
        <select v-model="selectedType" @change="loadReports">
          <option value="all">{{ pt('common.all') }}</option>
          <option value="maintenance">{{ pt('types.maintenance') }}</option>
          <option value="equipment">{{ pt('types.equipment') }}</option>
          <option value="alerts">{{ pt('types.alerts') }}</option>
        </select>
      </div>
    </div>

    <div v-if="reports.length === 0" class="no-reports">
      <div class="empty-icon">📄</div>
      <p>{{ pt('noReports') }}</p>
    </div>

    <div v-else class="reports-list">
      <div 
        v-for="report in reports" 
        :key="report.id"
        class="report-card"
      >
        <div class="report-header">
          <h3>{{ report.title }}</h3>
          <span class="report-type">{{ getTypeText(report.type) }}</span>
        </div>
        <div class="report-content">
          <p>{{ report.description }}</p>
          <div class="report-meta">
            <span>{{ pt('common.date') }}: {{ formatDate(report.created_at) }}</span>
            <span>{{ pt('filters.dateRange') }}: {{ getPeriodText(report.period) }}</span>
          </div>
        </div>
        <div class="report-actions">
          <button @click="exportReport(report)" class="btn-export">
            📥 {{ pt('actions.export') }}
          </button>
          <button @click="printReport(report)" class="btn-print">
            �️ {{ pt('actions.print') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ReportsView',
  
  data() {
    return {
      reports: [],
      selectedPeriod: 'month',
      selectedType: 'all'
    }
  },
  
  mounted() {
    this.loadReports()
  },
  
  methods: {
    // Helper pour les traductions du plugin
    pt(key) {
      return this.$t(`plugins.reports.${key}`)
    },
    
    // Helper pour vérifier si une clé de traduction existe
    pte(key) {
      return this.$te(`plugins.reports.${key}`)
    },
    
    async loadReports() {
      try {
        // Simuler le chargement de rapports
        this.reports = [
          {
            id: 1,
            title: 'Rapport de maintenance mensuel',
            description: 'Synthèse des interventions de maintenance',
            type: 'maintenance',
            period: 'month',
            created_at: new Date()
          },
          {
            id: 2,
            title: 'Analyse des équipements',
            description: 'État et performance des équipements',
            type: 'equipment',
            period: 'week',
            created_at: new Date()
          }
        ]
      } catch (error) {
        console.error('Erreur lors du chargement des rapports:', error)
      }
    },
    
    getTypeText(type) {
      return this.pt(`types.${type}`)
    },
    
    getPeriodText(period) {
      // Utiliser les clés de traduction maintenant disponibles
      return this.pt(`periods.${period}`)
    },
    
    exportReport(report) {
      console.log('Export du rapport:', report.title)
      // Implémentation de l'export
    },
    
    printReport(report) {
      console.log('Impression du rapport:', report.title)
      // Implémentation de l'impression
    },
    
    formatDate(date) {
      return new Date(date).toLocaleDateString()
    }
  }
}
</script>

<style scoped src="../styles/reports.scss" lang="scss"></style>
