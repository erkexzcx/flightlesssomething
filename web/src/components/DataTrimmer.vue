<template>
  <div class="data-trimmer card mb-3">
    <div class="card-body">
      <div class="d-flex justify-content-between align-items-center mb-3">
        <h6 class="card-title mb-0">
          <i class="fa-solid fa-scissors"></i> Data Trimming
        </h6>
        <button
          v-if="hasTrim"
          type="button"
          class="btn btn-sm btn-outline-warning"
          @click="resetTrim"
          title="Reset to show all data"
        >
          <i class="fa-solid fa-undo"></i> Reset
        </button>
      </div>
      
      <p class="text-muted small mb-3">
        <i class="fa-solid fa-info-circle"></i>
        Trim the data to exclude unwanted sections (like loading screens) from calculations.
        Drag the sliders or enter values manually.
      </p>

      <!-- Run selector for multi-run benchmarks -->
      <div v-if="runCount > 1" class="mb-3">
        <label class="form-label">Select Run to Trim:</label>
        <select 
          class="form-select form-select-sm"
          v-model="selectedRunIndex"
          @change="loadRunTrimSettings"
        >
          <option
            v-for="(label, index) in runLabels"
            :key="index"
            :value="index"
          >
            Run {{ index + 1 }}: {{ label }}
          </option>
        </select>
      </div>

      <!-- Trim range controls -->
      <div class="trim-controls">
        <div class="row g-3 mb-3">
          <div class="col-md-6">
            <label class="form-label">Start Sample:</label>
            <input
              type="number"
              class="form-control form-control-sm"
              v-model.number="localTrimStart"
              :min="0"
              :max="maxSampleIndex"
              @input="validateAndEmit"
            />
            <small class="text-muted">First sample to include (0-{{ maxSampleIndex }})</small>
          </div>
          <div class="col-md-6">
            <label class="form-label">End Sample:</label>
            <input
              type="number"
              class="form-control form-control-sm"
              v-model.number="localTrimEnd"
              :min="0"
              :max="maxSampleIndex"
              @input="validateAndEmit"
            />
            <small class="text-muted">Last sample to include (0-{{ maxSampleIndex }})</small>
          </div>
        </div>

        <!-- Visual range slider -->
        <div class="range-slider mb-3">
          <label class="form-label">Range:</label>
          <div class="position-relative">
            <input
              type="range"
              class="form-range trim-start"
              v-model.number="localTrimStart"
              :min="0"
              :max="maxSampleIndex"
              @input="validateAndEmit"
            />
            <input
              type="range"
              class="form-range trim-end"
              v-model.number="localTrimEnd"
              :min="0"
              :max="maxSampleIndex"
              @input="validateAndEmit"
            />
          </div>
          <div class="d-flex justify-content-between text-muted small">
            <span>0</span>
            <span>{{ maxSampleIndex }}</span>
          </div>
        </div>

        <!-- Info display -->
        <div class="alert alert-info small mb-0">
          <div class="row">
            <div class="col-md-6">
              <strong>Total Samples:</strong> {{ totalSamples }}
            </div>
            <div class="col-md-6">
              <strong>Selected Range:</strong> {{ localTrimStart }} - {{ localTrimEnd }} ({{ selectedSamples }} samples)
            </div>
          </div>
          <div v-if="percentageExcluded > 0" class="mt-1">
            <i class="fa-solid fa-exclamation-triangle"></i>
            <strong>Excluded:</strong> {{ percentageExcluded.toFixed(1) }}% of data
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  runIndex: {
    type: Number,
    required: true
  },
  trimStart: {
    type: Number,
    default: 0
  },
  trimEnd: {
    type: Number,
    default: 0
  },
  totalSamples: {
    type: Number,
    required: true
  },
  runCount: {
    type: Number,
    default: 1
  },
  runLabels: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:trimStart', 'update:trimEnd', 'update:runIndex'])

const selectedRunIndex = ref(props.runIndex)
const localTrimStart = ref(props.trimStart)
// Use explicit check for undefined/null instead of truthy evaluation
const localTrimEnd = ref(props.trimEnd !== undefined && props.trimEnd !== null ? props.trimEnd : props.totalSamples - 1)

const maxSampleIndex = computed(() => props.totalSamples - 1)

const selectedSamples = computed(() => {
  if (localTrimStart.value > localTrimEnd.value) return 0
  return localTrimEnd.value - localTrimStart.value + 1
})

const percentageExcluded = computed(() => {
  if (props.totalSamples === 0) return 0
  const excluded = props.totalSamples - selectedSamples.value
  return (excluded / props.totalSamples) * 100
})

const hasTrim = computed(() => {
  return localTrimStart.value !== 0 || localTrimEnd.value !== maxSampleIndex.value
})

function validateAndEmit() {
  // Ensure start is not greater than end
  if (localTrimStart.value > localTrimEnd.value) {
    localTrimEnd.value = localTrimStart.value
  }

  // Ensure within bounds
  if (localTrimStart.value < 0) localTrimStart.value = 0
  if (localTrimStart.value > maxSampleIndex.value) localTrimStart.value = maxSampleIndex.value
  if (localTrimEnd.value < 0) localTrimEnd.value = 0
  if (localTrimEnd.value > maxSampleIndex.value) localTrimEnd.value = maxSampleIndex.value

  emit('update:trimStart', localTrimStart.value)
  emit('update:trimEnd', localTrimEnd.value)
}

function resetTrim() {
  localTrimStart.value = 0
  localTrimEnd.value = maxSampleIndex.value
  validateAndEmit()
}

function loadRunTrimSettings() {
  emit('update:runIndex', selectedRunIndex.value)
}

// Watch for prop changes
watch(() => props.trimStart, (newVal) => {
  localTrimStart.value = newVal
})

watch(() => props.trimEnd, (newVal) => {
  // Use explicit check for undefined/null
  localTrimEnd.value = newVal !== undefined && newVal !== null ? newVal : maxSampleIndex.value
})

watch(() => props.runIndex, (newVal) => {
  selectedRunIndex.value = newVal
})
</script>

<style scoped>
.data-trimmer {
  background-color: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.range-slider {
  position: relative;
}

.form-range.trim-start,
.form-range.trim-end {
  position: absolute;
  width: 100%;
  pointer-events: none;
}

.form-range.trim-start::-webkit-slider-thumb,
.form-range.trim-end::-webkit-slider-thumb {
  pointer-events: all;
}

.form-range.trim-start::-moz-range-thumb,
.form-range.trim-end::-moz-range-thumb {
  pointer-events: all;
}

.form-range.trim-start {
  z-index: 3;
}

.form-range.trim-end {
  z-index: 2;
}

.alert-info {
  background-color: rgba(13, 202, 240, 0.1);
  border-color: rgba(13, 202, 240, 0.2);
  color: #ffffff;
}
</style>
