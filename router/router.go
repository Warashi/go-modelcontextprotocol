// Package router provides a flexible URI routing system with support for dynamic parameters
// and pattern matching.
package router

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ErrNotFound is returned when no matching route is found.
var ErrNotFound = errors.New("route not found")

// Request represents an incoming request with query parameters and path parameters.
type Request struct {
	// Query contains the query parameters from the URL.
	Query map[string]string
	// Params contains the dynamic parameters extracted from the URL path and host.
	Params map[string]string
}

// Handler is an interface that processes requests and returns results of type T.
type Handler[T any] interface {
	Handle(ctx context.Context, req *Request) (T, error)
}

// HandlerFunc is a function type that implements Handler[T].
type HandlerFunc[T any] func(ctx context.Context, req *Request) (T, error)

// Handle implements the Handler interface for HandlerFunc.
func (f HandlerFunc[T]) Handle(ctx context.Context, req *Request) (T, error) {
	return f(ctx, req)
}

// route represents a registered route with its pattern and handler.
type route[T any] struct {
	scheme string // stored in lowercase (fixed values only)
	// host parameters
	hostIsParam   bool
	hostParamName string // used when hostIsParam == true
	host          string // used when hostIsParam == false (lowercase)
	// path segments
	pathSegments []pathSegment
	// query parameters (fixed keys and fixed values only)
	query map[string]string
	// handler for processing requests
	handler Handler[T]
}

// pathSegment represents a single segment of a URL path, which can be either
// static (literal) or dynamic (parameter).
type pathSegment struct {
	isParam   bool
	paramName string // used when isParam = true
	literal   string // used when isParam = false
}

// Mux is a request multiplexer that matches incoming requests against registered
// patterns and calls the corresponding handler.
type Mux[T any] struct {
	routes          []route[T]
	notFoundHandler Handler[T]
}

// NewMux creates a new Mux instance.
func NewMux[T any]() *Mux[T] {
	return &Mux[T]{}
}

// SetNotFoundHandler sets the handler to be called when no matching route is found.
func (m *Mux[T]) SetNotFoundHandler(h Handler[T]) {
	m.notFoundHandler = h
}

// SetNotFoundHandlerFunc sets a function to be called when no matching route is found.
func (m *Mux[T]) SetNotFoundHandlerFunc(f func(ctx context.Context, req *Request) (T, error)) {
	m.notFoundHandler = HandlerFunc[T](f)
}

// HandleFunc registers a new route with a handler function.
func (m *Mux[T]) HandleFunc(uri string, f func(context.Context, *Request) (T, error)) error {
	return m.Handle(uri, HandlerFunc[T](f))
}

// Handle registers a new route with a handler.
// uri is a URI string that will be matched against registered routes.
// uri can contains dynamic parameters like {param} in path and host.
// uri can contains query parameters like ?key=value.
// for example, "http://example.com/users/{id}" is a valid uri.
func (m *Mux[T]) Handle(uri string, h Handler[T]) error {
	// Parse URI → Convert to internal route[T] structure
	r, err := m.parseRoute(uri, h)
	if err != nil {
		return err
	}
	// Check for duplicates and conflicts during registration
	if err := m.checkConflict(r); err != nil {
		return err
	}
	// Add
	m.routes = append(m.routes, r)
	return nil
}

// Execute processes an incoming request URI and calls the appropriate handler.
// It returns ErrNotFound if no matching route is found and no notFoundHandler is set.
func (m *Mux[T]) Execute(ctx context.Context, rawURI string) (T, error) {
	var zero T

	// Parse
	req, parsed, err := m.parseRequest(rawURI)
	if err != nil {
		// Return the parse error directly instead of treating it as a route mismatch
		return zero, err
	}

	// Scan all routes to find the one with highest match score
	var candidates []matchedRoute[T]
	for _, rt := range m.routes {
		params, match := m.matchRoute(rt, parsed)
		if match {
			// Add to candidates with acquired parameters
			score := calcStaticScore(rt) // Use number of static segments as score
			candidates = append(candidates, matchedRoute[T]{
				route:  rt,
				params: params,
				score:  score,
			})
		}
	}
	if len(candidates) == 0 {
		// Not found, return notFoundHandler or ErrNotFound
		return m.handleNotFound(ctx, req)
	}

	// Sort by static score in descending order and select highest score
	// This ensures "static routes are prioritized"
	best := pickBestMatch(candidates)

	// Set parameters to Request
	for k, v := range best.params {
		req.Params[k] = v
	}

	return best.route.handler.Handle(ctx, req)
}

// matchedRoute holds a matched route along with its extracted parameters and match score.
type matchedRoute[T any] struct {
	route  route[T]
	params map[string]string
	score  int
}

