package router

import (
	"context"
	"strings"
	"testing"
)

func TestMux_Handle(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid static route",
			uri:     "http://example.com/users",
			wantErr: false,
		},
		{
			name:    "valid route with path param",
			uri:     "http://example.com/users/{id}",
			wantErr: false,
		},
		{
			name:    "valid route with host param",
			uri:     "http://{subdomain}.example.com/users",
			wantErr: false,
		},
		{
			name:    "valid route with query params",
			uri:     "http://example.com/users?name=john",
			wantErr: false,
		},
		{
			name:    "valid route with multiple query params",
			uri:     "http://example.com/users?name=john&age=25",
			wantErr: false,
		},
		{
			name:        "invalid route with duplicate query params",
			uri:         "http://example.com/users?name=john&name=jane",
			wantErr:     true,
			errContains: "duplicate query key",
		},
		{
			name:        "invalid route with empty query key",
			uri:         "http://example.com/users?=value",
			wantErr:     true,
			errContains: "query key cannot be empty",
		},
		{
			name:        "invalid route with dynamic query param",
			uri:         "http://example.com/users?{name}=value",
			wantErr:     true,
			errContains: "dynamic param in query is not allowed",
		},
		{
			name:        "invalid route with malformed query string",
			uri:         "http://example.com/users?name=%ZZ",
			wantErr:     true,
			errContains: "invalid URI",
		},
		{
			name:    "valid route with URL-encoded query params",
			uri:     "http://example.com/search?q=hello%20world&type=web%2Fpage",
			wantErr: false,
		},
		{
			name:        "invalid URI - missing scheme",
			uri:         "example.com/users",
			wantErr:     true,
			errContains: "scheme is required",
		},
		{
			name:        "invalid URI - missing host",
			uri:         "http:///users",
			wantErr:     true,
			errContains: "host is required",
		},
		{
			name:        "invalid param name in path",
			uri:         "http://example.com/users/{invalid@param}",
			wantErr:     true,
			errContains: "invalid path param name",
		},
		{
			name:        "duplicate param names",
			uri:         "http://example.com/users/{param}/posts/{param}",
			wantErr:     true,
			errContains: "param name duplicated",
		},
		{
			name:        "invalid route with malformed query string",
			uri:         "http://example.com/search?q=%ZZ",
			wantErr:     true,
			errContains: "invalid URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMux[string]()
			err := m.Handle(tt.uri, HandlerFunc[string](func(ctx context.Context, req *Request) (string, error) {
				return "ok", nil
			}))

			if (err != nil) != tt.wantErr {
				t.Errorf("Mux.Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("Mux.Handle() error = %v, want error containing %v", err, tt.errContains)
			}
		})
	}
}

