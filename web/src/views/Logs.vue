<template>
  <div>
    <h2>Audit Logs</h2>
    <p class="text-muted">Admin only - View system activity and user actions</p>

    <!-- Filter section -->
    <div class="mb-3">
      <div class="row g-2">
        <div class="col-md-4">
          <input
            type="search"
            v-model="filterAction"
            class="form-control"
            placeholder="Filter by action..."
            @input="handleFilter"
          />
        </div>
        <div class="col-md-4">
          <select v-model="filterTargetType" class="form-select" @change="handleFilter">
            <option value="">All target types</option>
            <option value="user">Users</option>
            <option value="benchmark">Benchmarks</option>
          </select>
        </div>
        <div class="col-md-4">
          <button class="btn btn-outline-secondary w-100" @click="resetFilters">
            <i class="fas fa-redo"></i> Reset Filters
          </button>
        </div>
      </div>
    </div>

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

    <!-- Logs list -->
    <div v-else>
      <p>
        <small>Total logs: {{ totalLogs }}</small>
      </p>

      <div class="table-responsive">
        <table class="table table-striped table-hover">
          <thead>
            <tr>
              <th style="width: 180px">Time</th>
              <th style="width: 150px">User</th>
              <th style="width: 200px">Action</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.ID">
              <td>
                <small class="text-muted" :title="formatFullDate(log.CreatedAt)">
                  {{ log.CreatedAtHumanized }}
                </small>
              </td>
              <td>
                <router-link
                  v-if="log.User"
                  :to="`/benchmarks?user_id=${log.UserID}`"
                  class="text-decoration-none"
                  :title="`View benchmarks by ${log.User.Username}`"
                >
                  {{ log.User.Username }}
                </router-link>
                <span v-else class="text-muted">Unknown</span>
              </td>
              <td>
                <span class="badge" :class="getActionBadgeClass(log.Action)">
                  {{ log.Action }}
                </span>
              </td>
              <td>
                <span v-html="renderDescription(log)"></span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="d-flex justify-content-center align-items-center gap-3 mt-3 flex-wrap">
        <nav aria-label="Logs pagination">
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
          <label for="page-input-logs" class="text-nowrap mb-0 small">Go to:</label>
          <input
            id="page-input-logs"
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from '../api/client'

const router = useRouter()
const route = useRoute()

const logs = ref([])
const loading = ref(true)
const error = ref(null)
const currentPage = ref(1)
const perPage = ref(50)
const totalLogs = ref(0)
const totalPages = ref(0)
const filterAction = ref('')
const filterTargetType = ref('')
const isInitialized = ref(false)
const showScrollTop = ref(false)
const windowWidth = ref(window.innerWidth)
const pageInputValue = ref(null)

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

// Keep old visiblePages for backwards compatibility (not used anymore)
const visiblePages = computed(() => {
  const pages = []
  const maxVisible = 5
  let start = Math.max(1, currentPage.value - Math.floor(maxVisible / 2))
  let end = Math.min(totalPages.value, start + maxVisible - 1)

  if (end - start < maxVisible - 1) {
    start = Math.max(1, end - maxVisible + 1)
  }

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  return pages
})

// Initialize state from URL query parameters
function initializeFromURL() {
  const pageParam = parseInt(route.query.page)
  if (pageParam && pageParam > 0) {
    currentPage.value = pageParam
  }
  
  if (route.query.action) {
    filterAction.value = route.query.action
  }
  
  if (route.query.target_type) {
    filterTargetType.value = route.query.target_type
  }
  
  isInitialized.value = true
}

// Update URL with current state
function updateURL() {
  if (!isInitialized.value) return
  
  const query = {}
  
  if (currentPage.value > 1) {
    query.page = currentPage.value
  }
  
  if (filterAction.value) {
    query.action = filterAction.value
  }
  
  if (filterTargetType.value) {
    query.target_type = filterTargetType.value
  }
  
  const currentQuery = route.query
  const queryChanged = JSON.stringify(query) !== JSON.stringify(currentQuery)
  
  if (queryChanged) {
    router.push({ query })
  }
}

