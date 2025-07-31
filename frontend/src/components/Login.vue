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

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.login-card {
  background: white;
  border-radius: 16px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  padding: 40px;
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h1 {
  color: #2c3e50;
  margin: 0 0 10px 0;
  font-size: 2.5rem;
  font-weight: 700;
}

.login-header p {
  color: #7f8c8d;
  margin: 0;
  font-size: 1rem;
}

.login-form {
  margin-bottom: 30px;
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

.login-btn {
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

.login-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.4);
}

.login-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}

.error-message {
  margin-top: 15px;
  padding: 12px;
  background-color: #fee;
  color: #c62828;
  border-radius: 8px;
  border: 1px solid #ffcdd2;
  text-align: center;
  font-size: 0.9rem;
}

.demo-accounts {
  border-top: 1px solid #e1e8ed;
  padding-top: 20px;
}

.demo-accounts h3 {
  color: #34495e;
  margin: 0 0 15px 0;
  font-size: 1rem;
  text-align: center;
}

.demo-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.demo-btn {
  flex: 1;
  padding: 10px 12px;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.2s ease;
  min-width: 0;
}

.demo-btn:hover {
  transform: translateY(-1px);
}

.demo-btn.admin {
  background-color: #e74c3c;
  color: white;
}

.demo-btn.tech {
  background-color: #f39c12;
  color: white;
}

.demo-btn.user {
  background-color: #27ae60;
  color: white;
}

@media (max-width: 480px) {
  .login-card {
    padding: 30px 20px;
    margin: 10px;
  }
  
  .demo-buttons {
    flex-direction: column;
  }
  
  .demo-btn {
    flex: none;
  }
}
</style>
