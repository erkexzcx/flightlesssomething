<template>
  <div>
    <div class="d-flex justify-content-between align-items-center mb-3">
      <h2>Benchmarks</h2>
      <router-link v-if="authStore.isAuthenticated" to="/benchmarks/new" class="btn btn-primary">
        <i class="fa-solid fa-plus"></i> New benchmark
      </router-link>
    </div>

    <!-- Search -->
    <form @submit.prevent="handleSearch">
      <div class="input-group rounded mb-3">
        <input
          type="search"
          v-model="searchQuery"
          class="form-control rounded"
          placeholder="Search title or description..."
          aria-label="Search"
          aria-describedby="search-addon"
          :disabled="route.query.user_id !== undefined"
        />
        <span class="input-group-text border-0" id="search-addon">
          <button type="submit" class="btn btn-link p-0 m-0" :disabled="route.query.user_id !== undefined">
            <i class="fas fa-search"></i>
          </button>
        </span>
      </div>
    </form>

    <!-- Filter indicator -->
    <div v-if="route.query.user_id || filterUserId" class="alert alert-info d-flex justify-content-between align-items-center mb-3" role="alert">
      <span>
        <i class="fas fa-filter"></i> Showing benchmarks by <strong>{{ filterUsername }}</strong>
      </span>
      <button type="button" class="btn btn-sm btn-outline-info" @click="clearFilter">
        <i class="fas fa-times"></i> Clear filter
      </button>
    </div>

    <p>
      <small>Benchmarks found: {{ totalBenchmarks }}</small>
    </p>

    <!-- Loading state -->
    <div v-if="loading" class="text-center my-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="alert alert-danger" role="alert">
      {{ error }}
    </div>

    <!-- Benchmarks list -->
    <div v-else-if="benchmarks.length > 0" class="list-group mt-1">
      <div
        v-for="benchmark in benchmarks"
        :key="benchmark.ID"
        class="list-group-item flex-column align-items-start"
        role="button"
        tabindex="0"
        :aria-label="`View benchmark: ${benchmark.Title}`"
        @click="navigateToBenchmark(benchmark.ID)"
        @keypress.enter="navigateToBenchmark(benchmark.ID)"
      >
        <div class="d-flex w-100 justify-content-between align-items-center">
          <div class="d-flex align-items-center gap-2 flex-grow-1" style="min-width: 0;">
            <h5 class="mb-1 text-truncate flex-shrink-1">{{ benchmark.Title }}</h5>
            <div 
              v-if="benchmark.run_count"
              class="badge-wrapper flex-shrink-0"
              @mouseenter="showPopover(benchmark.ID)"
              @mouseleave="hidePopover(benchmark.ID)"
              @click.stop="togglePopover(benchmark.ID)"
            >
              <span class="badge badge-outline-white rounded-pill">
                {{ benchmark.run_count }}
              </span>
              <div 
                v-if="activePopover === benchmark.ID" 
                :ref="el => popoverRef = el"
                class="custom-popover"
                @mouseenter="keepPopoverOpen(benchmark.ID)"
                @mouseleave="hidePopover(benchmark.ID)"
                @click.stop
              >
                <div class="popover-header">Runs ({{ benchmark.run_count }})</div>
                <div class="popover-body">
                  <div 
                    v-for="(label, idx) in benchmark.run_labels" 
                    :key="idx"
                    class="run-label-item"
                  >
                    <span class="run-number">{{ idx + 1 }}.</span>
                    <span class="run-label">{{ label }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <small class="text-nowrap flex-shrink-0 ms-2">
            <span v-if="benchmark.UpdatedAt !== benchmark.CreatedAt" :title="`Created: ${formatRelativeDate(benchmark.CreatedAt)}`">
              {{ formatRelativeDate(benchmark.UpdatedAt) }}
            </span>
            <span v-else>
              {{ formatRelativeDate(benchmark.CreatedAt) }}
            </span>
          </small>
        </div>
        <div class="d-flex w-100 justify-content-between">
          <p class="mb-1 text-truncate">
            <small>{{ benchmark.Description || 'No description' }}</small>
          </p>
          <small class="text-nowrap">
            By <template v-if="benchmark.User">
              <a 
                v-if="!filterUserId && !route.query.user_id"
                href="#" 
                class="username-link" 
                @click.stop.prevent="filterByUser(benchmark.User)"
              >
                <b>{{ benchmark.User.Username }}<span v-if="benchmark.User.IsAdmin" class="admin-asterisk" title="Admin">*</span></b>
              </a>
              <b v-else>{{ benchmark.User.Username }}<span v-if="benchmark.User.IsAdmin" class="admin-asterisk" title="Admin">*</span></b>
            </template>
            <b v-else>Unknown</b>
          </small>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="alert alert-info" role="alert">
      No benchmarks found. {{ authStore.isAuthenticated ? 'Create your first benchmark!' : 'Login to create benchmarks.' }}
    </div>

    <!-- Pagination -->
    <div class="d-flex justify-content-center mt-3">
      <ul class="pagination">
        <li class="page-item" :class="{ disabled: currentPage <= 1 || totalBenchmarks <= perPage }">
          <a class="page-link" href="#" @click.prevent="goToPage(currentPage - 1)">
            Previous
          </a>
        </li>
        <li class="page-item disabled">
          <a class="page-link" href="#">{{ currentPage }} / {{ totalPages }}</a>
        </li>
        <li class="page-item" :class="{ disabled: currentPage >= totalPages || totalBenchmarks <= perPage }">
          <a class="page-link" href="#" @click.prevent="goToPage(currentPage + 1)">
            Next
          </a>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import { formatRelativeDate } from '../utils/dateFormatter'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const benchmarks = ref([])
const loading = ref(false)
const error = ref(null)
const currentPage = ref(1)
const totalPages = ref(1)
const totalBenchmarks = ref(0)
const perPage = ref(10)
const searchQuery = ref('')
const activePopover = ref(null)
const pinnedPopover = ref(null)
const hideTimeout = ref(null)
const filterUserId = ref(null)
const filterUsername = ref('')
const popoverRef = ref(null)

async function loadBenchmarks() {
  try {
    loading.value = true
    error.value = null

    // Check if filtering by user_id from URL
    const userId = route.query.user_id
    
    // Update local state based on URL
    if (userId && !filterUserId.value) {
      filterUserId.value = userId
    }

    let response
    if (userId) {
      response = await api.benchmarks.listByUser(
        userId,
        currentPage.value,
        perPage.value
      )
      // Update filterUsername from response if we have benchmarks and don't already have username
      if (!filterUsername.value && response.benchmarks && response.benchmarks.length > 0 && response.benchmarks[0].User) {
        filterUsername.value = response.benchmarks[0].User.Username
      }
    } else {
      response = await api.benchmarks.list(
        currentPage.value,
        perPage.value,
        searchQuery.value
      )
    }

    benchmarks.value = response.benchmarks || []
    totalBenchmarks.value = response.total || 0
    totalPages.value = response.total_pages || 1
  } catch (err) {
    error.value = err.message || 'Failed to load benchmarks'
    benchmarks.value = []
  } finally {
    loading.value = false
  }
}

function goToPage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    loadBenchmarks()
  }
}