async function fetchLogs() {
  loading.value = true
  error.value = null

  try {
    const filters = {}
    if (filterAction.value) {
      filters.action = filterAction.value
    }
    if (filterTargetType.value) {
      filters.targetType = filterTargetType.value
    }

    const response = await api.admin.listLogs(currentPage.value, perPage.value, filters)

    logs.value = response.logs || []
    totalLogs.value = response.total
    totalPages.value = response.total_pages
    currentPage.value = response.page
  } catch (err) {
    if (err.status === 403) {
      error.value = 'Admin privileges required'
    } else {
      error.value = err.message || 'Failed to load audit logs'
    }
  } finally {
    loading.value = false
  }
}

function goToPage(page) {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  updateURL()
  fetchLogs()
  // Scroll to top when changing pages for better UX
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function goToCustomPage() {
  if (isValidPageInput.value) {
    goToPage(pageInputValue.value)
    pageInputValue.value = null // Clear input after navigation
  }
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

function handleFilter() {
  currentPage.value = 1
  updateURL()
  fetchLogs()
}

function resetFilters() {
  filterAction.value = ''
  filterTargetType.value = ''
  currentPage.value = 1
  updateURL()
  fetchLogs()
}

function formatDate(dateString) {
  const date = new Date(dateString)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
}

function formatFullDate(dateString) {
  const date = new Date(dateString)
  return date.toLocaleString()
}

function getActionBadgeClass(action) {
  const actionLower = action.toLowerCase()
  
  if (actionLower.includes('created')) {
    return 'bg-success'
  } else if (actionLower.includes('deleted')) {
    return 'bg-danger'
  } else if (actionLower.includes('updated') || actionLower.includes('edited')) {
    return 'bg-info'
  } else if (actionLower.includes('banned')) {
    return 'bg-danger'
  } else if (actionLower.includes('unbanned')) {
    return 'bg-success'
  } else if (actionLower.includes('admin granted')) {
    return 'bg-warning text-dark'
  } else if (actionLower.includes('admin revoked')) {
    return 'bg-secondary'
  }
  
  return 'bg-secondary'
}

function renderDescription(log) {
  let description = escapeHtml(log.Description)
  
  // Make benchmark IDs clickable - safe because we control the format
  // Benchmark IDs are always numeric and come from the backend
  description = description.replace(
    /#(\d+)/g,
    '<a href="/benchmarks/$1" class="text-decoration-none">#$1</a>'
  )
  
  // For user actions, create a link based on the target_id if it's a user
  // This is safer than parsing the description string
  if (log.TargetType === 'user' && log.TargetID) {
    // Find the username in the description (it's always at the end after "user: ")
    const match = description.match(/user: ([^<]+)$/)
    if (match) {
      const escapedUsername = match[1]
      description = description.replace(
        /user: ([^<]+)$/,
        `user: <a href="/benchmarks?user_id=${log.TargetID}" class="text-decoration-none">${escapedUsername}</a>`
      )
    }
  }
  
  return description
}

function escapeHtml(text) {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

onMounted(() => {
  initializeFromURL()
  fetchLogs()
  window.addEventListener('scroll', handleScroll)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleResize)
})

// Watch for route query changes (for browser back/forward)
watch(() => route.query, (newQuery) => {
  if (!isInitialized.value) return
  
  const newPage = parseInt(newQuery.page) || 1
  const newAction = newQuery.action || ''
  const newTargetType = newQuery.target_type || ''
  
  let shouldFetch = false
  
  if (newPage !== currentPage.value) {
    currentPage.value = newPage
    shouldFetch = true
  }
  
  if (newAction !== filterAction.value) {
    filterAction.value = newAction
    currentPage.value = 1
    shouldFetch = true
  }
  
  if (newTargetType !== filterTargetType.value) {
    filterTargetType.value = newTargetType
    currentPage.value = 1
    shouldFetch = true
  }
  
  if (shouldFetch) {
    fetchLogs()
  }
}, { deep: true })
</script>

<style scoped>
.table td {
  vertical-align: middle;
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
