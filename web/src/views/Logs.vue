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
      <nav v-if="totalPages > 1" aria-label="Logs pagination">
        <ul class="pagination justify-content-center">
          <li class="page-item" :class="{ disabled: currentPage === 1 }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage - 1)">
              Previous
            </a>
          </li>
          <li
            v-for="page in visiblePages"
            :key="page"
            class="page-item"
            :class="{ active: page === currentPage }"
          >
            <a class="page-link" href="#" @click.prevent="changePage(page)">
              {{ page }}
            </a>
          </li>
          <li class="page-item" :class="{ disabled: currentPage === totalPages }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage + 1)">
              Next
            </a>
          </li>
        </ul>
      </nav>
    </div>
  </div>
</template>

<script>
import { api } from '../api/client'

export default {
  name: 'Logs',
  data() {
    return {
      logs: [],
      loading: true,
      error: null,
      currentPage: 1,
      perPage: 50,
      totalLogs: 0,
      totalPages: 0,
      filterAction: '',
      filterTargetType: '',
    }
  },
  computed: {
    visiblePages() {
      const pages = []
      const maxVisible = 5
      let start = Math.max(1, this.currentPage - Math.floor(maxVisible / 2))
      let end = Math.min(this.totalPages, start + maxVisible - 1)

      if (end - start < maxVisible - 1) {
        start = Math.max(1, end - maxVisible + 1)
      }

      for (let i = start; i <= end; i++) {
        pages.push(i)
      }
      return pages
    }
  },
  mounted() {
    this.fetchLogs()
  },
  methods: {
    async fetchLogs() {
      this.loading = true
      this.error = null

      try {
        const filters = {}
        if (this.filterAction) {
          filters.action = this.filterAction
        }
        if (this.filterTargetType) {
          filters.targetType = this.filterTargetType
        }

        const response = await api.admin.listLogs(this.currentPage, this.perPage, filters)

        this.logs = response.logs || []
        this.totalLogs = response.total
        this.totalPages = response.total_pages
        this.currentPage = response.page
      } catch (err) {
        if (err.status === 403) {
          this.error = 'Admin privileges required'
        } else {
          this.error = err.message || 'Failed to load audit logs'
        }
      } finally {
        this.loading = false
      }
    },
    changePage(page) {
      if (page < 1 || page > this.totalPages) return
      this.currentPage = page
      this.fetchLogs()
    },
    handleFilter() {
      this.currentPage = 1
      this.fetchLogs()
    },
    resetFilters() {
      this.filterAction = ''
      this.filterTargetType = ''
      this.currentPage = 1
      this.fetchLogs()
    },
    formatDate(dateString) {
      const date = new Date(dateString)
      return date.toLocaleDateString() + ' ' + date.toLocaleTimeString()
    },
    formatFullDate(dateString) {
      const date = new Date(dateString)
      return date.toLocaleString()
    },
    getActionBadgeClass(action) {
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
    },
    renderDescription(log) {
      let description = this.escapeHtml(log.Description)
      
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
    },
    escapeHtml(text) {
      const div = document.createElement('div')
      div.textContent = text
      return div.innerHTML
    }
  }
}
</script>

<style scoped>
.table td {
  vertical-align: middle;
}
</style>
