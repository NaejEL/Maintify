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

<style src="@/css/components/user-management.scss" lang="scss" scoped></style>
