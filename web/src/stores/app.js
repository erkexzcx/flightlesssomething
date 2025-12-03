import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAppStore = defineStore('app', () => {
  const version = ref('')
  const loading = ref(false)
  
  // Calculation mode: 'original' or 'mangohud'
  const calculationMode = ref(localStorage.getItem('calculationMode') || 'original')

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

  // Set calculation mode
  function setCalculationMode(mode) {
    calculationMode.value = mode
    localStorage.setItem('calculationMode', mode)
  }

  return {
    version,
    loading,
    calculationMode,
    fetchVersion,
    setCalculationMode,
  }
})
