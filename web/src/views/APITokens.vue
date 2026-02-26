<template>
  <div class="container">
    <div class="d-flex justify-content-between align-items-center mb-4">
      <h2>API Tokens & MCP</h2>
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
        <a href="https://github.com/erkexzcx/flightlesssomething/blob/main/docs/api.md" target="_blank" rel="noopener noreferrer">
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

    <!-- MCP Server Configuration -->
    <div class="mt-5">
      <h3 class="mb-3"><i class="fa-solid fa-plug"></i> MCP Server</h3>
      <p class="text-muted">
        This server exposes an <a href="https://github.com/erkexzcx/flightlesssomething/blob/main/docs/mcp.md" target="_blank" rel="noopener noreferrer">MCP (Model Context Protocol)</a> endpoint that AI assistants can use to interact with your benchmarks. 
        Connect without authentication for read-only access, or enable authentication below for read-write access.
      </p>

      <div class="form-check mb-3">
        <input class="form-check-input" type="checkbox" id="mcpAuthEnabled" v-model="mcpAuthEnabled" :disabled="tokens.length === 0">
        <label class="form-check-label" for="mcpAuthEnabled">
          Enable authentication (read-write access)
        </label>
        <div v-if="tokens.length === 0" class="form-text text-muted">Create an API token above to enable authenticated access.</div>
      </div>

      <div v-if="mcpAuthEnabled && tokens.length > 0" class="mb-3">
        <label for="mcpTokenSelect" class="form-label fw-bold">Select token for MCP configuration:</label>
        <select id="mcpTokenSelect" class="form-select" v-model="selectedMCPTokenId">
          <option v-for="token in tokens" :key="token.ID" :value="token.ID">{{ token.Name }}</option>
        </select>
      </div>

      <!-- VS Code -->
      <div class="card mb-3">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span><i class="fa-solid fa-code"></i> Visual Studio Code</span>
          <div>
            <button 
              class="btn btn-sm"
              :class="copiedMCP.vscode ? 'btn-success' : 'btn-outline-secondary'"
              @click="copyMCPConfig('vscode')"
            >
              <i class="fa-solid" :class="copiedMCP.vscode ? 'fa-check' : 'fa-copy'"></i> Copy
            </button>
            <button 
              class="btn btn-sm btn-outline-primary ms-1"
              @click="openInVSCode"
              title="Open in VS Code (requires VS Code with MCP support)"
            >
              <i class="fa-solid fa-up-right-from-square"></i> Install in VS Code
            </button>
          </div>
        </div>
        <div class="card-body">
          <p class="small text-muted mb-2">Add this to your <code>.vscode/mcp.json</code> file in your workspace, or to your VS Code user settings under <code>"mcp"</code>:</p>
          <pre class="mb-0 p-3 rounded" style="background-color: var(--bs-tertiary-bg); overflow-x: auto;"><code>{{ vscodeConfig }}</code></pre>
        </div>
      </div>

      <!-- Claude Desktop -->
      <div class="card mb-3">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span><i class="fa-solid fa-robot"></i> Claude Desktop</span>
          <button 
            class="btn btn-sm"
            :class="copiedMCP.claude ? 'btn-success' : 'btn-outline-secondary'"
            @click="copyMCPConfig('claude')"
          >
            <i class="fa-solid" :class="copiedMCP.claude ? 'fa-check' : 'fa-copy'"></i> Copy
          </button>
        </div>
        <div class="card-body">
          <p class="small text-muted mb-2">Add this to your Claude Desktop config file (<code>claude_desktop_config.json</code>), inside the <code>"mcpServers"</code> object:</p>
          <pre class="mb-0 p-3 rounded" style="background-color: var(--bs-tertiary-bg); overflow-x: auto;"><code>{{ claudeConfig }}</code></pre>
        </div>
      </div>

      <p class="small text-muted">
        <i class="fa-solid fa-circle-info"></i>
        Tool call approval is controlled by your MCP client (e.g. VS Code, Claude Desktop) and applies to all tools 
        from this server — there is no per-tool granularity. This is a client-side setting, not an MCP server feature.
      </p>
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
import { ref, computed, onMounted, reactive, watch } from 'vue'
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

// MCP configuration
const selectedMCPTokenId = ref(null)
const mcpAuthEnabled = ref(false)
const copiedMCP = reactive({ vscode: false, claude: false })

const selectedMCPToken = computed(() => {
  return tokens.value.find(t => t.ID === selectedMCPTokenId.value)
})

const mcpServerUrl = computed(() => {
  return `${window.location.origin}/mcp`
})

const vscodeConfig = computed(() => {
  const server = {
    type: "http",
    url: mcpServerUrl.value,
  }
  if (mcpAuthEnabled.value) {
    server.headers = { "Authorization": "Bearer ${input:flightlesssomething_api_token}" }
    const config = {
      inputs: [
        {
          type: "promptString",
          id: "flightlesssomething_api_token",
          description: "FlightlessSomething API Token",
          password: true
        }
      ],
      servers: { "flightlesssomething": server }
    }
    return JSON.stringify(config, null, 2)
  }
  return JSON.stringify({ servers: { "flightlesssomething": server } }, null, 2)
})

const claudeConfig = computed(() => {
  const server = {
    type: "http",
    url: mcpServerUrl.value,
  }
  if (mcpAuthEnabled.value) {
    const token = selectedMCPToken.value?.Token || 'YOUR_API_TOKEN'
    server.headers = { "Authorization": `Bearer ${token}` }
  }
  return JSON.stringify({ "flightlesssomething": server }, null, 2)
})

watch(tokens, (newTokens) => {
  if (newTokens.length > 0 && !selectedMCPTokenId.value) {
    selectedMCPTokenId.value = newTokens[0].ID
  }
})

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

async function copyMCPConfig(type) {
  try {
    const text = type === 'vscode' ? vscodeConfig.value : claudeConfig.value
    await navigator.clipboard.writeText(text)
    copiedMCP[type] = true
    setTimeout(() => {
      copiedMCP[type] = false
    }, 1000)
  } catch (err) {
    console.error('Failed to copy MCP config:', err)
  }
}

function openInVSCode() {
  const config = {
    name: "flightlesssomething",
    type: "http",
    url: mcpServerUrl.value,
  }
  if (mcpAuthEnabled.value) {
    config.headers = { "Authorization": "Bearer ${input:flightlesssomething_api_token}" }
    config.inputs = [
      {
        type: "promptString",
        id: "flightlesssomething_api_token",
        description: "FlightlessSomething API Token",
        password: true
      }
    ]
  }
  const uri = `vscode:mcp/install?${encodeURIComponent(JSON.stringify(config))}`
  window.open(uri, '_blank')
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
