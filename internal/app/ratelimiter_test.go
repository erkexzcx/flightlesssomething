package app

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a rate limiter: 3 attempts per 1 second
	rl := NewRateLimiter(3, 1*time.Second)

	// First 3 attempts should succeed
	for i := 0; i < 3; i++ {
		if !rl.Allow("test-key") {
			t.Errorf("Attempt %d should be allowed", i+1)
		}
	}

	// 4th attempt should fail
	if rl.Allow("test-key") {
		t.Error("4th attempt should be blocked")
	}

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("test-key") {
		t.Error("After window expired, attempt should be allowed")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)

	// Use up limit for key1
	if !rl.Allow("key1") {
		t.Error("First attempt for key1 should be allowed")
	}
	if !rl.Allow("key1") {
		t.Error("Second attempt for key1 should be allowed")
	}
	if rl.Allow("key1") {
		t.Error("Third attempt for key1 should be blocked")
	}

	// key2 should still be allowed
	if !rl.Allow("key2") {
		t.Error("First attempt for key2 should be allowed")
	}
	if !rl.Allow("key2") {
		t.Error("Second attempt for key2 should be allowed")
	}
	if rl.Allow("key2") {
		t.Error("Third attempt for key2 should be blocked")
	}
}

func TestRateLimiter_IsLocked(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)

	// Initially not locked
	if rl.IsLocked("test-key") {
		t.Error("Should not be locked initially")
	}

	// Use up the limit
	rl.Allow("test-key")
	rl.Allow("test-key")

	// Should be locked now
	if !rl.IsLocked("test-key") {
		t.Error("Should be locked after reaching limit")
	}

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// Should not be locked anymore
	if rl.IsLocked("test-key") {
		t.Error("Should not be locked after window expired")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)

	// Use up the limit
	rl.Allow("test-key")
	rl.Allow("test-key")

	// Should be locked
	if !rl.IsLocked("test-key") {
		t.Error("Should be locked after reaching limit")
	}

	// Reset
	rl.Reset("test-key")

	// Should not be locked anymore
	if rl.IsLocked("test-key") {
		t.Error("Should not be locked after reset")
	}

	// Should be allowed again
	if !rl.Allow("test-key") {
		t.Error("Should be allowed after reset")
	}
}

func TestRateLimiter_GetRemainingTime(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)

	// Initially no remaining time
	if remaining := rl.GetRemainingTime("test-key"); remaining != 0 {
		t.Errorf("Initially remaining time should be 0, got %v", remaining)
	}

	// Use up the limit
	rl.Allow("test-key")
	rl.Allow("test-key")

	// Should have remaining time
	remaining := rl.GetRemainingTime("test-key")
	if remaining <= 0 || remaining > 1*time.Second {
		t.Errorf("Remaining time should be between 0 and 1 second, got %v", remaining)
	}

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Remaining time should be less
	newRemaining := rl.GetRemainingTime("test-key")
	if newRemaining >= remaining {
		t.Errorf("Remaining time should decrease, was %v, now %v", remaining, newRemaining)
	}
}

func TestRateLimiter_CleanupExpired(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Add some entries
	rl.Allow("key1")
	rl.Allow("key2")

	// Verify entries exist
	if len(rl.entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(rl.entries))
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Cleanup
	rl.CleanupExpired()

	// Entries should be cleaned up
	if len(rl.entries) != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", len(rl.entries))
	}
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)

	// Make 2 attempts
	rl.Allow("test-key")
	rl.Allow("test-key")

	// Wait 600ms
	time.Sleep(600 * time.Millisecond)

	// Make 1 more attempt (total 3 in window)
	if !rl.Allow("test-key") {
		t.Error("3rd attempt within window should be allowed")
	}

	// 4th attempt should fail
	if rl.Allow("test-key") {
		t.Error("4th attempt should be blocked")
	}

	// Wait 500ms more (total 1100ms from first attempt)
	time.Sleep(500 * time.Millisecond)

	// First 2 attempts should have expired, should be allowed now
	if !rl.Allow("test-key") {
		t.Error("After first 2 attempts expired, should be allowed")
	}
}

func TestInitRateLimiters(t *testing.T) {
	// Call InitRateLimiters
	InitRateLimiters()

	// Check that limiters are initialized
	if benchmarkUploadLimiter == nil {
		t.Error("benchmarkUploadLimiter should be initialized")
	}

	if adminLoginLimiter == nil {
		t.Error("adminLoginLimiter should be initialized")
	}

	// Check GetBenchmarkUploadLimiter
	if GetBenchmarkUploadLimiter() == nil {
		t.Error("GetBenchmarkUploadLimiter should return initialized limiter")
	}

	// Check GetAdminLoginLimiter
	if GetAdminLoginLimiter() == nil {
		t.Error("GetAdminLoginLimiter should return initialized limiter")
	}
}
