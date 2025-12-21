<template>
  <div>
    <div class="d-flex justify-content-between align-items-center mb-3">
      <h2>{{ isMyBenchmarksPage ? 'My Benchmarks' : 'Benchmarks' }}</h2>
      <router-link v-if="authStore.isAuthenticated" to="/benchmarks/new" class="btn btn-primary">
        <i class="fa-solid fa-plus"></i> New benchmark
      </router-link>
    </div>

    <!-- Search -->
    <form @submit.prevent="handleSearch" v-if="!isMyBenchmarksPage">
      <div class="input-group rounded mb-3">
        <input
          type="search"
          v-model="searchQuery"
          class="form-control rounded search-input"
          placeholder="Search title, description, or username..."
          aria-label="Search"
          aria-describedby="search-addon"
          :disabled="route.query.user_id !== undefined"
        />
        <span class="input-group-text border-0 search-btn-icon" id="search-addon">
          <button type="submit" class="btn btn-link p-0 m-0" :disabled="route.query.user_id !== undefined">
            <i class="fas fa-search"></i>
          </button>
        </span>
      </div>
    </form>

    <!-- Filter indicator -->
    <div v-if="(route.query.user_id || filterUserId) && !isMyBenchmarksPage" class="alert alert-info d-flex justify-content-between align-items-center mb-3" role="alert">
      <span>
        <i class="fas fa-filter"></i> Showing benchmarks by <strong>{{ filterUsername }}</strong>
      </span>
      <button type="button" class="btn btn-sm btn-outline-info" @click="clearFilter">
        <i class="fas fa-times"></i> Clear filter
      </button>
    </div>

    <!-- Sort buttons -->
    <div class="mb-3 d-flex gap-2 align-items-center">
      <small class="text-muted me-2">Sort by:</small>
      <button
        @click="toggleSort('name')"
        :class="['btn', 'btn-sm', sortKey === 'name' ? 'btn-primary' : 'btn-outline-secondary', 'sort-btn']"
      >
        <i class="fas fa-sort-alpha-down"></i> Name
        <span v-if="sortKey === 'name'" class="ms-1">
          <i :class="['fas', sortDirection === 'asc' ? 'fa-arrow-up' : 'fa-arrow-down']"></i>
        </span>
      </button>
      <button
        @click="toggleSort('date')"
        :class="['btn', 'btn-sm', sortKey === 'date' ? 'btn-primary' : 'btn-outline-secondary', 'sort-btn']"
      >
        <i class="fas fa-calendar-alt"></i> Date
        <span v-if="sortKey === 'date'" class="ms-1">
          <i :class="['fas', sortDirection === 'asc' ? 'fa-arrow-up' : 'fa-arrow-down']"></i>
        </span>
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
        class="list-group-item flex-column align-items-start benchmark-card"
        role="button"
        tabindex="0"
        :aria-label="`View benchmark: ${benchmark.Title}`"
        @click="navigateToBenchmark(benchmark.ID)"
        @keypress.enter="navigateToBenchmark(benchmark.ID)"
      >
        <div class="d-flex w-100 justify-content-between align-items-center benchmark-first-row">
          <h5 class="mb-1 text-truncate flex-grow-1" style="min-width: 0;">{{ benchmark.Title }}</h5>
          <small class="text-nowrap flex-shrink-0 benchmark-date-author-desktop">
            <span v-if="benchmark.UpdatedAt !== benchmark.CreatedAt" :title="`Created: ${formatRelativeDate(benchmark.CreatedAt)}`">
              {{ formatRelativeDate(benchmark.UpdatedAt) }}
            </span>
            <span v-else>
              {{ formatRelativeDate(benchmark.CreatedAt) }}
            </span>
            by <template v-if="benchmark.User">
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
        <div class="d-flex w-100 justify-content-between align-items-start benchmark-second-row">
          <p class="text-truncate benchmark-description">
            <small>{{ benchmark.Description || 'No description' }}</small>
          </p>
          <div class="benchmark-meta-group">
            <small v-if="benchmark.run_count" class="text-muted benchmark-metadata text-nowrap">
              {{ benchmark.run_count }} <i class="fa-solid fa-play"></i>
            </small>
            <small class="text-nowrap benchmark-date-author-mobile">
              <span v-if="benchmark.UpdatedAt !== benchmark.CreatedAt" :title="`Created: ${formatRelativeDate(benchmark.CreatedAt)}`">
                {{ formatRelativeDate(benchmark.UpdatedAt) }}
              </span>
              <span v-else>
                {{ formatRelativeDate(benchmark.CreatedAt) }}
              </span>
              by <template v-if="benchmark.User">
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
    </div>

    <!-- Empty state -->
    <div v-else class="alert alert-info" role="alert">
      <template v-if="isMyBenchmarksPage">
        You haven't created any benchmarks yet. 
        <router-link to="/benchmarks/new" class="alert-link">Create your first benchmark!</router-link>
      </template>
      <template v-else>
        No benchmarks found. {{ authStore.isAuthenticated ? 'Create your first benchmark!' : 'Login to create benchmarks.' }}
      </template>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="d-flex justify-content-center align-items-center gap-3 mt-3 flex-wrap">
      <nav aria-label="Benchmark pagination">
        <ul class="pagination mb-0">
          <!-- Previous button -->
          <li class="page-item" :class="{ disabled: currentPage <= 1 }">
            <a class="page-link" href="#" @click.prevent="goToPage(currentPage - 1)" aria-label="Previous page">
              <span aria-hidden="true">&laquo;</span>
            </a>
          </li>
          
          <!-- First page -->
          <li v-if="paginationPages.length > 0 && paginationPages[0] !== 1" class="page-item">
            <a class="page-link" href="#" @click.prevent="goToPage(1)">1</a>
          </li>
          
          <!-- Left ellipsis -->
          <li v-if="paginationPages.length > 0 && paginationPages[0] > 2" class="page-item disabled">
            <span class="page-link">...</span>
          </li>
          
          <!-- Page numbers -->
          <li 
            v-for="page in paginationPages" 
            :key="page" 
            class="page-item" 
            :class="{ active: page === currentPage }"
          >
            <a 
              class="page-link" 
              href="#" 
              @click.prevent="goToPage(page)"
              :aria-label="`Go to page ${page}`"
              :aria-current="page === currentPage ? 'page' : undefined"
            >
              {{ page }}
            </a>
          </li>
          
          <!-- Right ellipsis -->
          <li v-if="paginationPages.length > 0 && paginationPages[paginationPages.length - 1] < totalPages - 1" class="page-item disabled">
            <span class="page-link">...</span>
          </li>
          
          <!-- Last page -->
          <li v-if="paginationPages.length > 0 && paginationPages[paginationPages.length - 1] !== totalPages" class="page-item">
            <a class="page-link" href="#" @click.prevent="goToPage(totalPages)">{{ totalPages }}</a>
          </li>
          
          <!-- Next button -->
          <li class="page-item" :class="{ disabled: currentPage >= totalPages }">
            <a class="page-link" href="#" @click.prevent="goToPage(currentPage + 1)" aria-label="Next page">
              <span aria-hidden="true">&raquo;</span>
            </a>
          </li>
        </ul>
      </nav>
      
      <!-- Custom page input -->
      <div class="d-flex align-items-center gap-2 page-jump">
        <label for="page-input" class="text-nowrap mb-0 small">Go to:</label>
        <input
          id="page-input"
          v-model.number="pageInputValue"
          type="number"
          class="form-control form-control-sm page-input"
          :min="1"
          :max="totalPages"
          :placeholder="`1-${totalPages}`"
          @keyup.enter="goToCustomPage"
          aria-label="Enter page number"
        />
        <button
          class="btn btn-sm btn-primary"
          @click="goToCustomPage"
          :disabled="!isValidPageInput"
          aria-label="Go to page"
        >
          Go
        </button>
      </div>
    </div>

    <!-- Scroll to top button -->
    <transition name="fade">
      <button
        v-if="showScrollTop"
        @click="scrollToTop"
        class="btn btn-primary scroll-top-btn"
        aria-label="Scroll to top"
      >
        <i class="fas fa-arrow-up"></i>
      </button>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
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
const filterUserId = ref(null)
const filterUsername = ref('')
const sortKey = ref(null)
const sortDirection = ref('asc')
const showScrollTop = ref(false)
const windowWidth = ref(window.innerWidth)
const pageInputValue = ref(null)
const isInitialized = ref(false)

