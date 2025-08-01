<template>
  <div class="plugin-nav">
    <router-link 
      v-for="item in pluginMenuItems" 
      :key="item.plugin + '-' + item.label"
      :to="getMenuPath(item)"
      :class="getMenuClass(item)"
      class="nav-link plugin-nav-link"
    >
      <span class="nav-icon">{{ item.icon }}</span>
      <span class="nav-text">{{ getMenuLabel(item) }}</span>
      <span v-if="item.plugin" class="plugin-badge">{{ item.plugin }}</span>
    </router-link>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'

export default {
  name: 'PluginNavigation',
  
  computed: {
    ...mapGetters('plugins', ['sortedMenuItems']),
    ...mapGetters('auth', ['user', 'isAdmin', 'isTechnician']),
    
    pluginMenuItems() {
      return this.sortedMenuItems.filter(item => this.canAccessMenuItem(item))
    }
  },
  
  methods: {
    getMenuPath(item) {
      // Trouver la route correspondante dans les plugins
      const pluginRoutes = this.$store.getters['plugins/routes']
      const route = pluginRoutes.find(r => r.plugin === item.plugin)
      return route ? route.path : '/'
    },
    
    getMenuClass(item) {
      const classes = []
      
      // Classe spéciale pour les plugins admin
      if (item.requiresAdmin || (item.meta && item.meta.requiresAdmin)) {
        classes.push('admin-link')
      }
      
      // Classe spéciale pour les plugins technicien
      if (item.requiresTechnician || (item.meta && item.meta.requiresTechnician)) {
        classes.push('tech-link')
      }
      
      return classes.join(' ')
    },
    
    getMenuLabel(item) {
      // Mapper navigation.{plugin} vers plugins.{plugin}.title pour les plugins
      if (item.label && item.label.startsWith('navigation.') && item.plugin) {
        const pluginName = item.label.replace('navigation.', '')
        const pluginKey = `plugins.${pluginName}.title`
        if (this.$te(pluginKey)) {
          return this.$t(pluginKey)
        }
      }
      
      // Essayer de traduire le label standard
      if (item.label && item.label.startsWith('navigation.')) {
        if (this.$te(item.label)) {
          return this.$t(item.label)
        }
      }
      
      // Essayer avec le namespace du plugin
      if (item.plugin && item.label) {
        const pluginKey = `plugins.${item.plugin}.${item.label}`
        if (this.$te(pluginKey)) {
          return this.$t(pluginKey)
        }
      }
      
      // Fallback sur le label brut
      return item.label || item.plugin || 'Plugin'
    },
    
    canAccessMenuItem(item) {
      // Vérifier les permissions d'accès
      if (item.requiresAdmin || (item.meta && item.meta.requiresAdmin)) {
        return this.isAdmin
      }
      
      if (item.requiresTechnician || (item.meta && item.meta.requiresTechnician)) {
        return this.isTechnician
      }
      
      // Par défaut, accessible à tous les utilisateurs connectés
      return true
    }
  }
}
</script>

<style scoped>
.plugin-nav {
  display: contents;
}

.plugin-nav-link {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-icon {
  font-size: 1.1rem;
}

.nav-text {
  flex: 1;
}

.plugin-badge {
  font-size: 0.7rem;
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.8);
  padding: 2px 6px;
  border-radius: 10px;
  font-weight: 500;
}

.tech-link {
  background-color: #e8f4fd;
  color: #1976d2;
}

.tech-link:hover {
  background-color: #bbdefb;
}

.admin-link {
  background-color: #fff3e0;
  color: #f57c00;
}

.admin-link:hover {
  background-color: #ffe0b2;
}
</style>
