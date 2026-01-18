package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/auth"
	"github.com/prashanta0234/vpsmyth/internal/db"
)

var (
	loginAttempts = make(map[string]int)
	loginMutex    sync.Mutex
	maxAttempts   = 5
	lockoutTime   = 15 * time.Minute
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple IP-based rate limiting
	ip := r.RemoteAddr
	loginMutex.Lock()
	if attempts, exists := loginAttempts[ip]; exists && attempts > 10 {
		loginMutex.Unlock()
		http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}
	loginAttempts[ip]++
	loginMutex.Unlock()

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Account is locked. Try again after %s", user.LockedUntil.Format("15:04:05")),
		})
		return
	}

	match, err := auth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil || !match {
		// Increment failed attempts
		db.IncrementFailedAttempts(req.Username, maxAttempts, lockoutTime)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Reset failed attempts on success
	db.ResetFailedAttempts(req.Username)

	token, err := auth.GenerateToken(user.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "vpsmyth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "vpsmyth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}
