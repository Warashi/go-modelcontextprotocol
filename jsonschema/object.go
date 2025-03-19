package jsonschema

import (
	"encoding/json"
	"fmt"
)

// Object is a JSON schema object.
type Object struct {
	Properties map[string]Schema
	Required   []string
}

// MarshalSchema marshals the object into a JSON schema.
func (o *Object) MarshalSchema() (json.RawMessage, error) {
	for _, r := range o.Required {
		if _, ok := o.Properties[r]; !ok {
			return nil, fmt.Errorf("required property %s not found", r)
		}
	}

	obj := map[string]any{
		"type": "object",
	}

	if len(o.Required) > 0 {
		obj["required"] = o.Required
	}

	if len(o.Properties) > 0 {
		properties := make(map[string]any)
		for k, v := range o.Properties {
			var err error
			properties[k], err = v.MarshalSchema()
			if err != nil {
				return nil, err
			}
		}
		obj["properties"] = properties
	}

	return json.Marshal(obj)
}
