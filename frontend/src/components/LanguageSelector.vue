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

<style scoped>
.language-selector {
  position: relative;
  display: inline-block;
}

.language-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #f8f9fa;
  border: 1px solid #e1e8ed;
  border-radius: 8px;
  color: #2c3e50;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.3s ease;
  min-width: 80px;
}

.language-btn:hover,
.language-btn.active {
  background: #e9ecef;
  border-color: #adb5bd;
}

.flag {
  font-size: 1.1rem;
}

.lang-code {
  font-weight: 500;
  font-size: 0.8rem;
}

.arrow {
  font-size: 0.7rem;
  margin-left: auto;
  transition: transform 0.3s ease;
}

.language-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 5px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
  z-index: 1000;
  min-width: 160px;
  overflow: hidden;
  animation: slideDown 0.2s ease-out;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.language-option {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 16px;
  background: none;
  border: none;
  color: #2c3e50;
  font-size: 0.9rem;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.language-option:hover {
  background-color: #f8f9fa;
}

.language-option.selected {
  background-color: #e3f2fd;
  color: #1976d2;
}

.check {
  margin-left: auto;
  color: #4caf50;
  font-weight: bold;
}

.language-option .name {
  flex: 1;
  text-align: left;
  font-weight: 500;
}

.language-option .check {
  color: #27ae60;
  font-weight: bold;
}

/* Responsive */
@media (max-width: 768px) {
  .language-btn {
    padding: 6px 10px;
    min-width: 70px;
  }
  
  .lang-code {
    display: none;
  }
  
  .language-dropdown {
    right: 0;
    left: auto;
  }
}
</style>
