package app

import (
	"time"

	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	gorm.Model
	DiscordID         string     `gorm:"size:20;uniqueIndex"`
	Username          string     `gorm:"size:32"`
	IsAdmin           bool       `gorm:"default:false"`
	IsBanned          bool       `gorm:"default:false"`
	LastWebActivityAt *time.Time `gorm:"default:null"`                       // Last activity via web session
	LastAPIActivityAt *time.Time `gorm:"default:null"`                       // Last activity via API token
	BenchmarkCount    int        `gorm:"-" json:"benchmark_count,omitempty"` // Number of benchmarks by user
	APITokenCount     int        `gorm:"-" json:"api_token_count,omitempty"` // Number of API tokens by user

	Benchmarks []Benchmark `gorm:"constraint:OnDelete:CASCADE;"`
	APITokens  []APIToken  `gorm:"constraint:OnDelete:CASCADE;"`
}

// APIToken represents an API token for a user
type APIToken struct {
	gorm.Model
	UserID     uint
	Token      string     `gorm:"size:64;uniqueIndex"`
	Name       string     `gorm:"size:100"`
	LastUsedAt *time.Time `gorm:"default:null"`

	User User `gorm:"foreignKey:UserID;"`
}

// Benchmark represents a benchmark record in the database
type Benchmark struct {
	gorm.Model
	UserID      uint
	Title       string `gorm:"size:100"`
	Description string `gorm:"size:5000"`
	
	// Searchable metadata extracted from benchmark data files
	RunNames       string `gorm:"type:text"` // Comma-separated list of run labels for search
	Specifications string `gorm:"type:text"` // Concatenated specifications (OS, CPU, GPU, etc.) for search
	DataSizeBytes  int64  `gorm:"default:0"` // Uncompressed data size in bytes

	CreatedAtHumanized string   `gorm:"-"`                             // Human readable "X h/m/s ago" version of CreatedAt
	UpdatedAtHumanized string   `gorm:"-"`                             // Human readable "X h/m/s ago" version of UpdatedAt
	RunCount           int      `gorm:"-" json:"run_count,omitempty"`  // Number of runs in benchmark
	RunLabels          []string `gorm:"-" json:"run_labels,omitempty"` // Labels of runs

	User User `gorm:"foreignKey:UserID;"`
}

// AfterFind is a GORM hook that is called after a record is found
func (b *Benchmark) AfterFind(tx *gorm.DB) (err error) {
	b.CreatedAtHumanized = humanize.Time(b.CreatedAt)
	b.UpdatedAtHumanized = humanize.Time(b.UpdatedAt)
	return nil
}

// BenchmarkMetadata represents lightweight metadata for a benchmark
type BenchmarkMetadata struct {
	RunCount  int
	RunLabels []string
}

// AuditLog represents an audit log entry for tracking user actions
type AuditLog struct {
	gorm.Model
	UserID      uint   // User who performed the action
	Action      string `gorm:"size:100"`  // Short action title
	Description string `gorm:"size:1000"` // Detailed description
	TargetType  string `gorm:"size:50"`   // Type of target (user, benchmark, etc.)
	TargetID    uint   // ID of the target (user ID, benchmark ID, etc.)

	CreatedAtHumanized string `gorm:"-"` // Human readable "X h/m/s ago" version of CreatedAt
	User               User   `gorm:"foreignKey:UserID;"`
}

// AfterFind is a GORM hook that is called after a record is found
func (a *AuditLog) AfterFind(tx *gorm.DB) (err error) {
	a.CreatedAtHumanized = humanize.Time(a.CreatedAt)
	return nil
}

// BenchmarkData represents the actual benchmark data stored separately
type BenchmarkData struct {
	Label string

	// System specs
	SpecOS             string
	SpecCPU            string
	SpecGPU            string
	SpecRAM            string
	SpecLinuxKernel    string
	SpecLinuxScheduler string

	// Performance data arrays
	DataFPS          []float64
	DataFrameTime    []float64
	DataCPULoad      []float64
	DataGPULoad      []float64
	DataCPUTemp      []float64
	DataCPUPower     []float64
	DataGPUTemp      []float64
	DataGPUCoreClock []float64
	DataGPUMemClock  []float64
	DataGPUVRAMUsed  []float64
	DataGPUPower     []float64
	DataRAMUsed      []float64
	DataSwapUsed     []float64
}
