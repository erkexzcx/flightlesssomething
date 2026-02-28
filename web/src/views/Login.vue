<template>
  <div class="row justify-content-center mt-5">
    <div class="col-md-6 col-lg-4">
      <h2 class="text-center mb-4">
        <i class="fa-solid fa-dove"></i> Login
      </h2>

      <!-- Discord Login -->
      <div class="card mb-3">
        <div class="card-body">
          <h5 class="card-title">Discord OAuth</h5>
          <p class="card-text">Login with your Discord account</p>
          <button @click="handleDiscordLogin" class="btn btn-primary w-100">
            <i class="fa-brands fa-discord"></i> Login with Discord
          </button>
        </div>
      </div>

      <!-- Admin Login -->
      <div class="card">
        <div class="card-body">
          <h5 class="card-title">Admin Login</h5>
          
          <form @submit.prevent="handleAdminLogin">
            <div class="mb-3">
              <label for="username" class="form-label">Username</label>
              <input
                type="text"
                class="form-control"
                id="username"
                v-model="username"
                required
                :disabled="loading"
              >
            </div>
            
            <div class="mb-3">
              <label for="password" class="form-label">Password</label>
              <input
                type="password"
                class="form-control"
                id="password"
                v-model="password"
                required
                :disabled="loading"
              >
            </div>

            <div v-if="error" class="alert alert-danger" role="alert">
              {{ error }}
            </div>

            <button type="submit" class="btn btn-success w-100" :disabled="loading">
              <span v-if="loading" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
              <i v-else class="fa-solid fa-right-to-bracket"></i>
              {{ loading ? 'Logging in...' : 'Login' }}
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref(null)

// Watch for authentication changes and redirect if authenticated
watch(() => authStore.isAuthenticated, (isAuth) => {
  if (isAuth) {
    router.replace('/benchmarks')
  }
}, { immediate: true })

function handleDiscordLogin() {
  authStore.loginDiscord()
}

async function handleAdminLogin() {
  try {
    loading.value = true
    error.value = null
    
    await authStore.loginAdmin(username.value, password.value)
    
    // Redirect to benchmarks page on success
    router.push('/benchmarks')
  } catch (err) {
    error.value = err.message || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.card {
  background-color: var(--bs-dark);
}

.card-title {
  color: var(--bs-light);
}
</style>
