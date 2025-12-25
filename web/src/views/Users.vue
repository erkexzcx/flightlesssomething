<template>
  <div>
    <h2>User Management</h2>
    <p class="text-muted">Admin only - Manage users and their content</p>

    <!-- Search bar -->
    <div class="mb-3">
      <form @submit.prevent="handleSearch">
        <div class="input-group">
          <input
            type="search"
            v-model="searchQuery"
            class="form-control"
            placeholder="Search by username or Discord ID..."
            aria-label="Search users"
          />
          <button type="submit" class="btn btn-outline-secondary">
            <i class="fas fa-search"></i> Search
          </button>
        </div>
      </form>
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

    <!-- Users list -->
    <div v-else>
      <p>
        <small>Total users: {{ totalUsers }}</small>
      </p>

      <div class="table-responsive">
        <table class="table table-striped">
          <thead>
            <tr>
              <th>ID</th>
              <th>Username</th>
              <th>Discord ID</th>
              <th>Benchmarks</th>
              <th>API Tokens</th>
              <th>Last Web Activity</th>
              <th>Last API Activity</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.ID">
              <td>{{ user.ID }}</td>
              <td>
                <router-link 
                  :to="`/benchmarks?user_id=${user.ID}`" 
                  class="text-decoration-none"
                  :title="`View benchmarks by ${user.Username}`"
                >
                  {{ user.Username }}
                </router-link>
                <span v-if="user.IsAdmin" class="badge bg-warning ms-1">Admin</span>
                <span v-if="user.IsBanned" class="badge bg-danger ms-1">Banned</span>
              </td>
              <td><small>{{ user.DiscordID }}</small></td>
              <td>{{ user.benchmark_count || 0 }}</td>
              <td>{{ user.api_token_count || 0 }}</td>
              <td>
                <small class="text-muted" v-if="user.LastWebActivityAt">
                  {{ formatRelativeDate(user.LastWebActivityAt, 'Never') }}
                </small>
                <small class="text-muted" v-else>Never</small>
              </td>
              <td>
                <small class="text-muted" v-if="user.LastAPIActivityAt">
                  {{ formatRelativeDate(user.LastAPIActivityAt, 'Never') }}
                </small>
                <small class="text-muted" v-else>Never</small>
              </td>
              <td>
                <small class="text-muted">
                  Joined {{ formatRelativeDate(user.CreatedAt) }}
                </small>
              </td>
              <td>
                <div class="btn-group btn-group-sm" role="group">
                  <button
                    class="btn btn-outline-danger"
                    @click="confirmDeleteBenchmarks(user)"
                    :disabled="!user.benchmark_count"
                    title="Delete all user's benchmarks"
                  >
                    <i class="fa-solid fa-trash"></i> Del Benchmarks
                  </button>
                  <button
                    v-if="!user.IsBanned"
                    class="btn btn-outline-warning"
                    @click="confirmBanUser(user)"
                    :disabled="isCurrentUser(user)"
                    :title="isCurrentUser(user) ? 'You cannot ban yourself' : 'Ban user from signing in'"
                  >
                    <i class="fa-solid fa-ban"></i> Ban
                  </button>
                  <button
                    v-else
                    class="btn btn-outline-success"
                    @click="confirmUnbanUser(user)"
                    :disabled="isCurrentUser(user)"
                    :title="isCurrentUser(user) ? 'You cannot unban yourself' : 'Unban user'"
                  >
                    <i class="fa-solid fa-check"></i> Unban
                  </button>
                  <button
                    v-if="!user.IsAdmin"
                    class="btn btn-outline-primary"
                    @click="toggleAdmin(user, true)"
                    title="Make user an admin"
                  >
                    <i class="fa-solid fa-star"></i> Make Admin
                  </button>
                  <button
                    v-else
                    class="btn btn-outline-secondary"
                    @click="toggleAdmin(user, false)"
                    title="Remove admin privileges"
                  >
                    <i class="fa-solid fa-star-half-stroke"></i> Remove Admin
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="d-flex justify-content-center align-items-center gap-3 mt-3 flex-wrap">
        <nav aria-label="User pagination">
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
          <label for="page-input-users" class="text-nowrap mb-0 small">Go to:</label>
          <input
            id="page-input-users"
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
import { useAuthStore } from '../stores/auth'
import { formatRelativeDate } from '../utils/dateFormatter'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const users = ref([])
const loading = ref(false)
const error = ref(null)
const currentPage = ref(1)
const perPage = ref(10)
const totalUsers = ref(0)
const totalPages = ref(0)
const searchQuery = ref('')
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

