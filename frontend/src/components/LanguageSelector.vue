<template>
  <div class="language-selector">
    <button 
      @click="toggleDropdown" 
      class="language-btn"
      :class="{ 'active': showDropdown }"
    >
      <span class="flag">🇫🇷</span>
      <span class="lang-code">FR</span>
      <span class="arrow">{{ showDropdown ? '▲' : '▼' }}</span>
    </button>
    
    <div v-if="showDropdown" class="language-dropdown">
      <button
        @click="selectLanguage('fr')"
        class="language-option"
      >
        <span class="flag">🇫🇷</span>
        <span class="name">Français</span>
      </button>
      <button
        @click="selectLanguage('en')"
        class="language-option"
      >
        <span class="flag">🇺🇸</span>
        <span class="name">English</span>
      </button>
      <button
        @click="selectLanguage('es')"
        class="language-option"
      >
        <span class="flag">🇪🇸</span>
        <span class="name">Español</span>
      </button>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LanguageSelector',
  
  data() {
    return {
      showDropdown: false
    }
  },
  
  methods: {
    toggleDropdown() {
      this.showDropdown = !this.showDropdown
    },
    
    selectLanguage(locale) {
      console.log('Changing language to:', locale)
      this.showDropdown = false
      
      // Changer la langue via Vue I18n
      if (this.$i18n) {
        this.$i18n.locale = locale
        localStorage.setItem('maintify_locale', locale)
        document.documentElement.lang = locale
      }
    },
    
    // Fermer le dropdown quand on clique ailleurs
    handleClickOutside(event) {
      if (!this.$el.contains(event.target)) {
        this.showDropdown = false
      }
    }
  },
  
  mounted() {
    document.addEventListener('click', this.handleClickOutside)
  },
  
  beforeUnmount() {
    document.removeEventListener('click', this.handleClickOutside)
  }
}
</script>

<style scoped src="@/css/components/language-selector.scss" lang="scss"></style>
