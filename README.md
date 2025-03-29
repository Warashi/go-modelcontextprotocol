# go-modelcontextprotocol

`go-modelcontextprotocol` is a Go library that implements the Model Context Protocol (MCP). It provides a framework for building AI/LLM service backends with support for tools, resources, and various transport mechanisms.

## Project Structure

- **mcp**: Core implementation of the Model Context Protocol with support for tools, resources, and server capabilities.
- **jsonrpc2**: JSON-RPC 2.0 protocol implementation for client-server communication.
- **jsonschema**: JSON schema validation tools for validating inputs and outputs.
- **transport**: Transport layer implementations including stdio, SSE (Server-Sent Events), and pipe-based communication.
- **router**: Flexible URI routing system with support for dynamic parameters and pattern matching.

## Usage

### Creating a Server

To create a new server, use the `NewServer` function:

```go
import (
	"context"
	"log"

	"github.com/Warashi/go-modelcontextprotocol/mcp"
)

func main() {
	// Create a new server with name and version
	server, err := mcp.NewServer("example", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Start the server using stdio transport
	ctx := context.Background()
	if err := server.ServeStdio(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

### Adding Tools

You can add tools to the server using the provided options:

```go
import (
	"context"
	"log"

	"github.com/Warashi/go-modelcontextprotocol/mcp"
	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func main() {
	// Create a tool with a name, description, input schema, and handler function
	tool := mcp.NewToolFunc(
		"exampleTool", 
		"An example tool", 
		jsonschema.Object{}, 
		func(ctx context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"result": "success"}, nil
		},
	)

	// Create a server with the tool
	server, err := mcp.NewServer("example", "1.0.0", mcp.WithTool(tool))
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Serve the server over stdin/stdout
	ctx := context.Background()
	if err := server.ServeStdio(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

### Adding Resources

You can add static resources and resource templates:

```go
import (
	"github.com/Warashi/go-modelcontextprotocol/mcp"
)

// Add a static resource
resource := mcp.Resource{
	URI:         "example://resource/doc",
	Name:        "Example Resource",
	Description: "An example resource",
	MimeType:    "text/plain",
}

// Add a resource template
template := mcp.ResourceTemplate{
	URITemplate: "example://resource/{id}",
	Name:        "Example Template",
	Description: "A template for accessing resources by ID",
	MimeType:    "text/plain",
}

// Create a resource reader to serve the resources
reader := mcp.NewResourceReaderMux()

// Create the server with resources
server, err := mcp.NewServer("example", "1.0.0",
	mcp.WithResource(resource),
	mcp.WithResourceTemplate(template),
	mcp.WithResourceReader(reader),
)
```

### Using HTTP/SSE Transport

For web applications, you can use the SSE transport:

```go
import (
	"context"
	"log"
	"net/http"

	"github.com/Warashi/go-modelcontextprotocol/mcp"
)

func main() {
	server, err := mcp.NewServer("example", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Create an SSE handler with a base URL
	handler, err := server.SSEHandler("http://localhost:8080/sse")
	if err != nil {
		log.Fatalf("Failed to create SSE handler: %v", err)
	}
	
	// Register the handler with your HTTP server
	http.Handle("/sse", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Running Tests

To run the tests, use the `go test` command:

```sh
go test ./...
```

## Server Implementations Using This Library

- [mcp-server-pipecd](https://github.com/Warashi/mcp-server-pipecd)

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
