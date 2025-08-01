<template>
  <div class="plugin-test">
    <h2>Test Système de Plugins - Phase 2A</h2>
    
    <div class="test-section">
      <h3>1. État du store des plugins</h3>
      <pre>{{ pluginStoreState }}</pre>
    </div>

    <div class="test-section">
      <h3>2. Routes dynamiques chargées</h3>
      <pre>{{ dynamicRoutes }}</pre>
    </div>

    <div class="test-section">
      <h3>3. Locales des plugins</h3>
      <pre>{{ pluginLocales }}</pre>
    </div>

    <div class="test-section">
      <h3>4. Navigation des plugins</h3>
      <PluginNavigation />
    </div>

    <div class="test-section">
      <h3>5. Test de traduction</h3>
      <p>Dashboard (fr): {{ $t('plugins.dashboard.title') }}</p>
      <p>Alerts (fr): {{ $t('plugins.alerts.title') }}</p>
      <p>Reports (fr): {{ $t('plugins.reports.title') }}</p>
    </div>

    <div class="test-section">
      <h3>6. Boutons de test</h3>
      <button @click="reloadPlugins" class="test-btn">Recharger les plugins</button>
      <button @click="testNavigation" class="test-btn">Tester la navigation</button>
    </div>
  </div>
</template>

<script>
import { mapState, mapActions } from 'vuex'
import PluginNavigation from './PluginNavigation.vue'

export default {
  name: 'PluginTest',
  components: {
    PluginNavigation
  },
  computed: {
    ...mapState('plugins', ['plugins', 'loaded', 'error']),
    pluginStoreState() {
      return {
        loaded: this.loaded,
        error: this.error,
        pluginCount: Object.keys(this.plugins).length,
        pluginNames: Object.keys(this.plugins)
      }
    },
    dynamicRoutes() {
      return this.$router.getRoutes().filter(route => route.meta?.plugin)
    },
    pluginLocales() {
      const i18n = this.$i18n
      if (!i18n || !i18n.messages || !i18n.locale) {
        return 'i18n non disponible'
      }
      
      const messages = i18n.messages[i18n.locale]
      if (!messages) {
        return `Messages non trouvés pour la locale: ${i18n.locale}`
      }
      
      return messages.plugins || `Aucune locale plugin trouvée pour: ${i18n.locale}`
    }
  },
  async mounted() {
    if (!this.loaded) {
      await this.initialize()
    }
  },
  methods: {
    ...mapActions('plugins', ['initialize']),
    async reloadPlugins() {
      await this.initialize()
    },
    testNavigation() {
      console.log('Test de navigation vers /dashboard')
      this.$router.push('/dashboard')
    }
  }
}
</script>

<style scoped>
.plugin-test {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.test-section {
  margin-bottom: 30px;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 5px;
  background: #f9f9f9;
}

.test-section h3 {
  margin-top: 0;
  color: #333;
}

pre {
  background: #fff;
  padding: 10px;
  border: 1px solid #ccc;
  border-radius: 3px;
  overflow-x: auto;
  font-size: 12px;
}

.test-btn {
  padding: 10px 15px;
  margin-right: 10px;
  background: #007cba;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.test-btn:hover {
  background: #005a87;
}
</style>
