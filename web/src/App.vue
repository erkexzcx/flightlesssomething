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
  // Check authentication state on app load
  authStore.checkAuth()
  // Fetch app version
  appStore.fetchVersion()
  // Apply theme on mount
  document.documentElement.setAttribute('data-bs-theme', appStore.theme)
})
</script>

<style>
/* Custom styles can go here if needed */
</style>