func TestMux_Execute(t *testing.T) {
	tests := []struct {
		name        string
		setupRoutes []string
		execURI     string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "static route match",
			setupRoutes: []string{"http://example.com/users"},
			execURI:     "http://example.com/users",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "path param match",
			setupRoutes: []string{"http://example.com/users/{id}"},
			execURI:     "http://example.com/users/123",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "host param match",
			setupRoutes: []string{"http://{subdomain}.example.com/users"},
			execURI:     "http://test.example.com/users",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match",
			setupRoutes: []string{"http://example.com/users?name=john"},
			execURI:     "http://example.com/users?name=john",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match with URL-encoded values",
			setupRoutes: []string{"http://example.com/search?q=hello%20world&type=web%2Fpage"},
			execURI:     "http://example.com/search?q=hello%20world&type=web%2Fpage",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match with mixed encoding",
			setupRoutes: []string{"http://example.com/search?q=hello world&type=web%2Fpage"},
			execURI:     "http://example.com/search?q=hello%20world&type=web%2Fpage",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "no match - different path",
			setupRoutes: []string{"http://example.com/users"},
			execURI:     "http://example.com/posts",
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name:        "no match - different host",
			setupRoutes: []string{"http://example.com/users"},
			execURI:     "http://other.com/users",
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name:        "no match - different query",
			setupRoutes: []string{"http://example.com/users?name=john"},
			execURI:     "http://example.com/users?name=jane",
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name: "static route prioritized over dynamic",
			setupRoutes: []string{
				"http://example.com/users/{id}",
				"http://example.com/users/profile",
			},
			execURI: "http://example.com/users/profile",
			want:    "ok",
			wantErr: false,
		},
		{
			name:        "invalid request URI",
			setupRoutes: []string{"http://example.com/users"},
			execURI:     "invalid-uri",
			wantErr:     true,
			errContains: "invalid URI",
		},
		{
			name:        "URL-encoded query param extraction",
			setupRoutes: []string{"http://example.com/search"},
			execURI:     "http://example.com/search?q=hello%20world&type=web%2Fpage",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match",
			setupRoutes: []string{"http://example.com/users?name=john"},
			execURI:     "http://example.com/users?name=john",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match with multiple params",
			setupRoutes: []string{"http://example.com/users?name=john&age=25"},
			execURI:     "http://example.com/users?name=john&age=25",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param match with extra params in request",
			setupRoutes: []string{"http://example.com/users?name=john"},
			execURI:     "http://example.com/users?name=john&age=25",
			want:        "ok",
			wantErr:     false,
		},
		{
			name:        "query param mismatch - missing required param",
			setupRoutes: []string{"http://example.com/users?name=john&age=25"},
			execURI:     "http://example.com/users?name=john",
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name:        "query param mismatch - different value",
			setupRoutes: []string{"http://example.com/users?name=john"},
			execURI:     "http://example.com/users?name=jane",
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name:        "query param order independence",
			setupRoutes: []string{"http://example.com/users?name=john&age=25"},
			execURI:     "http://example.com/users?age=25&name=john",
			want:        "ok",
			wantErr:     false,
		},
		{
			name: "multiple routes with different query params",
			setupRoutes: []string{
				"http://example.com/users?role=admin",
				"http://example.com/users?role=user",
			},
			execURI: "http://example.com/users?role=admin",
			want:    "ok",
			wantErr: false,
		},
		{
			name: "static path with query params prioritized over dynamic path",
			setupRoutes: []string{
				"http://example.com/users/{id}?role=admin",
				"http://example.com/users/profile?role=admin",
			},
			execURI: "http://example.com/users/profile?role=admin",
			want:    "ok",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMux[string]()

			// Setup routes
			for _, uri := range tt.setupRoutes {
				err := m.Handle(uri, HandlerFunc[string](func(ctx context.Context, req *Request) (string, error) {
					return "ok", nil
				}))
				if err != nil {
					t.Fatalf("Failed to setup route %s: %v", uri, err)
				}
			}

			// Execute request
			got, err := m.Execute(context.Background(), tt.execURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mux.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("Mux.Execute() error = %v, want error containing %v", err, tt.errContains)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("Mux.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMux_ParamExtraction(t *testing.T) {
	tests := []struct {
		name        string
		routeURI    string
		requestURI  string
		wantParams  map[string]string
		wantQuery   map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:       "path param extraction",
			routeURI:   "http://example.com/users/{id}/posts/{postId}",
			requestURI: "http://example.com/users/123/posts/456",
			wantParams: map[string]string{
				"id":     "123",
				"postId": "456",
			},
			wantQuery: map[string]string{},
		},
		{
			name:       "host param extraction",
			routeURI:   "http://{subdomain}.example.com/users",
			requestURI: "http://test.example.com/users",
			wantParams: map[string]string{
				"subdomain": "test",
			},
			wantQuery: map[string]string{},
		},
		{
			name:       "query param extraction",
			routeURI:   "http://example.com/users",
			requestURI: "http://example.com/users?name=john&age=25",
			wantParams: map[string]string{},
			wantQuery: map[string]string{
				"name": "john",
				"age":  "25",
			},
		},
		{
			name:       "combined param extraction",
			routeURI:   "http://{subdomain}.example.com/users/{id}?role=admin",
			requestURI: "http://test.example.com/users/123?role=admin&extra=value",
			wantParams: map[string]string{
				"subdomain": "test",
				"id":        "123",
			},
			wantQuery: map[string]string{
				"role":  "admin",
				"extra": "value",
			},
		},
		{
			name:       "URL-encoded query param extraction",
			routeURI:   "http://example.com/search",
			requestURI: "http://example.com/search?q=hello%20world&type=web%2Fpage",
			wantParams: map[string]string{},
			wantQuery: map[string]string{
				"q":    "hello world",
				"type": "web/page",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMux[string]()
			var capturedReq *Request

			err := m.Handle(tt.routeURI, HandlerFunc[string](func(ctx context.Context, req *Request) (string, error) {
				capturedReq = req
				return "ok", nil
			}))
			if err != nil {
				t.Fatalf("Failed to setup route: %v", err)
			}

			_, err = m.Execute(context.Background(), tt.requestURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mux.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("Mux.Execute() error = %v, want error containing %v", err, tt.errContains)
				return
			}
			if err == nil {
				// Check params
				if !mapsEqual(capturedReq.Params, tt.wantParams) {
					t.Errorf("Params = %v, want %v", capturedReq.Params, tt.wantParams)
				}
				// Check query
				if !mapsEqual(capturedReq.Query, tt.wantQuery) {
					t.Errorf("Query = %v, want %v", capturedReq.Query, tt.wantQuery)
				}
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func mapsEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		if v2, ok := m2[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}
