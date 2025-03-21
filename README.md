# go-modelcontextprotocol

`go-modelcontextprotocol` is a Go library that implements a model context protocol. It provides a framework for managing resources, tools, and server capabilities in a structured and extensible manner.

## Project Structure

- **mcp**: Contains the core implementation of the model context protocol.
- **jsonrpc2**: Implements JSON-RPC 2.0 protocol support.
- **jsonschema**: Provides JSON schema validation support.

## Usage

### Creating a Server

To create a new server, use the `NewServer` function:

```go
import (
	"context"

	"github.com/Warashi/go-modelcontextprotocol/mcp"
)

func main() {
	server := mcp.NewStdioServer("example", "1.0.0")
	ctx := context.Background()
	if err := server.Serve(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

### Adding Tools

You can add resources and tools to the server using the provided options:

```go
tool := mcp.NewToolFunc("exampleTool", "An example tool", jsonschema.Object{}, func(ctx context.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{"result": "success"}, nil
})

server := mcp.NewStdioServer("example", "1.0.0",
	mcp.WithTool(tool),
)
```

### Running Tests

To run the tests, use the `go test` command:

```sh
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
