package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgtype"
)

// Search service errors
var (
	ErrInvalidSearchQuery = errors.New("invalid search query")
)

// SearchResult represents a generic search result
type SearchResult struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type SearchService struct {
	queries *store.Queries
	cache   *redis.Client
}

func NewSearchService(queries *store.Queries, cache *redis.Client) *SearchService {
	return &SearchService{
		queries: queries,
		cache:   cache,
	}
}

// SearchEntities performs a search across entities
func (s *SearchService) SearchEntities(ctx context.Context, userID, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return nil, ErrInvalidSearchQuery
	}

	if limit <= 0 {
		limit = 20
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var queryText pgtype.Text
	if err := queryText.Scan(query); err != nil {
		return nil, fmt.Errorf("invalid query format: %w", err)
	}

	results, err := s.queries.SearchEntities(ctx, store.SearchEntitiesParams{
		OwnerID: userUUID,
		Column2: queryText,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	// Convert to search results
	searchResults := make([]SearchResult, 0, len(results))
	for _, r := range results {
		result := SearchResult{
			Type:        r.EntityType,
			ID:          r.EntityID.String(),
			Name:        r.EntityName,
			Description: r.EntityDescription.String,
			CreatedAt:   r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}

		if r.ParentID.Valid {
			result.ParentID = r.ParentID.String()
		}

		searchResults = append(searchResults, result)
	}

	return searchResults, nil
}
