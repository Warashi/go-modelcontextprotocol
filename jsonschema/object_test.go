package jsonschema_test

import (
	"testing"

	"github.com/Warashi/go-modelcontextprotocol/jsonschema"
)

func TestObject_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		object  jsonschema.Object
		want    string
		wantErr bool
	}{
		{
			name: "valid object",
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{
					"prop1": &jsonschema.String{},
				},
				Required: []string{"prop1"},
			},
			want:    `{"properties":{"prop1":{"type":"string"}},"required":["prop1"],"type":"object"}`,
			wantErr: false,
		},
		{
			name: "missing required property",
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{},
				Required:   []string{"prop1"},
			},
			want:    "",
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
			if got := string(got); got != tt.want {
				t.Errorf("Object.MarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		object  jsonschema.Object
		value   any
		wantErr bool
	}{
		{
			name: "valid object",
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{
					"prop1": &jsonschema.String{},
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
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{
					"prop1": &jsonschema.String{},
				},
				Required: []string{"prop1"},
			},
			value:   map[string]any{},
			wantErr: true,
		},
		{
			name: "unexpected property",
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{
					"prop1": &jsonschema.String{},
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
			object: jsonschema.Object{
				Properties: map[string]jsonschema.Schema{
					"prop1": &jsonschema.String{},
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
			if err := tt.object.Validate(tt.value); (err != nil) != tt.wantErr {
				t.Errorf("Object.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
