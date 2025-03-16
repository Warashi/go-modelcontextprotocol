package mcp

import (
	"reflect"
	"testing"
)

func TestResult_MarshalJSON(t *testing.T) {
	type Params struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	tests := []struct {
		name         string
		notification Result[Params]
		want         string
		wantErr      bool
	}{
		{
			name: "with meta",
			notification: Result[Params]{
				Meta: map[string]any{"meta1": "value1"},
				Params: Params{
					Field1: "value1",
					Field2: 2,
				},
			},
			want:    `{"field1":"value1","field2":2,"_meta":{"meta1":"value1"}}`,
			wantErr: false,
		},
		{
			name: "without meta",
			notification: Result[Params]{
				Params: Params{
					Field1: "value1",
					Field2: 2,
				},
			},
			want:    `{"field1":"value1","field2":2}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.notification.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestResult_UnmarshalJSON(t *testing.T) {
	type Params struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2"`
	}

	tests := []struct {
		name    string
		data    string
		want    Result[Params]
		wantErr bool
	}{
		{
			name: "with meta",
			data: `{"field1":"value1","field2":2,"_meta":{"meta1":"value1"}}`,
			want: Result[Params]{
				Meta: map[string]any{"meta1": "value1"},
				Params: Params{
					Field1: "value1",
					Field2: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "without meta",
			data: `{"field1":"value1","field2":2}`,
			want: Result[Params]{
				Params: Params{
					Field1: "value1",
					Field2: 2,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Result[Params]
			if err := got.UnmarshalJSON([]byte(tt.data)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
