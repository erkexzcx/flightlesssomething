<template>
  <div>
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

    <!-- Benchmark details -->
    <div v-else-if="benchmark">
      <!-- Header with actions -->
      <div class="benchmark-header mb-3">
        <!-- Title and metadata -->
        <div class="benchmark-title-section">
          <h2>{{ benchmark.Title }}</h2>
          <p class="text-muted">
            By 
            <router-link 
              v-if="benchmark.User"
              :to="{ path: '/benchmarks', query: { user_id: benchmark.User.ID } }"
              class="username-link"
            >
              <strong>{{ benchmark.User.Username }}<span v-if="benchmark.User.IsAdmin" class="admin-asterisk" title="Admin">*</span></strong>
            </router-link>
            <strong v-else>Unknown</strong> •
            Created {{ formatRelativeDate(benchmark.CreatedAt) }}
            <span v-if="benchmark.UpdatedAt !== benchmark.CreatedAt">
              • Updated {{ formatRelativeDate(benchmark.UpdatedAt) }}
            </span>
          </p>
        </div>
        
        <!-- Action buttons -->
        <div class="btn-group benchmark-actions" role="group">
          <!-- Owner-only buttons -->
          <template v-if="isOwner">
            <button
              type="button"
              class="btn btn-outline-primary"
              @click="toggleEditMode"
              :disabled="deleting"
            >
              <i class="fa-solid fa-edit"></i> {{ editMode ? 'Cancel' : 'Edit' }}
            </button>
          </template>
          
          <!-- Download button - available for everyone -->
          <button
            type="button"
            class="btn btn-outline-success"
            @click="downloadBenchmark"
            :disabled="deleting"
          >
            <i class="fa-solid fa-download"></i> Download
          </button>
          
          <!-- Owner-only delete button -->
          <template v-if="isOwner">
            <button
              type="button"
              class="btn btn-outline-danger"
              @click="confirmDelete"
              :disabled="deleting"
            >
              <i class="fa-solid fa-trash"></i> {{ deleting ? 'Deleting...' : 'Delete' }}
            </button>
          </template>
        </div>
      </div>

      <!-- Edit form -->
      <div v-if="editMode" class="card mb-3">
        <div class="card-body">
          <h5 class="card-title">Edit Benchmark</h5>
          <form @submit.prevent="handleUpdate">
            <div class="mb-3">
              <label for="editTitle" class="form-label">Title</label>
              <input
                type="text"
                class="form-control"
                id="editTitle"
                v-model="editTitle"
                required
                maxlength="100"
              />
              <div class="form-text">{{ editTitle.length }}/100 characters</div>
            </div>

            <div class="mb-3">
              <label for="editDescription" class="form-label">Description</label>
              <textarea
                class="form-control font-monospace"
                id="editDescription"
                v-model="editDescription"
                rows="8"
                maxlength="5000"
                placeholder="Optional description of your benchmark setup... (supports Markdown)"
              ></textarea>
              <div class="form-text">{{ editDescription.length }}/5000 characters • Markdown supported</div>
            </div>

            <!-- Labels editing section -->
            <div v-if="editLabels.length > 0" class="mb-3">
              <label class="form-label">Run Labels</label>
              <p class="text-muted small">
                <i class="fa-solid fa-info-circle"></i> 
                Edit the labels for each benchmark run. These labels are used to identify datasets in charts.
              </p>
              <div v-for="(label, index) in editLabels" :key="index" class="mb-2">
                <div class="input-group input-group-sm">
                  <span class="input-group-text">Run {{ index + 1 }}:</span>
                  <input
                    type="text"
                    class="form-control"
                    v-model="editLabels[index]"
                    placeholder="Enter label"
                    maxlength="100"
                  />
                  <button
                    type="button"
                    class="btn btn-sm btn-danger"
                    @click="confirmDeleteRun(index)"
                    :disabled="editLabels.length === 1 || updating"
                    :title="editLabels.length === 1 ? 'Cannot delete the last run' : 'Delete this run'"
                  >
                    <i class="fa-solid fa-trash"></i>
                  </button>
                </div>
              </div>
            </div>

            <!-- Add new runs section -->
            <div class="mb-3">
              <label class="form-label">Add New Runs</label>
              <p class="text-muted small">
                <i class="fa-solid fa-info-circle"></i> 
                Upload additional benchmark files to add more runs to this benchmark.
                <br>
                <i class="fa-solid fa-info-circle"></i> <strong>Note:</strong> Combined benchmarks must not exceed 1 million data lines (excluding headers).
              </p>
              <input
                type="file"
                class="form-control"
                ref="fileInput"
                @change="handleFileSelect"
                accept=".csv,.hml"
                multiple
                :disabled="updating"
              />
              <div v-if="selectedFiles.length > 0" class="mt-2">
                <h6 class="small">Selected Files ({{ selectedFiles.length }})</h6>
                <ul class="list-group list-group-sm">
                  <li
                    v-for="(fileObj, index) in selectedFiles"
                    :key="index"
                    class="list-group-item py-2"
                  >
                    <div class="row align-items-center">
                      <div class="col-md-5">
                        <small class="text-muted">
                          <i class="fa-solid fa-file"></i> {{ fileObj.originalName }}
                        </small>
                      </div>
                      <div class="col-md-6">
                        <div class="input-group input-group-sm">
                          <span class="input-group-text">Label:</span>
                          <input
                            type="text"
                            class="form-control"
                            v-model="fileObj.label"
                            placeholder="Enter label for this file"
                            maxlength="100"
                            :disabled="updating"
                          />
                        </div>
                      </div>
                      <div class="col-md-1 text-end">
                        <button
                          type="button"
                          class="btn btn-sm btn-danger"
                          @click="removeFile(index)"
                          :disabled="updating"
                        >
                          <i class="fa-solid fa-trash"></i>
                        </button>
                      </div>
                    </div>
                  </li>
                </ul>
              </div>
            </div>

            <div class="d-flex gap-2">
              <button type="submit" class="btn btn-primary" :disabled="updating">
                <span v-if="updating">
                  <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
                  Saving...
                </span>
                <span v-else>
                  <i class="fa-solid fa-save"></i> Save Changes
                </span>
              </button>
              <button type="button" class="btn btn-secondary" @click="cancelEdit" :disabled="updating">
                Cancel
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Description -->
      <div v-if="!editMode && benchmark.Description" class="card mb-3">
        <div class="card-body">
          <div class="d-flex justify-content-between align-items-center mb-2">
            <h5 class="card-title mb-0">Description</h5>
            <button
              v-if="shouldShowCollapseButton"
              type="button"
              class="btn btn-sm btn-outline-secondary"
              @click="toggleDescriptionExpanded"
            >
              <i :class="descriptionExpanded ? 'fa-solid fa-chevron-up' : 'fa-solid fa-chevron-down'"></i>
              {{ descriptionExpanded ? 'Collapse' : 'Expand' }}
            </button>
          </div>
          <div 
            ref="descriptionContentRef"
            class="markdown-content"
            :class="{ 'collapsed': !descriptionExpanded && shouldShowCollapseButton }"
            v-html="renderedDescription"
          ></div>
        </div>
      </div>

      <!-- Benchmark data visualization -->
      <div class="card">
        <div class="card-body">
          <h5 class="card-title">Benchmark Data</h5>
          
          <!-- Loading state with progress -->
          <div v-if="loadingData" class="text-center my-4">
            <div class="spinner-border" role="status">
              <span class="visually-hidden">Loading data...</span>
            </div>
            <p class="text-muted mt-2">
              <span v-if="downloadProgress">
                Downloading benchmark data... {{ downloadProgress.percentage }}%
                <br>
                <small>({{ formatBytes(downloadProgress.loaded) }} / {{ formatBytes(downloadProgress.total) }})</small>
              </span>
              <span v-else>Loading benchmark data...</span>
            </p>
            <!-- Progress bar -->
            <div v-if="downloadProgress" class="progress mx-auto mt-3" style="max-width: 400px;">
              <div 
                class="progress-bar progress-bar-striped progress-bar-animated" 
                role="progressbar" 
                :style="{ width: downloadProgress.percentage + '%' }"
                :aria-valuenow="downloadProgress.percentage"
                aria-valuemin="0" 
                aria-valuemax="100"
              >
                {{ downloadProgress.percentage }}%
              </div>
            </div>
          </div>

          <!-- Error state -->
          <div v-else-if="dataError" class="alert alert-warning" role="alert">
            <i class="fa-solid fa-exclamation-triangle"></i> {{ dataError }}
          </div>

          <!-- Data visualization -->
          <div v-else-if="benchmarkData && benchmarkData.length > 0">
            <BenchmarkCharts :benchmarkData="benchmarkData" />
          </div>

          <!-- No data state -->
          <div v-else class="text-muted">
            <p>No benchmark data available.</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete confirmation modal -->
    <div v-if="showDeleteModal" class="modal fade show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.5);">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Confirm Delete</h5>
            <button type="button" class="btn-close" @click="showDeleteModal = false"></button>
          </div>
          <div class="modal-body">
            <p>Are you sure you want to delete this benchmark?</p>
            <p class="text-danger"><strong>This action cannot be undone.</strong></p>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showDeleteModal = false">Cancel</button>
            <button type="button" class="btn btn-danger" @click="handleDelete">Delete</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete run confirmation modal -->
    <div v-if="showDeleteRunModal" class="modal fade show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.5);">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Confirm Delete Run</h5>
            <button type="button" class="btn-close" @click="showDeleteRunModal = false"></button>
          </div>
          <div class="modal-body">
            <p>Are you sure you want to delete run "{{ runToDelete !== null ? editLabels[runToDelete] : '' }}"?</p>
            <p class="text-danger"><strong>This action cannot be undone.</strong></p>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="showDeleteRunModal = false">Cancel</button>
            <button type="button" class="btn btn-danger" @click="handleDeleteRun">Delete Run</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { api } from '../api/client'
