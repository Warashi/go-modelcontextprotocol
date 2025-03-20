package mcp

import (
	"encoding/base64"
	"encoding/json"
)

// IsContent is an interface for the content of the tool call result.
type IsContent interface {
	isContent()
}

// TextContent is the text content of the tool call result.
// TODO: add Annotations field
type TextContent struct {
	Text string `json:"text"`
}

// isContent implements isContent.
func (TextContent) isContent() {}

// MarshalJSON implements json.Marshaler.
func (t TextContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": "text",
		"text": t.Text,
	})
}

// ImageContent is the image content of the tool call result.
// TODO: add Annotations field
type ImageContent struct {
	Data     []byte `json:"data"`
	MimeType string `json:"mimeType"`
}

// isContent implements isContent.
func (ImageContent) isContent() {}

// MarshalJSON implements json.Marshaler.
func (t ImageContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":     "image",
		"data":     base64.StdEncoding.EncodeToString(t.Data),
		"mimeType": t.MimeType,
	})
}

// EmbeddedResource is the embedded resource content of the tool call result.
// TODO: add Annotations field
type EmbeddedResource struct {
	Resource IsResourceContents `json:"resource"`
}

// isContent implements isContent.
func (EmbeddedResource) isContent() {}

// MarshalJSON implements json.Marshaler.
func (t EmbeddedResource) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":     "resource",
		"resource": t.Resource,
	})
}
