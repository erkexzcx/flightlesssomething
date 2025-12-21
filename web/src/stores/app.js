import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAppStore = defineStore('app', () => {
  const version = ref('')
  const loading = ref(false)

  // Theme: 'light' or 'dark'
  const storedTheme = localStorage.getItem('theme')
  const theme = ref(storedTheme || 'dark')
  
  // Initialize theme if not set
  if (!storedTheme) {
    localStorage.setItem('theme', 'dark')
  }

  // Fetch version from backend
  async function fetchVersion() {
    try {
      loading.value = true
      const data = await api.health()
      if (data && data.version) {
        version.value = data.version
      }
    } catch (err) {
      console.error('Failed to fetch version:', err)
      // Keep version empty on error
    } finally {
      loading.value = false
    }
  }

  // Set theme
  function setTheme(newTheme) {
    theme.value = newTheme
    localStorage.setItem('theme', newTheme)
    // Update the data-bs-theme attribute on the html element
    document.documentElement.setAttribute('data-bs-theme', newTheme)
  }

  // Toggle theme between light and dark
  function toggleTheme() {
    const newTheme = theme.value === 'dark' ? 'light' : 'dark'
    setTheme(newTheme)
  }

  return {
    version,
    loading,
    theme,
    fetchVersion,
    setTheme,
    toggleTheme,
  }
})
