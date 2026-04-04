package app

import (
	"time"

	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

// BaseModel replaces gorm.Model with snake_case JSON field names.
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents a user in the system
type User struct {
	BaseModel
	DiscordID         string     `gorm:"size:20;uniqueIndex" json:"discord_id"`
	Username          string     `gorm:"size:32" json:"username"`
	IsAdmin           bool       `gorm:"default:false" json:"is_admin"`
	IsBanned          bool       `gorm:"default:false" json:"is_banned"`
	LastWebActivityAt *time.Time `gorm:"default:null" json:"last_web_activity_at"`
	LastAPIActivityAt *time.Time `gorm:"default:null" json:"last_api_activity_at"`
	BenchmarkCount    int        `gorm:"-" json:"benchmark_count,omitempty"`
	APITokenCount     int        `gorm:"-" json:"api_token_count,omitempty"`

	Benchmarks []Benchmark `gorm:"constraint:OnDelete:CASCADE;" json:"benchmarks,omitempty"`
	APITokens  []APIToken  `gorm:"constraint:OnDelete:CASCADE;" json:"api_tokens,omitempty"`
}

// APIToken represents an API token for a user
type APIToken struct {
	BaseModel
	UserID     uint       `gorm:"index" json:"user_id"`
	Token      string     `gorm:"size:64;uniqueIndex" json:"token"`
	Name       string     `gorm:"size:100" json:"name"`
	LastUsedAt *time.Time `gorm:"default:null" json:"last_used_at"`

	User User `gorm:"foreignKey:UserID;" json:"user,omitempty"`
}

// Benchmark represents a benchmark record in the database
type Benchmark struct {
	BaseModel
	UserID      uint   `gorm:"index" json:"user_id"`
	Title       string `gorm:"size:100" json:"title"`
	Description string `gorm:"size:5000" json:"description"`

	RunNames       string `gorm:"type:text" json:"run_names"`
	Specifications string `gorm:"type:text" json:"specifications"`

	CreatedAtHumanized string   `gorm:"-" json:"created_at_humanized"`
	UpdatedAtHumanized string   `gorm:"-" json:"updated_at_humanized"`
	RunCount           int      `gorm:"-" json:"run_count,omitempty"`
	RunLabels          []string `gorm:"-" json:"run_labels,omitempty"`

	User User `gorm:"foreignKey:UserID;" json:"user,omitempty"`
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