// Computed property to check if we're on "My Benchmarks" page
const isMyBenchmarksPage = computed(() => {
  return route.path === '/benchmarks/my'
})

// Initialize state from URL query parameters
function initializeFromURL() {
  // Page number
  const pageParam = parseInt(route.query.page)
  if (pageParam && pageParam > 0) {
    currentPage.value = pageParam
  }
  
  // Search query
  if (route.query.search) {
    searchQuery.value = route.query.search
  }
  
  // Sort parameters
  if (route.query.sort) {
    sortKey.value = route.query.sort
  }
  if (route.query.order && (route.query.order === 'asc' || route.query.order === 'desc')) {
    sortDirection.value = route.query.order
  }
  
  // User filter parameter
  if (route.query.user_id) {
    filterUserId.value = route.query.user_id
    // Note: filterUsername will be populated when benchmarks are loaded
  } else {
    filterUserId.value = null
    filterUsername.value = ''
  }
  
  isInitialized.value = true
}

// Update URL with current state
function updateURL() {
  if (!isInitialized.value) return
  
  const query = {}
  
  // Add page parameter if not on first page
  if (currentPage.value > 1) {
    query.page = currentPage.value
  }
  
  // Add search parameter if present
  if (searchQuery.value && !filterUserId.value) {
    query.search = searchQuery.value
  }
  
  // Add sort parameters if set
  if (sortKey.value) {
    query.sort = sortKey.value
    query.order = sortDirection.value
  }
  
  // Add user_id parameter if filter is active
  if (filterUserId.value) {
    query.user_id = filterUserId.value
  }
  
  // Only push if query actually changed
  const currentQuery = route.query
  const queryChanged = JSON.stringify(query) !== JSON.stringify(currentQuery)
  
  if (queryChanged) {
    router.push({ query })
  }
}

