package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	saltLength    = 16
	keyLength     = 32
	timeIteration = 4
	memory        = 32 * 1024
	parallelism   = 4
)

func HashPassword(password string) (string, string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, uint32(timeIteration), uint32(memory), uint8(parallelism), uint32(keyLength))

	hashB64 := base64.RawStdEncoding.EncodeToString(hash)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)

	return saltB64, hashB64, nil
}

func VerifyPassword(salt, password, hash string) (bool, error) {
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false, fmt.Errorf("invalid salt: %w", err)
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false, fmt.Errorf("invalid hash: %w", err)
	}

	computedHash := argon2.IDKey([]byte(password), saltBytes, uint32(timeIteration), uint32(memory), uint8(parallelism), uint32(keyLength))

	if len(computedHash) != len(hashBytes) {
		return false, errors.New("password does not match")
	}
	for i := range computedHash {
		if computedHash[i] != hashBytes[i] {
			return false, errors.New("password does not match")
		}
	}

	return true, nil
}

// GenerateSecureToken creates a cryptographically secure random token
func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
