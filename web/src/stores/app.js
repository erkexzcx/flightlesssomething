import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useAppStore = defineStore('app', () => {
  const version = ref('')
  const loading = ref(false)

  // Theme: 'light' or 'dark'
  const validThemes = ['light', 'dark']
  const storedTheme = localStorage.getItem('theme')
  const theme = ref(validThemes.includes(storedTheme) ? storedTheme : 'dark')
  
  // Initialize theme if not set
  if (!storedTheme) {
    localStorage.setItem('theme', 'dark')
  }

  // Calculation method: 'linear-interpolation' or 'mangohud-threshold'
  const validCalculationMethods = ['linear-interpolation', 'mangohud-threshold']
  const storedCalculationMethod = localStorage.getItem('calculationMethod')
  const calculationMethod = ref(validCalculationMethods.includes(storedCalculationMethod) ? storedCalculationMethod : 'linear-interpolation')
  
  // Initialize calculation method if not set
  if (!storedCalculationMethod) {
    localStorage.setItem('calculationMethod', 'linear-interpolation')
  }

  // Comparison mode: 'percentage', 'numbers', or 'numbers-diff'
  const validComparisonModes = ['percentage', 'numbers', 'numbers-diff']
  const storedComparisonMode = localStorage.getItem('comparisonMode')
  const comparisonMode = ref(validComparisonModes.includes(storedComparisonMode) ? storedComparisonMode : 'percentage')
  
  // Initialize comparison mode if not set
  if (!storedComparisonMode) {
    localStorage.setItem('comparisonMode', 'percentage')
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

  // Set calculation method
  function setCalculationMethod(newMethod) {
    calculationMethod.value = newMethod
    localStorage.setItem('calculationMethod', newMethod)
  }

  // Set comparison mode
  function setComparisonMode(newMode) {
    comparisonMode.value = newMode
    localStorage.setItem('comparisonMode', newMode)
  }

  return {
    version,
    loading,
    theme,
    calculationMethod,
    comparisonMode,
    fetchVersion,
    setTheme,
    toggleTheme,
    setCalculationMethod,
    setComparisonMode,
  }
})