// Computed property to validate page input
const isValidPageInput = computed(() => {
  const value = pageInputValue.value
  return value && value >= 1 && value <= totalPages.value && value !== currentPage.value
})

// Computed property to calculate which page numbers to show
const paginationPages = computed(() => {
  const pages = []
  const current = currentPage.value
  const total = totalPages.value
  
  // Show max 7 page numbers (mobile: 5)
  const maxVisible = windowWidth.value <= 768 ? 5 : 7
  const halfVisible = Math.floor(maxVisible / 2)
  
  if (total <= maxVisible) {
    // Show all pages if total is less than max
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    // Calculate start and end of visible range
    let start = Math.max(1, current - halfVisible)
    let end = Math.min(total, current + halfVisible)
    
    // Adjust if we're near the beginning or end
    if (current <= halfVisible) {
      end = maxVisible
    } else if (current >= total - halfVisible) {
      start = total - maxVisible + 1
    }
    
    for (let i = start; i <= end; i++) {
      pages.push(i)
    }
  }
  
  return pages
})

async function loadBenchmarks() {
  try {
    loading.value = true
    error.value = null

    // Check if we're on "My Benchmarks" page
    let userId = route.query.user_id
    
    if (isMyBenchmarksPage.value) {
      // For "My Benchmarks", use current user's ID
      if (!authStore.user?.user_id) {
        // Not authenticated, redirect to login
        router.push('/login')
        return
      }
      userId = authStore.user.user_id
      filterUserId.value = userId
      filterUsername.value = authStore.user.username || 'You'
    } else if (userId && !filterUserId.value) {
      // Update local state based on URL query parameter
      filterUserId.value = userId
    }

    // Prepare sort parameters for backend
    let sortByParam = ''
    let sortOrderParam = ''
    if (sortKey.value === 'name') {
      sortByParam = 'title'
      sortOrderParam = sortDirection.value
    } else if (sortKey.value === 'date') {
      sortByParam = 'updated_at'
      sortOrderParam = sortDirection.value
    }

    let response
    if (userId) {
      response = await api.benchmarks.listByUser(
        userId,
        currentPage.value,
        perPage.value,
        sortByParam,
        sortOrderParam
      )
      // Update filterUsername from response if we have benchmarks and don't already have username
      if (!isMyBenchmarksPage.value && !filterUsername.value && response.benchmarks && response.benchmarks.length > 0 && response.benchmarks[0].User) {
        filterUsername.value = response.benchmarks[0].User.Username
      }
    } else {
      response = await api.benchmarks.list(
        currentPage.value,
        perPage.value,
        searchQuery.value,
        sortByParam,
        sortOrderParam
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

function toggleSort(key) {
  if (sortKey.value === key) {
    // Toggle direction if same key
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    // New sort key, default to ascending
    sortKey.value = key
    sortDirection.value = 'asc'
  }
  // Update URL with new sort order
  updateURL()
  // Reload benchmarks with new sort order from backend
  loadBenchmarks()
}

function goToPage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    // Update URL with new page
    updateURL()
    loadBenchmarks()
    // Scroll to top when changing pages for better UX
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

function goToCustomPage() {
  if (isValidPageInput.value) {
    goToPage(pageInputValue.value)
    pageInputValue.value = null // Clear input after navigation
  }
}

function handleSearch() {
  currentPage.value = 1
  filterUserId.value = null
  filterUsername.value = ''
  // Update URL - this will remove user_id and add search if present
  updateURL()
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
  searchQuery.value = ''
  // If on "My Benchmarks" page, go to regular benchmarks page
  if (isMyBenchmarksPage.value) {
    router.push('/')
  } else {
    // Update URL to clear all filters
    updateURL()
    loadBenchmarks()
  }
}

function navigateToBenchmark(id) {
  router.push(`/benchmarks/${id}`)
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handleScroll() {
  showScrollTop.value = window.scrollY > 300
}

function handleResize() {
  windowWidth.value = window.innerWidth
}

onMounted(() => {
  // Initialize state from URL parameters
  initializeFromURL()
  // Load benchmarks with initialized state
  loadBenchmarks()
  window.addEventListener('scroll', handleScroll)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleResize)
})

// Watch for route query changes (for browser back/forward and direct URL changes)
watch(() => route.query, (newQuery, oldQuery) => {
  if (!isInitialized.value) return
  
  // Handle page changes
  const newPage = parseInt(newQuery.page) || 1
  if (newPage !== currentPage.value) {
    currentPage.value = newPage
    loadBenchmarks()
    return
  }
  
  // Handle search changes
  const newSearch = newQuery.search || ''
  if (newSearch !== searchQuery.value) {
    searchQuery.value = newSearch
    currentPage.value = 1
    loadBenchmarks()
    return
  }
  
  // Handle sort changes
  const newSort = newQuery.sort || null
  const newOrder = newQuery.order || 'asc'
  if (newSort !== sortKey.value || newOrder !== sortDirection.value) {
    sortKey.value = newSort
    sortDirection.value = newOrder
    loadBenchmarks()
    return
  }
  
  // Handle user_id changes
  const newUserId = newQuery.user_id
  const oldUserId = oldQuery.user_id
  if (newUserId !== oldUserId) {
    // Sync filter state with URL
    if (newUserId) {
      filterUserId.value = newUserId
      // filterUsername will be populated when benchmarks are loaded
    } else {
      filterUserId.value = null
      filterUsername.value = ''
    }
    currentPage.value = 1
    loadBenchmarks()
  }
}, { deep: true })

// Watch for route path changes (e.g., switching between / and /benchmarks/my)
watch(() => route.path, (newPath, oldPath) => {
  if (newPath !== oldPath) {
    currentPage.value = 1
    filterUserId.value = null
    filterUsername.value = ''
    searchQuery.value = ''
    sortKey.value = null
    sortDirection.value = 'asc'
    // Re-initialize from URL for the new path
    initializeFromURL()
    loadBenchmarks()
  }
})
</script>

<style scoped>
/* Enhanced search input */
.search-input {
  transition: box-shadow 0.3s ease, border-color 0.3s ease;
}

.search-input:focus {
  box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
  border-color: #86b7fe;
}

.search-btn-icon {
  background-color: transparent;
  transition: background-color 0.3s ease;
}

.search-btn-icon:hover {
  background-color: var(--bs-tertiary-bg);
}

/* Sort buttons */
.sort-btn {
  transition: all 0.2s ease;
  border-radius: 0.375rem;
}

.sort-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.sort-btn:active {
  transform: translateY(0);
}

/* Enhanced benchmark cards */
.benchmark-card {
  cursor: pointer;
  transition: all 0.3s ease;
  border-radius: 0.5rem;
  margin-bottom: 0.75rem;
  background: var(--bs-secondary-bg);
  border: 1px solid var(--bs-border-color);
  position: relative;
  overflow: visible;
}

.benchmark-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(145deg, rgba(13, 110, 253, 0.05), transparent);
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}

.benchmark-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.3), 0 0 20px rgba(13, 110, 253, 0.1);
  border-color: rgba(13, 110, 253, 0.3);
  z-index: 1025;
}

.benchmark-card:hover::before {
  opacity: 1;
}

.benchmark-card:active {
  transform: translateY(-2px);
}

/* Scroll to top button */
.scroll-top-btn {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  width: 3rem;
  height: 3rem;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
  z-index: 1000;
  border: none;
}

.scroll-top-btn:hover {
  transform: translateY(-4px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.4);
}

.scroll-top-btn:active {
  transform: translateY(-2px);
}

/* Fade transition for scroll button */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: scale(0.8);
}

