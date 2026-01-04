// Client-side quality validation for benchmarks
// This mirrors the server-side quality detection logic

const MIN_TITLE_LENGTH = 10
const MIN_DESCRIPTION_LENGTH = 15
const MAX_RUN_NAME_LENGTH = 25
const MIN_DATA_LINES = 100

// DateTime patterns that indicate auto-generated filenames
const DATE_TIME_PATTERNS = [
  // Date patterns
  /\d{4}-\d{2}-\d{2}/, // YYYY-MM-DD
  /\d{4}_\d{2}_\d{2}/, // YYYY_MM_DD
  /\d{2}\/\d{2}\/\d{4}/, // MM/DD/YYYY or DD/MM/YYYY
  /\d{2}-\d{2}-\d{4}/, // MM-DD-YYYY or DD-MM-YYYY
  
  // Time patterns
  /\d{2}:\d{2}:\d{2}/, // HH:MM:SS
  /\d{2}_\d{2}_\d{2}/, // HH_MM_SS
  /\d{2}-\d{2}-\d{2}/, // HH-MM-SS
  
  // Combined datetime patterns
  /\d{4}-\d{2}-\d{2}[T_\s]\d{2}:\d{2}:\d{2}/, // ISO-like datetime
  /\d{4}\d{2}\d{2}_?\d{6}/, // Compact format: YYYYMMDD_HHMMSS
]

/**
 * Check if a string contains a datetime pattern
 */
function containsDateTimePattern(str) {
  return DATE_TIME_PATTERNS.some(pattern => pattern.test(str))
}

/**
 * Check if run names have low quality (datetime patterns or too long)
 */
function hasLowQualityRunNames(runLabels) {
  for (const label of runLabels) {
    const trimmed = label.trim()
    
    // Check if too long
    if (trimmed.length > MAX_RUN_NAME_LENGTH) {
      return true
    }
    
    // Check for datetime patterns
    if (containsDateTimePattern(trimmed)) {
      return true
    }
  }
  return false
}

/**
 * Check if there are duplicate run names
 */
function hasDuplicateRunNames(runLabels) {
  const seen = new Set()
  for (const label of runLabels) {
    const trimmed = label.trim()
    if (seen.has(trimmed)) {
      return true
    }
    seen.add(trimmed)
  }
  return false
}

/**
 * Calculate all quality indicators for a benchmark
 */
export function calculateQualityIndicators(title, description, runLabels) {
  const isSingleRun = runLabels.length === 1
  const hasLowQualityTitle = title.trim().length < MIN_TITLE_LENGTH
  const hasLowQualityDescription = description.trim().length < MIN_DESCRIPTION_LENGTH
  const hasLowQualityRunNamesFlag = hasLowQualityRunNames(runLabels)
  const hasDuplicateRuns = hasDuplicateRunNames(runLabels)
  
  return {
    isSingleRun,
    hasLowQualityRunNames: hasLowQualityRunNamesFlag,
    hasLowQualityDescription,
    hasLowQualityTitle,
    hasDuplicateRuns,
    // Note: hasInsufficientData cannot be determined client-side without parsing files
    hasInsufficientData: false,
  }
}

/**
 * Get a list of specific quality issues for display
 */
export function getQualityIssues(title, description, runLabels) {
  const issues = []
  const indicators = calculateQualityIndicators(title, description, runLabels)
  
  if (indicators.isSingleRun) {
    issues.push('Single run benchmark (consider adding more runs for comparison)')
  }
  
  if (indicators.hasLowQualityTitle) {
    issues.push('Title is too short (less than 10 characters)')
  }
  
  if (indicators.hasLowQualityDescription) {
    issues.push('Description is too short or missing (less than 15 characters)')
  }
  
  if (indicators.hasLowQualityRunNames) {
    // Provide specific details about each problematic run name
    for (const label of runLabels) {
      const trimmed = label.trim()
      // Check length first (cheaper)
      if (trimmed.length > MAX_RUN_NAME_LENGTH) {
        issues.push(`Run name too long: "${trimmed}" (over 25 characters)`)
      } else if (containsDateTimePattern(trimmed)) {
        issues.push(`Run name contains date/time pattern: "${trimmed}"`)
      }
    }
  }
  
  if (indicators.hasDuplicateRuns) {
    issues.push('Benchmark has duplicate run names')
  }
  
  return issues
}

/**
 * Check if a benchmark is considered low quality
 */
export function isLowQuality(title, description, runLabels) {
  const indicators = calculateQualityIndicators(title, description, runLabels)
  return indicators.isSingleRun || 
         indicators.hasLowQualityRunNames || 
         indicators.hasLowQualityDescription || 
         indicators.hasLowQualityTitle ||
         indicators.hasDuplicateRuns
}
