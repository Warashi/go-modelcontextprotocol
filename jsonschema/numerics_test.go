package jsonschema

import (
	"encoding/json"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

func TestNumber_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Number
		value   json.RawMessage
		wantErr bool
	}{
		{
			name:   "valid number",
			schema: Number{},
			value:  json.RawMessage(`42.5`),
		},
		{
			name: "number within range",
			schema: Number{
				Minimum: ptr(10.0),
				Maximum: ptr(100.0),
			},
			value: json.RawMessage(`50.0`),
		},
		{
			name: "number at minimum",
			schema: Number{
				Minimum: ptr(10.0),
			},
			value: json.RawMessage(`10.0`),
		},
		{
			name: "number at maximum",
			schema: Number{
				Maximum: ptr(100.0),
			},
			value: json.RawMessage(`100.0`),
		},
		{
			name: "number below minimum",
			schema: Number{
				Minimum: ptr(10.0),
			},
			value:   json.RawMessage(`5.0`),
			wantErr: true,
		},
		{
			name: "number above maximum",
			schema: Number{
				Maximum: ptr(100.0),
			},
			value:   json.RawMessage(`150.0`),
			wantErr: true,
		},
		{
			name: "number at exclusive minimum",
			schema: Number{
				ExclusiveMinimum: ptr(10.0),
			},
			value:   json.RawMessage(`10.0`),
			wantErr: true,
		},
		{
			name: "number at exclusive maximum",
			schema: Number{
				ExclusiveMaximum: ptr(100.0),
			},
			value:   json.RawMessage(`100.0`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Number{},
			value:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Number.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInteger_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Integer
		value   json.RawMessage
		wantErr bool
	}{
		{
			name:   "valid integer",
			schema: Integer{},
			value:  json.RawMessage(`42`),
		},
		{
			name: "integer within range",
			schema: Integer{
				Minimum: ptr(int64(10)),
				Maximum: ptr(int64(100)),
			},
			value: json.RawMessage(`50`),
		},
		{
			name: "integer at minimum",
			schema: Integer{
				Minimum: ptr(int64(10)),
			},
			value: json.RawMessage(`10`),
		},
		{
			name: "integer at maximum",
			schema: Integer{
				Maximum: ptr(int64(100)),
			},
			value: json.RawMessage(`100`),
		},
		{
			name: "integer below minimum",
			schema: Integer{
				Minimum: ptr(int64(10)),
			},
			value:   json.RawMessage(`5`),
			wantErr: true,
		},
		{
			name: "integer above maximum",
			schema: Integer{
				Maximum: ptr(int64(100)),
			},
			value:   json.RawMessage(`150`),
			wantErr: true,
		},
		{
			name: "integer at exclusive minimum",
			schema: Integer{
				ExclusiveMinimum: ptr(int64(10)),
			},
			value:   json.RawMessage(`10`),
			wantErr: true,
		},
		{
			name: "integer at exclusive maximum",
			schema: Integer{
				ExclusiveMaximum: ptr(int64(100)),
			},
			value:   json.RawMessage(`100`),
			wantErr: true,
		},
		{
			name:    "non-integer number",
			schema:  Integer{},
			value:   json.RawMessage(`42.5`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			schema:  Integer{},
			value:   json.RawMessage(`invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Integer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumber_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Number
		want    string
		wantErr bool
	}{
		{
			name:   "empty schema",
			schema: Number{},
			want:   `{"type":"number"}`,
		},
		{
			name: "with description",
			schema: Number{
				Description: "test description",
			},
			want: `{"type":"number","description":"test description"}`,
		},
		{
			name: "with minimum and maximum",
			schema: Number{
				Minimum: ptr(10.0),
				Maximum: ptr(100.0),
			},
			want: `{"type":"number","minimum":10,"maximum":100}`,
		},
		{
			name: "with exclusive minimum and maximum",
			schema: Number{
				ExclusiveMinimum: ptr(10.0),
				ExclusiveMaximum: ptr(100.0),
			},
			want: `{"type":"number","exclusiveMinimum":10,"exclusiveMaximum":100}`,
		},
		{
			name: "with all fields",
			schema: Number{
				Description:      "test description",
				Minimum:          ptr(10.0),
				Maximum:          ptr(100.0),
				ExclusiveMinimum: ptr(20.0),
				ExclusiveMaximum: ptr(90.0),
			},
			want: `{"type":"number","description":"test description","minimum":10,"maximum":100,"exclusiveMinimum":20,"exclusiveMaximum":90}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Number.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare JSON strings after normalizing them
			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Errorf("failed to unmarshal got: %v", err)
				return
			}
			if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
				t.Errorf("failed to unmarshal want: %v", err)
				return
			}

			gotJSON, err := json.Marshal(gotMap)
			if err != nil {
				t.Errorf("failed to marshal got: %v", err)
				return
			}
			wantJSON, err := json.Marshal(wantMap)
			if err != nil {
				t.Errorf("failed to marshal want: %v", err)
				return
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Number.MarshalJSON() = %v, want %v", string(gotJSON), string(wantJSON))
			}
		})
	}
}

func TestInteger_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		schema  Integer
		want    string
		wantErr bool
	}{
		{
			name:   "empty schema",
			schema: Integer{},
			want:   `{"type":"integer"}`,
		},
		{
			name: "with description",
			schema: Integer{
				Description: "test description",
			},
			want: `{"type":"integer","description":"test description"}`,
		},
		{
			name: "with minimum and maximum",
			schema: Integer{
				Minimum: ptr(int64(10)),
				Maximum: ptr(int64(100)),
			},
			want: `{"type":"integer","minimum":10,"maximum":100}`,
		},
		{
			name: "with exclusive minimum and maximum",
			schema: Integer{
				ExclusiveMinimum: ptr(int64(10)),
				ExclusiveMaximum: ptr(int64(100)),
			},
			want: `{"type":"integer","exclusiveMinimum":10,"exclusiveMaximum":100}`,
		},
		{
			name: "with all fields",
			schema: Integer{
				Description:      "test description",
				Minimum:          ptr(int64(10)),
				Maximum:          ptr(int64(100)),
				ExclusiveMinimum: ptr(int64(20)),
				ExclusiveMaximum: ptr(int64(90)),
			},
			want: `{"type":"integer","description":"test description","minimum":10,"maximum":100,"exclusiveMinimum":20,"exclusiveMaximum":90}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.schema.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Integer.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Compare JSON strings after normalizing them
			var gotMap, wantMap map[string]interface{}
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Errorf("failed to unmarshal got: %v", err)
				return
			}
			if err := json.Unmarshal([]byte(tt.want), &wantMap); err != nil {
				t.Errorf("failed to unmarshal want: %v", err)
				return
			}

			gotJSON, err := json.Marshal(gotMap)
			if err != nil {
				t.Errorf("failed to marshal got: %v", err)
				return
			}
			wantJSON, err := json.Marshal(wantMap)
			if err != nil {
				t.Errorf("failed to marshal want: %v", err)
				return
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Integer.MarshalJSON() = %v, want %v", string(gotJSON), string(wantJSON))
			}
		})
	}
}
