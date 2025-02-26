package middleware

import (
	"net/http"

	"github.com/Bethel-nz/tickit/internal/database/store"
)

// OwnershipMiddleware ensures that the authenticated user owns the project.
// It expects the project id to be provided as a URL query parameter "project_id".
func OwnershipMiddleware(queries *store.Queries, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(string)

		if !ok || userID == "" {
			http.Error(w, "Unauthorized: user not found in context", http.StatusUnauthorized)
			return
		}

		projectID := r.URL.Query().Get("project_id")
		if projectID == "" {
			http.Error(w, "Bad Request: missing project_id", http.StatusBadRequest)
			return
		}

		project, err := queries.GetProject(r.Context(), projectID)
		if err != nil {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}

		if project.OwnerID != userID {
			http.Error(w, "Forbidden: you are not the owner of this project", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
