package jsonschema

import (
	"testing"
)

func TestObject_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		object  Object
		want    string
		wantErr bool
	}{
		{
			name: "valid object",
			object: Object{
				Properties: map[string]Schema{
					"prop1": String{},
				},
				Required: []string{"prop1"},
			},
			want:    `{"additionalProperties":false,"properties":{"prop1":{"type":"string"}},"required":["prop1"],"type":"object"}`,
			wantErr: false,
		},
		{
			name: "missing required property",
			object: Object{
				Properties: map[string]Schema{},
				Required:   []string{"prop1"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.object.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Object.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestObject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		object  Object
		value   any
		wantErr bool
	}{
		{
			name: "valid object",
			object: Object{
				Properties: map[string]Schema{
					"prop1": String{},
				},
				Required: []string{"prop1"},
			},
			value: map[string]any{
				"prop1": "value1",
			},
			wantErr: false,
		},
		{
			name: "missing required property",
			object: Object{
				Properties: map[string]Schema{
					"prop1": String{},
				},
				Required: []string{"prop1"},
			},
			value:   map[string]any{},
			wantErr: true,
		},
		{
			name: "unexpected property",
			object: Object{
				Properties: map[string]Schema{
					"prop1": String{},
				},
				Required: []string{"prop1"},
			},
			value: map[string]any{
				"prop1": "value1",
				"prop2": "value2",
			},
			wantErr: true,
		},
		{
			name: "invalid property value",
			object: Object{
				Properties: map[string]Schema{
					"prop1": String{},
				},
				Required: []string{"prop1"},
			},
			value: map[string]any{
				"prop1": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.object.validate(tt.value); (err != nil) != tt.wantErr {
				t.Errorf("Object.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
