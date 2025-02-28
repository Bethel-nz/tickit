package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	saltLength    = 16
	keyLength     = 32
	timeIteration = 4
	parallelism   = 4
	memory        = 32 * 1024
)

func HashPassword(password string) (string, string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.Key(salt, []byte(password), timeIteration, parallelism, memory, keyLength, argon2.Argon2id)
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

	verified := argon2.VerifyBytes(saltBytes, []byte(password), hashBytes, timeIteration, parallelism, memory, keyLength, argon2.Argon2id)
	if !verified {
		return false, errors.New("password does not match")
	}

	return true, nil
}
