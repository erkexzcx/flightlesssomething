<template>
  <div>
    <h2><i class="fa-solid fa-plus"></i> Create New Benchmark</h2>
    
    <!-- Error Alert -->
    <div v-if="error" class="alert alert-danger alert-dismissible fade show" role="alert">
      {{ error }}
      <button type="button" class="btn-close" @click="error = null"></button>
    </div>

    <!-- Upload Form -->
    <div class="card mt-3">
      <div class="card-body">
        <h5 class="card-title">1. Upload Benchmark Files</h5>
        <p class="text-muted">
          Upload MangoHud CSV or Afterburner HML files. Multiple files will be combined.
          <br>
          <i class="fa-solid fa-circle-info"></i> Need help capturing benchmarks? 
          <a href="https://github.com/erkexzcx/flightlesssomething/blob/main/docs/benchmarks.md" target="_blank" rel="noopener noreferrer">
            Read the benchmark guide <i class="fa-solid fa-external-link-alt fa-xs"></i>
          </a>
          <br>
          <i class="fa-solid fa-info-circle"></i> <strong>Note:</strong> Combined benchmarks must not exceed 1 million data lines (excluding headers).
        </p>
        
        <div class="mb-3">
          <input
            type="file"
            class="form-control"
            ref="fileInput"
            @change="handleFileSelect"
            accept=".csv,.hml"
            multiple
          />
        </div>

        <!-- File List -->
        <div v-if="selectedFiles.length > 0" class="mt-3">
          <h6>Selected Files ({{ selectedFiles.length }})</h6>
          <p class="text-muted small">
            <i class="fa-solid fa-info-circle"></i> 
            You can edit the label for each file below. The label will be used to identify this dataset in charts.
          </p>
          <ul class="list-group">
            <li
              v-for="(fileObj, index) in selectedFiles"
              :key="index"
              class="list-group-item"
            >
              <div class="row align-items-center">
                <div class="col-md-5">
                  <small class="text-muted">
                    <i class="fa-solid fa-file"></i> {{ fileObj.originalName }}
                    <br>
                    <span class="badge bg-secondary">{{ formatFileSize(fileObj.file.size) }}</span>
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
                    />
                  </div>
                </div>
                <div class="col-md-1 text-end">
                  <button
                    type="button"
                    class="btn btn-sm btn-danger"
                    @click="removeFile(index)"
                  >
                    <i class="fa-solid fa-trash"></i>
                  </button>
                </div>
              </div>
            </li>
          </ul>

          <!-- Date/Time Warning -->
          <div v-if="hasDateTimeWarning" class="alert alert-warning mt-3" role="alert">
            <h6 class="alert-heading">
              <i class="fa-solid fa-exclamation-triangle"></i> 
              <strong>Warning - Default Filenames Detected</strong>
            </h6>
            <p class="mb-0">
              {{ dateTimeWarningMessage }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Metadata Form -->
    <div v-if="selectedFiles.length > 0" class="card mt-3">
      <div class="card-body">
        <h5 class="card-title">2. Add Benchmark Details</h5>
        
        <form @submit.prevent="handleSubmit">
          <div class="mb-3">
            <label for="title" class="form-label">Title <span class="text-danger">*</span></label>
            <input
              type="text"
              class="form-control"
              id="title"
              v-model="title"
              required
              maxlength="100"
              placeholder="e.g., Cyberpunk 2077 - High Settings"
            />
            <div class="form-text">{{ title.length }}/100 characters</div>
          </div>

          <div class="mb-3">
            <label for="description" class="form-label">Description</label>
            <textarea
              class="form-control font-monospace"
              id="description"
              v-model="description"
              rows="8"
              maxlength="5000"
              placeholder="Optional description of your benchmark setup... (supports Markdown)"
            ></textarea>
            <div class="form-text">{{ description.length }}/5000 characters • Markdown supported</div>
          </div>

          <div class="d-flex gap-2">
            <button type="submit" class="btn btn-primary" :disabled="uploading">
              <span v-if="uploading">
                <span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
                Uploading...
              </span>
              <span v-else>
                <i class="fa-solid fa-upload"></i> Upload Benchmark
              </span>
            </button>
            <button type="button" class="btn btn-secondary" @click="resetForm" :disabled="uploading">
              <i class="fa-solid fa-rotate-left"></i> Reset
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/client'
import { hasAnyDateTimePattern, getDateTimeWarningMessage } from '../utils/filenameValidator'

const router = useRouter()

const fileInput = ref(null)
const selectedFiles = ref([])
const title = ref('')
const description = ref('')
const uploading = ref(false)
const error = ref(null)

// Computed property to check if any filenames have date/time patterns
const hasDateTimeWarning = computed(() => {
  const labels = selectedFiles.value.map(fileObj => fileObj.label)
  return hasAnyDateTimePattern(labels)
})

const dateTimeWarningMessage = getDateTimeWarningMessage()

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
  const FILE_EXTENSIONS = /\.(csv|hml)$/i
  return filename.replace(FILE_EXTENSIONS, '')
}

function removeFile(index) {
  selectedFiles.value.splice(index, 1)
  // Reset file input
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

function formatFileSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function resetForm() {
  selectedFiles.value = []
  title.value = ''
  description.value = ''
  error.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

async function handleSubmit() {
  if (selectedFiles.value.length === 0) {
    error.value = 'Please select at least one file'
    return
  }

  if (!title.value.trim()) {
    error.value = 'Please enter a title'
    return
  }

  try {
    uploading.value = true
    error.value = null

    // Create FormData
    const formData = new FormData()
    formData.append('title', title.value.trim())
    if (description.value.trim()) {
      formData.append('description', description.value.trim())
    }
    
    // Add all files with their custom labels
    // We need to rename files to use the custom labels
    selectedFiles.value.forEach(fileObj => {
      // Get the original extension
      const FILE_EXTENSIONS = /\.(csv|hml)$/i
      const ext = fileObj.originalName.match(FILE_EXTENSIONS)?.[0] || '.csv'
      // Create a new File with the custom label as name
      const renamedFile = new File([fileObj.file], fileObj.label + ext, { type: fileObj.file.type })
      formData.append('files', renamedFile)
    })

    // Upload
    const result = await api.benchmarks.create(formData)
    
    // Navigate to the created benchmark
    router.push(`/benchmarks/${result.ID}`)
  } catch (err) {
    error.value = err.message || 'Failed to upload benchmark'
    uploading.value = false
  }
}
</script>

<style scoped>
.card {
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.list-group-item {
  background-color: rgba(0, 0, 0, 0.2);
  border-color: rgba(255, 255, 255, 0.1);
}
</style>
