package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Bethel-nz/tickit/internal/env"
	"github.com/golang-jwt/jwt/v4"
)

var secretKey = env.String("TICKIT_JWT_KEY", "", env.Require).Get()

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for the given user ID
// This is the primary JWT generation function to use
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

// ValidateJWT validates a JWT token and returns the claims if valid
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

// GenerateJWT is an alias for GenerateToken for backward compatibility
// Consider deprecating this in favor of GenerateToken for consistency
func GenerateJWT(userID string) (string, error) {
	return GenerateToken(userID)
}
