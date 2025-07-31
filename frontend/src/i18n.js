import { createI18n } from 'vue-i18n'
import fr from './locales/fr.json'
import en from './locales/en.json'
import es from './locales/es.json'

// Fonction pour détecter la langue du navigateur
function getDefaultLocale() {
  const stored = localStorage.getItem('maintify_locale')
  if (stored) return stored
  
  const browserLang = navigator.language.split('-')[0]
  const supportedLocales = ['fr', 'en', 'es']
  
  return supportedLocales.includes(browserLang) ? browserLang : 'fr'
}

const i18n = createI18n({
  legacy: false, // Utilise la nouvelle API Composition
  locale: getDefaultLocale(),
  fallbackLocale: 'fr',
  globalInjection: true, // Permet d'utiliser $t dans les templates
  messages: {
    fr,
    en,
    es
  }
})

export default i18n

// Helper pour changer la langue
export function setLocale(locale) {
  i18n.global.locale.value = locale
  localStorage.setItem('maintify_locale', locale)
  document.documentElement.lang = locale
}

// Helper pour obtenir la langue actuelle
export function getCurrentLocale() {
  return i18n.global.locale.value
}

// Helper pour obtenir la liste des langues disponibles
export function getAvailableLocales() {
  return [
    { code: 'fr', name: 'Français', flag: '🇫🇷' },
    { code: 'en', name: 'English', flag: '🇺🇸' },
    { code: 'es', name: 'Español', flag: '🇪🇸' }
  ]
}
