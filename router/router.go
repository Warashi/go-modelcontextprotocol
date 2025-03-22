package router

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ----------------------------------------
// Error Definitions
// ----------------------------------------
var ErrNotFound = errors.New("route not found")

// ----------------------------------------
// Request Type
// ----------------------------------------
type Request struct {
	Query  map[string]string
	Params map[string]string
}

// ----------------------------------------
// Handler[T] Interface
// ----------------------------------------
type Handler[T any] interface {
	Handle(ctx context.Context, req *Request) (T, error)
}

// HandlerFunc[T] is a type to treat functions as Handler[T]
type HandlerFunc[T any] func(ctx context.Context, req *Request) (T, error)

// Interface implementation
func (f HandlerFunc[T]) Handle(ctx context.Context, req *Request) (T, error) {
	return f(ctx, req)
}

// ----------------------------------------
// Internal Structure: struct representing registered routes
// ----------------------------------------
type route[T any] struct {
	scheme string // stored in lowercase (fixed values only)
	// whether host is dynamic
	hostIsParam   bool
	hostParamName string // used when hostIsParam == true
	host          string // used when hostIsParam == false (lowercase)
	// path information
	pathSegments []pathSegment
	// query information (fixed keys and fixed values only)
	query map[string]string
	// handler
	handler Handler[T]
}

// pathSegment distinguishes between static and dynamic segments
type pathSegment struct {
	isParam   bool
	paramName string // used when isParam = true
	literal   string // used when isParam = false
}

// ----------------------------------------
// Mux[T] Main Body
// ----------------------------------------
type Mux[T any] struct {
	routes          []route[T]
	notFoundHandler Handler[T]
}

// ----------------------------------------
// Mux[T] Constructor (as needed)
// ----------------------------------------
func NewMux[T any]() *Mux[T] {
	return &Mux[T]{}
}

// ----------------------------------------
// NotFoundHandler Configuration
// ----------------------------------------
func (m *Mux[T]) SetNotFoundHandler(h Handler[T]) {
	m.notFoundHandler = h
}

func (m *Mux[T]) SetNotFoundHandlerFunc(f func(ctx context.Context, req *Request) (T, error)) {
	m.notFoundHandler = HandlerFunc[T](f)
}

// ----------------------------------------
// Handler Registration (Handle, HandleFunc)
// ----------------------------------------
func (m *Mux[T]) HandleFunc(uri string, f func(context.Context, *Request) (T, error)) error {
	return m.Handle(uri, HandlerFunc[T](f))
}

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

