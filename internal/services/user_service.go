package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Bethel-nz/tickit/internal/auth"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/go-redis/redis/v8"
)

type UserService struct {
	queries *store.Queries
	cache   *redis.Client
}

func NewUserService(queries *store.Queries, cache *redis.Client) *UserService {
	return &UserService{
		queries: queries,
		cache:   cache,
	}
}

func (s *UserService) CreateUser(ctx context.Context, email, password string) (*store.CreateUserRow, error) {
	salt, hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, store.CreateUserParams{
		Email:    email,
		Password: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userJSON, err := json.Marshal(struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Created string `json:"created"`
	}{
		ID:      user.ID.String(),
		Email:   user.Email,
		Created: user.Created.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	// Cache for 1 hour
	cacheKey := fmt.Sprintf("user:%s", user.ID.String())
	if err := s.cache.Set(ctx, cacheKey, userJSON, time.Hour).Err(); err != nil {
		log.Printf("Failed to cache user: %v", err)
	}

	return user, nil
}
