package flightlesssomething

import (
	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	DiscordID string
	Username  string

	Benchmarks []Benchmark `gorm:"constraint:OnDelete:CASCADE;"`
}

type Benchmark struct {
	gorm.Model
	UserID      uint
	Title       string
	Description string

	CreatedAtHumanized string `gorm:"-"` // Human readable "X h/m/s ago" version of CreatedAt (filled automatically)

	User User `gorm:"foreignKey:UserID;"`
}

// AfterFind is a GORM hook that is called after a record is found
func (b *Benchmark) AfterFind(tx *gorm.DB) (err error) {
	b.CreatedAtHumanized = humanize.Time(b.CreatedAt)
	return nil
}
