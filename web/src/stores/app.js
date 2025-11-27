import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAppStore = defineStore('app', () => {
  const version = ref('dev')
  const loading = ref(false)

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
      // Keep default 'dev' version on error
    } finally {
      loading.value = false
    }
  }

  return {
    version,
    loading,
    fetchVersion,
  }
})
