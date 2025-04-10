package router

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
)

// Context wraps http.ResponseWriter and *http.Request with additional utilities
type Context struct {
	http.ResponseWriter
	Request *http.Request
	Params  map[string]string
	path    string // store the matched path pattern
}

// Param returns a route parameter by key
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// Query returns a query parameter by key
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// JSON sends a JSON response with the specified status code and data
func (c *Context) JSON(status int, v interface{}) {
	c.Header().Set("Content-Type", "application/json")
	c.WriteHeader(status)
	if err := json.NewEncoder(c).Encode(v); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		if status < 400 {
			c.Write([]byte(`{"error": "Internal server error during response encoding"}`))
		}
	}
}

// Status sends a response with the specified status code and an optional message
func (c *Context) Status(code int, message ...string) {
	c.WriteHeader(code)
	if len(message) > 0 {
		c.Header().Set("Content-Type", "text/plain")
		c.Write([]byte(message[0]))
	}
}

// Pattern represents a route pattern split into segments
type Pattern struct {
	segments []string
}

// NewPattern creates a Pattern from a path string
func NewPattern(path string) *Pattern {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 1 && segments[0] == "" {
		segments = []string{}
	}
	return &Pattern{segments: segments}
}

// ParamNames extracts parameter names from the pattern
func (p *Pattern) ParamNames() []string {
	var names []string
	for _, seg := range p.segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			names = append(names, strings.Trim(seg, "{}"))
		}
	}
	return names
}

// LiteralCount returns the number of non-parameter segments for sorting precedence
func (p *Pattern) LiteralCount() int {
	count := 0
	for _, seg := range p.segments {
		if !strings.HasPrefix(seg, "{") {
			count++
		}
	}
	return count
}

// Route defines a single route
type Route struct {
	Method     string
	Path       string
	Pattern    *Pattern
	Handler    func(*Context)
	Middleware []func(http.Handler) http.Handler
	paramNames []string
}

// RouterGroup holds routes and subgroups with a common prefix
type RouterGroup struct {
	prefix     string
	middleware []func(http.Handler) http.Handler
	routes     []Route
	groups     []*RouterGroup
}

// NewRouter initializes a root router group
func NewRouter() *RouterGroup {
	return &RouterGroup{
		prefix:     "",
		middleware: []func(http.Handler) http.Handler{},
		routes:     []Route{},
		groups:     []*RouterGroup{},
	}
}

// Group creates a subgroup with a prefix and optional middleware
func (rg *RouterGroup) Group(prefix string, middleware ...func(http.Handler) http.Handler) *RouterGroup {
	fullPrefix := strings.TrimRight(rg.prefix, "/") + "/" + strings.TrimLeft(prefix, "/")
	if fullPrefix == "/" {
		fullPrefix = ""
	}
	group := &RouterGroup{
		prefix:     fullPrefix,
		middleware: append([]func(http.Handler) http.Handler{}, middleware...),
		routes:     []Route{},
		groups:     []*RouterGroup{},
	}
	rg.groups = append(rg.groups, group)
	return group
}

// Handle registers a route with a method, path, handler, and optional middleware
func (rg *RouterGroup) Handle(method, path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	fullPath := strings.TrimRight(rg.prefix, "/") + "/" + strings.TrimLeft(path, "/")
	if fullPath == "/" {
		fullPath = ""
	}
	pattern := NewPattern(fullPath)
	route := Route{
		Method:     method,
		Path:       fullPath,
		Pattern:    pattern,
		Handler:    handler,
		Middleware: middleware,
		paramNames: pattern.ParamNames(),
	}
	rg.routes = append(rg.routes, route)
	return rg
}

// HTTP Method Helpers

// GET registers a GET route. For overlapping paths with the same method, the first registered route takes precedence
func (rg *RouterGroup) GET(path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("GET", path, handler, middleware...)
}

// POST registers a POST route
func (rg *RouterGroup) POST(path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("POST", path, handler, middleware...)
}

// PUT registers a PUT route
func (rg *RouterGroup) PUT(path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("PUT", path, handler, middleware...)
}

// DELETE registers a DELETE route
func (rg *RouterGroup) DELETE(path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("DELETE", path, handler, middleware...)
}

// PATCH registers a PATCH route
func (rg *RouterGroup) PATCH(path string, handler func(*Context), middleware ...func(http.Handler) http.Handler) *RouterGroup {
	return rg.Handle("PATCH", path, handler, middleware...)
}

// TrieNode represents a node in the trie structure
type TrieNode struct {
	staticChildren map[string]*TrieNode
	paramChild     *TrieNode
	routes         map[string]*Route
}

// Trie manages the trie structure for route matching
type Trie struct {
	root *TrieNode
}

// NewTrie initializes a new Trie
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{
			staticChildren: make(map[string]*TrieNode),
			routes:         make(map[string]*Route),
		},
	}
}