// Keep old displayPages for backwards compatibility (not used anymore)
const displayPages = computed(() => {
  const pages = []
  const maxDisplay = 5
  let start = Math.max(1, currentPage.value - Math.floor(maxDisplay / 2))
  let end = Math.min(totalPages.value, start + maxDisplay - 1)

  if (end - start < maxDisplay - 1) {
    start = Math.max(1, end - maxDisplay + 1)
  }

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }
  return pages
})

// Helper function to check if a user is the current logged-in user
const isCurrentUser = (user) => {
  // authStore.user.user_id is from the session API (snake_case)
  // user.ID is from the users list API (PascalCase from GORM)
  if (!authStore.user?.user_id || !user.ID) {
    return false
  }
  return Number(authStore.user.user_id) === Number(user.ID)
}

// Initialize state from URL query parameters
function initializeFromURL() {
  const pageParam = parseInt(route.query.page)
  if (pageParam && pageParam > 0) {
    currentPage.value = pageParam
  }
  
  if (route.query.search) {
    searchQuery.value = route.query.search
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
  
  if (searchQuery.value) {
    query.search = searchQuery.value
  }
  
  const currentQuery = route.query
  const queryChanged = JSON.stringify(query) !== JSON.stringify(currentQuery)
  
  if (queryChanged) {
    router.push({ query })
  }
}

async function fetchUsers() {
  try {
    loading.value = true
    error.value = null

    const data = await api.admin.listUsers(currentPage.value, perPage.value, searchQuery.value)

    users.value = data.users || []
    totalUsers.value = data.total || 0
    totalPages.value = data.total_pages || 0
  } catch (err) {
    error.value = err.message || 'Failed to fetch users'
    console.error('Error fetching users:', err)
  } finally {
    loading.value = false
  }
}

function goToPage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    updateURL()
    fetchUsers()
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

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function handleScroll() {
  showScrollTop.value = window.scrollY > 300
}

function handleResize() {
  windowWidth.value = window.innerWidth
}

function handleSearch() {
  currentPage.value = 1
  updateURL()
  fetchUsers()
}

async function toggleAdmin(user, makeAdmin) {
  const action = makeAdmin ? 'make an admin' : 'remove admin privileges from'
  const message = `Are you sure you want to ${action} user "${user.Username}"?`
  
  if (!confirm(message)) {
    return
  }

  try {
    loading.value = true
    await api.admin.toggleUserAdmin(user.ID, makeAdmin)
    await fetchUsers()
  } catch (err) {
    error.value = err.message || 'Failed to update admin privileges'
    console.error('Error updating admin privileges:', err)
  } finally {
    loading.value = false
  }
}

async function confirmDeleteBenchmarks(user) {
  if (!confirm(`Are you sure you want to delete ALL benchmarks for user "${user.Username}"? This cannot be undone.`)) {
    return
  }

  try {
    loading.value = true
    await api.admin.deleteUserBenchmarks(user.ID)
    alert(`All benchmarks for user "${user.Username}" have been deleted.`)
    await fetchUsers()
  } catch (err) {
    error.value = err.message || 'Failed to delete benchmarks'
    console.error('Error deleting benchmarks:', err)
  } finally {
    loading.value = false
  }
}

async function confirmBanUser(user) {
  if (!confirm(`Are you sure you want to BAN user "${user.Username}"? They will not be able to sign in or upload anything.`)) {
    return
  }

  try {
    loading.value = true
    await api.admin.banUser(user.ID, true)
    alert(`User "${user.Username}" has been banned.`)
    await fetchUsers()
  } catch (err) {
    error.value = err.message || 'Failed to ban user'
    console.error('Error banning user:', err)
  } finally {
    loading.value = false
  }
}

async function confirmUnbanUser(user) {
  if (!confirm(`Are you sure you want to UNBAN user "${user.Username}"?`)) {
    return
  }

  try {
    loading.value = true
    await api.admin.banUser(user.ID, false)
    alert(`User "${user.Username}" has been unbanned.`)
    await fetchUsers()
  } catch (err) {
    error.value = err.message || 'Failed to unban user'
    console.error('Error unbanning user:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  initializeFromURL()
  fetchUsers()
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
  const newSearch = newQuery.search || ''
  
  let shouldFetch = false
  
  if (newPage !== currentPage.value) {
    currentPage.value = newPage
    shouldFetch = true
  }
  
  if (newSearch !== searchQuery.value) {
    searchQuery.value = newSearch
    currentPage.value = 1
    shouldFetch = true
  }
  
  if (shouldFetch) {
    fetchUsers()
  }
}, { deep: true })
</script>

<style scoped>
.table {
  margin-top: 1rem;
}

.btn-group-sm {
  white-space: nowrap;
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
