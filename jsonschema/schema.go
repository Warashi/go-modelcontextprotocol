package jsonschema

import "encoding/json"

type Schema interface {
	MarshalSchema() (json.RawMessage, error)
}
