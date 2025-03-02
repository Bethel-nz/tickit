package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgtype"
)

// Team service errors
var (
	ErrTeamNotFound      = errors.New("team not found")
	ErrInvalidTeamData   = errors.New("invalid team data")
	ErrNotTeamMember     = errors.New("user is not a team member")
	ErrInsufficientRoles = errors.New("insufficient permissions for this operation")
	ErrUnauthorized      = errors.New("unauthorized action")
	ErrNotMember         = errors.New("user is not a team member")
)

// TeamMemberInfo represents a team member with role information
type TeamMemberInfo struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role"`
}

// TeamInfo represents team information returned to clients
type TeamInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	Role        string `json:"role,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type TeamService struct {
	queries *store.Queries
	cache   *redis.Client
}

func NewTeamService(queries *store.Queries, cache *redis.Client) *TeamService {
	return &TeamService{
		queries: queries,
		cache:   cache,
	}
}

// CreateTeam creates a new team with the provided information
func (s *TeamService) CreateTeam(ctx context.Context, params store.CreateTeamParams, ownerID string) (*store.Team, error) {

	if params.Name == "" {
		return nil, fmt.Errorf("%w: team name is required", ErrInvalidTeamData)
	}

	if len(params.Name) > 100 {
		return nil, fmt.Errorf("%w: team name cannot exceed 100 characters", ErrInvalidTeamData)
	}

	team, err := s.queries.CreateTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	var ownerUUID pgtype.UUID
	if err := ownerUUID.Scan(ownerID); err != nil {
		return nil, fmt.Errorf("invalid owner ID: %w", err)
	}

	err = s.queries.AddUserToTeam(ctx, store.AddUserToTeamParams{
		TeamID: team.ID,
		UserID: ownerUUID,
		Role:   pgtype.Text{String: "owner", Valid: true},
	})
	if err != nil {
		if delErr := s.queries.DeleteTeam(ctx, team.ID); delErr != nil {
			log.Printf("Failed to delete team after adding owner failed: %v", delErr)
		}
		return nil, fmt.Errorf("failed to add owner to team: %w", err)
	}

	s.cacheTeam(ctx, &team)

	return &team, nil
}

func (s *TeamService) GetTeamByID(ctx context.Context, teamID string) (*store.Team, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return nil, fmt.Errorf("invalid team ID: %w", err)
	}

	cacheKey := fmt.Sprintf("team:%s", teamID)
	cachedTeam, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var team store.Team
		if err := json.Unmarshal([]byte(cachedTeam), &team); err == nil {
			return &team, nil
		}
	}

	team, err := s.queries.GetTeamByID(ctx, teamUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	s.cacheTeam(ctx, &team)

	return &team, nil
}

// UpdateTeam updates team information
func (s *TeamService) UpdateTeam(ctx context.Context, params store.UpdateTeamParams, userID string) error {

	if params.Name != "" && len(params.Name) > 100 {
		return fmt.Errorf("%w: team name cannot exceed 100 characters", ErrInvalidTeamData)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	role, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: params.ID,
		UserID: userUUID,
	})
	if err != nil {
		return fmt.Errorf("%w: user is not a member of this team", ErrNotTeamMember)
	}

	if role.String != "owner" && role.String != "admin" {
		return ErrInsufficientRoles
	}

	if err := s.queries.UpdateTeam(ctx, params); err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	cacheKey := fmt.Sprintf("team:%s", params.ID.String())
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate team cache: %v", err)
	}

	return nil
}

// DeleteTeam deletes a team
func (s *TeamService) DeleteTeam(ctx context.Context, teamID, userID string) error {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	role, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: userUUID,
	})
	if err != nil {
		return fmt.Errorf("%w: user is not a member of this team", ErrNotTeamMember)
	}

	if role.String != "owner" {
		return ErrInsufficientRoles
	}

	if err := s.queries.DeleteTeam(ctx, teamUUID); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	cacheKey := fmt.Sprintf("team:%s", teamID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate team cache: %v", err)
	}

	return nil
}

// AddUserToTeam adds a user to a team
func (s *TeamService) AddUserToTeam(ctx context.Context, teamID, userIDToAdd, adderUserID, role string) error {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	var userToAddUUID pgtype.UUID
	if err := userToAddUUID.Scan(userIDToAdd); err != nil {
		return fmt.Errorf("invalid user ID to add: %w", err)
	}

	var adderUserUUID pgtype.UUID
	if err := adderUserUUID.Scan(adderUserID); err != nil {
		return fmt.Errorf("invalid adder user ID: %w", err)
	}

	adderRole, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: adderUserUUID,
	})
	if err != nil {
		return fmt.Errorf("%w: adder is not a member of this team", ErrNotTeamMember)
	}

	if adderRole.String != "owner" && adderRole.String != "admin" {
		return ErrInsufficientRoles
	}

	validRoles := map[string]bool{
		"admin":  true,
		"editor": true,
		"viewer": true,
	}

	if !validRoles[role] {
		return fmt.Errorf("%w: invalid role '%s'", ErrInvalidTeamData, role)
	}

	isMember, err := s.queries.CheckTeamMembership(ctx, store.CheckTeamMembershipParams{
		TeamID: teamUUID,
		UserID: userToAddUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to check team membership: %w", err)
	}

	if isMember {
		return s.queries.UpdateTeamMemberRole(ctx, store.UpdateTeamMemberRoleParams{
			TeamID: teamUUID,
			UserID: userToAddUUID,
			Role:   pgtype.Text{String: role, Valid: true},
		})
	}

	err = s.queries.AddUserToTeam(ctx, store.AddUserToTeamParams{
		TeamID: teamUUID,
		UserID: userToAddUUID,
		Role:   pgtype.Text{String: role, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to add user to team: %w", err)
	}

	return nil
}

// RemoveUserFromTeam removes a user from a team
func (s *TeamService) RemoveUserFromTeam(ctx context.Context, teamID, userIDToRemove, removerUserID string) error {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	var userToRemoveUUID pgtype.UUID
	if err := userToRemoveUUID.Scan(userIDToRemove); err != nil {
		return fmt.Errorf("invalid user ID to remove: %w", err)
	}

	var removerUserUUID pgtype.UUID
	if err := removerUserUUID.Scan(removerUserID); err != nil {
		return fmt.Errorf("invalid remover user ID: %w", err)
	}

	isMember, err := s.queries.CheckTeamMembership(ctx, store.CheckTeamMembershipParams{
		TeamID: teamUUID,
		UserID: userToRemoveUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to check team membership: %w", err)
	}

	if !isMember {
		return fmt.Errorf("%w: user is not a member of this team", ErrNotTeamMember)
	}

	if userIDToRemove != removerUserID {
		removerRole, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
			TeamID: teamUUID,
			UserID: removerUserUUID,
		})
		if err != nil {
			return fmt.Errorf("%w: remover is not a member of this team", ErrNotTeamMember)
		}

		userToRemoveRole, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
			TeamID: teamUUID,
			UserID: userToRemoveUUID,
		})
		if err != nil {
			return fmt.Errorf("failed to get user role: %w", err)
		}

		if userToRemoveRole.String == "owner" && removerRole.String != "owner" {
			return ErrInsufficientRoles
		}

		if removerRole.String != "owner" && removerRole.String != "admin" {
			return ErrInsufficientRoles
		}
	}

	err = s.queries.RemoveUserFromTeam(ctx, store.RemoveUserFromTeamParams{
		TeamID: teamUUID,
		UserID: userToRemoveUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove user from team: %w", err)
	}

	return nil
}

func (s *TeamService) UpdateTeamMemberRole(ctx context.Context, teamID, userIDToUpdate, updaterUserID, newRole string) error {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	var userToUpdateUUID pgtype.UUID
	if err := userToUpdateUUID.Scan(userIDToUpdate); err != nil {
		return fmt.Errorf("invalid user ID to update: %w", err)
	}

	var updaterUserUUID pgtype.UUID
	if err := updaterUserUUID.Scan(updaterUserID); err != nil {
		return fmt.Errorf("invalid updater user ID: %w", err)
	}

	validRoles := map[string]bool{
		"admin":  true,
		"editor": true,
		"viewer": true,
	}

	if newRole == "owner" || !validRoles[newRole] {
		return fmt.Errorf("%w: invalid role '%s'", ErrInvalidTeamData, newRole)
	}

	isMember, err := s.queries.CheckTeamMembership(ctx, store.CheckTeamMembershipParams{
		TeamID: teamUUID,
		UserID: userToUpdateUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to check team membership: %w", err)
	}

	if !isMember {
		return fmt.Errorf("%w: user is not a member of this team", ErrNotTeamMember)
	}

	updaterRole, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: updaterUserUUID,
	})
	if err != nil {
		return fmt.Errorf("%w: updater is not a member of this team", ErrNotTeamMember)
	}

	currentRole, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: userToUpdateUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to get user role: %w", err)
	}

	if currentRole.String == "owner" && updaterRole.String != "owner" {
		return ErrInsufficientRoles
	}

	// Only owner or admin can update roles
	if updaterRole.String != "owner" && updaterRole.String != "admin" {
		return ErrInsufficientRoles
	}

	// Update role
	err = s.queries.UpdateTeamMemberRole(ctx, store.UpdateTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: userToUpdateUUID,
		Role:   pgtype.Text{String: newRole, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update team member role: %w", err)
	}

	return nil
}

// GetTeamMembers retrieves all members of a team
func (s *TeamService) GetTeamMembers(ctx context.Context, teamID, requestorID string) ([]TeamMemberInfo, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return nil, fmt.Errorf("invalid team ID: %w", err)
	}

	var requestorUUID pgtype.UUID
	if err := requestorUUID.Scan(requestorID); err != nil {
		return nil, fmt.Errorf("invalid requestor ID: %w", err)
	}

	// Check if requestor is a team member
	isMember, err := s.queries.CheckTeamMembership(ctx, store.CheckTeamMembershipParams{
		TeamID: teamUUID,
		UserID: requestorUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check team membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("%w: requestor is not a member of this team", ErrNotTeamMember)
	}

	// Try to get from cache
	cacheKey := fmt.Sprintf("team:%s:members", teamID)
	cachedMembers, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		
		var members []TeamMemberInfo
		if err := json.Unmarshal([]byte(cachedMembers), &members); err == nil {
			return members, nil
		}
	}

	dbMembers, err := s.queries.GetTeamMembers(ctx, teamUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	members := make([]TeamMemberInfo, len(dbMembers))
	for i, m := range dbMembers {
		members[i] = TeamMemberInfo{
			UserID:    m.ID.String(),
			Email:     m.Email,
			Name:      m.Name.String,
			Username:  m.Username.String,
			AvatarURL: m.AvatarUrl.String,
			Role:      m.Role.String,
		}
	}

	membersJSON, err := json.Marshal(members)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, membersJSON, 5*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache team members: %v", err)
		}
	}

	return members, nil
}

// GetUserTeams retrieves all teams a user is a member of
func (s *TeamService) GetUserTeams(ctx context.Context, userID string) ([]TeamInfo, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s:teams", userID)
	cachedTeams, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var teams []TeamInfo
		if err := json.Unmarshal([]byte(cachedTeams), &teams); err == nil {
			return teams, nil
		}
	}

	dbTeams, err := s.queries.GetUserTeams(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	teams := make([]TeamInfo, len(dbTeams))
	for i, t := range dbTeams {
		teams[i] = TeamInfo{
			ID:          t.ID.String(),
			Name:        t.Name,
			Description: t.Description.String,
			AvatarURL:   t.AvatarUrl.String,
			Role:        t.Role.String,
		}
	}

	// Cache the result
	teamsJSON, err := json.Marshal(teams)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, teamsJSON, 10*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache user teams: %v", err)
		}
	}

	return teams, nil
}

// CheckTeamMembership checks if a user is a member of a team
func (s *TeamService) CheckTeamMembership(ctx context.Context, teamID, userID string) (bool, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return false, fmt.Errorf("invalid team ID: %w", err)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.queries.CheckTeamMembership(ctx, store.CheckTeamMembershipParams{
		TeamID: teamUUID,
		UserID: userUUID,
	})
}

// GetTeamMemberRole gets a user's role in a team
func (s *TeamService) GetTeamMemberRole(ctx context.Context, teamID, userID string) (string, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return "", fmt.Errorf("invalid team ID: %w", err)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	role, err := s.queries.GetTeamMemberRole(ctx, store.GetTeamMemberRoleParams{
		TeamID: teamUUID,
		UserID: userUUID,
	})
	if err != nil {
		return "", fmt.Errorf("%w: user is not a member of this team", ErrNotTeamMember)
	}

	return role.String, nil
}

// Helper method to cache a team
func (s *TeamService) cacheTeam(_ context.Context, team *store.Team) {
	if s.cache == nil {
		return
	}

	teamJSON, err := json.Marshal(team)
	if err != nil {
		log.Printf("Failed to marshal team: %v", err)
		return
	}

	cacheKey := fmt.Sprintf("team:%s", team.ID.String())
	if err := s.cache.Set(context.Background(), cacheKey, teamJSON, time.Hour).Err(); err != nil {
		log.Printf("Failed to cache team: %v", err)
	}
}

// AddMember adds a new member to a team with the specified role
func (s *TeamService) AddMember(ctx context.Context, teamID, userToAddID, role, requestingUserID string) error {
	
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	if _, err := s.queries.GetTeamByID(ctx, teamUUID); err != nil {
		return ErrTeamNotFound
	}

	var requestingUserUUID pgtype.UUID
	if err := requestingUserUUID.Scan(requestingUserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	isAdmin, err := s.isTeamAdmin(ctx, teamID, requestingUserID)
	if err != nil {
		return err
	}

	if !isAdmin {
		return ErrUnauthorized
	}

	var userToAddUUID pgtype.UUID
	if err := userToAddUUID.Scan(userToAddID); err != nil {
		return fmt.Errorf("invalid user ID for new member: %w", err)
	}

	var roleText pgtype.Text
	if err := roleText.Scan(role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	isMember, err := s.CheckTeamMembership(ctx, teamID, userToAddID)
	if err != nil {
		return fmt.Errorf("failed to check team membership: %w", err)
	}

	if isMember {
		err = s.queries.UpdateTeamMemberRole(ctx, store.UpdateTeamMemberRoleParams{
			TeamID: teamUUID,
			UserID: userToAddUUID,
			Role:   roleText,
		})
	} else {
		err = s.queries.AddUserToTeam(ctx, store.AddUserToTeamParams{
			TeamID: teamUUID,
			UserID: userToAddUUID,
			Role:   roleText,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}

	return nil
}

// RemoveMember removes a user from a team
func (s *TeamService) RemoveMember(ctx context.Context, teamID, memberID, requestingUserID string) error {

	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return fmt.Errorf("invalid team ID: %w", err)
	}

	if _, err := s.queries.GetTeamByID(ctx, teamUUID); err != nil {
		return ErrTeamNotFound
	}

	var requestingUserUUID pgtype.UUID
	if err := requestingUserUUID.Scan(requestingUserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	isAdmin, err := s.isTeamAdmin(ctx, teamID, requestingUserID)
	if err != nil {
		return err
	}

	isSelf := requestingUserID == memberID

	if !isAdmin && !isSelf {
		return ErrUnauthorized
	}

	if isAdmin && memberID != requestingUserID {
		isLastAdmin, err := s.isLastAdmin(ctx, teamID, memberID)
		if err != nil {
			return fmt.Errorf("failed to check admin status: %w", err)
		}
		if isLastAdmin {
			return fmt.Errorf("cannot remove the last admin from the team")
		}
	}

	var memberUUID pgtype.UUID
	if err := memberUUID.Scan(memberID); err != nil {
		return fmt.Errorf("invalid member ID: %w", err)
	}

	if err := s.queries.RemoveUserFromTeam(ctx, store.RemoveUserFromTeamParams{
		TeamID: teamUUID,
		UserID: memberUUID,
	}); err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}

	return nil
}

// Helper method to check if a user is the last admin of a team
func (s *TeamService) isLastAdmin(ctx context.Context, teamID, userID string) (bool, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return false, fmt.Errorf("invalid team ID: %w", err)
	}

	admins, err := s.queries.GetTeamAdmins(ctx, teamUUID)
	if err != nil {
		return false, fmt.Errorf("failed to get team admins: %w", err)
	}

	if len(admins) <= 1 {
		if len(admins) == 1 {
			admin := admins[0]
			if admin.UserID.String() == userID {
				return true, nil
			}
		}
		return false, nil
	}

	return false, nil
}

func (s *TeamService) isTeamAdmin(ctx context.Context, teamID, userID string) (bool, error) {
	isMember, role, err := s.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		return false, err
	}

	if !isMember {
		return false, ErrNotMember
	}

	return role == "admin", nil
}

func (s *TeamService) GetMemberRole(ctx context.Context, teamID, userID string) (bool, string, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return false, "", fmt.Errorf("invalid team ID: %w", err)
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return false, "", fmt.Errorf("invalid user ID: %w", err)
	}

	member, err := s.queries.GetTeamMember(ctx, store.GetTeamMemberParams{
		TeamID: teamUUID,
		UserID: userUUID,
	})

	if err != nil {
		return false, "", nil
	}

	return true, member.Role.String, nil
}
