package jsonschema

import "encoding/json"

type Schema interface {
	SchemaValidator
	json.Marshaler
}

type SchemaValidator interface {
	Validate(v any) error
}