// Insert adds a route to the trie
func (t *Trie) Insert(route *Route) {
	node := t.root
	for _, seg := range route.Pattern.segments {
		isParam := strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
		if isParam {
			if node.paramChild == nil {
				node.paramChild = &TrieNode{
					staticChildren: make(map[string]*TrieNode),
					routes:         make(map[string]*Route),
				}
			}
			node = node.paramChild
		} else {
			if node.staticChildren == nil {
				node.staticChildren = make(map[string]*TrieNode)
			}
			if child, ok := node.staticChildren[seg]; !ok {
				child = &TrieNode{
					staticChildren: make(map[string]*TrieNode),
					routes:         make(map[string]*Route),
				}
				node.staticChildren[seg] = child
			}
			node = node.staticChildren[seg]
		}
	}
	if _, ok := node.routes[route.Method]; !ok {
		node.routes[route.Method] = route
	}
}

// Match finds a matching route for a method and path
func (t *Trie) Match(method, path string) (*Route, []string, bool) {
	normalizedPath := strings.Trim(path, "/")
	if normalizedPath == "" {
		if route, ok := t.root.routes[method]; ok {
			return route, []string{}, true
		}
		return nil, nil, false
	}
	segments := strings.Split(normalizedPath, "/")

	if path == "/" && len(segments) == 1 && segments[0] == "" {
		segments = []string{}
	}

	// Special case for root path
	if len(segments) == 0 {
		if route, ok := t.root.routes[method]; ok {
			return route, []string{}, true
		}
		return nil, nil, false
	}

	// Try standard matching first
	node := t.root
	var paramValues []string
	var lastParamNode *TrieNode
	var paramsSoFar []string

	for i, seg := range segments {
		// Remember last parameter node we encounter
		if node.paramChild != nil {
			lastParamNode = node
			paramsSoFar = make([]string, len(paramValues))
			copy(paramsSoFar, paramValues)
		}

		// Static match
		if child, ok := node.staticChildren[seg]; ok {
			node = child
			continue
		}

		// Parameter match
		if node.paramChild != nil {
			node = node.paramChild
			paramValues = append(paramValues, seg)
			continue
		}

		// If we reach here, normal matching failed
		// Check if we have a parameter that should capture all remaining segments
		if lastParamNode != nil && lastParamNode.paramChild != nil {
			if route, ok := lastParamNode.paramChild.routes[method]; ok {
				// Find position of the last parameter
				pattern := route.Pattern
				if len(pattern.segments) > 0 {
					lastSeg := pattern.segments[len(pattern.segments)-1]
					if strings.HasPrefix(lastSeg, "{") && strings.HasSuffix(lastSeg, "}") {
						// Last segment is a parameter - treat it as greedy
						remainingSegs := segments[i-1:]
						remainingPath := strings.Join(remainingSegs, "/")

						// Use the parameters up to this point
						result := append(paramsSoFar, remainingPath)
						return route, result, true
					}
				}
			}
		}

		// No match found
		return nil, nil, false
	}

	// Normal match at the end of the path
	if route, ok := node.routes[method]; ok {
		return route, paramValues, true
	}

	return nil, nil, false
}

// Build flattens the router group into a list of routes
func (rg *RouterGroup) Build() []Route {
	routes := rg.buildRoutes(nil)
	// Sort routes by literal count (descending) for precedence
	sort.Slice(routes, func(i, j int) bool {
		countI := routes[i].Pattern.LiteralCount()
		countJ := routes[j].Pattern.LiteralCount()
		return countI > countJ
	})
	return routes
}

// buildRoutes recursively collects all routes with inherited middleware
func (rg *RouterGroup) buildRoutes(parentMiddleware []func(http.Handler) http.Handler) []Route {
	currentMiddleware := append(parentMiddleware, rg.middleware...)
	var result []Route
	for _, route := range rg.routes {
		newRoute := route
		newRoute.Middleware = append(currentMiddleware, newRoute.Middleware...)
		result = append(result, newRoute)
	}
	for _, group := range rg.groups {
		result = append(result, group.buildRoutes(currentMiddleware)...)
	}
	return result
}

// ServeMux creates an http.ServeMux with trie-based route matching
func ServeMux(rg *RouterGroup) *http.ServeMux {
	routes := rg.Build()
	trie := NewTrie()
	for i := range routes {
		trie.Insert(&routes[i])
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, paramValues, ok := trie.Match(r.Method, r.URL.Path)
		if ok {
			c := &Context{
				ResponseWriter: w,
				Request:        r,
				Params:         make(map[string]string),
				path:           route.Path,
			}
			// Populate params from trie matching
			if len(route.paramNames) == len(paramValues) {
				for i, name := range route.paramNames {
					c.Params[name] = paramValues[i]
				}
			}

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				route.Handler(c)
			})
			for i := len(route.Middleware) - 1; i >= 0; i-- {
				handler = http.HandlerFunc(route.Middleware[i](handler).ServeHTTP)
			}
			handler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}
