<template>
  <div>
    <h2>Settings</h2>
    <p class="text-muted mb-4">
      <i class="fa-solid fa-info-circle"></i> 
      These settings control how benchmark data is processed and displayed in charts.
      <strong>All settings are automatically saved and persist across page reloads.</strong>
    </p>

    <div class="card mb-4">
      <div class="card-body">
        <h5 class="card-title">
          <i class="fa-solid fa-filter"></i> Advanced Filter Settings
        </h5>
        
        <div class="filter-section mt-4">
          <h6 class="mb-3">FPS Calculation Filters</h6>
          
          <div class="form-check mb-3">
            <input 
              class="form-check-input" 
              type="checkbox" 
              id="filterExtremeSpikesCb"
              v-model="localFilterExtremeSpikes"
              @change="handleFilterChange"
            >
            <label class="form-check-label" for="filterExtremeSpikesCb">
              <strong>Filter extreme FPS spikes</strong>
              <span v-if="settingsChanged" class="badge bg-warning text-dark ms-2">
                <i class="fa-solid fa-sync-alt"></i> Setting changed
              </span>
            </label>
            <div class="form-text mt-2">
              <p class="mb-2">
                When enabled, extreme non-gameplay frames (e.g., loading screens, menus, cutscenes) 
                are filtered out when calculating high percentile FPS values (90th percentile and above, 
                including 97th percentile).
              </p>
              <p class="mb-2">
                <strong>How it works:</strong>
              </p>
              <ul class="mb-2">
                <li>For FPS-capped runs (e.g., 60 FPS): Filters frames above cap × 1.5</li>
                <li>For uncapped runs: Filters frames above median × 3</li>
              </ul>
              <p class="mb-2">
                <strong>Why this matters:</strong> Extreme spikes can inflate 97th percentile metrics, 
                making performance appear better than actual gameplay. This filter ensures fair comparisons 
                and more accurate representation of in-game performance.
              </p>
              <p class="mb-0">
                <strong>When to disable:</strong> If you want to see completely unfiltered data, 
                or if you believe the filtering is removing valid gameplay frames for your specific use case.
              </p>
              <p class="mb-0 mt-2 text-info">
                <i class="fa-solid fa-lightbulb"></i> 
                <strong>Recommendation:</strong> Keep this enabled (default) for best comparability. 
                Users should avoid benchmarking during loading screens and menus to ensure accurate results.
              </p>
            </div>
          </div>

          <div v-if="settingsChanged" class="alert alert-info mt-3" role="alert">
            <i class="fa-solid fa-info-circle"></i> 
            <strong>Settings saved!</strong> Your filter preferences have been saved and will persist across sessions.
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { useAppStore } from '../stores/app'

const appStore = useAppStore()

// Advanced filter settings
const localFilterExtremeSpikes = ref(appStore.filterExtremeSpikes)
const settingsChanged = ref(false)
let settingsChangedTimeout = null

// Handle filter setting changes
function handleFilterChange() {
  // Save to store (which persists to localStorage)
  appStore.setFilterExtremeSpikes(localFilterExtremeSpikes.value)
  
  // Show "settings changed" feedback
  settingsChanged.value = true
  
  // Clear any existing timeout
  if (settingsChangedTimeout) {
    clearTimeout(settingsChangedTimeout)
  }
  
  // Hide the "settings changed" message after 3 seconds
  settingsChangedTimeout = setTimeout(() => {
    settingsChanged.value = false
  }, 3000)
}

onUnmounted(() => {
  // Clear settings changed timeout
  if (settingsChangedTimeout) {
    clearTimeout(settingsChangedTimeout)
  }
})
</script>

<style scoped>
.filter-section {
  border-left: 3px solid var(--bs-primary);
  padding-left: 1.5rem;
  margin-left: 0.5rem;
}

.filter-section h6 {
  color: var(--bs-primary);
  font-weight: 600;
}

.form-check-input:checked {
  background-color: var(--bs-primary);
  border-color: var(--bs-primary);
}

.badge {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
}
</style>
