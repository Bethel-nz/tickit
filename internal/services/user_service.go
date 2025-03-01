package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Bethel-nz/tickit/internal/auth"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/email"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgtype"
)

// Service errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already in use")
	ErrInvalidUserData    = errors.New("invalid user data")
)

// UserProfile represents the user profile data returned to clients
type UserProfile struct {
	ID        pgtype.UUID      `json:"id"`
	Email     string           `json:"email"`
	Name      string           `json:"name,omitempty"`
	Username  string           `json:"username,omitempty"`
	AvatarURL string           `json:"avatar_url,omitempty"`
	Bio       string           `json:"bio,omitempty"`
	CreatedAt pgtype.Timestamp `json:"created_at"`
	UpdatedAt pgtype.Timestamp `json:"updated_at,omitempty"`
}

// UserProfileUpdate contains fields that can be updated
type UserProfileUpdate struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

type UserService struct {
	queries      *store.Queries
	cache        *redis.Client
	emailService *email.EmailService
}

func NewUserService(queries *store.Queries, cache *redis.Client, emailService *email.EmailService) *UserService {
	return &UserService{
		queries:      queries,
		cache:        cache,
		emailService: emailService,
	}
}

// CreateUser creates a new user with the provided information
func (s *UserService) CreateUser(ctx context.Context, params store.CreateUserParams) (*store.CreateUserRow, error) {
	// Hash password
	password := params.Password
	salt, hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password in params with the hashed version
	params.Password = fmt.Sprintf("%s:%s", salt, hashedPassword)

	// Create user in database
	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Send welcome email
	if s.emailService != nil {
		userName := ""
		if params.Name.Valid {
			userName = params.Name.String
		}

		go func() {
			if err := s.emailService.SendWelcomeEmail(params.Email, userName); err != nil {
				log.Printf("Failed to send welcome email: %v", err)
			}
		}()
	}

	// Cache the user
	userJSON, err := json.Marshal(struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name,omitempty"`
		Username  string `json:"username,omitempty"`
		AvatarUrl string `json:"avatar_url,omitempty"`
		Bio       string `json:"bio,omitempty"`
		CreatedAt string `json:"created"`
	}{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name.String,
		Username:  user.Username.String,
		AvatarUrl: user.AvatarUrl.String,
		Bio:       user.Bio.String,
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s", user.ID.String())
	if err := s.cache.Set(ctx, cacheKey, userJSON, time.Hour).Err(); err != nil {
		log.Printf("Failed to cache user: %v", err)
	}

	return &user, nil
}

// DeleteAccount removes a user account and related data
func (s *UserService) DeleteAccount(ctx context.Context, userID string) error {
	var scannedUserId pgtype.UUID
	if err := scannedUserId.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	user, err := s.queries.GetUserByID(ctx, scannedUserId)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	activeProjects, err := s.queries.GetActiveProjectsCount(ctx, scannedUserId)
	if err == nil && activeProjects > 0 {
		log.Printf("Warning: Deleting user with %d active projects", activeProjects)
	}

	if err := s.queries.DeleteUser(ctx, scannedUserId); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to remove user from cache: %v", err)
	}

	log.Printf("User account deleted - ID: %s, Email: %s, Time: %s",
		userID, user.Email, time.Now().Format(time.RFC3339))

	return nil
}

// GetUserProfile retrieves user profile information
func (s *UserService) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	var scannedUserId pgtype.UUID
	if err := scannedUserId.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	cachedUser, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var profile UserProfile
		if err := json.Unmarshal([]byte(cachedUser), &profile); err == nil {
			return &profile, nil
		}
	}

	user, err := s.queries.GetUserByID(ctx, scannedUserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	profile := &UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name.String,
		Username:  user.Username.String,
		AvatarURL: user.AvatarUrl.String,
		Bio:       user.Bio.String,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	profileJSON, err := json.Marshal(profile)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, profileJSON, time.Hour).Err(); err != nil {
			log.Printf("Failed to cache user profile: %v", err)
		}
	}

	return profile, nil
}

// UpdateUserProfile updates user profile information
func (s *UserService) UpdateUserProfile(ctx context.Context, userID string, updates UserProfileUpdate) error {
	var scannedUserId pgtype.UUID
	if err := scannedUserId.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	_, err := s.queries.GetUserByID(ctx, scannedUserId)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := s.queries.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
		ID:        scannedUserId,
		Name:      pgtype.Text{String: updates.Name, Valid: updates.Name != ""},
		Email:     updates.Email,
		Username:  pgtype.Text{String: updates.Username, Valid: updates.Username != ""},
		AvatarUrl: pgtype.Text{String: updates.AvatarURL, Valid: updates.AvatarURL != ""},
		Bio:       pgtype.Text{String: updates.Bio, Valid: updates.Bio != ""},
	}); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate user cache: %v", err)
	}

	return nil
}

// ChangePassword handles password changes
func (s *UserService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	var scannedUserId pgtype.UUID
	if err := scannedUserId.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}

	user, err := s.queries.GetUserByEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	parts := strings.Split(user.Password, ":")
	if len(parts) != 2 {
		return errors.New("invalid password format in database")
	}
	salt, storedHash := parts[0], parts[1]

	valid, err := auth.VerifyPassword(salt, currentPassword, storedHash)
	if err != nil || !valid {
		return ErrInvalidCredentials
	}

	newSalt, newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	newPasswordStore := fmt.Sprintf("%s:%s", newSalt, newHash)

	if err := s.queries.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
		ID:       scannedUserId,
		Password: newPasswordStore,
	}); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ForgotPassword initiates the password reset process
func (s *UserService) ForgotPassword(ctx context.Context, email string) error {

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		log.Printf("Password reset requested for non-existent email: %s", email)
		return nil
	}

	token := auth.GenerateSecureToken(32)

	resetKey := fmt.Sprintf("password_reset:%s", token)
	if err := s.cache.Set(ctx, resetKey, user.ID.String(), 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	resetLink := fmt.Sprintf("https://acme.example.com/reset-password?token=%s", token)

	if s.emailService != nil {
		if err := s.emailService.SendPasswordResetEmail(email, resetLink); err != nil {
			log.Printf("Failed to send password reset email: %v", err)
		}
	} else {
		log.Printf("Password reset link for %s: %s", email, resetLink)
	}

	return nil
}

// ResetPassword completes the password reset process
func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {

	resetKey := fmt.Sprintf("password_reset:%s", token)
	userID, err := s.cache.Get(ctx, resetKey).Result()
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	var scannedUserId pgtype.UUID
	if err := scannedUserId.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID in token: %w", err)
	}

	salt, hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	passwordStore := fmt.Sprintf("%s:%s", salt, hash)

	if err := s.queries.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
		ID:       scannedUserId,
		Password: passwordStore,
	}); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if err := s.cache.Del(ctx, resetKey).Err(); err != nil {
		log.Printf("Failed to delete reset token: %v", err)
	}

	return nil
}
