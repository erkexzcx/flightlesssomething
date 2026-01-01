// API client for backend communication
const API_BASE = ''

class APIError extends Error {
  constructor(message, status) {
    super(message)
    this.status = status
    this.name = 'APIError'
  }
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(API_BASE + url, {
    ...options,
    credentials: 'include', // Include cookies for session management
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    throw new APIError(errorData.error || 'Request failed', response.status)
  }

  return response.json()
}

export const api = {
  // Health check
  async health() {
    const response = await fetch(API_BASE + '/health', { credentials: 'include' })
    return response.json()
  },

  // Auth endpoints
  auth: {
    async adminLogin(username, password) {
      return fetchJSON('/auth/admin/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
    },

    async logout() {
      return fetchJSON('/auth/logout', {
        method: 'POST',
      })
    },

    async getCurrentUser() {
      return fetchJSON('/api/auth/me')
    },

    // For Discord OAuth, redirect to /auth/login
    discordLogin() {
      window.location.href = '/auth/login'
    },
  },

  // Benchmark endpoints
  benchmarks: {
    async list(page = 1, perPage = 10, search = '', sortBy = '', sortOrder = '', searchFields = []) {
      const params = new URLSearchParams({
        page: page.toString(),
        per_page: perPage.toString(),
      })
      if (search) {
        params.append('search', search)
      }
      if (sortBy) {
        params.append('sort_by', sortBy)
      }
      if (sortOrder) {
        params.append('sort_order', sortOrder)
      }
      if (searchFields.length > 0) {
        params.append('search_fields', searchFields.join(','))
      }
      return fetchJSON(`/api/benchmarks?${params}`)
    },

    async listByUser(userId, page = 1, perPage = 10, sortBy = '', sortOrder = '') {
      const params = new URLSearchParams({
        page: page.toString(),
        per_page: perPage.toString(),
        user_id: userId.toString(),
      })
      if (sortBy) {
        params.append('sort_by', sortBy)
      }
      if (sortOrder) {
        params.append('sort_order', sortOrder)
      }
      return fetchJSON(`/api/benchmarks?${params}`)
    },

    async get(id) {
      return fetchJSON(`/api/benchmarks/${id}`)
    },

    async create(formData) {
      const response = await fetch(API_BASE + '/api/benchmarks', {
        method: 'POST',
        credentials: 'include',
        body: formData, // FormData for file uploads
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new APIError(errorData.error || 'Failed to create benchmark', response.status)
      }

      return response.json()
    },

    async update(id, data) {
      return fetchJSON(`/api/benchmarks/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      })
    },

    async delete(id) {
      return fetchJSON(`/api/benchmarks/${id}`, {
        method: 'DELETE',
      })
    },

    async getData(id, onProgress) {
      // Use custom fetch with progress tracking for large data downloads
      const response = await fetch(API_BASE + `/api/benchmarks/${id}/data`, {
        credentials: 'include',
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new APIError(errorData.error || 'Failed to fetch data', response.status)
      }

      // Get content length for progress calculation
      const contentLength = response.headers.get('content-length')
      const total = contentLength ? parseInt(contentLength, 10) : 0

      // If no progress callback or body not readable, use simple method
      if (!onProgress || !response.body) {
        return response.json()
      }

      // Read response with progress tracking
      const reader = response.body.getReader()
      const chunks = []
      let receivedLength = 0

      while (true) {
        const { done, value } = await reader.read()

        if (done) break

        chunks.push(value)
        receivedLength += value.length

        // Call progress callback
        onProgress({
          loaded: receivedLength,
          total: total, // May be 0 if no Content-Length
          percentage: total > 0 ? Math.round((receivedLength / total) * 100) : 0
        })
      }

      // Combine chunks and parse JSON
      const chunksAll = new Uint8Array(receivedLength)
      let position = 0
      for (const chunk of chunks) {
        chunksAll.set(chunk, position)
        position += chunk.length
      }

      const text = new TextDecoder('utf-8').decode(chunksAll)
      return JSON.parse(text)
    },

    getDataUrl(id) {
      return `${API_BASE}/api/benchmarks/${id}/data`
    },

    async deleteRun(id, runIndex) {
      return fetchJSON(`/api/benchmarks/${id}/runs/${runIndex}`, {
        method: 'DELETE',
      })
    },

    async addRuns(id, formData) {
      const response = await fetch(API_BASE + `/api/benchmarks/${id}/runs`, {
        method: 'POST',
        credentials: 'include',
        body: formData, // FormData for file uploads
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new APIError(errorData.error || 'Failed to add runs', response.status)
      }

      return response.json()
    },
  },

  // Admin endpoints
  admin: {
    async listUsers(page = 1, perPage = 10, search = '') {
      const params = new URLSearchParams({
        page: page.toString(),
        per_page: perPage.toString(),
      })
      if (search) {
        params.append('search', search)
      }
      return fetchJSON(`/api/admin/users?${params}`)
    },

    async deleteUser(id, deleteData = false) {
      const params = deleteData ? '?delete_data=true' : ''
      return fetchJSON(`/api/admin/users/${id}${params}`, {
        method: 'DELETE',
      })
    },

    async deleteUserBenchmarks(id) {
      return fetchJSON(`/api/admin/users/${id}/benchmarks`, {
        method: 'DELETE',
      })
    },

    async banUser(id, banned) {
      return fetchJSON(`/api/admin/users/${id}/ban`, {
        method: 'PUT',
        body: JSON.stringify({ banned }),
      })
    },

    async toggleUserAdmin(id, isAdmin) {
      return fetchJSON(`/api/admin/users/${id}/admin`, {
        method: 'PUT',
        body: JSON.stringify({ is_admin: isAdmin }),
      })
    },

    async listLogs(page = 1, perPage = 50, filters = {}) {
      const params = new URLSearchParams({
        page: page.toString(),
        per_page: perPage.toString(),
      })
      if (filters.action) {
        params.append('action', filters.action)
      }
      if (filters.targetType) {
        params.append('target_type', filters.targetType)
      }
      if (filters.userId) {
        params.append('user_id', filters.userId.toString())
      }
      return fetchJSON(`/api/admin/logs?${params}`)
    },
  },

  // API Token endpoints
  tokens: {
    async list() {
      return fetchJSON('/api/tokens')
    },

    async create(name) {
      return fetchJSON('/api/tokens', {
        method: 'POST',
        body: JSON.stringify({ name }),
      })
    },

    async delete(id) {
      return fetchJSON(`/api/tokens/${id}`, {
        method: 'DELETE',
      })
    },
  },
}

export { APIError }