// pickBestMatch selects the route with the highest score from the candidates.
func pickBestMatch[T any](candidates []matchedRoute[T]) matchedRoute[T] {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best
}

// calcStaticScore calculates a score for a route based on its static segments.
// Static segments and fixed hosts contribute to a higher score.
func calcStaticScore[T any](r route[T]) int {
	score := 0
	// +1 if host is fixed
	if !r.hostIsParam {
		score++
	}
	// +1 for each static path segment
	for _, seg := range r.pathSegments {
		if !seg.isParam {
			score++
		}
	}
	return score
}

// handleNotFound processes requests when no matching route is found.
func (m *Mux[T]) handleNotFound(ctx context.Context, req *Request) (T, error) {
	var zero T
	if m.notFoundHandler != nil {
		return m.notFoundHandler.Handle(ctx, req)
	}
	return zero, ErrNotFound
}

// parseRoute converts a URI string into a route structure.
func (m *Mux[T]) parseRoute(uri string, h Handler[T]) (route[T], error) {
	// Extract scheme and host manually first
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return route[T]{}, fmt.Errorf("scheme is required")
	}
	scheme := strings.ToLower(parts[0])

	// Split host and path+query
	remainder := parts[1]
	slashIndex := strings.Index(remainder, "/")
	var host, pathAndQuery string
	if slashIndex == -1 {
		host = remainder
		pathAndQuery = "/"
	} else {
		host = remainder[:slashIndex]
		pathAndQuery = remainder[slashIndex:]
	}

	if host == "" {
		return route[T]{}, fmt.Errorf("host is required")
	}

	// Check if host contains parameter
	hostIsParam := false
	hostParamName := ""
	if strings.Contains(host, "{") && strings.Contains(host, "}") {
		// Extract parameter name
		start := strings.Index(host, "{")
		end := strings.Index(host, "}")
		if start == -1 || end == -1 || end <= start {
			return route[T]{}, fmt.Errorf("invalid host parameter format")
		}
		hostParamName = host[start+1 : end]
		if !isValidParamName(hostParamName) {
			return route[T]{}, fmt.Errorf("invalid host param name: %s", hostParamName)
		}
		hostIsParam = true
	}

	// Parse path and query
	u, err := url.Parse(scheme + "://example.com" + pathAndQuery)
	if err != nil {
		return route[T]{}, fmt.Errorf("invalid URI: %w", err)
	}

	// path → normalize (consecutive slashes/trailing slash etc.)
	pathSegs, err := parsePath(u.Path)
	if err != nil {
		return route[T]{}, err
	}

	// query (fixed key-value pairs only)
	q, err := parseQueryForRegistration(u.RawQuery)
	if err != nil {
		return route[T]{}, err
	}

	// Check for duplicate parameter names if dynamic parameters are included
	if err := checkParamNameDuplication(hostParamName, pathSegs); err != nil {
		return route[T]{}, err
	}

	r := route[T]{
		scheme:        scheme,
		hostIsParam:   hostIsParam,
		hostParamName: hostParamName,
		host:          strings.ToLower(host),
		pathSegments:  pathSegs,
		query:         q,
		handler:       h,
	}
	return r, nil
}

// checkConflict verifies that a new route doesn't conflict with existing routes.
func (m *Mux[T]) checkConflict(newRoute route[T]) error {
	for _, rt := range m.routes {
		// Check if identical static routes are not duplicated (same scheme, host fixed/param, path, query pattern)
		// Or if dynamic routes cover the same pattern
		if isSameCoverage(rt, newRoute) {
			// Already have same (or same coverage) route
			return fmt.Errorf("conflict route: %v", routeToString(newRoute))
		}
	}
	return nil
}

// isSameCoverage determines if two routes would match the same requests.
// Routes are considered to have the same coverage if:
// - They have identical static segments
// - They have dynamic segments in the same positions
// - They have the same query parameters
func isSameCoverage[T any](a, b route[T]) bool {
	// 1) scheme is already lowercase, compare as is
	if a.scheme != b.scheme {
		return false
	}

	// 2) host
	if a.hostIsParam != b.hostIsParam {
		// One is dynamic, other is fixed → not same coverage
		return false
	} else {
		if a.hostIsParam && b.hostIsParam {
			// Both dynamic means same coverage
		} else {
			// Both fixed, check if values match
			if a.host != b.host {
				return false
			}
		}
	}

	// 3) path
	if len(a.pathSegments) != len(b.pathSegments) {
		return false
	}
	for i := 0; i < len(a.pathSegments); i++ {
		sA, sB := a.pathSegments[i], b.pathSegments[i]
		if sA.isParam != sB.isParam {
			// One is static, other is dynamic means different coverage
			return false
		} else {
			if sA.isParam && sB.isParam {
				// Both dynamic means same coverage
				// → Even if parameter names differ, same coverage
			} else {
				// Both static, check if strings match
				if sA.literal != sB.literal {
					return false
				}
			}
		}
	}

	// 4) query
	//   - Same coverage if exact match of key-value pairs
	//   - In this case, "dynamic query" is prohibited by spec so checking fixed values is sufficient
	if len(a.query) != len(b.query) {
		return false
	}
	for k, v := range a.query {
		if v2, ok := b.query[k]; !ok || v != v2 {
			return false
		}
	}

	return true
}

