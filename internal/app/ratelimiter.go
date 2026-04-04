package app

import (
	"sync"
	"time"
)

// RateLimiter provides in-memory rate limiting functionality
type RateLimiter struct {
	mu          sync.RWMutex
	entries     map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

// NewRateLimiter creates a new rate limiter
// maxAttempts: maximum number of attempts allowed
// window: time window for rate limiting
func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries:     make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// Allow checks if the key is allowed to proceed
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get existing attempts for this key
	attempts := rl.entries[key]

	// Remove expired attempts in-place (no allocation)
	n := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			attempts[n] = t
			n++
		}
	}
	attempts = attempts[:n]

	// Check if rate limit exceeded
	if len(attempts) >= rl.maxAttempts {
		rl.entries[key] = attempts // persist compacted slice
		return false
	}

	// Record this attempt
	attempts = append(attempts, now)
	rl.entries[key] = attempts

	return true
}

// AllowWithRemaining atomically checks whether key is allowed and, if not,
// returns the duration until the rate limit lifts. This avoids the TOCTOU
// race between a separate Allow() + GetRemainingTime() call pair.
// Returns (true, 0) when allowed; (false, remaining) when rate-limited.
func (rl *RateLimiter) AllowWithRemaining(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	attempts := rl.entries[key]

	// Remove expired attempts in-place
	n := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			attempts[n] = t
			n++
		}
	}
	attempts = attempts[:n]

	// If rate limit exceeded, calculate remaining time under the same lock
	if len(attempts) >= rl.maxAttempts {
		oldestValid := time.Time{}
		for _, t := range attempts {
			if oldestValid.IsZero() || t.Before(oldestValid) {
				oldestValid = t
			}
		}
		rl.entries[key] = attempts
		var remaining time.Duration
		if !oldestValid.IsZero() {
			if r := oldestValid.Add(rl.window).Sub(now); r > 0 {
				remaining = r
			}
		}
		return false, remaining
	}

	// Record this attempt
	attempts = append(attempts, now)
	rl.entries[key] = attempts
	return true, 0
}

// Reset clears all entries for a specific key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.entries, key)
}

// IsLocked checks if a key is currently rate limited without recording an attempt
func (rl *RateLimiter) IsLocked(key string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	attempts := rl.entries[key]
	validAttempts := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			validAttempts++
		}
	}

	return validAttempts >= rl.maxAttempts
}

// GetRemainingTime returns the duration until the rate limit is lifted
// Returns 0 if not currently rate limited
func (rl *RateLimiter) GetRemainingTime(key string) time.Duration {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	attempts := rl.entries[key]
	// Count only valid (non-expired) entries
	validCount := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			validCount++
		}
	}
	if validCount < rl.maxAttempts {
		return 0
	}

	// Find the oldest valid attempt
	oldestValid := time.Time{} // Zero value
	for _, t := range attempts {
		if t.After(cutoff) {
			if oldestValid.IsZero() || t.Before(oldestValid) {
				oldestValid = t
			}
		}
	}

	if oldestValid.IsZero() {
		return 0
	}

	// Calculate when the oldest attempt will expire
	unlockTime := oldestValid.Add(rl.window)
	remaining := unlockTime.Sub(now)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// CleanupExpired removes all expired entries to free memory
// Should be called periodically
func (rl *RateLimiter) CleanupExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for key, attempts := range rl.entries {
		n := 0
		for _, t := range attempts {
			if t.After(cutoff) {
				attempts[n] = t
				n++
			}
		}

		if n == 0 {
			delete(rl.entries, key)
		} else {
			rl.entries[key] = attempts[:n]
		}
	}
}

var (
	// Global rate limiters
	benchmarkUploadLimiter *RateLimiter
	adminLoginLimiter      *RateLimiter
	debugCalcLimiter       *RateLimiter
	cleanupOnce            sync.Once
)

// InitRateLimiters initializes the global rate limiters
func InitRateLimiters() {
	// 5 benchmark uploads per 10 minutes per user
	benchmarkUploadLimiter = NewRateLimiter(5, 10*time.Minute)

	// 3 failed admin login attempts per source IP locks for 10 minutes
	adminLoginLimiter = NewRateLimiter(3, 10*time.Minute)

	// 30 debug calc requests per minute per IP
	debugCalcLimiter = NewRateLimiter(30, time.Minute)

	// Start cleanup goroutine (only once)
	// Note: This goroutine runs for the lifetime of the application.
	// It will be terminated naturally when the application shuts down.
	cleanupOnce.Do(func() {
		// Capture limiter values at goroutine start to avoid data races
		// if InitRateLimiters() is called again (e.g., in tests).
		bl := benchmarkUploadLimiter
		al := adminLoginLimiter
		dl := debugCalcLimiter
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				bl.CleanupExpired()
				al.CleanupExpired()
				dl.CleanupExpired()
			}
		}()
	})
}

// GetBenchmarkUploadLimiter returns the global benchmark upload limiter
func GetBenchmarkUploadLimiter() *RateLimiter {
	return benchmarkUploadLimiter
}

// GetAdminLoginLimiter returns the global admin login limiter
func GetAdminLoginLimiter() *RateLimiter {
	return adminLoginLimiter
}

// GetDebugCalcLimiter returns the global debug calc rate limiter
func GetDebugCalcLimiter() *RateLimiter {
	return debugCalcLimiter
}
