package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/services"
)

// searchService is retrieved from the application's dependency container
var searchService *services.SearchService

// SetSearchService sets the search service for handlers
func SetSearchService(service *services.SearchService) {
	searchService = service
}

// SearchEntities performs a search across multiple entity types
func SearchEntities(c *router.Context) {
	if searchService == nil {
		c.Status(http.StatusInternalServerError, "Search service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	query := c.Query("q")
	if query == "" {
		c.Status(http.StatusBadRequest, "Search query is required")
		return
	}

	limitStr := c.Query("limit")
	limit := 20 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	results, err := searchService.SearchEntities(c.Request.Context(), userID, query, limit)
	if err != nil {
		if errors.Is(err, services.ErrInvalidSearchQuery) {
			c.Status(http.StatusBadRequest, "Invalid search query")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to perform search")
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   query,
	})
}
