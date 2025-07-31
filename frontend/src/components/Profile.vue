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

<style scoped>
.profile-container {
  padding: 30px;
  max-width: 1000px;
  margin: 0 auto;
}

.profile-header {
  text-align: center;
  margin-bottom: 40px;
}

.profile-header h1 {
  color: #2c3e50;
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 10px;
}

.profile-header p {
  color: #7f8c8d;
  font-size: 1.1rem;
}

.profile-content {
  display: grid;
  grid-template-columns: 1fr;
  gap: 30px;
}

.profile-card {
  background: white;
  border-radius: 12px;
  padding: 30px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.profile-info {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e1e8ed;
}

.avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 1.8rem;
  font-weight: 700;
}

.user-details h2 {
  color: #2c3e50;
  margin: 0 0 8px 0;
  font-size: 1.5rem;
}

.user-role {
  display: inline-block;
  background-color: #e3f2fd;
  color: #1565c0;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.user-email {
  color: #7f8c8d;
  margin: 0;
  font-size: 1rem;
}

.profile-form h3 {
  color: #2c3e50;
  margin-bottom: 25px;
  font-size: 1.3rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #34495e;
  font-weight: 600;
  font-size: 0.9rem;
}

.form-group input {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e1e8ed;
  border-radius: 8px;
  font-size: 1rem;
  transition: border-color 0.3s ease;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #667eea;
}

.form-group input:disabled {
  background-color: #f8f9fa;
  cursor: not-allowed;
}

.update-btn {
  width: 100%;
  padding: 14px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.update-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
}

.update-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

@media (max-width: 768px) {
  .profile-container {
    padding: 20px 15px;
  }
  
  .form-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
  
  .profile-info {
    flex-direction: column;
    text-align: center;
  }
  
  .avatar {
    width: 60px;
    height: 60px;
    font-size: 1.4rem;
  }
}
</style>
