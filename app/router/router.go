package router

import (
	"context"
	"net/http"
	"sort"
	"strings"
)

// Pattern represents a route pattern with segments
type Pattern struct {
	segments []string // Each segment is either a literal or a parameter (e.g., ":id")
}

// NewPattern creates a new Pattern from a path string
func NewPattern(path string) *Pattern {
	segments := strings.Split(path, "/")
	var cleaned []string
	for _, seg := range segments {
		if seg != "" { // Remove empty segments from leading/trailing slashes
			cleaned = append(cleaned, seg)
		}
	}
	return &Pattern{segments: cleaned}
}

// LiteralCount returns the number of literal (non-parameter) segments
func (p *Pattern) LiteralCount() int {
	count := 0
	for _, seg := range p.segments {
		if !strings.HasPrefix(seg, ":") {
			count++
		}
	}
	return count
}

// Route represents a single HTTP route with its handler and middleware
type Route struct {
	Method     string
	Path       string // Original path for reference
	Pattern    *Pattern
	Handler    http.HandlerFunc
	Middleware []func(http.Handler) http.Handler
}

// Match checks if the route matches the given method and path
func (p *Pattern) Match(reqPath string) (bool, map[string]string) {
	reqSegments := strings.Split(reqPath, "/")
	var cleanedReq []string
	for _, seg := range reqSegments {
		if seg != "" {
			cleanedReq = append(cleanedReq, seg)
		}
	}

	params := make(map[string]string)
	pi, ri := 0, 0

	for pi < len(p.segments) && ri < len(cleanedReq) {
		seg := p.segments[pi]

		if strings.HasPrefix(seg, ":") && strings.HasSuffix(seg, "*") {
			// Wildcard parameter (e.g., ":path*")
			paramName := seg[1 : len(seg)-1]
			params[paramName] = strings.Join(cleanedReq[ri:], "/")
			return true, params
		} else if strings.HasPrefix(seg, ":") {
			// Regular parameter
			paramName := seg[1:]
			params[paramName] = cleanedReq[ri]
			pi++
			ri++
		} else if seg == cleanedReq[ri] {
			// Exact match
			pi++
			ri++
		} else {
			// Mismatch
			return false, nil
		}
	}

	// Check if all segments matched
	return pi == len(p.segments) && ri == len(cleanedReq), params
}

// RouterGroup represents a group of routes with a common path prefix and middleware
type RouterGroup struct {
	prefix     string
	middleware []func(http.Handler) http.Handler
	routes     []Route
	groups     []*RouterGroup
}

// NewRoutes creates a new root router group
func NewRoutes() *RouterGroup {
	return &RouterGroup{
		prefix:     "",
		middleware: []func(http.Handler) http.Handler{},
		routes:     []Route{},
		groups:     []*RouterGroup{},
	}
}

// In the RouterGroup struct, change the Group method:
func (rg *RouterGroup) Group(prefix string, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// Use proper path joining
	fullPrefix := rg.prefix + prefix
	if rg.prefix != "" && strings.HasSuffix(rg.prefix, "/") {
		fullPrefix = rg.prefix + strings.TrimPrefix(prefix, "/")
	}

	group := &RouterGroup{
		prefix:     fullPrefix, // Replace path.Join with manual handling
		middleware: append([]func(http.Handler) http.Handler{}, middleware...),
		routes:     []Route{},
		groups:     []*RouterGroup{},
	}

	rg.groups = append(rg.groups, group)
	return group
}

// And in the Handle method:
func (rg *RouterGroup) Handle(method, path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Manual path concatenation instead of path.Join
	fullPath := rg.prefix + path
	if rg.prefix != "" && strings.HasSuffix(rg.prefix, "/") {
		fullPath = rg.prefix + strings.TrimPrefix(path, "/")
	}

	route := Route{
		Method:     method,
		Path:       fullPath,
		Pattern:    NewPattern(fullPath),
		Handler:    handler,
		Middleware: middleware,
	}

	rg.routes = append(rg.routes, route)
	return rg
}

// GET is a shorthand for Handle("GET", path, handler, middleware...)
func (rg *RouterGroup) GET(path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("GET", path, handler, middleware...)
}

// POST is a shorthand for Handle("POST", path, handler, middleware...)
func (rg *RouterGroup) POST(path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("POST", path, handler, middleware...)
}

// PUT is a shorthand for Handle("PUT", path, handler, middleware...)
func (rg *RouterGroup) PUT(path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("PUT", path, handler, middleware...)
}

// DELETE is a shorthand for Handle("DELETE", path, handler, middleware...)
func (rg *RouterGroup) DELETE(path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("DELETE", path, handler, middleware...)
}

// PATCH is a shorthand for Handle("PATCH", path, handler, middleware...)
func (rg *RouterGroup) PATCH(path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("PATCH", path, handler, middleware...)
}

func (r *Route) Match(method, path string) (bool, map[string]string) {
	if r.Method != method {
		return false, nil
	}
	return r.Pattern.Match(path)
}

// buildRoutes recursively builds a flat list of all routes in the group and its subgroups
func (rg *RouterGroup) buildRoutes(parentMiddleware []func(http.Handler) http.Handler) []Route {
	// Combine parent middleware with current group's middleware
	currentMiddleware := append(parentMiddleware, rg.middleware...)

	result := make([]Route, 0, len(rg.routes))

	// Apply current middleware to this group's routes
	for _, route := range rg.routes {
		newRoute := route
		newRoute.Middleware = append(currentMiddleware, newRoute.Middleware...)
		result = append(result, newRoute)
	}

	// Process subgroups
	for _, group := range rg.groups {
		groupRoutes := group.buildRoutes(currentMiddleware)
		result = append(result, groupRoutes...)
	}

	return result
}

// Build returns all routes from this group and its subgroups, sorted by specificity
func (rg *RouterGroup) Build() []Route {
	routes := rg.buildRoutes(nil)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Pattern.LiteralCount() > routes[j].Pattern.LiteralCount()
	})
	return routes
}

// ServeMux creates an http.ServeMux from the router group's routes
func ServeMux(rg *RouterGroup) *http.ServeMux {
	routes := rg.Build()
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, route := range routes {
			if match, params := route.Match(r.Method, r.URL.Path); match {
				ctx := context.WithValue(r.Context(), "routeParams", params)
				r = r.WithContext(ctx)
				handler := http.Handler(route.Handler)
				for i := len(route.Middleware) - 1; i >= 0; i-- {
					handler = route.Middleware[i](handler)
				}
				handler.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	return mux
}

// GetParams retrieves the route parameters from the request context
func GetParams(r *http.Request) map[string]string {
	params, _ := r.Context().Value("routeParams").(map[string]string)
	return params
}