// routeToString converts a route to its string representation for debugging.
func routeToString[T any](r route[T]) string {
	var sb strings.Builder
	sb.WriteString(r.scheme)
	sb.WriteString("://")
	if r.hostIsParam {
		sb.WriteString("{" + r.hostParamName + "}")
	} else {
		sb.WriteString(r.host)
	}
	sb.WriteString("/")
	for i, seg := range r.pathSegments {
		if i > 0 {
			sb.WriteString("/")
		}
		if seg.isParam {
			sb.WriteString("{" + seg.paramName + "}")
		} else {
			sb.WriteString(seg.literal)
		}
	}
	if len(r.query) > 0 {
		sb.WriteString("?")
		i := 0
		for k, v := range r.query {
			if i > 0 {
				sb.WriteString("&")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			i++
		}
	}
	return sb.String()
}

// parsedURI represents a parsed URI with its components separated.
type parsedURI struct {
	scheme   string
	host     string
	pathSegs []string
	query    map[string]string
	rawQuery map[string]string // All actual received queries (including additional keys)
}

// parseRequest parses a raw URI string into a Request object and internal parsedURI structure.
func (m *Mux[T]) parseRequest(rawURI string) (*Request, *parsedURI, error) {
	// Extract scheme and host manually first
	parts := strings.SplitN(rawURI, "://", 2)
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid URI: scheme is required")
	}
	scheme := strings.ToLower(parts[0])

	// Split host and path+query
	remainder := parts[1]
	slashIndex := strings.Index(remainder, "/")
	var host, pathAndQuery string
	if slashIndex == -1 {
		host = remainder
		pathAndQuery = ""
	} else {
		host = remainder[:slashIndex]
		pathAndQuery = remainder[slashIndex:]
	}

	if host == "" {
		return nil, nil, fmt.Errorf("invalid URI: host is required")
	}
	host = strings.ToLower(host)

	// Parse path and query
	var u *url.URL
	var err error
	if pathAndQuery == "" {
		u = &url.URL{
			Scheme: scheme,
			Host:   host,
			Path:   "/",
		}
	} else {
		u, err = url.Parse(scheme + "://" + host + pathAndQuery)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid URI: %w", err)
		}
	}

	// path → normalize (consecutive slashes/trailing slash etc.)
	pathSegs, err := parsePathForRequest(u.Path)
	if err != nil {
		return nil, nil, err
	}

	// Parse query → key/value
	reqQuery, err := parseQueryForRequest(u.RawQuery)
	if err != nil {
		return nil, nil, err
	}

	// Request structure
	req := &Request{
		Query:  make(map[string]string),
		Params: make(map[string]string),
	}
	// Put all received queries into Request.Query (match determination is separate)
	for k, v := range reqQuery {
		req.Query[k] = v
	}

	p := &parsedURI{
		scheme:   scheme,
		host:     host,
		pathSegs: pathSegs,
		query:    reqQuery, // Same object
		rawQuery: reqQuery,
	}

	return req, p, nil
}

// matchRoute checks if a route matches a parsed URI and returns extracted parameters.
func (m *Mux[T]) matchRoute(rt route[T], parsed *parsedURI) (map[string]string, bool) {
	// 1) scheme
	if rt.scheme != parsed.scheme {
		return nil, false
	}

	params := make(map[string]string)

	// 2) host
	if rt.hostIsParam {
		// For host parameters, we need to check if the host follows the same pattern
		routeHost := rt.host       // e.g. "{subdomain}.example.com"
		requestHost := parsed.host // e.g. "test.example.com"

		// Extract the parameter part and the static parts
		start := strings.Index(routeHost, "{")
		end := strings.Index(routeHost, "}")
		if start != -1 && end != -1 && end > start {
			prefix := routeHost[:start]
			suffix := routeHost[end+1:]

			// Check if the request host matches the pattern
			if !strings.HasPrefix(requestHost, prefix) || !strings.HasSuffix(requestHost, suffix) {
				return nil, false
			}

			// Extract the parameter value
			paramValue := requestHost[len(prefix) : len(requestHost)-len(suffix)]
			if paramValue == "" {
				return nil, false
			}
			params[rt.hostParamName] = paramValue
		} else {
			return nil, false
		}
	} else {
		if rt.host != parsed.host {
			return nil, false
		}
	}

	// 3) path
	if len(rt.pathSegments) != len(parsed.pathSegs) {
		return nil, false
	}
	for i, seg := range rt.pathSegments {
		got := parsed.pathSegs[i]
		if seg.isParam {
			// Set parameter
			params[seg.paramName] = got
		} else {
			// Match against static segment (case-sensitive)
			if seg.literal != got {
				return nil, false
			}
		}
	}

	// 4) query
	//   - For registered keys, must match exactly
	//   - Extra keys in request are allowed
	for k, v := range rt.query {
		got, ok := parsed.query[k]
		if !ok {
			return nil, false
		}
		if got != v {
			return nil, false
		}
	}

	return params, true
}

