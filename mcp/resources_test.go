package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/router"
)

func TestResourceReaderMux(t *testing.T) {
	t.Run("ReadResource", func(t *testing.T) {
		mux := NewResourceReaderMux()
		ctx := context.Background()

		// Test case 1: Basic resource reading
		expectedResult := &Result[ReadResourceResultData]{
			Data: ReadResourceResultData{
				Contents: []IsResourceContents{
					&TextResourceContents{
						URI:      "test://example.com/text",
						MimeType: "text/plain",
						Text:     "Hello, World!",
					},
				},
			},
		}

		err := mux.HandleFunc("test://example.com/text", func(ctx context.Context, req *router.Request) (*Result[ReadResourceResultData], error) {
			return expectedResult, nil
		})
		if err != nil {
			t.Fatalf("failed to handle func: %v", err)
		}

		result, err := mux.ReadResource(ctx, &Request[ReadResourceRequestParams]{
			Params: ReadResourceRequestParams{
				URI: "test://example.com/text",
			},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(expectedResult, result) {
			t.Errorf("result mismatch: want %+v, got %+v", expectedResult, result)
		}

		// Test case 2: Not found handler
		notFoundResult := &Result[ReadResourceResultData]{
			Data: ReadResourceResultData{
				Contents: []IsResourceContents{},
			},
		}
		mux.SetNotFoundHandlerFunc(func(ctx context.Context, req *router.Request) (*Result[ReadResourceResultData], error) {
			return notFoundResult, nil
		})

		result, err = mux.ReadResource(ctx, &Request[ReadResourceRequestParams]{
			Params: ReadResourceRequestParams{
				URI: "test://example.com/nonexistent",
			},
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(notFoundResult, result) {
			t.Errorf("result mismatch: want %+v, got %+v", notFoundResult, result)
		}
	})
}

func TestServer_ListResources(t *testing.T) {
	ctx := context.Background()
	resources := []Resource{
		{
			URI:         "test://example.com/resource1",
			Name:        "Resource 1",
			Description: "Test resource 1",
			MimeType:    "text/plain",
			Size:        100,
		},
		{
			URI:         "test://example.com/resource2",
			Name:        "Resource 2",
			Description: "Test resource 2",
			MimeType:    "application/json",
			Size:        200,
		},
	}

	server := &Server{
		resources: resources,
	}

	result, err := server.ListResources(ctx, &Request[ListResourcesRequestParams]{
		Params: ListResourcesRequestParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := &Result[ListResourcesResultData]{
		Data: ListResourcesResultData{
			Resources: resources,
		},
	}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("result mismatch: want %+v, got %+v", expected, result)
	}
}

func TestServer_ListResourceTemplates(t *testing.T) {
	ctx := context.Background()
	templates := []ResourceTemplate{
		{
			URITemplate: "test://example.com/template1/{param}",
			Name:        "Template 1",
			Description: "Test template 1",
			MimeType:    "text/plain",
		},
		{
			URITemplate: "test://example.com/template2/{param}",
			Name:        "Template 2",
			Description: "Test template 2",
			MimeType:    "application/json",
		},
	}

	server := &Server{
		resourceTemplates: templates,
	}

	result, err := server.ListResourceTemplates(ctx, &Request[ListResourceTemplatesRequestParams]{
		Params: ListResourceTemplatesRequestParams{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := &Result[ListResourceTemplatesResultData]{
		Data: ListResourceTemplatesResultData{
			ResourceTemplates: templates,
		},
	}
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("result mismatch: want %+v, got %+v", expected, result)
	}
}

func TestServer_ReadResource(t *testing.T) {
	ctx := context.Background()
	expectedResult := &Result[ReadResourceResultData]{
		Data: ReadResourceResultData{
			Contents: []IsResourceContents{
				&TextResourceContents{
					URI:      "test://example.com/text",
					MimeType: "text/plain",
					Text:     "Hello, World!",
				},
			},
		},
	}

	mux := NewResourceReaderMux()
	err := mux.HandleFunc("test://example.com/text", func(ctx context.Context, req *router.Request) (*Result[ReadResourceResultData], error) {
		return expectedResult, nil
	})
	if err != nil {
		t.Fatalf("failed to handle func: %v", err)
	}

	server := &Server{
		resourceReader: mux,
	}

	result, err := server.ReadResource(ctx, &Request[ReadResourceRequestParams]{
		Params: ReadResourceRequestParams{
			URI: "test://example.com/text",
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("result mismatch: want %+v, got %+v", expectedResult, result)
	}
}

func TestBlobResourceContents_MarshalJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    BlobResourceContents
		expected string
	}{
		{
			name: "basic blob resource",
			input: BlobResourceContents{
				URI:      "test://example.com/image",
				MimeType: "image/png",
				Blob:     []byte{0x89, 0x50, 0x4E, 0x47},
			},
			expected: `{"uri":"test://example.com/image","mimeType":"image/png","blob":"iVBORw=="}`,
		},
		{
			name: "empty blob",
			input: BlobResourceContents{
				URI:      "test://example.com/empty",
				MimeType: "application/octet-stream",
				Blob:     []byte{},
			},
			expected: `{"uri":"test://example.com/empty","mimeType":"application/octet-stream","blob":""}`,
		},
		{
			name: "blob without mimetype",
			input: BlobResourceContents{
				URI:  "test://example.com/data",
				Blob: []byte{0x01, 0x02, 0x03, 0x04},
			},
			expected: `{"uri":"test://example.com/data","blob":"AQIDBA=="}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(data) != tc.expected {
				t.Errorf("result mismatch: want %s, got %s", tc.expected, string(data))
			}
		})
	}
}

func TestBlobResourceContents_Implementation(t *testing.T) {
	// Test that BlobResourceContents implements IsResourceContents
	var blob BlobResourceContents
	var _ IsResourceContents = blob

	// Test isResourceContents method doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("isResourceContents panicked: %v", r)
			}
		}()
		blob.isResourceContents()
	}()
}

func TestResourceReaderWithBlob(t *testing.T) {
	mux := NewResourceReaderMux()
	ctx := context.Background()

	binaryData := []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F} // "Hello" in binary
	expectedResult := &Result[ReadResourceResultData]{
		Data: ReadResourceResultData{
			Contents: []IsResourceContents{
				&BlobResourceContents{
					URI:      "test://example.com/binary",
					MimeType: "application/octet-stream",
					Blob:     binaryData,
				},
			},
		},
	}

	err := mux.HandleFunc("test://example.com/binary", func(ctx context.Context, req *router.Request) (*Result[ReadResourceResultData], error) {
		return expectedResult, nil
	})
	if err != nil {
		t.Fatalf("failed to handle func: %v", err)
	}

	result, err := mux.ReadResource(ctx, &Request[ReadResourceRequestParams]{
		Params: ReadResourceRequestParams{
			URI: "test://example.com/binary",
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("result mismatch: want %+v, got %+v", expectedResult, result)
	}
}

func TestBlobResourceContents_Roundtrip(t *testing.T) {
	testCases := []struct {
		name  string
		input BlobResourceContents
	}{
		{
			name: "PNG image data",
			input: BlobResourceContents{
				URI:      "test://example.com/image.png",
				MimeType: "image/png",
				Blob:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			},
		},
		{
			name: "PDF document",
			input: BlobResourceContents{
				URI:      "test://example.com/document.pdf",
				MimeType: "application/pdf",
				Blob:     []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x35},
			},
		},
		{
			name: "Large binary data",
			input: BlobResourceContents{
				URI:      "test://example.com/large",
				MimeType: "application/octet-stream",
				Blob:     generateLargeByteSlice(1024), // 1KB of data
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal to JSON
			marshaledJSON, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal using custom struct to simulate JSON unmarshaling
			var unmarshaled struct {
				URI      string `json:"uri"`
				MimeType string `json:"mimeType"`
				Blob     string `json:"blob"`
			}
			err = json.Unmarshal(marshaledJSON, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify URI and MimeType
			if unmarshaled.URI != tc.input.URI {
				t.Errorf("URI mismatch: want %s, got %s", tc.input.URI, unmarshaled.URI)
			}
			if unmarshaled.MimeType != tc.input.MimeType {
				t.Errorf("MimeType mismatch: want %s, got %s", tc.input.MimeType, unmarshaled.MimeType)
			}

			// Decode base64 and verify blob content
			decodedBlob, err := base64.StdEncoding.DecodeString(unmarshaled.Blob)
			if err != nil {
				t.Fatalf("Failed to decode base64: %v", err)
			}
			if !reflect.DeepEqual(decodedBlob, tc.input.Blob) {
				t.Errorf("Blob mismatch: want %v, got %v", tc.input.Blob, decodedBlob)
			}
		})
	}
}

// Helper function to generate a byte slice of specified size
func generateLargeByteSlice(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}
	return data
}
