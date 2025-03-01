package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	t.Run("Basic route matching", func(t *testing.T) {
		rg := NewRouter()
		rg.GET("/users/{id}", func(c *Context) {
			c.Write([]byte(c.Param("id")))
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
		rg := NewRouter()
		called := false

		rg.GET("/users/{id}", func(c *Context) {
			t.Error("Parameterized route should not be called")
		})

		rg.GET("/users/profile", func(c *Context) {
			called = true
			c.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/users/profile", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if !called {
			t.Error("Static route was not called")
		}
	})

	t.Run("Middleware execution", func(t *testing.T) {
		rg := NewRouter()
		var executionOrder []string

		rg.Group("/admin", func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "group")
				h.ServeHTTP(w, r)
			})
		}).GET("/dashboard", func(c *Context) {
			executionOrder = append(executionOrder, "handler")
			c.WriteHeader(http.StatusOK)
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
		rg := NewRouter()
		api := rg.Group("/api/v1")

		api.GET("/users/{id}", func(c *Context) {
			c.Write([]byte(c.Param("id")))
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
				path: "/{category}/{id}",
				url:  "/books/123",
				expected: map[string]string{
					"category": "books",
					"id":       "123",
				},
			},
			{
				path: "/files/{path}",
				url:  "/files/images/logo.png",
				expected: map[string]string{
					"path": "images/logo.png",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.path, func(t *testing.T) {
				rg := NewRouter()
				rg.GET(tt.path, func(c *Context) {
					for k, v := range tt.expected {
						if c.Param(k) != v {
							t.Errorf("Param %s: got %s want %s", k, c.Param(k), v)
						}
					}
					c.WriteHeader(http.StatusOK)
				})

				req := httptest.NewRequest("GET", tt.url, nil)
				rr := httptest.NewRecorder()
				ServeMux(rg).ServeHTTP(rr, req)

				if status := rr.Code; status != http.StatusOK {
					t.Errorf("handler returned wrong status for %s: got %v want %v",
						tt.path, status, http.StatusOK)
				}
			})
		}
	})

	t.Run("404 handling", func(t *testing.T) {
		rg := NewRouter()
		rg.GET("/existing", func(c *Context) {})

		req := httptest.NewRequest("GET", "/not-found", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Method validation", func(t *testing.T) {
		rg := NewRouter()
		rg.POST("/users", func(c *Context) {})

		req := httptest.NewRequest("GET", "/users", nil)
		rr := httptest.NewRecorder()
		ServeMux(rg).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status for method mismatch: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("SortRoutes", func(t *testing.T) {
		rg := NewRouter()
		rg.GET("/users/{id}", func(c *Context) {})   // 1 literal
		rg.GET("/static/about", func(c *Context) {}) // 2 literals
		rg.GET("/{catchall}", func(c *Context) {})   // 0 literals

		routes := rg.Build()
		expectedOrder := []string{
			"/static/about",
			"/users/{id}",
			"/{catchall}",
		}

		for i, route := range routes {
			if route.Path != expectedOrder[i] {
				t.Errorf("Route order mismatch at index %d: got %s want %s", i, route.Path, expectedOrder[i])
			}
		}
	})

	t.Run("Catchall parameter parsing", func(t *testing.T) {
		rg := NewRouter()
		rg.GET("/drive/files/{path}", func(c *Context) {
			c.Write([]byte(c.Param("path")))
		})

		tests := []struct {
			url      string
			expected string
		}{
			{
				url:      "/drive/files/image.jpg",
				expected: "image.jpg",
			},
			{
				url:      "/drive/files/docs/report.pdf",
				expected: "docs/report.pdf",
			},
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()
			ServeMux(rg).ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status for %s: got %v want %v",
					tt.url, status, http.StatusOK)
			}

			if rr.Body.String() != tt.expected {
				t.Errorf("handler returned unexpected body for %s: got %v want %v",
					tt.url, rr.Body.String(), tt.expected)
			}
		}
	})

	t.Run("Greedy parameter parsing", func(t *testing.T) {
		rg := NewRouter()
		rg.GET("/api/{all}", func(c *Context) {
			c.Write([]byte(c.Param("all")))
		})

		tests := []struct {
			url      string
			expected string
		}{
			{
				url:      "/api/users",
				expected: "users",
			},
			{
				url:      "/api/users/123/profile",
				expected: "users/123/profile",
			},
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", tt.url, nil)
			rr := httptest.NewRecorder()
			ServeMux(rg).ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status for %s: got %v want %v",
					tt.url, status, http.StatusOK)
			}

			if rr.Body.String() != tt.expected {
				t.Errorf("handler returned unexpected body for %s: got %v want %v",
					tt.url, rr.Body.String(), tt.expected)
			}
		}
	})
}