function handleSearch() {
  currentPage.value = 1
  filterUserId.value = null
  filterUsername.value = ''
  // Remove user_id from URL
  router.push({ query: {} })
  loadBenchmarks()
}

function filterByUser(user) {
  if (user && user.ID) {
    filterUserId.value = user.ID
    filterUsername.value = user.Username
    searchQuery.value = ''
    currentPage.value = 1
    // Update URL to use user_id query parameter
    router.push({ query: { user_id: user.ID } })
  }
}

function clearFilter() {
  filterUserId.value = null
  filterUsername.value = ''
  currentPage.value = 1
  // Remove user_id from URL
  router.push({ query: {} })
}

function showPopover(benchmarkId) {
  // Cancel any pending hide timeout
  if (hideTimeout.value) {
    clearTimeout(hideTimeout.value)
    hideTimeout.value = null
  }
  
  // Always show on hover (unless a different one is pinned)
  if (pinnedPopover.value === null || pinnedPopover.value === benchmarkId) {
    activePopover.value = benchmarkId
    // Position popover on mobile after DOM update
    nextTick(() => {
      positionPopoverOnMobile()
    })
  }
}

function hidePopover(benchmarkId) {
  // Only hide on mouse leave if not pinned
  if (pinnedPopover.value !== benchmarkId && activePopover.value === benchmarkId) {
    // Cancel any existing timeout
    if (hideTimeout.value) {
      clearTimeout(hideTimeout.value)
    }
    
    // Small delay to allow moving between badge and popover
    hideTimeout.value = setTimeout(() => {
      if (activePopover.value === benchmarkId && pinnedPopover.value !== benchmarkId) {
        activePopover.value = null
      }
      hideTimeout.value = null
    }, 100)
  }
}