/* List group items */
.list-group-item {
  transition: background-color 0.2s;
}

.list-group-item:hover {
  background-color: var(--bs-tertiary-bg);
}

.stretched-link {
  text-decoration: none;
}

.gap-2 {
  gap: 0.5rem;
}

.benchmark-metadata {
  font-size: 0.8125rem;
  color: var(--bs-secondary-color);
}

.benchmark-description {
  flex: 1;
  min-width: 0;
  margin-right: 0.5rem;
}

.benchmark-meta-group {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

.benchmark-second-row {
  gap: 0.5rem;
}

/* Desktop: show date+author on first line, hide mobile version */
.benchmark-date-author-mobile {
  display: none;
}

.benchmark-date-author-desktop {
  display: inline;
  margin-left: 0.5rem;
}

/* Mobile responsive: hide date+author from first line, show on third line with metadata */
@media (max-width: 768px) {
  /* Hide desktop date+author, show mobile version */
  .benchmark-date-author-desktop {
    display: none;
  }
  
  .benchmark-date-author-mobile {
    display: inline;
  }
  
  .benchmark-second-row {
    flex-direction: column;
    align-items: flex-start !important;
  }
  
  .benchmark-meta-group {
    width: 100%;
    justify-content: space-between;
  }
  
  .benchmark-description {
    margin-right: 0;
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

/* Pagination styles */
.pagination {
  gap: 0.25rem;
}

.pagination .page-link {
  transition: all 0.2s ease;
  border-radius: 0.375rem;
  min-width: 2.5rem;
  text-align: center;
}

.pagination .page-item.active .page-link {
  background-color: var(--bs-primary);
  border-color: var(--bs-primary);
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(13, 110, 253, 0.3);
}

.pagination .page-item:not(.disabled) .page-link:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.pagination .page-item.disabled .page-link {
  cursor: not-allowed;
}

/* Page jump input styles */
.page-jump {
  font-size: 0.875rem;
}

.page-jump .page-input {
  width: 70px;
  text-align: center;
  transition: all 0.2s ease;
}

.page-jump .page-input:focus {
  box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
  border-color: #86b7fe;
}

.page-jump .btn {
  transition: all 0.2s ease;
}

.page-jump .btn:not(:disabled):hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

/* Mobile pagination adjustments */
@media (max-width: 768px) {
  .pagination {
    gap: 0.15rem;
  }
  
  .pagination .page-link {
    min-width: 2rem;
    padding: 0.375rem 0.5rem;
    font-size: 0.875rem;
  }
}
</style>
