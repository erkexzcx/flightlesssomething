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
      <div v-if="totalPages > 1" class="d-flex justify-content-center mt-3">
        <ul class="pagination">
          <li class="page-item" :class="{ disabled: currentPage <= 1 }">
            <a class="page-link" href="#" @click.prevent="goToPage(currentPage - 1)">
              Previous
            </a>
          </li>
          <li class="page-item disabled">
            <a class="page-link" href="#">{{ currentPage }} / {{ totalPages }}</a>
          </li>
          <li class="page-item" :class="{ disabled: currentPage >= totalPages }">
            <a class="page-link" href="#" @click.prevent="goToPage(currentPage + 1)">
              Next
            </a>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
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
  }
}

function navigateToBenchmark(id) {
  router.push(`/benchmarks/${id}`)
}

onMounted(() => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }
  loadBenchmarks()
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
</style>
