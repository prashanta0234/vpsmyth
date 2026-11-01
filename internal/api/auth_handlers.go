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

type ipFailures struct {
	count int
	first time.Time
}

var (
	loginAttempts = make(map[string]*ipFailures)
	loginMutex    sync.Mutex
	maxAttempts   = 5
	maxIPFailures = 10
	lockoutTime   = 15 * time.Minute
)

// ipBlocked reports whether ip has hit maxIPFailures within lockoutTime.
func ipBlocked(ip string) bool {
	loginMutex.Lock()
	defer loginMutex.Unlock()
	f := loginAttempts[ip]
	if f != nil && time.Since(f.first) > lockoutTime {
		delete(loginAttempts, ip)
		return false
	}
	return f != nil && f.count >= maxIPFailures
}

func recordLoginFailure(ip string) {
	loginMutex.Lock()
	defer loginMutex.Unlock()
	if len(loginAttempts) > 10000 {
		for k, f := range loginAttempts {
			if time.Since(f.first) > lockoutTime {
				delete(loginAttempts, k)
			}
		}
	}
	if f := loginAttempts[ip]; f != nil {
		f.count++
	} else {
		loginAttempts[ip] = &ipFailures{count: 1, first: time.Now()}
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// RemoteAddr is always Nginx's loopback; rate-limit on the real client IP.
	ip := realIP(r)
	if ipBlocked(ip) {
		http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
		return
	}

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
		auth.DummyVerify(req.Password)
		recordLoginFailure(ip)
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
		recordLoginFailure(ip)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Reset failed attempts on success
	db.ResetFailedAttempts(req.Username)
	loginMutex.Lock()
	delete(loginAttempts, ip)
	loginMutex.Unlock()

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
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("vpsmyth_token"); err == nil && cookie.Value != "" {
		auth.RevokeToken(cookie.Value)
	}
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
