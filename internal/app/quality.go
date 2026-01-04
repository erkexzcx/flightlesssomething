package app

import (
	"fmt"
	"regexp"
	"strings"
)

// Quality detection thresholds
const (
	minTitleLength       = 10  // Minimum characters for a good title
	minDescriptionLength = 15  // Minimum characters for a good description
	maxRunNameLength     = 25  // Maximum characters for a run name before it's considered too long
	minDataLines         = 100 // Minimum data lines per run
)

// dateTimePatterns are regex patterns that match common datetime formats in run names
var dateTimePatterns = []*regexp.Regexp{
	// Date patterns
	regexp.MustCompile(`\d{4}-\d{2}-\d{2}`),                    // YYYY-MM-DD
	regexp.MustCompile(`\d{4}_\d{2}_\d{2}`),                    // YYYY_MM_DD
	regexp.MustCompile(`\d{2}/\d{2}/\d{4}`),                    // MM/DD/YYYY or DD/MM/YYYY
	regexp.MustCompile(`\d{2}-\d{2}-\d{4}`),                    // MM-DD-YYYY or DD-MM-YYYY
	
	// Time patterns
	regexp.MustCompile(`\d{2}:\d{2}:\d{2}`),                    // HH:MM:SS
	regexp.MustCompile(`\d{2}_\d{2}_\d{2}`),                    // HH_MM_SS
	regexp.MustCompile(`\d{2}-\d{2}-\d{2}`),                    // HH-MM-SS
	
	// Combined datetime patterns (common in auto-generated filenames)
	regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T_\s]\d{2}:\d{2}:\d{2}`), // ISO-like datetime
	regexp.MustCompile(`\d{4}\d{2}\d{2}_?\d{6}`),                    // Compact format: YYYYMMDD_HHMMSS
}

// CalculateQualityIndicators determines quality flags for a benchmark
func CalculateQualityIndicators(title, description string, runLabels []string) (isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle bool) {
	// Check if single run
	isSingleRun = len(runLabels) == 1
	
	// Check title quality
	hasLowQualityTitle = len(strings.TrimSpace(title)) < minTitleLength
	
	// Check description quality
	descTrimmed := strings.TrimSpace(description)
	hasLowQualityDescription = len(descTrimmed) < minDescriptionLength
	
	// Check run names quality
	hasLowQualityRunNames = HasLowQualityRunNames(runLabels)
	
	return
}

// CalculateQualityIndicatorsWithData determines quality flags for a benchmark including data analysis
func CalculateQualityIndicatorsWithData(title, description string, benchmarkData []*BenchmarkData) (isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle, hasDuplicateRuns, hasInsufficientData bool) {
	runLabels := make([]string, len(benchmarkData))
	for i, data := range benchmarkData {
		runLabels[i] = data.Label
	}
	
	// Use existing checks
	isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle = CalculateQualityIndicators(title, description, runLabels)
	
	// Check for duplicate runs
	hasDuplicateRuns = HasDuplicateRuns(benchmarkData)
	
	// Check for insufficient data
	hasInsufficientData = HasInsufficientData(benchmarkData)
	
	return
}

// HasLowQualityRunNames checks if any run name contains datetime patterns or is too long
func HasLowQualityRunNames(runLabels []string) bool {
	for _, label := range runLabels {
		trimmedLabel := strings.TrimSpace(label)
		
		// Check if run name is too long
		if len(trimmedLabel) > maxRunNameLength {
			return true
		}
		
		// Check for datetime patterns
		if containsDateTimePattern(trimmedLabel) {
			return true
		}
	}
	return false
}

// containsDateTimePattern checks if a string contains any datetime pattern
func containsDateTimePattern(s string) bool {
	for _, pattern := range dateTimePatterns {
		if pattern.MatchString(s) {
			return true
		}
	}
	return false
}

// HasDuplicateRuns checks if benchmark has duplicate run names or identical data
func HasDuplicateRuns(benchmarkData []*BenchmarkData) bool {
	if len(benchmarkData) <= 1 {
		return false
	}
	
	// Check for duplicate run names
	labelsSeen := make(map[string]bool)
	for _, data := range benchmarkData {
		trimmedLabel := strings.TrimSpace(data.Label)
		if labelsSeen[trimmedLabel] {
			return true // Found duplicate name
		}
		labelsSeen[trimmedLabel] = true
	}
	
	// Check for identical data (comparing FPS data as a signature)
	// We compare the length and first/last few values to detect duplicates without full comparison
	type dataSignature struct {
		length int
		first  float64
		last   float64
		sum    float64
	}
	
	signatures := make(map[dataSignature]bool)
	for _, data := range benchmarkData {
		if len(data.DataFPS) == 0 {
			continue
		}
		
		// Calculate signature
		sum := 0.0
		for _, v := range data.DataFPS {
			sum += v
		}
		
		sig := dataSignature{
			length: len(data.DataFPS),
			first:  data.DataFPS[0],
			last:   data.DataFPS[len(data.DataFPS)-1],
			sum:    sum,
		}
		
		if signatures[sig] {
			return true // Found duplicate data
		}
		signatures[sig] = true
	}
	
	return false
}

// HasInsufficientData checks if any run has less than minimum required data lines
func HasInsufficientData(benchmarkData []*BenchmarkData) bool {
	for _, data := range benchmarkData {
		// Check FPS data length as it's the primary metric
		if len(data.DataFPS) < minDataLines {
			return true
		}
	}
	return false
}

// GetQualityIssues returns a list of quality issues for display to users
func GetQualityIssues(isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle bool, runLabels []string) []string {
	var issues []string
	
	if isSingleRun {
		issues = append(issues, "Single run benchmark (consider adding more runs for comparison)")
	}
	
	if hasLowQualityTitle {
		issues = append(issues, "Title is too short (less than 10 characters)")
	}
	
	if hasLowQualityDescription {
		issues = append(issues, "Description is too short or missing (less than 15 characters)")
	}
	
	if hasLowQualityRunNames {
		// Only provide specific details about run name issues if requested
		// Re-check each label to provide specific feedback (this is for UI display only)
		for _, label := range runLabels {
			trimmedLabel := strings.TrimSpace(label)
			// Check length first as it's cheaper
			if len(trimmedLabel) > maxRunNameLength {
				issues = append(issues, "Run name too long: \""+trimmedLabel+"\" (over 25 characters)")
			} else if containsDateTimePattern(trimmedLabel) {
				issues = append(issues, "Run name contains date/time pattern: \""+trimmedLabel+"\"")
			}
		}
	}
	
	return issues
}

// GetQualityIssuesWithData returns a list of quality issues including data-based checks
func GetQualityIssuesWithData(isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle, hasDuplicateRuns, hasInsufficientData bool, benchmarkData []*BenchmarkData) []string {
	runLabels := make([]string, len(benchmarkData))
	for i, data := range benchmarkData {
		runLabels[i] = data.Label
	}
	
	// Get base issues
	issues := GetQualityIssues(isSingleRun, hasLowQualityRunNames, hasLowQualityDescription, hasLowQualityTitle, runLabels)
	
	if hasDuplicateRuns {
		issues = append(issues, "Benchmark has duplicate run names or identical data")
	}
	
	if hasInsufficientData {
		// Provide specific details about which runs have insufficient data
		for _, data := range benchmarkData {
			if len(data.DataFPS) < minDataLines {
				issues = append(issues, fmt.Sprintf("Run \"%s\" has insufficient data (%d lines, minimum %d required)", data.Label, len(data.DataFPS), minDataLines))
			}
		}
	}
	
	return issues
}
