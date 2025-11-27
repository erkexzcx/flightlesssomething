import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const isAuthenticated = ref(false)
  const isAdmin = ref(false)
  const loading = ref(false)
  const error = ref(null)

  // Check authentication state by making an API call
  async function checkAuth() {
    try {
      loading.value = true
      error.value = null
      
      // Check authentication with the new /api/auth/me endpoint
      const data = await api.auth.getCurrentUser()
      
      // Validate response data before setting auth state
      if (data && data.username !== undefined) {
        isAuthenticated.value = true
        user.value = {
          user_id: data.user_id,
          username: data.username,
          isAdmin: data.is_admin,
        }
        isAdmin.value = data.is_admin || false
      } else {
        // Invalid response data
        isAuthenticated.value = false
        user.value = null
        isAdmin.value = false
      }
    } catch (err) {
      // If API call fails (e.g., 401 unauthorized), user is not authenticated
      isAuthenticated.value = false
      user.value = null
      isAdmin.value = false
    } finally {
      loading.value = false
    }
  }

  async function loginAdmin(username, password) {
    try {
      loading.value = true
      error.value = null
      
      await api.auth.adminLogin(username, password)
      
      // After successful login, get user data from session
      await checkAuth()
      
      return true
    } catch (err) {
      error.value = err.message || 'Login failed'
      isAuthenticated.value = false
      user.value = null
      isAdmin.value = false
      throw err
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      loading.value = true
      error.value = null
      
      await api.auth.logout()
      
      isAuthenticated.value = false
      user.value = null
      isAdmin.value = false
    } catch (err) {
      error.value = err.message || 'Logout failed'
      throw err
    } finally {
      loading.value = false
    }
  }

  function loginDiscord() {
    api.auth.discordLogin()
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    loading,
    error,
    checkAuth,
    loginAdmin,
    logout,
    loginDiscord,
  }
})
