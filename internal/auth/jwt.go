package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Bethel-nz/tickit/internal/env"
	"github.com/golang-jwt/jwt/v4"
)

var secretKey = env.String("TICKIT_JWT_KEY", "", env.Require).Get()

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "your-issuer",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, errors.New("invalid JWT claims")
	}
}

// GenerateToken creates a JWT token for the given user ID
func GenerateToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tickit-api",
		},
	}

	// Create token with claims and sign with secret key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// Helper function to get token secret from environment or use a default
func getTokenSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = env.String("JWT_SECRET", "super-secret-key-change-in-production", env.Optional).Get()
	}
	return secret
}
