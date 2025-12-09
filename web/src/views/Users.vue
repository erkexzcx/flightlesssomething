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
      <nav v-if="totalPages > 1" aria-label="User pagination">
        <ul class="pagination justify-content-center">
          <li class="page-item" :class="{ disabled: currentPage === 1 }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage - 1)">Previous</a>
          </li>
          <li
            v-for="page in displayPages"
            :key="page"
            class="page-item"
            :class="{ active: page === currentPage }"
          >
            <a class="page-link" href="#" @click.prevent="changePage(page)">{{ page }}</a>
          </li>
          <li class="page-item" :class="{ disabled: currentPage === totalPages }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage + 1)">Next</a>
          </li>
        </ul>
      </nav>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
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

function changePage(page) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    updateURL()
    fetchUsers()
  }
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
</style>
