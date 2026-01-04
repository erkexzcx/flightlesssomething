package app

import (
	"regexp"
	"strings"
)

// Quality detection thresholds
const (
	minTitleLength       = 10  // Minimum characters for a good title
	minDescriptionLength = 15  // Minimum characters for a good description
	maxRunNameLength     = 25  // Maximum characters for a run name before it's considered too long
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
