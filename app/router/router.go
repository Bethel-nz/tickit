package router

import (
	"net/http"
	"path"
	"strings"
)

// Route represents a single HTTP route with its handler and middleware
type Route struct {
	Method     string
	Path       string
	Handler    http.HandlerFunc
	Middleware []func(http.Handler) http.Handler
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

// Group creates a new router group with the given path prefix
func (rg *RouterGroup) Group(prefix string, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	group := &RouterGroup{
		prefix:     path.Join(rg.prefix, prefix),
		middleware: append([]func(http.Handler) http.Handler{}, middleware...),
		routes:     []Route{},
		groups:     []*RouterGroup{},
	}

	rg.groups = append(rg.groups, group)
	return group
}

// Handle adds a new route to the router group
func (rg *RouterGroup) Handle(method, path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	fullPath := path
	if path != "/" {
		fullPath = path.Join(rg.prefix, path)
	} else if rg.prefix != "" {
		fullPath = rg.prefix
	}

	route := Route{
		Method:     method,
		Path:       fullPath,
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

// buildRoutes recursively builds a flat list of all routes in the group and its subgroups
func (rg *RouterGroup) buildRoutes() []Route {
	result := append([]Route{}, rg.routes...)

	for _, group := range rg.groups {
		// Add group middleware to each route in the group
		groupRoutes := group.buildRoutes()
		for i := range groupRoutes {
			groupRoutes[i].Middleware = append(rg.middleware, groupRoutes[i].Middleware...)
		}
		result = append(result, groupRoutes...)
	}

	return result
}

// Build returns all routes from this group and its subgroups
func (rg *RouterGroup) Build() []Route {
	return rg.buildRoutes()
}
