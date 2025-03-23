package mcp

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestTextContent_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content TextContent
		want    string
	}{
		{
			name: "basic text",
			content: TextContent{
				Text: "Hello, World!",
			},
			want: `{"type":"text","text":"Hello, World!"}`,
		},
		{
			name: "empty text",
			content: TextContent{
				Text: "",
			},
			want: `{"type":"text","text":""}`,
		},
		{
			name: "text with special characters",
			content: TextContent{
				Text: "Hello\nWorld\t!\"\\",
			},
			want: `{"type":"text","text":"Hello\nWorld\t!\"\\"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestImageContent_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content ImageContent
		want    string
	}{
		{
			name: "basic image",
			content: ImageContent{
				Data:     []byte("test data"),
				MimeType: "image/png",
			},
			want: `{"type":"image","data":"dGVzdCBkYXRh","mimeType":"image/png"}`,
		},
		{
			name: "empty image",
			content: ImageContent{
				Data:     []byte{},
				MimeType: "image/jpeg",
			},
			want: `{"type":"image","data":"","mimeType":"image/jpeg"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.want, string(got))

			// Verify that the base64-encoded data can be decoded back
			var result map[string]interface{}
			if err := json.Unmarshal(got, &result); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			dataStr, ok := result["data"].(string)
			if !ok {
				t.Fatal("data field is not a string")
			}

			decoded, err := base64.StdEncoding.DecodeString(dataStr)
			if err != nil {
				t.Fatalf("failed to decode base64 data: %v", err)
			}

			if string(decoded) != string(tt.content.Data) {
				t.Errorf("decoded data mismatch: got %q, want %q", decoded, tt.content.Data)
			}
		})
	}
}

func TestEmbeddedResource_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		content EmbeddedResource
		want    string
	}{
		{
			name: "text resource",
			content: EmbeddedResource{
				Resource: &TextResourceContents{
					URI:      "test://example.com/text",
					MimeType: "text/plain",
					Text:     "Hello, World!",
				},
			},
			want: `{"type":"resource","resource":{"uri":"test://example.com/text","mimeType":"text/plain","text":"Hello, World!"}}`,
		},
		{
			name: "blob resource",
			content: EmbeddedResource{
				Resource: &BlobResourceContents{
					URI:      "test://example.com/blob",
					MimeType: "application/octet-stream",
					Blob:     []byte("test data"),
				},
			},
			want: `{"type":"resource","resource":{"uri":"test://example.com/blob","mimeType":"application/octet-stream","blob":"dGVzdCBkYXRh"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestIsContent_Interface(t *testing.T) {
	var _ IsContent = TextContent{}
	var _ IsContent = ImageContent{}
	var _ IsContent = EmbeddedResource{}
}
