package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

const (
	saltLength  = 16
	timeCost    = 1
	memory      = 64 * 1024
	threads     = 4
	keyLength   = 32
	tokenExpiry = 24 * time.Hour
)

// JWTSecret is random per process until LoadOrCreateSecret pins it to disk.
var JWTSecret = func() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}()

// LoadOrCreateSecret reads the JWT signing key from path, or saves the
// current random key there (0600) on first run.
func LoadOrCreateSecret(path string) error {
	data, err := os.ReadFile(path)
	if err == nil && len(data) >= 32 {
		JWTSecret = data
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, JWTSecret, 0600)
}

// ponytail: in-memory denylist — survives nothing but a restart (which rotates
// nothing; tokens stay valid). Move to the DB if this ever runs multi-process.
var (
	revoked   = map[string]time.Time{}
	revokedMu sync.Mutex
)

// RevokeToken invalidates a token for its maximum remaining lifetime.
func RevokeToken(tokenString string) {
	revokedMu.Lock()
	defer revokedMu.Unlock()
	now := time.Now()
	for t, exp := range revoked {
		if now.After(exp) {
			delete(revoked, t)
		}
	}
	revoked[tokenString] = now.Add(tokenExpiry)
}

func isRevoked(tokenString string) bool {
	revokedMu.Lock()
	defer revokedMu.Unlock()
	exp, ok := revoked[tokenString]
	return ok && time.Now().Before(exp)
}

// DummyVerify burns the same time as a real check so unknown usernames
// can't be detected by response timing.
func DummyVerify(password string) {
	argon2.IDKey([]byte(password), make([]byte, saltLength), timeCost, memory, threads, keyLength)
}

// HashPassword hashes a password using Argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLength)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, timeCost, threads, b64Salt, b64Hash)
	return encoded, nil
}

// VerifyPassword compares a password with an Argon2id hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}

	var m, t, p uint32
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p)
	if err != nil {
		return false, err
	}
	if p == 0 || p > 255 {
		return false, errors.New("invalid hash parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	if len(decodedHash) == 0 || len(decodedHash) > 1024 {
		return false, errors.New("invalid hash length")
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(decodedHash))) // #nosec G115 -- p and len guarded above

	return subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1, nil
}

// GenerateToken generates a JWT for a user.
func GenerateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(tokenExpiry).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateToken validates a JWT and returns the username.
func ValidateToken(tokenString string) (string, error) {
	if isRevoked(tokenString) {
		return "", errors.New("token revoked")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("invalid token claims")
		}
		return username, nil
	}

	return "", errors.New("invalid token")
}