function togglePopover(benchmarkId) {
  // Cancel any pending hide timeout
  if (hideTimeout.value) {
    clearTimeout(hideTimeout.value)
    hideTimeout.value = null
  }
  
  // Toggle pin on click/touch
  if (pinnedPopover.value === benchmarkId) {
    pinnedPopover.value = null
    activePopover.value = null
  } else {
    pinnedPopover.value = benchmarkId
    activePopover.value = benchmarkId
    // Position popover on mobile after DOM update
    nextTick(() => {
      positionPopoverOnMobile()
    })
  }
}

function positionPopoverOnMobile() {
  if (!popoverRef.value) return
  
  // Get the badge wrapper element (parent of popover)
  const badgeWrapper = popoverRef.value.parentElement
  if (!badgeWrapper) return
  
  // Get position of badge wrapper relative to viewport
  const badgeRect = badgeWrapper.getBoundingClientRect()
  
  // On mobile (width <= 768px), use fixed positioning
  if (window.innerWidth <= 768) {
    // Calculate top position (below the badge)
    const top = badgeRect.bottom + 8
    
    // Set the top position as inline style
    popoverRef.value.style.top = `${top}px`
  } else {
    // On desktop, adjust if popover would go off-screen
    const popoverRect = popoverRef.value.getBoundingClientRect()
    const popoverWidth = popoverRect.width
    
    // Calculate where the centered popover would be
    const badgeCenter = badgeRect.left + (badgeRect.width / 2)
    const popoverLeft = badgeCenter - (popoverWidth / 2)
    
    // Check if it goes off-screen and adjust
    if (popoverLeft < 16) {
      // Would go off left edge, align to left with margin
      popoverRef.value.style.left = 'auto'
      popoverRef.value.style.right = 'auto'
      popoverRef.value.style.transform = 'none'
      popoverRef.value.style.marginLeft = `${16 - badgeRect.left}px`
    } else if (popoverLeft + popoverWidth > window.innerWidth - 16) {
      // Would go off right edge, align to right with margin
      popoverRef.value.style.left = 'auto'
      popoverRef.value.style.right = 'auto'
      popoverRef.value.style.transform = 'none'
      popoverRef.value.style.marginLeft = `${window.innerWidth - 16 - popoverWidth - badgeRect.left}px`
    }
    // Otherwise, leave it centered (default CSS handles this)
  }
}

function keepPopoverOpen(benchmarkId) {
  // Cancel any pending hide timeout when mouse enters popover
  if (hideTimeout.value) {
    clearTimeout(hideTimeout.value)
    hideTimeout.value = null
  }
  
  // Keep popover open when mouse enters it (for scrolling)
  activePopover.value = benchmarkId
}

