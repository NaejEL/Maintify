<template>
  <div class="placeholder-container">
    <div class="placeholder-content">
      <div class="placeholder-icon">🚧</div>
      <h2 class="placeholder-title">{{ displayTitle }}</h2>
      <p class="placeholder-message">
        {{ displayMessage }}
      </p>
      <div v-if="pluginInfo" class="plugin-info">
        <h3>{{ $t('plugins.info.title') }}</h3>
        <ul>
          <li><strong>{{ $t('plugins.info.name') }} :</strong> {{ pluginInfo.displayName || pluginInfo.name }}</li>
          <li><strong>{{ $t('plugins.info.version') }} :</strong> {{ pluginInfo.version }}</li>
          <li><strong>{{ $t('plugins.info.description') }} :</strong> {{ pluginInfo.description }}</li>
          <li><strong>{{ $t('plugins.info.author') }} :</strong> {{ pluginInfo.author }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'PlaceholderView',
  props: {
    title: {
      type: String,
      default: ''
    },
    message: {
      type: String,
      default: ''
    },
    pluginName: {
      type: String,
      default: ''
    }
  },
  computed: {
    displayTitle() {
      return this.title || this.$t('plugins.placeholder.defaultTitle')
    },
    displayMessage() {
      return this.message || this.$t('plugins.placeholder.defaultMessage')
    },
    pluginInfo() {
      if (!this.pluginName) return null
      
      // On peut récupérer les infos du plugin depuis le store ou le plugin manager
      // Pour l'instant, on retourne null - sera implémenté avec le plugin manager
      return null
    }
  }
}
</script>

<style scoped src="@/css/components/placeholder-view.scss" lang="scss"></style>