import BenchmarkCharts from '../components/BenchmarkCharts.vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { formatRelativeDate } from '../utils/dateFormatter'

// Configure marked for security
marked.setOptions({
  breaks: true,
  gfm: true,
  // Disable HTML for security
  mangle: false,
  headerIds: false,
})

// Use a custom renderer extension to add security attributes to links
marked.use({
  renderer: {
    link({ href, title, tokens }) {
      // Parse the tokens to get the link text
      const text = this.parser.parseInline(tokens)
      
      // Build the link HTML with security attributes
      let html = `<a href="${href}"`
      if (title) {
        html += ` title="${title}"`
      }
      html += ' rel="noopener noreferrer nofollow" target="_blank"'
      html += `>${text}</a>`
      
      return html
    }
  }
})

// Constants
const FILE_EXTENSIONS = /\.(csv|hml)$/i
const COLLAPSE_HEIGHT_THRESHOLD = 150 // px - should match .markdown-content.collapsed max-height in CSS

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const benchmark = ref(null)
const loading = ref(false)
const error = ref(null)
const editMode = ref(false)
const editTitle = ref('')
const editDescription = ref('')
const editLabels = ref([])
const updating = ref(false)
const deleting = ref(false)
const showDeleteModal = ref(false)
const showDeleteRunModal = ref(false)
const runToDelete = ref(null)
const benchmarkData = ref(null)
const loadingData = ref(false)
const dataError = ref(null)
const downloadProgress = ref(null) // { loaded, total, percentage }
const descriptionExpanded = ref(false)
const fileInput = ref(null)
const selectedFiles = ref([])
const descriptionContentRef = ref(null)
const shouldShowCollapseButton = ref(false)

