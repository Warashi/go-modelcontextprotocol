package jsonschema

import (
	"encoding/json"
	"reflect"
	"testing"
)

type testStruct struct {
	String     string         `json:"string" jsonschema:"required,description=A string field"`
	Int        int            `json:"int" jsonschema:"description=An integer field"`
	Bool       bool           `json:"bool"`
	Float      float64        `json:"float"`
	Slice      []string       `json:"slice" jsonschema:"description=A slice of strings"`
	Map        map[string]int `json:"map"`
	Nested     nestedStruct   `json:"nested"`
	unexported string
}

type nestedStruct struct {
	Value string `json:"value" jsonschema:"required"`
}

func TestFromStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    Object
		wantErr bool
	}{
		{
			name:  "basic struct",
			input: testStruct{},
			want: Object{
				Properties: map[string]Schema{
					"string": String{Description: "A string field"},
					"int":    Integer{Description: "An integer field"},
					"bool":   Boolean{},
					"float":  Number{},
					"slice":  Array{Items: String{}, Description: "A slice of strings"},
					"map":    Map{AdditionalProperties: Integer{}},
					"nested": Object{
						Properties: map[string]Schema{
							"value": String{},
						},
						Required: []string{"value"},
					},
				},
				Required: []string{"string"},
			},
		},
		{
			name:    "non-struct input",
			input:   "not a struct",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare JSON representations to check equality
			wantJSON, err := json.Marshal(tt.want)
			if err != nil {
				t.Errorf("Failed to marshal want: %v", err)
				return
			}
			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Errorf("Failed to marshal got: %v", err)
				return
			}

			if string(wantJSON) != string(gotJSON) {
				t.Errorf("FromStruct() = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestParseSchemaTag(t *testing.T) {
	tests := []struct {
		name string
		tag  reflect.StructTag
		want SchemaTag
	}{
		{
			name: "empty tag",
			tag:  ``,
			want: SchemaTag{},
		},
		{
			name: "required only",
			tag:  `jsonschema:"required"`,
			want: SchemaTag{Required: true},
		},
		{
			name: "description only",
			tag:  `jsonschema:"description=test description"`,
			want: SchemaTag{Description: "test description"},
		},
		{
			name: "both required and description",
			tag:  `jsonschema:"required,description=test description"`,
			want: SchemaTag{Required: true, Description: "test description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSchemaTag(tt.tag)
			if got.Required != tt.want.Required || got.Description != tt.want.Description {
				t.Errorf("ParseSchemaTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
