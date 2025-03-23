package mcp

import (
	"context"
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
