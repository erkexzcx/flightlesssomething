<template>
  <div class="container">
    <Navbar />
    <router-view />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import Navbar from './components/Navbar.vue'
import { useAuthStore } from './stores/auth'
import { useAppStore } from './stores/app'

const authStore = useAuthStore()
const appStore = useAppStore()

onMounted(() => {
  // Use idempotent init() instead of checkAuth() to avoid triggering a second
  // /api/auth/me request when the router guard has already called init().
  authStore.init()
  // Fetch app version
  appStore.fetchVersion()
  // Apply theme on mount
  document.documentElement.setAttribute('data-bs-theme', appStore.theme)
})
</script>

<style>
/* Custom styles can go here if needed */
</style>
