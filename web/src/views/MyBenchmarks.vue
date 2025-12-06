<template>
  <div>
    <h2>My Benchmarks</h2>
    
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
    <div v-else>
      <p>
        <small>Your benchmarks: {{ totalBenchmarks }}</small>
      </p>

      <div v-if="benchmarks.length > 0" class="list-group mt-1">
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
          <div class="d-flex w-100 justify-content-between">
            <h5 class="mb-1 text-truncate">{{ benchmark.Title }}</h5>
            <small class="text-nowrap">
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
              By <span v-if="benchmark.User">
                <b>{{ benchmark.User.Username }}<span v-if="benchmark.User.IsAdmin" class="admin-asterisk" title="Admin">*</span></b>
              </span>
              <b v-else>Unknown</b>
            </small>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else class="alert alert-info" role="alert">
        You haven't created any benchmarks yet. 
        <router-link to="/benchmarks/new" class="alert-link">Create your first benchmark!</router-link>
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
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import { formatRelativeDate } from '../utils/dateFormatter'

const router = useRouter()
const authStore = useAuthStore()

const benchmarks = ref([])
const loading = ref(false)
const error = ref(null)
const currentPage = ref(1)
const totalPages = ref(1)
const totalBenchmarks = ref(0)
const perPage = ref(10)
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

async function loadBenchmarks() {
  try {
    loading.value = true
    error.value = null

    // Get current user's benchmarks
    const response = await api.benchmarks.listByUser(
      authStore.user?.user_id,
      currentPage.value,
      perPage.value
    )

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

function navigateToBenchmark(id) {
  router.push(`/benchmarks/${id}`)
}

function handleResize() {
  windowWidth.value = window.innerWidth
}

onMounted(() => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }
  loadBenchmarks()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
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

.text-truncate {
  max-width: 100%;
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
