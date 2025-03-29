package mcp

import (
	"context"
	"reflect"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func TestServer_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		server  *Server
		request *Request[InitializationRequestParams]
		want    *Result[InitializationResponseData]
	}{
		{
			name:   "empty server",
			server: mustNewServer(t, "test", "1.0.0"),
			request: &Request[InitializationRequestParams]{
				Params: InitializationRequestParams{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities:    InitializationRequestCapabilities{},
					ClientInfo: ClientInfoData{
						Name:    "test-client",
						Version: "1.0.0",
					},
				},
			},
			want: &Result[InitializationResponseData]{
				Data: InitializationResponseData{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities:    Capabilities{},
					ServerInfo: ServerInfoData{
						Name:    "test",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			name: "server with tools",
			server: mustNewServer(t, "test", "1.0.0",
				WithTool(NewToolFunc("test", "Test tool", jsonschema.Object{}, func(ctx context.Context, input string) (string, error) {
					return "success", nil
				})),
			),
			request: &Request[InitializationRequestParams]{
				Params: InitializationRequestParams{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities:    InitializationRequestCapabilities{},
					ClientInfo: ClientInfoData{
						Name:    "test-client",
						Version: "1.0.0",
					},
				},
			},
			want: &Result[InitializationResponseData]{
				Data: InitializationResponseData{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities: Capabilities{
						Tools: &ToolsCapabilities{},
					},
					ServerInfo: ServerInfoData{
						Name:    "test",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			name: "server with resources",
			server: mustNewServer(t, "test", "1.0.0",
				WithResource(Resource{
					URI:         "test://example.com/resource",
					Name:        "Test Resource",
					Description: "Test resource",
					MimeType:    "text/plain",
				}),
				WithResourceReader(NewResourceReaderMux()),
			),
			request: &Request[InitializationRequestParams]{
				Params: InitializationRequestParams{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities:    InitializationRequestCapabilities{},
					ClientInfo: ClientInfoData{
						Name:    "test-client",
						Version: "1.0.0",
					},
				},
			},
			want: &Result[InitializationResponseData]{
				Data: InitializationResponseData{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities: Capabilities{
						Resources: &ResourcesCapabilities{},
					},
					ServerInfo: ServerInfoData{
						Name:    "test",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			name: "server with resource templates",
			server: mustNewServer(t, "test", "1.0.0",
				WithResourceTemplate(ResourceTemplate{
					URITemplate: "test://example.com/template/{param}",
					Name:        "Test Template",
					Description: "Test template",
					MimeType:    "text/plain",
				}),
				WithResourceReader(NewResourceReaderMux()),
			),
			request: &Request[InitializationRequestParams]{
				Params: InitializationRequestParams{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities:    InitializationRequestCapabilities{},
					ClientInfo: ClientInfoData{
						Name:    "test-client",
						Version: "1.0.0",
					},
				},
			},
			want: &Result[InitializationResponseData]{
				Data: InitializationResponseData{
					ProtocolVersion: SupportedProtocolVersion,
					Capabilities: Capabilities{
						Resources: &ResourcesCapabilities{},
					},
					ServerInfo: ServerInfoData{
						Name:    "test",
						Version: "1.0.0",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.server.Initialize(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Server.Initialize() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
