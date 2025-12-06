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
          class="form-control rounded search-input"
          placeholder="Search title or description..."
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
    <div v-if="route.query.user_id || filterUserId" class="alert alert-info d-flex justify-content-between align-items-center mb-3" role="alert">
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
    <div v-if="totalPages > 1" class="d-flex justify-content-center mt-3">
      <nav aria-label="Benchmark pagination">
        <ul class="pagination">
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
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
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
const sortKey = ref(null)
const sortDirection = ref('asc')
const showScrollTop = ref(false)
const windowWidth = ref(window.innerWidth)

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

    // Check if filtering by user_id from URL
    const userId = route.query.user_id
    
    // Update local state based on URL
    if (userId && !filterUserId.value) {
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
      if (!filterUsername.value && response.benchmarks && response.benchmarks.length > 0 && response.benchmarks[0].User) {
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
  // Reload benchmarks with new sort order from backend
  loadBenchmarks()
}

function goToPage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    loadBenchmarks()
    // Scroll to top when changing pages for better UX
    window.scrollTo({ top: 0, behavior: 'smooth' })
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

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handleScroll() {
  showScrollTop.value = window.scrollY > 300
}

function handleResize() {
  windowWidth.value = window.innerWidth
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
  window.addEventListener('scroll', handleScroll)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleResize)
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
  overflow: hidden;
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
  border: 1px solid var(--bs-body-color);
  color: var(--bs-body-color);
}

.custom-popover {
  position: absolute;
  top: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  min-width: 200px;
  max-width: min(600px, calc(100vw - 2rem));
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
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
  border-bottom: 6px solid var(--bs-border-color);
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
  border-bottom: 5px solid var(--bs-body-bg);
}

.popover-header {
  padding: 0.5rem 0.75rem;
  background: var(--bs-secondary-bg);
  border-bottom: 1px solid var(--bs-border-color);
  border-radius: 0.375rem 0.375rem 0 0;
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--bs-body-color);
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