function navigateToBenchmark(id) {
  router.push(`/benchmarks/${id}`)
}

// Handle clicking outside to close pinned popover
function handleClickOutside(event) {
  if (pinnedPopover.value !== null) {
    // Check if click is outside the popover
    const popoverElements = document.querySelectorAll('.badge-wrapper, .custom-popover')
    let clickedInside = false
    
    popoverElements.forEach(el => {
      if (el.contains(event.target)) {
        clickedInside = true
      }
    })
    
    if (!clickedInside) {
      pinnedPopover.value = null
      activePopover.value = null
    }
  }
}

onMounted(() => {
  loadBenchmarks()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// Watch for route query changes
watch(() => route.query.user_id, (newUserId, oldUserId) => {
  if (newUserId !== oldUserId) {
    currentPage.value = 1
    loadBenchmarks()
  }
})
</script>

<style scoped>
.list-group-item {
  cursor: pointer;
  transition: background-color 0.2s;
}

.list-group-item:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.stretched-link {
  text-decoration: none;
}

.gap-2 {
  gap: 0.5rem;
}

.badge-wrapper {
  position: relative;
  display: inline-block;
}

.badge {
  cursor: help;
  white-space: nowrap;
}

.badge-outline-white {
  background: transparent;
  border: 1px solid white;
  color: white;
}

.custom-popover {
  position: absolute;
  top: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  min-width: 200px;
  max-width: min(600px, calc(100vw - 2rem));
  background: var(--bs-dark);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 0.375rem;
  box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.5);
  z-index: 1050;
  animation: fadeIn 0.15s ease-in;
}

/* Adjust popover positioning on mobile */
@media (max-width: 768px) {
  /* On mobile, ensure list items don't cause horizontal overflow */
  .list-group-item {
    overflow: visible;
  }
  
  .custom-popover {
    /* On mobile, use different positioning strategy */
    position: fixed;
    top: auto;
    left: 1rem;
    right: 1rem;
    bottom: auto;
    transform: none;
    max-width: none;
    min-width: auto;
    /* Position below click point with margin */
    margin-top: 0.5rem;
  }
  
  /* Hide arrows on mobile since popover is fixed */
  .custom-popover::before,
  .custom-popover::after {
    display: none;
  }
}

.custom-popover::before {
  content: '';
  position: absolute;
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 6px solid transparent;
  border-right: 6px solid transparent;
  border-bottom: 6px solid rgba(255, 255, 255, 0.2);
}

.custom-popover::after {
  content: '';
  position: absolute;
  top: -5px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 5px solid transparent;
  border-right: 5px solid transparent;
  border-bottom: 5px solid var(--bs-dark);
}

.popover-header {
  padding: 0.5rem 0.75rem;
  background: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem 0.375rem 0 0;
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--bs-light);
}

.popover-body {
  padding: 0.5rem 0.75rem;
  max-height: calc(100vh - 150px);
  overflow-y: auto;
}

.run-label-item {
  display: flex;
  gap: 0.5rem;
  padding: 0.25rem 0;
  font-size: 0.875rem;
  color: var(--bs-light);
  white-space: nowrap;
}

.run-number {
  color: var(--bs-primary);
  font-weight: 500;
  min-width: 1.5rem;
  flex-shrink: 0;
}

.run-label {
  white-space: nowrap;
}

.run-label {
  word-break: break-word;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateX(-50%) translateY(-5px);
  }
  to {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
}

.text-truncate {
  max-width: 100%;
}

.username-link {
  color: var(--bs-primary);
  text-decoration: none;
  transition: color 0.2s;
}

.username-link:hover {
  color: var(--bs-primary);
  text-decoration: underline;
  opacity: 0.8;
}

.username-link b {
  font-weight: 600;
}

.admin-asterisk {
  color: var(--bs-warning);
  font-weight: bold;
  cursor: help;
  margin-left: 2px;
}
</style>
