<template>
  <div class="container">
    <div class="d-flex justify-content-between align-items-center mb-4">
      <h2>API Tokens</h2>
      <button class="btn btn-primary" @click="showCreateModal = true" :disabled="tokens.length >= 10">
        <i class="fa-solid fa-plus"></i> Create Token
      </button>
    </div>

    <div v-if="tokens.length >= 10" class="alert alert-warning" role="alert">
      <i class="fa-solid fa-exclamation-triangle"></i> You have reached the maximum number of API tokens (10).
    </div>

    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <div v-else-if="error" class="alert alert-danger" role="alert">
      {{ error }}
    </div>

    <div v-else-if="tokens.length === 0" class="text-center py-5 text-muted">
      <i class="fa-solid fa-key fa-3x mb-3"></i>
      <p>No API tokens yet. Create one to get started with automated benchmark uploads.</p>
      <p>
        <a href="https://github.com/erkexzcx/flightlesssomething/blob/main/docs/api.md#api-token-management" target="_blank" rel="noopener noreferrer">
          <i class="fa-solid fa-book"></i> View API Documentation
        </a>
      </p>
    </div>

    <div v-else class="row g-3">
      <div v-for="token in tokens" :key="token.ID" class="col-12">
        <div class="card">
          <div class="card-body">
            <div class="d-flex justify-content-between align-items-start">
              <div class="flex-grow-1">
                <h5 class="card-title mb-2">{{ token.Name }}</h5>
                
                <div class="input-group mb-2">
                  <input 
                    :type="visibleTokens[token.ID] ? 'text' : 'password'" 
                    class="form-control font-monospace" 
                    :value="token.Token" 
                    readonly
                  >
                  <button 
                    class="btn btn-outline-secondary" 
                    type="button" 
                    @click="toggleTokenVisibility(token.ID)"
                  >
                    <i class="fa-solid" :class="visibleTokens[token.ID] ? 'fa-eye-slash' : 'fa-eye'"></i>
                  </button>
                  <button 
                    class="btn" 
                    :class="copiedTokens[token.ID] ? 'btn-success' : 'btn-outline-secondary'"
                    type="button" 
                    @click="copyToken(token.Token, token.ID)"
                  >
                    <i class="fa-solid" :class="copiedTokens[token.ID] ? 'fa-check' : 'fa-copy'"></i>
                  </button>
                </div>

                <div class="text-muted small">
                  <div><strong>Created:</strong> {{ formatDate(token.CreatedAt) }}</div>
                  <div>
                    <strong>Last used:</strong> 
                    {{ token.LastUsedAt ? formatDate(token.LastUsedAt) : 'Never' }}
                  </div>
                </div>
              </div>
              
              <button 
                class="btn btn-outline-danger btn-sm ms-3" 
                @click="confirmDelete(token)"
              >
                <i class="fa-solid fa-trash"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Token Modal -->
    <div class="modal fade" :class="{ show: showCreateModal, 'd-block': showCreateModal }" tabindex="-1" style="background-color: rgba(0,0,0,0.5);" v-if="showCreateModal">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Create API Token</h5>
            <button type="button" class="btn-close" @click="closeCreateModal"></button>
          </div>
          <div class="modal-body">
            <form @submit.prevent="createToken">
              <div class="mb-3">
                <label for="tokenName" class="form-label">Token Name</label>
                <input 
                  type="text" 
                  class="form-control" 
                  id="tokenName" 
                  v-model="newTokenName"
                  placeholder="e.g., My Automation Script"
                  required
                  maxlength="100"
                >
                <div class="form-text">Give your token a descriptive name to remember what it's for.</div>
              </div>
              
              <div v-if="createError" class="alert alert-danger" role="alert">
                {{ createError }}
              </div>
            </form>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="closeCreateModal">Cancel</button>
            <button type="button" class="btn btn-primary" @click="createToken" :disabled="!newTokenName || creating">
              <span v-if="creating">
                <span class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
                Creating...
              </span>
              <span v-else>Create Token</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div class="modal fade" :class="{ show: showDeleteModal, 'd-block': showDeleteModal }" tabindex="-1" style="background-color: rgba(0,0,0,0.5);" v-if="showDeleteModal">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Delete API Token</h5>
            <button type="button" class="btn-close" @click="showDeleteModal = false"></button>
          </div>
          <div class="modal-body">
            <p>Are you sure you want to delete the token <strong>{{ tokenToDelete?.Name }}</strong>?</p>
            <p class="text-danger mb-0">This action cannot be undone. Any scripts using this token will stop working.</p>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showDeleteModal = false">Cancel</button>
            <button type="button" class="btn btn-danger" @click="deleteToken" :disabled="deleting">
              <span v-if="deleting">
                <span class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
                Deleting...
              </span>
              <span v-else>Delete</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { api } from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

const tokens = ref([])
const loading = ref(false)
const error = ref(null)
const visibleTokens = reactive({})
const copiedTokens = reactive({})

const showCreateModal = ref(false)
const newTokenName = ref('')
const creating = ref(false)
const createError = ref(null)

const showDeleteModal = ref(false)
const tokenToDelete = ref(null)
const deleting = ref(false)

onMounted(async () => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }
  await loadTokens()
})

async function loadTokens() {
  loading.value = true
  error.value = null
  
  try {
    tokens.value = await api.tokens.list()
  } catch (err) {
    error.value = err.message || 'Failed to load tokens'
  } finally {
    loading.value = false
  }
}

function toggleTokenVisibility(tokenId) {
  visibleTokens[tokenId] = !visibleTokens[tokenId]
}

async function copyToken(token, tokenId) {
  try {
    await navigator.clipboard.writeText(token)
    copiedTokens[tokenId] = true
    setTimeout(() => {
      copiedTokens[tokenId] = false
    }, 1000)
  } catch (err) {
    console.error('Failed to copy token:', err)
  }
}

function closeCreateModal() {
  showCreateModal.value = false
  newTokenName.value = ''
  createError.value = null
}

async function createToken() {
  if (!newTokenName.value) return
  
  creating.value = true
  createError.value = null
  
  try {
    await api.tokens.create(newTokenName.value)
    await loadTokens()
    closeCreateModal()
  } catch (err) {
    createError.value = err.message || 'Failed to create token'
  } finally {
    creating.value = false
  }
}

function confirmDelete(token) {
  tokenToDelete.value = token
  showDeleteModal.value = true
}

async function deleteToken() {
  if (!tokenToDelete.value) return
  
  deleting.value = true
  
  try {
    await api.tokens.delete(tokenToDelete.value.ID)
    await loadTokens()
    showDeleteModal.value = false
    tokenToDelete.value = null
  } catch (err) {
    error.value = err.message || 'Failed to delete token'
  } finally {
    deleting.value = false
  }
}

function formatDate(dateString) {
  const date = new Date(dateString)
  return date.toLocaleString()
}
</script>

<style scoped>
.modal.show {
  display: block;
}

.font-monospace {
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.9em;
}
</style>
