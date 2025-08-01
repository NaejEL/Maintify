import { nextTick } from 'vue'

class PluginI18nManager {
  constructor(i18nInstance) {
    this.i18n = i18nInstance
    this.pluginMessages = new Map()
    
    // Debug : vérifier la structure de l'instance i18n
    console.log('🔍 Debug i18n instance:', {
      hasGlobal: !!i18nInstance?.global,
      availableLocales: i18nInstance?.global?.availableLocales,
      directLocales: i18nInstance?.availableLocales
    })
  }

  /**
   * Enregistre les messages de traduction d'un plugin
   */
  registerPluginMessages(pluginName, messages) {
    try {
      this.pluginMessages.set(pluginName, messages)
      
      // Vérifier que l'instance i18n est bien initialisée
      if (!this.i18n || !this.i18n.global || !this.i18n.global.availableLocales) {
        console.warn('⚠️ Instance i18n non prête, les messages seront ajoutés plus tard:', pluginName)
        return
      }
      
      // Ajouter les messages à toutes les locales disponibles
      Object.keys(messages).forEach(locale => {
        if (this.i18n.global.availableLocales.includes(locale)) {
          const pluginKey = `plugins.${pluginName}`
          
          // Fusion des messages existants avec les nouveaux messages du plugin
          const currentMessages = this.i18n.global.getLocaleMessage(locale)
          const updatedMessages = {
            ...currentMessages,
            plugins: {
              ...currentMessages.plugins,
              [pluginName]: messages[locale]
            }
          }
          
          this.i18n.global.setLocaleMessage(locale, updatedMessages)
        }
      })
      
      console.log(`📝 Messages i18n enregistrés pour le plugin: ${pluginName}`)
      return true
    } catch (error) {
      console.error(`❌ Erreur lors de l'enregistrement des messages i18n pour ${pluginName}:`, error)
      return false
    }
  }

  /**
   * Charge les messages d'un plugin depuis un fichier ou objet
   */
  async loadPluginMessages(pluginName, messagesOrPath) {
    try {
      let messages
      
      if (typeof messagesOrPath === 'string') {
        // Si c'est un chemin, on essaie de charger le fichier
        // Pour l'instant on simule, plus tard on peut implémenter un vrai chargement
        console.warn(`Chargement de fichier i18n non implémenté: ${messagesOrPath}`)
        return false
      } else {
        // Si c'est déjà un objet, on l'utilise directement
        messages = messagesOrPath
      }
      
      return this.registerPluginMessages(pluginName, messages)
    } catch (error) {
      console.error(`❌ Erreur lors du chargement des messages pour ${pluginName}:`, error)
      return false
    }
  }

  /**
   * Supprime les messages d'un plugin
   */
  unregisterPluginMessages(pluginName) {
    try {
      this.pluginMessages.delete(pluginName)
      
      // Supprimer les messages de toutes les locales
      this.i18n.global.availableLocales.forEach(locale => {
        const currentMessages = this.i18n.global.getLocaleMessage(locale)
        if (currentMessages.plugins && currentMessages.plugins[pluginName]) {
          delete currentMessages.plugins[pluginName]
          this.i18n.global.setLocaleMessage(locale, currentMessages)
        }
      })
      
      console.log(`🗑️ Messages i18n supprimés pour le plugin: ${pluginName}`)
      return true
    } catch (error) {
      console.error(`❌ Erreur lors de la suppression des messages pour ${pluginName}:`, error)
      return false
    }
  }

  /**
   * Retourne les messages d'un plugin
   */
  getPluginMessages(pluginName) {
    return this.pluginMessages.get(pluginName)
  }

  /**
   * Retourne tous les plugins avec des messages i18n
   */
  getAllPluginMessages() {
    return Object.fromEntries(this.pluginMessages)
  }

  /**
   * Vérifie si un plugin a des messages enregistrés
   */
  hasPluginMessages(pluginName) {
    return this.pluginMessages.has(pluginName)
  }

  /**
   * Traduit une clé pour un plugin spécifique
   */
  t(pluginName, key, params = {}) {
    const fullKey = `plugins.${pluginName}.${key}`
    return this.i18n.global.t(fullKey, params)
  }

  /**
   * Vérifie si une clé de traduction existe pour un plugin
   */
  te(pluginName, key) {
    const fullKey = `plugins.${pluginName}.${key}`
    return this.i18n.global.te(fullKey)
  }

  /**
   * Recharge tous les messages en attente (pour quand i18n devient prêt)
   */
  retryPendingMessages() {
    console.log('🔄 Rechargement des messages en attente...')
    const pendingMessages = new Map(this.pluginMessages)
    
    for (const [pluginName, messages] of pendingMessages) {
      this.registerPluginMessages(pluginName, messages)
    }
  }
}

// Instance globale (sera initialisée dans main.js)
let pluginI18nManager = null

export function createPluginI18nManager(i18nInstance) {
  pluginI18nManager = new PluginI18nManager(i18nInstance)
  return pluginI18nManager
}

export function getPluginI18nManager() {
  if (!pluginI18nManager) {
    throw new Error('PluginI18nManager non initialisé. Appelez createPluginI18nManager() d\'abord.')
  }
  return pluginI18nManager
}

export default PluginI18nManager
