package schema

import (
	"reflect"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		data   map[string]any
		want   map[string]string
	}{
		{
			name:   "missing required key",
			schema: map[string]any{"required": []any{"ticket", "repo"}},
			data:   map[string]any{"ticket": "MED2-5322"},
			want:   map[string]string{"repo": "missing required key"},
		},
		{
			name:   "all required present",
			schema: map[string]any{"required": []any{"ticket", "repo"}},
			data:   map[string]any{"ticket": "X", "repo": "Y"},
			want:   map[string]string{},
		},
		{
			name:   "wrong type string",
			schema: map[string]any{"properties": map[string]any{"ticket": map[string]any{"type": "string"}}},
			data:   map[string]any{"ticket": 42},
			want:   map[string]string{"ticket": "expected string"},
		},
		{
			name:   "integer accepts whole-valued float 1.0",
			schema: map[string]any{"properties": map[string]any{"n": map[string]any{"type": "integer"}}},
			data:   map[string]any{"n": 1.0},
			want:   map[string]string{},
		},
		{
			name:   "integer rejects 1.5",
			schema: map[string]any{"properties": map[string]any{"n": map[string]any{"type": "integer"}}},
			data:   map[string]any{"n": 1.5},
			want:   map[string]string{"n": "expected integer"},
		},
		{
			name:   "union string|null accepts string",
			schema: map[string]any{"properties": map[string]any{"x": map[string]any{"type": []any{"string", "null"}}}},
			data:   map[string]any{"x": "hello"},
			want:   map[string]string{},
		},
		{
			name:   "union string|null accepts null",
			schema: map[string]any{"properties": map[string]any{"x": map[string]any{"type": []any{"string", "null"}}}},
			data:   map[string]any{"x": nil},
			want:   map[string]string{},
		},
		{
			name:   "union string|null rejects number",
			schema: map[string]any{"properties": map[string]any{"x": map[string]any{"type": []any{"string", "null"}}}},
			data:   map[string]any{"x": 42},
			want:   map[string]string{"x": "expected string or null"},
		},
		{
			name:   "nil schema is pass-through",
			schema: nil,
			data:   map[string]any{"anything": 1},
			want:   map[string]string{},
		},
		{
			name:   "empty schema is pass-through",
			schema: map[string]any{},
			data:   map[string]any{"anything": 1},
			want:   map[string]string{},
		},
		{
			name: "required and type together",
			schema: map[string]any{
				"required": []any{"ticket"},
				"properties": map[string]any{
					"ticket": map[string]any{"type": "string"},
					"count":  map[string]any{"type": "integer"},
				},
			},
			data: map[string]any{"ticket": "X", "count": "not-a-number"},
			want: map[string]string{"count": "expected integer"},
		},
		{
			name:   "type check skipped for absent optional field",
			schema: map[string]any{"properties": map[string]any{"count": map[string]any{"type": "integer"}}},
			data:   map[string]any{},
			want:   map[string]string{},
		},
		{
			name: "boolean array object number all valid",
			schema: map[string]any{"properties": map[string]any{
				"b": map[string]any{"type": "boolean"},
				"a": map[string]any{"type": "array"},
				"o": map[string]any{"type": "object"},
				"f": map[string]any{"type": "number"},
			}},
			data: map[string]any{
				"b": true,
				"a": []any{1, 2},
				"o": map[string]any{"k": "v"},
				"f": 1.5,
			},
			want: map[string]string{},
		},
		{
			name:   "unknown type does not fail",
			schema: map[string]any{"properties": map[string]any{"x": map[string]any{"type": "weird"}}},
			data:   map[string]any{"x": 123},
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.schema, tt.data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
