<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1>🔧 {{ $t('app.name') }}</h1>
        <p>{{ $t('app.tagline') }}</p>
      </div>
      
      <form @submit.prevent="handleLogin" class="login-form">
        <div class="form-group">
          <label for="username">{{ $t('auth.login.username') }}</label>
          <input
            id="username"
            v-model="credentials.username"
            type="text"
            required
            :placeholder="$t('auth.login.username')"
            :disabled="loading"
          />
        </div>
        
        <div class="form-group">
          <label for="password">{{ $t('auth.login.password') }}</label>
          <input
            id="password"
            v-model="credentials.password"
            type="password"
            required
            :placeholder="$t('auth.login.password')"
            :disabled="loading"
          />
        </div>
        
        <button type="submit" class="login-btn" :disabled="loading">
          <span v-if="loading">{{ $t('common.loading') }}</span>
          <span v-else>{{ $t('auth.login.submit') }}</span>
        </button>
        
        <div v-if="error" class="error-message">
          {{ error }}
        </div>
      </form>
      
      <div class="demo-accounts">
        <h3>{{ $t('auth.login.demoAccounts') }}:</h3>
        <div class="demo-buttons">
          <button @click="fillCredentials('admin', 'admin123')" class="demo-btn admin">
            👑 {{ $t('users.roles.admin') }}
          </button>
          <button @click="fillCredentials('technicien', 'tech123')" class="demo-btn tech">
            🔧 {{ $t('users.roles.technician') }}
          </button>
          <button @click="fillCredentials('utilisateur', 'user123')" class="demo-btn user">
            👤 {{ $t('users.roles.user') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Login',
  data() {
    return {
      credentials: {
        username: '',
        password: ''
      },
      loading: false,
      error: null
    }
  },
  methods: {
    async handleLogin() {
      this.loading = true;
      this.error = null;
      
      try {
        await this.$store.dispatch('auth/login', this.credentials);
        this.$router.push('/');
      } catch (error) {
        this.error = error.message || 'Erreur de connexion';
      } finally {
        this.loading = false;
      }
    },
    
    fillCredentials(username, password) {
      this.credentials.username = username;
      this.credentials.password = password;
    }
  }
}
</script>

<style src="@/css/components/login.scss" lang="scss" scoped></style>
