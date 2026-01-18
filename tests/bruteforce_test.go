package tests

import (
	"os"
	"testing"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/db"
)

func TestBruteForceLockout(t *testing.T) {
	dbPath := "test_bruteforce.db"
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	if err := db.InitDB(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	username := "testuser"
	password := "password123"
	hash, _ := auth.HashPassword(password)
	db.CreateUser(username, hash)

	maxAttempts := 3
	lockoutDuration := 2 * time.Second

	// 1. Fail login multiple times
	for i := 0; i < maxAttempts; i++ {
		err := db.IncrementFailedAttempts(username, maxAttempts, lockoutDuration)
		if err != nil {
			t.Fatalf("Failed to increment attempts: %v", err)
		}
	}

	// 2. Check if locked
	user, _ := db.GetUserByUsername(username)
	if user.FailedAttempts != maxAttempts {
		t.Errorf("Expected %d failed attempts, got %d", maxAttempts, user.FailedAttempts)
	}
	if user.LockedUntil == nil {
		t.Error("Account should be locked")
	}
	if !time.Now().Before(*user.LockedUntil) {
		t.Error("Account should be locked in the future")
	}

	// 3. Wait for lockout to expire
	time.Sleep(lockoutDuration + 100*time.Millisecond)

	// 4. Check if still locked (it should be, but time.Now().Before will be false)
	user, _ = db.GetUserByUsername(username)
	if time.Now().Before(*user.LockedUntil) {
		t.Error("Account should be expired now")
	}

	// 5. Successful login should reset
	err := db.ResetFailedAttempts(username)
	if err != nil {
		t.Fatalf("Failed to reset attempts: %v", err)
	}

	user, _ = db.GetUserByUsername(username)
	if user.FailedAttempts != 0 {
		t.Errorf("Expected 0 failed attempts, got %d", user.FailedAttempts)
	}
	if user.LockedUntil != nil {
		t.Error("Account should be unlocked")
	}
}
