package mcp

import "encoding/json"

type Notification[Params any] struct {
	Meta   map[string]any
	Params Params
}

// MarshalJSON implements the json.Marshaler interface.
func (r *Notification[Params]) MarshalJSON() ([]byte, error) {
	v := make(map[string]json.RawMessage)

	b, err := json.Marshal(r.Params)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	if r.Meta != nil {
		data, err := json.Marshal(r.Meta)
		if err != nil {
			return nil, err
		}
		v["_meta"] = data
	}

	return json.Marshal(v)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *Notification[Params]) UnmarshalJSON(data []byte) error {
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
