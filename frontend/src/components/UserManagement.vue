<template>
  <div class="user-management-container">
    <div class="header">
      <h1>👥 {{ $t('users.title') }}</h1>
      <p>{{ $t('users.subtitle') }}</p>
    </div>
    
    <div class="controls">
      <div class="search-box">
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="$t('users.search')"
          @input="filterUsers"
        />
      </div>
      
      <button @click="showCreateModal = true" class="add-user-btn">
        ➕ {{ $t('users.add') }}
      </button>
    </div>
    
    <div class="users-grid">
      <div
        v-for="user in filteredUsers"
        :key="user.id"
        class="user-card"
      >
        <div class="user-avatar">
          {{ getUserInitials(user) }}
        </div>
        
        <div class="user-info">
          <h3>{{ user.full_name }}</h3>
          <p class="user-email">{{ user.email }}</p>
          <span class="user-role" :class="getRoleClass(user.role)">
            {{ formatRole(user.role) }}
          </span>
        </div>
        
        <div class="user-actions">
          <button @click="editUser(user)" class="edit-btn">
            ✏️ Modifier
          </button>
          <button 
            @click="deleteUser(user)"
            class="delete-btn"
            :disabled="user.id === currentUser.id"
          >
            🗑️ Supprimer
          </button>
        </div>
      </div>
    </div>
    
    <!-- Modal de création/édition -->
    <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click="closeModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h2>{{ showCreateModal ? 'Ajouter un utilisateur' : 'Modifier l\'utilisateur' }}</h2>
          <button @click="closeModal" class="close-btn">✕</button>
        </div>
        
        <form @submit.prevent="saveUser" class="user-form">
          <div class="form-row">
            <div class="form-group">
              <label for="firstName">Prénom</label>
              <input
                id="firstName"
                v-model="userForm.first_name"
                type="text"
                required
              />
            </div>
            
            <div class="form-group">
              <label for="lastName">Nom</label>
              <input
                id="lastName"
                v-model="userForm.last_name"
                type="text"
                required
              />
            </div>
          </div>
          
          <div class="form-group">
            <label for="email">Email</label>
            <input
              id="email"
              v-model="userForm.email"
              type="email"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="phone">Téléphone</label>
            <input
              id="phone"
              v-model="userForm.phone"
              type="tel"
            />
          </div>
          
          <div class="form-group">
            <label for="role">Rôle</label>
            <select id="role" v-model="userForm.role" required>
              <option value="UserRole.USER">Utilisateur</option>
              <option value="UserRole.TECHNICIAN">Technicien</option>
              <option value="UserRole.ADMIN">Administrateur</option>
            </select>
          </div>
          
          <div v-if="showCreateModal" class="form-group">
            <label for="password">Mot de passe</label>
            <input
              id="password"
              v-model="userForm.password"
              type="password"
              required
            />
          </div>
          
          <div class="form-actions">
            <button type="button" @click="closeModal" class="cancel-btn">
              Annuler
            </button>
            <button type="submit" class="save-btn" :disabled="loading">
              {{ loading ? 'Sauvegarde...' : 'Sauvegarder' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import axios from 'axios'

export default {
  name: 'UserManagement',
  data() {
    return {
      users: [],
      filteredUsers: [],
      searchQuery: '',
      loading: false,
      showCreateModal: false,
      showEditModal: false,
      userForm: {
        first_name: '',
        last_name: '',
        email: '',
        phone: '',
        role: 'UserRole.USER',
        password: ''
      },
      editingUser: null
    }
  },
  
  computed: {
    ...mapGetters('auth', ['user']),
    
    currentUser() {
      return this.user
    }
  },
  
  mounted() {
    this.fetchUsers()
  },
  
  methods: {
    async fetchUsers() {
      try {
        const response = await axios.get('/api/auth/users')
        this.users = response.data.users
        this.filteredUsers = [...this.users]
      } catch (error) {
        this.$store.dispatch('showNotification', {
          type: 'error',
          message: 'Erreur lors du chargement des utilisateurs'
        })
      }
    },
    
    filterUsers() {
      const query = this.searchQuery.toLowerCase()
      this.filteredUsers = this.users.filter(user =>
        user.full_name.toLowerCase().includes(query) ||
        user.email.toLowerCase().includes(query)
      )
    },
    
    getUserInitials(user) {
      const firstName = user.first_name || ''
      const lastName = user.last_name || ''
      return (firstName.charAt(0) + lastName.charAt(0)).toUpperCase()
    },
    
    formatRole(role) {
      return role?.replace('UserRole.', '') || 'USER'
    },
    
    getRoleClass(role) {
      const roleType = this.formatRole(role).toLowerCase()
      return `role-${roleType}`
    },
    
    editUser(user) {
      this.editingUser = user
      this.userForm = {
        first_name: user.first_name,
        last_name: user.last_name,
        email: user.email,
        phone: user.phone || '',
        role: user.role,
        password: ''
      }
      this.showEditModal = true
    },
    
    async deleteUser(user) {
      if (confirm(`Êtes-vous sûr de vouloir supprimer ${user.full_name} ?`)) {
        try {
          await axios.delete(`/api/auth/users/${user.id}`)
          this.$store.dispatch('showNotification', {
            type: 'success',
            message: 'Utilisateur supprimé avec succès'
          })
          this.fetchUsers()
        } catch (error) {
          this.$store.dispatch('showNotification', {
            type: 'error',
            message: 'Erreur lors de la suppression'
          })
        }
      }
    },
    
    async saveUser() {
      this.loading = true
      
      try {
        if (this.showCreateModal) {
          await axios.post('/api/auth/register', this.userForm)
          this.$store.dispatch('showNotification', {
            type: 'success',
            message: 'Utilisateur créé avec succès'
          })
        } else {
          const updateData = { ...this.userForm }
          delete updateData.password // Ne pas inclure le mot de passe vide
          await axios.put(`/api/auth/users/${this.editingUser.id}`, updateData)
          this.$store.dispatch('showNotification', {
            type: 'success',
            message: 'Utilisateur modifié avec succès'
          })
        }
        
        this.closeModal()
        this.fetchUsers()
      } catch (error) {
        this.$store.dispatch('showNotification', {
          type: 'error',
          message: error.response?.data?.message || 'Erreur lors de la sauvegarde'
        })
      } finally {
        this.loading = false
      }
    },
    
    closeModal() {
      this.showCreateModal = false
      this.showEditModal = false
      this.editingUser = null
      this.userForm = {
        first_name: '',
        last_name: '',
        email: '',
        phone: '',
        role: 'UserRole.USER',
        password: ''
      }
    }
  }
}
</script>

<style scoped>
.user-management-container {
  padding: 30px;
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  text-align: center;
  margin-bottom: 40px;
}

.header h1 {
  color: #2c3e50;
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 10px;
}

.header p {
  color: #7f8c8d;
  font-size: 1.1rem;
}

.controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  gap: 20px;
}

.search-box input {
  padding: 12px 16px;
  border: 2px solid #e1e8ed;
  border-radius: 8px;
  font-size: 1rem;
  width: 300px;
  transition: border-color 0.3s ease;
}

.search-box input:focus {
  outline: none;
  border-color: #667eea;
}

.add-user-btn {
  padding: 12px 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.add-user-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
}

.users-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
}

