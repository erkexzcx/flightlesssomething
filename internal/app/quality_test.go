package app

import (
	"testing"
)

func TestCalculateQualityIndicators(t *testing.T) {
	tests := []struct {
		name                     string
		title                    string
		description              string
		runLabels                []string
		wantIsSingleRun          bool
		wantHasLowQualityRunNames bool
		wantHasLowQualityDescription bool
		wantHasLowQualityTitle   bool
	}{
		{
			name:                     "high quality benchmark",
			title:                    "Cyberpunk 2077 - High Settings",
			description:              "Testing performance with high graphics settings on my gaming rig",
			runLabels:                []string{"High Settings", "Ultra Settings"},
			wantIsSingleRun:          false,
			wantHasLowQualityRunNames: false,
			wantHasLowQualityDescription: false,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "single run benchmark",
			title:                    "Quick Test Benchmark",
			description:              "Just a quick performance test",
			runLabels:                []string{"Test Run"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: false,
			wantHasLowQualityDescription: false,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "datetime in run name",
			title:                    "Performance Test",
			description:              "Testing game performance",
			runLabels:                []string{"2026-01-02_20-36-40_summary", "Another Run"},
			wantIsSingleRun:          false,
			wantHasLowQualityRunNames: true,
			wantHasLowQualityDescription: false,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "long run name",
			title:                    "Performance Test",
			description:              "Testing game performance",
			runLabels:                []string{"This is a very long run name that exceeds twenty five characters"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: true,
			wantHasLowQualityDescription: false,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "short title",
			title:                    "Test",
			description:              "A proper description here",
			runLabels:                []string{"Run 1"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: false,
			wantHasLowQualityDescription: false,
			wantHasLowQualityTitle:   true,
		},
		{
			name:                     "short description",
			title:                    "Benchmark Title Here",
			description:              "Short",
			runLabels:                []string{"Run 1"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: false,
			wantHasLowQualityDescription: true,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "empty description",
			title:                    "Benchmark Title",
			description:              "",
			runLabels:                []string{"Run 1"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: false,
			wantHasLowQualityDescription: true,
			wantHasLowQualityTitle:   false,
		},
		{
			name:                     "multiple quality issues",
			title:                    "Test",
			description:              "",
			runLabels:                []string{"2026-01-02_summary"},
			wantIsSingleRun:          true,
			wantHasLowQualityRunNames: true,
			wantHasLowQualityDescription: true,
			wantHasLowQualityTitle:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsSingleRun, gotHasLowQualityRunNames, gotHasLowQualityDescription, gotHasLowQualityTitle := CalculateQualityIndicators(
				tt.title, tt.description, tt.runLabels,
			)

			if gotIsSingleRun != tt.wantIsSingleRun {
				t.Errorf("IsSingleRun = %v, want %v", gotIsSingleRun, tt.wantIsSingleRun)
			}
			if gotHasLowQualityRunNames != tt.wantHasLowQualityRunNames {
				t.Errorf("HasLowQualityRunNames = %v, want %v", gotHasLowQualityRunNames, tt.wantHasLowQualityRunNames)
			}
			if gotHasLowQualityDescription != tt.wantHasLowQualityDescription {
				t.Errorf("HasLowQualityDescription = %v, want %v", gotHasLowQualityDescription, tt.wantHasLowQualityDescription)
			}
			if gotHasLowQualityTitle != tt.wantHasLowQualityTitle {
				t.Errorf("HasLowQualityTitle = %v, want %v", gotHasLowQualityTitle, tt.wantHasLowQualityTitle)
			}
		})
	}
}

func TestContainsDateTimePattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"ISO date format", "2026-01-02", true},
		{"ISO datetime", "2026-01-02T20:36:40", true},
		{"Underscore date", "2026_01_02", true},
		{"Time format", "20:36:40", true},
		{"Compact datetime", "20260102_203640", true},
		{"US date format", "01/02/2026", true},
		{"Filename with datetime", "tlou-ii_2026-01-02_20-36-40_summary", true},
		{"No datetime", "High Settings", false},
		{"No datetime 2", "Ultra Performance", false},
		{"Run 1", "Run 1", false},
		{"Numbers but not datetime", "Test123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsDateTimePattern(tt.input)
			if got != tt.want {
				t.Errorf("containsDateTimePattern(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasLowQualityRunNames(t *testing.T) {
	tests := []struct {
		name      string
		runLabels []string
		want      bool
	}{
		{
			name:      "good run names",
			runLabels: []string{"High", "Medium", "Low"},
			want:      false,
		},
		{
			name:      "datetime in one name",
			runLabels: []string{"High", "2026-01-02_summary"},
			want:      true,
		},
		{
			name:      "long run name",
			runLabels: []string{"This is a very long run name exceeding the limit"},
			want:      true,
		},
		{
			name:      "exactly 25 chars is ok",
			runLabels: []string{"ABCDEFGHIJKLMNOPQRSTUVWXY"}, // 25 chars, no datetime
			want:      false,
		},
		{
			name:      "26 chars is too long",
			runLabels: []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ"}, // 26 chars
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasLowQualityRunNames(tt.runLabels)
			if got != tt.want {
				t.Errorf("HasLowQualityRunNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetQualityIssues(t *testing.T) {
	tests := []struct {
		name                     string
		isSingleRun              bool
		hasLowQualityRunNames    bool
		hasLowQualityDescription bool
		hasLowQualityTitle       bool
		runLabels                []string
		wantMinIssues            int // Minimum number of issues expected
	}{
		{
			name:                     "no issues",
			isSingleRun:              false,
			hasLowQualityRunNames:    false,
			hasLowQualityDescription: false,
			hasLowQualityTitle:       false,
			runLabels:                []string{"Run 1", "Run 2"},
			wantMinIssues:            0,
		},
		{
			name:                     "single run only",
			isSingleRun:              true,
			hasLowQualityRunNames:    false,
			hasLowQualityDescription: false,
			hasLowQualityTitle:       false,
			runLabels:                []string{"Run 1"},
			wantMinIssues:            1,
		},
		{
			name:                     "datetime in run name",
			isSingleRun:              false,
			hasLowQualityRunNames:    true,
			hasLowQualityDescription: false,
			hasLowQualityTitle:       false,
			runLabels:                []string{"2026-01-02_summary"},
			wantMinIssues:            1,
		},
		{
			name:                     "all quality issues",
			isSingleRun:              true,
			hasLowQualityRunNames:    true,
			hasLowQualityDescription: true,
			hasLowQualityTitle:       true,
			runLabels:                []string{"2026-01-02_20-36-40_very_long_name_here"},
			wantMinIssues:            4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := GetQualityIssues(
				tt.isSingleRun,
				tt.hasLowQualityRunNames,
				tt.hasLowQualityDescription,
				tt.hasLowQualityTitle,
				tt.runLabels,
			)
			if len(issues) < tt.wantMinIssues {
				t.Errorf("GetQualityIssues() returned %d issues, want at least %d. Issues: %v",
					len(issues), tt.wantMinIssues, issues)
			}
		})
	}
}
