package middleware

import (
	"net/http"

	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// NewOwnershipMiddleware creates a middleware that ensures the authenticated user owns the project.
// This follows the standard middleware pattern used in the router.
func NewOwnershipMiddleware(queries *store.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			projectID := r.PathValue("id")
			if projectID == "" {
				http.Error(w, "Missing project ID", http.StatusBadRequest)
				return
			}

			userID := r.Context().Value("user_id").(string)
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			var scannedUserId pgtype.UUID
			if err := scannedUserId.Scan(userID); err != nil {
				http.Error(w, "Invalid User ID format", http.StatusBadRequest)
				return
			}

			var scannedProjectId pgtype.UUID
			if err := scannedProjectId.Scan(projectID); err != nil {
				http.Error(w, "Invalid Project ID format", http.StatusBadRequest)
				return
			}

			project, err := queries.GetProjectByID(r.Context(), scannedProjectId)
			if err != nil {
				http.Error(w, "Project not found", http.StatusNotFound)
				return
			}

			if project.OwnerID != scannedUserId {
				http.Error(w, "Forbidden: you are not the owner of this project", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