// parsePath normalizes and parses a path string into path segments.
// It removes consecutive slashes, trailing slashes, and empty segments.
func parsePath(p string) ([]pathSegment, error) {
	// Remove leading/trailing slashes first for easier normalization
	trimmed := strings.TrimRight(p, "/")
	// Split and rebuild to combine double slashes into one
	rawSegs := strings.Split(trimmed, "/")
	var segs []string
	for _, s := range rawSegs {
		if s == "" {
			// Ignore between consecutive slashes
			continue
		}
		segs = append(segs, s)
	}

	// Convert to pathSegment
	ps := make([]pathSegment, 0, len(segs))
	for _, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			// Dynamic parameter
			paramName := strings.TrimSuffix(strings.TrimPrefix(s, "{"), "}")
			if !isValidParamName(paramName) {
				return nil, fmt.Errorf("invalid path param name: %s", paramName)
			}
			ps = append(ps, pathSegment{
				isParam:   true,
				paramName: paramName,
			})
		} else {
			ps = append(ps, pathSegment{
				isParam: false,
				literal: s,
			})
		}
	}
	return ps, nil
}

// parsePathForRequest normalizes and parses a path string for incoming requests.
func parsePathForRequest(p string) ([]string, error) {
	trimmed := strings.TrimRight(p, "/")
	rawSegs := strings.Split(trimmed, "/")
	var segs []string
	for _, s := range rawSegs {
		if s == "" {
			continue
		}
		segs = append(segs, s)
	}
	return segs, nil
}

// parseQueryForRegistration parses query parameters during route registration.
// It enforces:
// - No duplicate keys
// - No empty keys
// - No dynamic parameters
func parseQueryForRegistration(q string) (map[string]string, error) {
	if q == "" {
		return map[string]string{}, nil
	}
	values, err := url.ParseQuery(q)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}
	result := make(map[string]string)
	for k, arr := range values {
		if k == "" {
			return nil, fmt.Errorf("query key cannot be empty")
		}
		if strings.HasPrefix(k, "{") && strings.HasSuffix(k, "}") {
			return nil, fmt.Errorf("dynamic param in query is not allowed: %s", k)
		}
		if len(arr) > 1 {
			// Multiple values for same key → Error
			return nil, fmt.Errorf("duplicate query key: %s", k)
		}
		result[k] = arr[0]
	}
	return result, nil
}

// parseQueryForRequest parses query parameters from incoming requests.
// It enforces:
// - No duplicate keys
// - No empty keys
func parseQueryForRequest(q string) (map[string]string, error) {
	if q == "" {
		return map[string]string{}, nil
	}
	values, err := url.ParseQuery(q)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for k, arr := range values {
		if k == "" {
			return nil, fmt.Errorf("query key cannot be empty")
		}
		if len(arr) > 1 {
			return nil, fmt.Errorf("duplicate query key: %s", k)
		}
		result[k] = arr[0]
	}
	return result, nil
}

// paramNamePattern defines the allowed characters in parameter names.
var paramNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// isValidParamName checks if a parameter name contains only allowed characters
// (alphanumeric, hyphen, underscore) and is not empty.
func isValidParamName(name string) bool {
	return name != "" && paramNamePattern.MatchString(name)
}

// checkParamNameDuplication verifies that parameter names are not duplicated
// within a route's host and path segments.
func checkParamNameDuplication(hostParamName string, pathSegs []pathSegment) error {
	used := make(map[string]struct{})
	if hostParamName != "" {
		used[hostParamName] = struct{}{}
	}
	for _, seg := range pathSegs {
		if seg.isParam {
			if _, exists := used[seg.paramName]; exists {
				return fmt.Errorf("param name duplicated: %s", seg.paramName)
			}
			used[seg.paramName] = struct{}{}
		}
	}
	return nil
}
