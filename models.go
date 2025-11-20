package flightlesssomething

import (
	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	DiscordID string `gorm:"size:20"`
	Username  string `gorm:"size:32"`
	IsAdmin   bool   `gorm:"default:false"`

	Benchmarks []Benchmark `gorm:"constraint:OnDelete:CASCADE;"`
}

type Benchmark struct {
	gorm.Model
	UserID      uint
	Title       string `gorm:"size:100"`
	Description string `gorm:"size:500"`
	AiSummary   string

	CreatedAtHumanized string `gorm:"-"` // Human readable "X h/m/s ago" version of CreatedAt (filled automatically)

	User User `gorm:"foreignKey:UserID;"`
}

// AfterFind is a GORM hook that is called after a record is found
func (b *Benchmark) AfterFind(tx *gorm.DB) (err error) {
	b.CreatedAtHumanized = humanize.Time(b.CreatedAt)
	return nil
}
