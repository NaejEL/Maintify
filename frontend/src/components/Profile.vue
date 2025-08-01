<template>
  <div class="profile-container">
    <div class="profile-header">
      <h1>👤 {{ $t('profile.title') }}</h1>
      <p>{{ $t('profile.subtitle') }}</p>
    </div>
    
    <div class="profile-content">
      <div class="profile-card">
        <div class="profile-info">
          <div class="avatar">
            {{ userInitials }}
          </div>
          <div class="user-details">
            <h2>{{ user.full_name }}</h2>
            <p class="user-role">{{ formattedRole }}</p>
            <p class="user-email">{{ user.email }}</p>
          </div>
        </div>
        
        <form @submit.prevent="updateProfile" class="profile-form">
          <h3>Informations personnelles</h3>
          
          <div class="form-row">
            <div class="form-group">
              <label for="firstName">Prénom</label>
              <input
                id="firstName"
                v-model="profileForm.first_name"
                type="text"
                required
                :disabled="loading"
              />
            </div>
            
            <div class="form-group">
              <label for="lastName">Nom</label>
              <input
                id="lastName"
                v-model="profileForm.last_name"
                type="text"
                required
                :disabled="loading"
              />
            </div>
          </div>
          
          <div class="form-group">
            <label for="email">Email</label>
            <input
              id="email"
              v-model="profileForm.email"
              type="email"
              required
              :disabled="loading"
            />
          </div>
          
          <div class="form-group">
            <label for="phone">Téléphone</label>
            <input
              id="phone"
              v-model="profileForm.phone"
              type="tel"
              :disabled="loading"
            />
          </div>
          
          <button type="submit" class="update-btn" :disabled="loading">
            <span v-if="loading">Mise à jour...</span>
            <span v-else>Mettre à jour</span>
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters, mapActions } from 'vuex'

export default {
  name: 'Profile',
  data() {
    return {
      profileForm: {
        first_name: '',
        last_name: '',
        email: '',
        phone: ''
      },
      loading: false
    }
  },
  
  computed: {
    ...mapGetters('auth', ['user']),
    
    userInitials() {
      if (!this.user) return '??'
      const firstName = this.user.first_name || ''
      const lastName = this.user.last_name || ''
      return (firstName.charAt(0) + lastName.charAt(0)).toUpperCase()
    },
    
    formattedRole() {
      return this.user?.role?.replace('UserRole.', '') || ''
    }
  },
  
  mounted() {
    this.initializeForm()
  },
  
  methods: {
    ...mapActions('auth', ['updateProfile']),
    
    initializeForm() {
      if (this.user) {
        this.profileForm = {
          first_name: this.user.first_name || '',
          last_name: this.user.last_name || '',
          email: this.user.email || '',
          phone: this.user.phone || ''
        }
      }
    },
    
    async updateProfile() {
      this.loading = true
      
      try {
        await this.updateProfile(this.profileForm)
        this.$store.dispatch('showNotification', {
          type: 'success',
          message: 'Profil mis à jour avec succès'
        })
      } catch (error) {
        this.$store.dispatch('showNotification', {
          type: 'error',
          message: error.message
        })
      } finally {
        this.loading = false
      }
    }
  }
}
</script>

<style src="@/css/components/profile.scss" lang="scss" scoped></style>
