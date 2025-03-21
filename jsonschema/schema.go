package jsonschema

import "encoding/json"

type Schema interface {
	SchemaValidator
	json.Marshaler
}

type SchemaValidator interface {
	Validate(v json.RawMessage) error
	validate(v any) error
}
