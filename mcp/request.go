package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ProgressToken represents a token that can be used to track the progress of a request.
// It can be either a string or an integer.
type ProgressToken struct {
	value any
}

// NewProgressToken creates a new ProgressToken from a string or an integer.
func NewProgressToken[T interface{ string | int }](v T) ProgressToken {
	return ProgressToken{value: v}
}

// IsNull returns true if the ProgressToken is null.
func (id ProgressToken) IsNull() bool {
	return id.value == nil
}

// String returns the ProgressToken as a string.
// If the ProgressToken is an integer, it is converted to a string.
// If the ProgressToken is null, it returns an empty string.
func (id ProgressToken) String() string {
	switch v := id.value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case nil:
		return ""
	default:
		panic("invalid ProgressToken type")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (id *ProgressToken) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		id.value = v
	case float64:
		id.value = int(v)
	case nil:
		id.value = v
	default:
		return fmt.Errorf("invalid ProgressToken type: %T", v)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (id ProgressToken) MarshalJSON() ([]byte, error) {
	switch v := id.value.(type) {
	case string:
		return json.Marshal(v)
	case int:
		return json.Marshal(v)
	case nil:
		return json.Marshal(v)
	default:
		return nil, errors.New("invalid ProgressToken type")
	}
}

type Request[Params any] struct {
	Meta struct {
		ProgressToken ProgressToken
	}
	Params Params
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Request[Params]) MarshalJSON() ([]byte, error) {
	v := make(map[string]json.RawMessage)

	b, err := json.Marshal(r.Params)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	if !r.Meta.ProgressToken.IsNull() {
		data, err := json.Marshal(r.Meta)
		if err != nil {
			return nil, err
		}
		v["_meta"] = data
	}

	return json.Marshal(v)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Request[Params]) UnmarshalJSON(data []byte) error {
	v := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if data, ok := v["_meta"]; ok {
		if err := json.Unmarshal(data, &r.Meta); err != nil {
			return err
		}
	}

	delete(v, "_meta")
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.Params)
}
