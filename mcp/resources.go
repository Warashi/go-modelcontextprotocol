package mcp

import "context"

// ListResourcesRequestParams is the parameters of the list resources request.
type ListResourcesRequestParams struct {
	Cursor string `json:"cursor"`
}

// ListResourcesResultData is the result of the list resources request.
type ListResourcesResultData struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty,omitzero"`
}

// ListResourceTemplatesRequestParams is the parameters of the list resource templates request.
type ListResourceTemplatesRequestParams struct {
	Cursor string `json:"cursor"`
}

// ListResourceTemplatesResultData is the result of the list resource templates request.
type ListResourceTemplatesResultData struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	NextCursor        string             `json:"nextCursor,omitempty,omitzero"`
}

// Resource is a resource that can be used in the model context.
// TODO: add Annotations field.
type Resource struct {
	// URI is the unique identifier of the resource.
	URI string `json:"uri"`
	// Name is human-readable name of the resource.
	Name string `json:"name"`
	// Description of what the resource is.
	Description string `json:"description,omitempty,omitzero"`
	// MimeType is the MIME type of the resource.
	MimeType string `json:"mimeType,omitempty,omitzero"`
	// Size is the size of the resource in bytes before base64 encoding or any tokenization.
	Size int64 `json:"size,omitempty,omitzero"`
}

// ResourceTemplate is the template of a resource.
// TODO: add Annotations field.
type ResourceTemplate struct {
	// URITemplate is the URI template of the resource.
	URITemplate string `json:"uriTemplate"`
	// Name is human-readable name of the resource.
	Name string `json:"name"`
	// Description of what the resource is.
	Description string `json:"description,omitempty,omitzero"`
	// MimeType is the MIME type of the resource.
	MimeType string `json:"mimeType,omitempty,omitzero"`
}

// ReadResourceRequestParams is the parameters of the read resource request.
type ReadResourceRequestParams struct {
	URI string `json:"uri"`
}

// ReadResourceResultData is the result of the read resource request.
type ReadResourceResultData struct {
	Contents []IsResourceContents `json:"contents"`
}

// IsResourceContents is an interface for the content of the read resource result.
type IsResourceContents interface {
	isResourceContents()
}

// TextResourceContents is the contents of a text resource.
type TextResourceContents struct {
	// URI is the unique identifier of the resource.
	URI string `json:"uri"`
	// MimeType is the MIME type of the resource.
	MimeType string `json:"mimeType,omitempty,omitzero"`
	// Text is the text content of the resource.
	Text string `json:"text"`
}

// isResourceContents implements isResourceContents.
func (TextResourceContents) isResourceContents() {}

// BlobResourceContents is the contents of a blob resource.
type BlobResourceContents struct {
	// URI is the unique identifier of the resource.
	URI string `json:"uri"`
	// MimeType is the MIME type of the resource.
	MimeType string `json:"mimeType,omitempty,omitzero"`
	// Blob is the binary data of the resource.
	// This field is base64 encoded when marshaling to JSON.
	Blob []byte `json:"blob"`
}

// isResourceContents implements isResourceContents.
func (BlobResourceContents) isResourceContents() {}

// ResourceHandler is the handler of the resource methods.
type ResourceReader interface {
	ReadResource(ctx context.Context, request *Request[ReadResourceRequestParams]) (*Result[ReadResourceResultData], error)
}

// ListResources lists resources.
func (s *Server) ListResources(ctx context.Context, request *Request[ListResourcesRequestParams]) (*Result[ListResourcesResultData], error) {
	return &Result[ListResourcesResultData]{
		Data: ListResourcesResultData{
			Resources: s.resources,
		},
	}, nil
}

// ListResourceTemplates lists resource templates.
func (s *Server) ListResourceTemplates(ctx context.Context, request *Request[ListResourceTemplatesRequestParams]) (*Result[ListResourceTemplatesResultData], error) {
	return &Result[ListResourceTemplatesResultData]{
		Data: ListResourceTemplatesResultData{
			ResourceTemplates: s.resourceTemplates,
		},
	}, nil
}

// ReadResource reads a resource.
func (s *Server) ReadResource(ctx context.Context, request *Request[ReadResourceRequestParams]) (*Result[ReadResourceResultData], error) {
	return s.resourceReader.ReadResource(ctx, request)
}
