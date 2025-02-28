package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	t.Run("Basic route matching", func(t *testing.T) {
		rg := NewRoutes()
		rg.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
			params := GetParams(r)
			w.Write([]byte(params["id"]))
		})

		req := httptest.NewRequest("GET", "/users/123", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status: got %v want %v", status, http.StatusOK)
		}

		expected := "123"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
		}
	})

	t.Run("Static route priority", func(t *testing.T) {
		rg := NewRoutes()
		called := false

		rg.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
			t.Error("Parameterized route should not be called")
		})

		rg.GET("/users/profile", func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/users/profile", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if !called {
			t.Error("Static route was not called")
		}
	})

	t.Run("Middleware execution", func(t *testing.T) {
		rg := NewRoutes()
		var executionOrder []string

		rg.Group("/admin", func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "group")
				h.ServeHTTP(w, r)
			})
		}).GET("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "handler")
			w.WriteHeader(http.StatusOK)
		}, func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "route")
				h.ServeHTTP(w, r)
			})
		})

		req := httptest.NewRequest("GET", "/admin/dashboard", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		expected := []string{"group", "route", "handler"}
		if len(executionOrder) != len(expected) {
			t.Fatalf("Unexpected middleware count: got %d want %d", len(executionOrder), len(expected))
		}

		for i, v := range executionOrder {
			if v != expected[i] {
				t.Errorf("Middleware order mismatch at index %d: got %s want %s", i, v, expected[i])
			}
		}
	})

	t.Run("Route groups", func(t *testing.T) {
		rg := NewRoutes()
		api := rg.Group("/api/v1")

		api.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
			params := GetParams(r)
			w.Write([]byte(params["id"]))
		})

		req := httptest.NewRequest("GET", "/api/v1/users/456", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status: got %v want %v", status, http.StatusOK)
		}

		expected := "456"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
		}
	})

	t.Run("Parameter parsing", func(t *testing.T) {
		tests := []struct {
			path     string
			url      string
			expected map[string]string
		}{
			{
				path: "/:category/:id",
				url:  "/books/123",
				expected: map[string]string{
					"category": "books",
					"id":       "123",
				},
			},
			{
				path: "/files/:path*",
				url:  "/files/images/logo.png",
				expected: map[string]string{
					"path": "images/logo.png",
				},
			},
		}

		for _, tt := range tests {
			rg := NewRoutes()
			rg.GET(tt.path, func(w http.ResponseWriter, r *http.Request) {
				params := GetParams(r)
				for k, v := range tt.expected {
					if params[k] != v {
						t.Errorf("Param %s: got %s want %s", k, params[k], v)
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()
			ServeMux(rg).ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status for %s: got %v want %v", tt.path, status, http.StatusOK)
			}
		}
	})

	t.Run("404 handling", func(t *testing.T) {
		rg := NewRoutes()
		rg.GET("/existing", func(w http.ResponseWriter, r *http.Request) {})

		req := httptest.NewRequest("GET", "/not-found", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Method validation", func(t *testing.T) {
		rg := NewRoutes()
		rg.POST("/users", func(w http.ResponseWriter, r *http.Request) {})

		req := httptest.NewRequest("GET", "/users", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status for method mismatch: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("SortRoutes", func(t *testing.T) {
		rg := NewRoutes()
		rg.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {})    // 1 literal
		rg.GET("/static/about", func(w http.ResponseWriter, r *http.Request) {}) // 2 literals
		rg.GET("/:catchall", func(w http.ResponseWriter, r *http.Request) {})    // 0 literals

		routes := rg.Build()
		expectedOrder := []string{
			"/static/about",
			"/users/:id",
			"/:catchall",
		}

		for i, route := range routes {
			if route.Path != expectedOrder[i] {
				t.Errorf("Route order mismatch at index %d: got %s want %s", i, route.Path, expectedOrder[i])
			}
		}
	})
}

func TestPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		matches  bool
		params   map[string]string
		literals int
	}{
		{
			pattern:  "/users/:id",
			path:     "/users/123",
			matches:  true,
			params:   map[string]string{"id": "123"},
			literals: 1,
		},
		{
			pattern:  "/posts/:slug/comments/:cid",
			path:     "/posts/hello-world/comments/456",
			matches:  true,
			params:   map[string]string{"slug": "hello-world", "cid": "456"},
			literals: 2,
		},
		{
			pattern:  "/static/about",
			path:     "/static/about",
			matches:  true,
			params:   map[string]string{},
			literals: 2,
		},
		{
			pattern:  "/users/:id",
			path:     "/users/123/profile",
			matches:  false,
			params:   nil,
			literals: 1,
		},
	}

	for _, tt := range tests {
		p := NewPattern(tt.pattern)
		if got := p.LiteralCount(); got != tt.literals {
			t.Errorf("LiteralCount(%q) = %d, want %d", tt.pattern, got, tt.literals)
		}

		match, params := p.Match(tt.path)
		if match != tt.matches {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, match, tt.matches)
		}

		if tt.matches {
			for k, v := range tt.params {
				if params[k] != v {
					t.Errorf("Param %q = %q, want %q", k, params[k], v)
				}
			}
		}
	}
}
