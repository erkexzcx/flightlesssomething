<template>
  <nav class="navbar navbar-expand-lg bg-body-tertiary rounded" aria-label="Main navigation">
    <div class="container-fluid">
      <router-link to="/benchmarks" class="navbar-brand" style="position: relative; display: inline-block;">
        <i class="fa-solid fa-dove"></i>
        FlightlessSomething
        <small style="font-size: 0.5em; color: gray; position: absolute; top: 3.1em; left: 2.65em;">{{ appStore.version }}</small>
      </router-link>

      <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav" aria-controls="navbarNav" aria-expanded="false" aria-label="Toggle navigation">
        <span class="navbar-toggler-icon"></span>
      </button>

      <div class="collapse navbar-collapse" id="navbarNav">
        <ul class="navbar-nav me-auto mb-2 mb-lg-0">
          <li class="nav-item">
            <router-link to="/benchmarks" class="nav-link" :class="{ active: $route.path === '/benchmarks' }">
              <i class="fa-solid fa-chart-line"></i> Benchmarks
            </router-link>
          </li>
          <li v-if="authStore.isAdmin" class="nav-item">
            <router-link to="/admin/users" class="nav-link" :class="{ active: $route.path === '/admin/users' }">
              <i class="fa-solid fa-users"></i> Users
            </router-link>
          </li>
          <li v-if="authStore.isAdmin" class="nav-item">
            <router-link to="/admin/logs" class="nav-link" :class="{ active: $route.path === '/admin/logs' }">
              <i class="fa-solid fa-clipboard-list"></i> Logs
            </router-link>
          </li>
        </ul>

        <ul class="navbar-nav">
          <li class="nav-item">
            <a class="nav-link" href="#" @click.prevent="appStore.toggleTheme()" title="Toggle theme">
              <i class="fa-solid" :class="appStore.theme === 'dark' ? 'fa-sun' : 'fa-moon'"></i>
            </a>
          </li>
          <li class="nav-item">
            <a class="nav-link" href="https://github.com/erkexzcx/flightlesssomething" target="_blank">
              <i class="fa-brands fa-github"></i> Source
            </a>
          </li>
          
          <template v-if="authStore.isAuthenticated">
            <li class="nav-item dropdown">
              <a class="nav-link dropdown-toggle" href="#" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                <i class="fa-solid fa-user"></i> {{ authStore.user?.username }}<span v-if="authStore.isAdmin" class="admin-asterisk" title="Admin">*</span>
              </a>
              <ul class="dropdown-menu dropdown-menu-end">
                <li>
                  <router-link to="/benchmarks/my" class="dropdown-item">
                    <i class="fa-solid fa-chart-simple"></i> My Benchmarks
                  </router-link>
                </li>
                <li>
                  <router-link to="/api-tokens" class="dropdown-item">
                    <i class="fa-solid fa-key"></i> API Tokens
                  </router-link>
                </li>
                <li><hr class="dropdown-divider"></li>
                <li>
                  <a class="dropdown-item" href="#" @click.prevent="handleLogout">
                    <i class="fa-solid fa-right-from-bracket"></i> Logout
                  </a>
                </li>
              </ul>
            </li>
          </template>
          
          <template v-else>
            <li class="nav-item">
              <router-link to="/login" class="nav-link">
                <i class="fa-brands fa-discord"></i> Login
              </router-link>
            </li>
          </template>
        </ul>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { useAuthStore } from '../stores/auth'
import { useAppStore } from '../stores/app'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const appStore = useAppStore()
const router = useRouter()

async function handleLogout() {
  try {
    await authStore.logout()
    router.push('/benchmarks')
  } catch (error) {
    console.error('Logout failed:', error)
    // In a production app, use a toast notification library like vue-toastification
    // For now, we'll just log the error and continue
  }
}
</script>

<style scoped>
.navbar {
  margin-bottom: 2rem;
}

.admin-asterisk {
  color: var(--bs-warning);
  font-weight: bold;
  cursor: help;
  margin-left: 2px;
}
</style>