const isOwner = computed(() => {
  if (!authStore.isAuthenticated || !benchmark.value) return false
  return benchmark.value.UserID === authStore.user?.user_id || authStore.isAdmin
})

const renderedDescription = computed(() => {
  if (!benchmark.value?.Description) return ''
  // Parse markdown and sanitize HTML
  const rawHtml = marked.parse(benchmark.value.Description)
  return DOMPurify.sanitize(rawHtml, {
    ALLOWED_TAGS: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'p', 'br', 'strong', 'em', 'code', 'pre', 'ul', 'ol', 'li', 'blockquote', 'a', 'hr'],
    ALLOWED_ATTR: ['href', 'rel', 'target'],
  })
})

// Smart collapsible check based on multiple criteria
function checkIfDescriptionShouldCollapse() {
  if (!benchmark.value?.Description) {
    shouldShowCollapseButton.value = false
    return
  }
  
  const description = benchmark.value.Description
  
  // Criterion 1: Character count > 300 (existing behavior)
  if (description.length > 300) {
    shouldShowCollapseButton.value = true
    return
  }
  
  // Criterion 2: Line count > 10 (new behavior for many short lines)
  const lineCount = description.split('\n').length
  if (lineCount > 10) {
    shouldShowCollapseButton.value = true
    return
  }
  
  // Criterion 3: Rendered height exceeds threshold (smart approach)
  // Only check this if criteria 1 and 2 are not met
  // This will be checked after the DOM is rendered
  nextTick(() => {
    if (descriptionContentRef.value) {
      const height = descriptionContentRef.value.scrollHeight
      shouldShowCollapseButton.value = height > COLLAPSE_HEIGHT_THRESHOLD
    }
  })
}

// Watch for changes to benchmark description
watch(() => benchmark.value?.Description, () => {
  checkIfDescriptionShouldCollapse()
}, { immediate: true })

function toggleDescriptionExpanded() {
  descriptionExpanded.value = !descriptionExpanded.value
}