// ----------------------------------------
// Lookup (Example of calling Handler from raw URI)
// Please modify method name and signature according to actual usage
// ----------------------------------------
func (m *Mux[T]) Lookup(ctx context.Context, rawURI string) (T, error) {
	// Parse
	req, parsed, err := m.parseRequest(rawURI)
	if err != nil {
		// Query duplicates or empty keys will result in errors at parseRequest
		// → Treat as route mismatch and call notFoundHandler
		return m.handleNotFound(ctx, req)
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

// ----------------------------------------
// Internal Helpers
// ----------------------------------------

// matchedRoute holds matched candidates during Lookup
type matchedRoute[T any] struct {
	route  route[T]
	params map[string]string
	score  int
}

// pickBestMatch returns the one with highest score
func pickBestMatch[T any](candidates []matchedRoute[T]) matchedRoute[T] {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best
}

// calcStaticScore: "static segment count + 1 if host is fixed, scheme is always fixed so +0" etc.
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

// handleNotFound calls notFoundHandler if exists, otherwise returns ErrNotFound
func (m *Mux[T]) handleNotFound(ctx context.Context, req *Request) (T, error) {
	var zero T
	if m.notFoundHandler != nil {
		return m.notFoundHandler.Handle(ctx, req)
	}
	return zero, ErrNotFound
}

// ----------------------------------------
// URI Parsing for Route Registration
// ----------------------------------------
func (m *Mux[T]) parseRoute(uri string, h Handler[T]) (route[T], error) {
	u, err := url.Parse(uri)
	if err != nil {
		return route[T]{}, fmt.Errorf("invalid URI: %w", err)
	}

	// scheme (lowercase)
	if u.Scheme == "" {
		return route[T]{}, fmt.Errorf("scheme is required")
	}
	scheme := strings.ToLower(u.Scheme)

	// host (lowercase)
	if u.Host == "" {
		return route[T]{}, fmt.Errorf("host is required")
	}
	// Check if it's "{param}"
	hostIsParam := false
	hostParamName := ""
	hostLower := strings.ToLower(u.Host)
	if strings.HasPrefix(u.Host, "{") && strings.HasSuffix(u.Host, "}") {
		// Dynamic host
		hostIsParam = true
		hostParamName = strings.TrimSuffix(strings.TrimPrefix(u.Host, "{"), "}")
		if !isValidParamName(hostParamName) {
			return route[T]{}, fmt.Errorf("invalid host param name: %s", hostParamName)
		}
	} else {
		hostLower = strings.ToLower(u.Host) // Store fixed host in lowercase
	}

	// path
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
		host:          hostLower,
		pathSegments:  pathSegs,
		query:         q,
		handler:       h,
	}
	return r, nil
}

// ----------------------------------------
// Check for Duplicates and Conflicts during Registration
// ----------------------------------------
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

// isSameCoverage checks if 2 routes cover "the same match range"
// Specification:
//   - static vs static: complete duplicate if identical
//   - dynamic vs dynamic: same pattern if at same position
//   - static vs dynamic: no conflict (even if same position can match both static and dynamic, not registration error)
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

// routeToString is for debugging
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

// ----------------------------------------
// URI Parsing for Requests
// ----------------------------------------
type parsedURI struct {
	scheme   string
	host     string
	pathSegs []string
	query    map[string]string
	rawQuery map[string]string // All actual received queries (including additional keys)
}

// parseRequest parses raw URI and returns Request and internal structure
func (m *Mux[T]) parseRequest(rawURI string) (*Request, *parsedURI, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return nil, nil, err
	}

	// scheme
	if u.Scheme == "" {
		return nil, nil, fmt.Errorf("scheme is missing in request")
	}
	scheme := strings.ToLower(u.Scheme)

	// host
	if u.Host == "" {
		return nil, nil, fmt.Errorf("host is missing in request")
	}
	host := strings.ToLower(u.Host)

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
		// Whether to keep same as rawQuery is optional
		rawQuery: reqQuery,
	}

	return req, p, nil
}

// ----------------------------------------
// Match Determination
// ----------------------------------------
func (m *Mux[T]) matchRoute(rt route[T], parsed *parsedURI) (map[string]string, bool) {
	// 1) scheme
	if rt.scheme != parsed.scheme {
		return nil, false
	}

	params := make(map[string]string)

	// 2) host
	if rt.hostIsParam {
		// Capture as dynamic host parameter
		params[rt.hostParamName] = parsed.host
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

// ----------------------------------------
// Path Normalization: Combine consecutive slashes into one, remove trailing slash, remove "" (empty) from start
// Example: //users///profile/ → ["users", "profile"]
// Used both during registration and request handling
// ----------------------------------------
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

// ----------------------------------------
// Request-time path normalization (returns []string)
// Almost same process as registration, can be combined if desired
// ----------------------------------------
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

// ----------------------------------------
// Query Parameter Parsing (for registration)
//   - Error if duplicate keys
//   - Error if empty key
//   - "%xx" is decoded by standard library
//   - "{param}" is prohibited (fixed values only)
//
// ----------------------------------------
func parseQueryForRegistration(q string) (map[string]string, error) {
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

// ----------------------------------------
// Request-time Query Parsing
//   - Error if duplicate keys or empty keys
//   - Error if multiple values
//
// ----------------------------------------
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

// ----------------------------------------
// Parameter Name Validation
//   - Only alphanumeric, hyphen, underscore allowed
//
// ----------------------------------------
var paramNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func isValidParamName(name string) bool {
	return name != "" && paramNamePattern.MatchString(name)
}

// ----------------------------------------
// Check for Parameter Name Duplication within Same Route
// Consider host if it's {xxx}
// ----------------------------------------
func checkParamNameDuplication(hostParamName string, pathSegs []pathSegment) error {
	used := make(map[string]bool)
	if hostParamName != "" {
		if used[hostParamName] {
			return fmt.Errorf("param name duplicated in host: %s", hostParamName)
		}
		used[hostParamName] = true
	}
	for _, seg := range pathSegs {
		if seg.isParam {
			if used[seg.paramName] {
				return fmt.Errorf("param name duplicated: %s", seg.paramName)
			}
			used[seg.paramName] = true
		}
	}
	return nil
}