.user-card {
  background: white;
  border-radius: 12px;
  padding: 25px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.user-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

.user-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 1.4rem;
  font-weight: 700;
  margin-bottom: 15px;
}

.user-info h3 {
  color: #2c3e50;
  margin: 0 0 8px 0;
  font-size: 1.2rem;
}

.user-email {
  color: #7f8c8d;
  margin: 0 0 10px 0;
  font-size: 0.9rem;
}

.user-role {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  margin-bottom: 15px;
}

.role-admin {
  background-color: #ffebee;
  color: #c62828;
}

.role-technician {
  background-color: #e8f5e8;
  color: #2e7d32;
}

.role-user {
  background-color: #e3f2fd;
  color: #1565c0;
}

.user-actions {
  display: flex;
  gap: 10px;
}

.edit-btn, .delete-btn {
  flex: 1;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: background-color 0.3s ease;
}

.edit-btn {
  background-color: #e3f2fd;
  color: #1565c0;
}

.edit-btn:hover {
  background-color: #bbdefb;
}

.delete-btn {
  background-color: #ffebee;
  color: #c62828;
}

.delete-btn:hover:not(:disabled) {
  background-color: #ffcdd2;
}

.delete-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 25px 30px 0;
  margin-bottom: 25px;
}

.modal-header h2 {
  color: #2c3e50;
  font-size: 1.5rem;
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #7f8c8d;
  padding: 5px;
}

.user-form {
  padding: 0 30px 30px;
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

.form-group input,
.form-group select {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e1e8ed;
  border-radius: 8px;
  font-size: 1rem;
  transition: border-color 0.3s ease;
  box-sizing: border-box;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: #667eea;
}

.form-actions {
  display: flex;
  gap: 15px;
  margin-top: 30px;
}

.cancel-btn, .save-btn {
  flex: 1;
  padding: 12px;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.3s ease;
}

.cancel-btn {
  background-color: #ecf0f1;
  color: #34495e;
}

.cancel-btn:hover {
  background-color: #d5dbdb;
}

.save-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.save-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.save-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .user-management-container {
    padding: 20px 15px;
  }
  
  .controls {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-box input {
    width: 100%;
  }
  
  .users-grid {
    grid-template-columns: 1fr;
  }
  
  .form-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
  
  .form-actions {
    flex-direction: column;
  }
}
</style>