async function loadBenchmark() {
  try {
    loading.value = true
    error.value = null
    
    const id = route.params.id
    benchmark.value = await api.benchmarks.get(id)
    
    // Initialize edit form values
    editTitle.value = benchmark.value.Title
    editDescription.value = benchmark.value.Description || ''
    
    // Load benchmark data
    await loadBenchmarkData(id)
  } catch (err) {
    error.value = err.message || 'Failed to load benchmark'
  } finally {
    loading.value = false
  }
}

async function loadBenchmarkData(id) {
  try {
    loadingData.value = true
    dataError.value = null
    downloadProgress.value = null
    
    // Load full data for accurate statistics (averages, percentiles, percentages)
    // The frontend chart component will downsample line charts as needed
    benchmarkData.value = await api.benchmarks.getData(id, (progress) => {
      downloadProgress.value = progress
    })
    
    // Initialize edit labels from loaded data
    editLabels.value = benchmarkData.value.map(d => d.Label || '')
  } catch (err) {
    dataError.value = err.message || 'Failed to load benchmark data'
  } finally {
    loadingData.value = false
    downloadProgress.value = null
  }
}

function toggleEditMode() {
  if (editMode.value) {
    cancelEdit()
  } else {
    editMode.value = true
    editTitle.value = benchmark.value.Title
    editDescription.value = benchmark.value.Description || ''
    editLabels.value = benchmarkData.value ? benchmarkData.value.map(d => d.Label || '') : []
  }
}

function cancelEdit() {
  editMode.value = false
  editTitle.value = benchmark.value.Title
  editDescription.value = benchmark.value.Description || ''
  editLabels.value = benchmarkData.value ? benchmarkData.value.map(d => d.Label || '') : []
  selectedFiles.value = []
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function handleFileSelect(event) {
  const files = Array.from(event.target.files)
  // Create file objects with editable labels
  const newFiles = files.map(file => ({
    file: file,
    label: getDefaultLabel(file.name),
    originalName: file.name
  }))
  selectedFiles.value = [...selectedFiles.value, ...newFiles]
}

function getDefaultLabel(filename) {
  // Remove extension (.csv or .hml)
  return filename.replace(FILE_EXTENSIONS, '')
}

function removeFile(index) {
  selectedFiles.value.splice(index, 1)
  // Reset file input only if all files are removed
  if (selectedFiles.value.length === 0 && fileInput.value) {
    fileInput.value.value = ''
  }
}

function confirmDeleteRun(index) {
  runToDelete.value = index
  showDeleteRunModal.value = true
}

async function handleDeleteRun() {
  if (runToDelete.value === null) return
  
  try {
    updating.value = true
    showDeleteRunModal.value = false
    
    await api.benchmarks.deleteRun(benchmark.value.ID, runToDelete.value)
    
    // Reload the benchmark data to reflect the deletion
    await loadBenchmarkData(benchmark.value.ID)
    
    // Reset state
    runToDelete.value = null
  } catch (err) {
    error.value = err.message || 'Failed to delete run'
    showDeleteRunModal.value = false
    runToDelete.value = null
  } finally {
    updating.value = false
  }
}

async function handleUpdate() {
  try {
    updating.value = true
    
    const data = {
      title: editTitle.value.trim(),
      description: editDescription.value.trim(),
    }
    
    // Add labels if they've been edited
    if (editLabels.value.length > 0) {
      const labels = {}
      editLabels.value.forEach((label, index) => {
        const trimmedLabel = label.trim()
        // Only include if it changed from original
        if (benchmarkData.value[index] && trimmedLabel !== benchmarkData.value[index].Label) {
          labels[index] = trimmedLabel
        }
      })
      if (Object.keys(labels).length > 0) {
        data.labels = labels
      }
    }
    
    const updated = await api.benchmarks.update(benchmark.value.ID, data)
    
    // Update local benchmark data
    benchmark.value.Title = updated.Title
    benchmark.value.Description = updated.Description
    
    // If labels were updated, reload the benchmark data
    if (data.labels) {
      await loadBenchmarkData(benchmark.value.ID)
    }
    
    // If there are new files to add, upload them
    if (selectedFiles.value.length > 0) {
      const formData = new FormData()
      
      // Add all files with their custom labels
      selectedFiles.value.forEach(fileObj => {
        // Get the original extension
        const ext = fileObj.originalName.match(FILE_EXTENSIONS)?.[0] || '.csv'
        // Create a new File with the custom label as name
        const renamedFile = new File([fileObj.file], fileObj.label + ext, { type: fileObj.file.type })
        formData.append('files', renamedFile)
      })
      
      await api.benchmarks.addRuns(benchmark.value.ID, formData)
      
      // Reload benchmark data to show new runs
      await loadBenchmarkData(benchmark.value.ID)
      
      // Clear selected files
      selectedFiles.value = []
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
    
    editMode.value = false
  } catch (err) {
    error.value = err.message || 'Failed to update benchmark'
  } finally {
    updating.value = false
  }
}

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

function confirmDelete() {
  showDeleteModal.value = true
}

async function handleDelete() {
  try {
    deleting.value = true
    showDeleteModal.value = false
    
    await api.benchmarks.delete(benchmark.value.ID)
    
    // Navigate back to benchmarks list
    router.push('/benchmarks')
  } catch (err) {
    error.value = err.message || 'Failed to delete benchmark'
    deleting.value = false
  }
}

function downloadBenchmark() {
  const url = `/api/benchmarks/${benchmark.value.ID}/download`
  window.open(url, '_blank')
}

onMounted(() => {
  loadBenchmark()
})
</script>

<style scoped>
/* Benchmark header layout */
.benchmark-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  flex-wrap: nowrap;
}

.benchmark-title-section {
  flex: 1;
  min-width: 0; /* Allow text to truncate if needed */
}

.benchmark-actions {
  flex-shrink: 0; /* Prevent buttons from shrinking */
  white-space: nowrap; /* Keep buttons on one line */
}

/* Mobile responsive: stack buttons above title */
@media (max-width: 768px) {
  .benchmark-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .benchmark-title-section {
    order: 2; /* Move title below buttons on mobile */
  }
  
  .benchmark-actions {
    order: 1; /* Move buttons above title on mobile */
    margin-bottom: 1rem;
    width: 100%;
  }
  
  .benchmark-actions .btn {
    flex: 1; /* Make buttons equal width on mobile */
  }
}

.card {
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.modal.show {
  display: block;
}

.markdown-content {
  line-height: 1.6;
  overflow-wrap: break-word;
  word-wrap: break-word;
}

.markdown-content.collapsed {
  max-height: 150px;
  overflow: hidden;
  position: relative;
}

.markdown-content.collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: linear-gradient(to bottom, transparent, var(--bs-body-bg));
  pointer-events: none;
}

/* Markdown styling */
.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4),
.markdown-content :deep(h5),
.markdown-content :deep(h6) {
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  font-weight: 600;
}

