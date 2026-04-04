import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, APIError } from '../api/client'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const isAuthenticated = ref(false)
  const isAdmin = ref(false)
  const initialized = ref(false)
  const loading = ref(false)
  const error = ref(null)
  let _initPromise = null

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
      // Only treat 401 Unauthorized as "not authenticated"
      // Network errors or server errors (500) should not log the user out
      if (err instanceof APIError && err.status === 401) {
        isAuthenticated.value = false
        user.value = null
        isAdmin.value = false
      }
      // Silently ignore other errors to prevent spurious logout on transient failures
    } finally {
      loading.value = false
    }
  }

  // Idempotent init — resolves once auth check completes.
  // Call from router guards to avoid racing initial navigation with checkAuth.
  function init() {
    if (!_initPromise) {
      _initPromise = checkAuth().finally(() => {
        initialized.value = true
      })
    }
    return _initPromise
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
    } catch (err) {
      error.value = err.message || 'Logout failed'
      throw err
    } finally {
      isAuthenticated.value = false
      user.value = null
      isAdmin.value = false
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
    initialized,
    loading,
    error,
    checkAuth,
    init,
    loginAdmin,
    logout,
    loginDiscord,
  }
})
