package jsonschema

import "encoding/json"

type Schema interface {
	SchemaValidator
	SchemaMarshaler
}

type SchemaValidator interface {
	Validate(v any) error
}

type SchemaMarshaler interface {
	MarshalSchema() (json.RawMessage, error)
}