.markdown-content :deep(h1) { font-size: 2rem; }
.markdown-content :deep(h2) { font-size: 1.5rem; }
.markdown-content :deep(h3) { font-size: 1.25rem; }
.markdown-content :deep(h4) { font-size: 1.1rem; }
.markdown-content :deep(h5) { font-size: 1rem; }
.markdown-content :deep(h6) { font-size: 0.9rem; }

.markdown-content :deep(p) {
  margin-bottom: 1rem;
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  margin-bottom: 1rem;
  padding-left: 2rem;
}

.markdown-content :deep(li) {
  margin-bottom: 0.25rem;
}

.markdown-content :deep(code) {
  background-color: rgba(0, 0, 0, 0.3);
  padding: 0.2rem 0.4rem;
  border-radius: 3px;
  font-family: monospace;
  font-size: 0.9em;
}

.markdown-content :deep(pre) {
  background-color: rgba(0, 0, 0, 0.3);
  padding: 1rem;
  border-radius: 5px;
  overflow-x: auto;
  margin-bottom: 1rem;
}

.markdown-content :deep(pre code) {
  background-color: transparent;
  padding: 0;
}

.markdown-content :deep(blockquote) {
  border-left: 4px solid rgba(255, 255, 255, 0.2);
  padding-left: 1rem;
  margin-left: 0;
  margin-bottom: 1rem;
  color: rgba(255, 255, 255, 0.7);
}

.markdown-content :deep(a) {
  color: var(--bs-link-color);
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
}

.markdown-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin-bottom: 1rem;
}

.markdown-content :deep(table th),
.markdown-content :deep(table td) {
  border: 1px solid rgba(255, 255, 255, 0.2);
  padding: 0.5rem;
}

.markdown-content :deep(table th) {
  background-color: rgba(0, 0, 0, 0.2);
  font-weight: 600;
}

.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid rgba(255, 255, 255, 0.2);
  margin: 1.5rem 0;
}

.font-monospace {
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}

.admin-asterisk {
  color: var(--bs-warning);
  font-weight: bold;
  cursor: help;
  margin-left: 2px;
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

.username-link :deep(strong) {
  font-weight: 600;
}
</style>
