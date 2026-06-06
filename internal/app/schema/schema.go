// Package schema is a pure JSON-Schema-subset validator. It is a direct port of
// the Clojure reference (validation.clj) with two deliberate fixes:
//
//   - "integer" accepts whole-valued floats (JSON numbers decode to float64, so
//     1.0 must count as an integer);
//   - "type" may be an array (e.g. ["string","null"]), validated as any-of.
//
// The package has no dependencies on other app layers.
package schema

import (
	"math"
	"strings"
)

// Validate checks data against a JSON-Schema-like contract and returns a map of
// {field: reason}. An empty (non-nil) map means valid. A nil or empty schema is
// a pass-through and always validates.
//
// It enforces "required" keys and the "type" of fields declared under
// "properties" that are present in data.
func Validate(schema map[string]any, data map[string]any) map[string]string {
	errs := map[string]string{}
	if len(schema) == 0 {
		return errs // pass-through
	}

	for _, field := range stringSlice(schema["required"]) {
		if _, ok := data[field]; !ok {
			errs[field] = "missing required key"
		}
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return errs
	}
	for field, sub := range props {
		subMap, ok := sub.(map[string]any)
		if !ok {
			continue
		}
		t, hasType := subMap["type"]
		if !hasType {
			continue
		}
		val, present := data[field]
		if !present {
			continue
		}
		if !typeMatches(t, val) {
			errs[field] = "expected " + typeName(t)
		}
	}
	return errs
}

// typeMatches reports whether val satisfies a "type", which may be a single
// type name (string) or an array of names validated as any-of.
func typeMatches(t, val any) bool {
	switch tt := t.(type) {
	case string:
		return typeOK(tt, val)
	case []any:
		for _, e := range tt {
			if s, ok := e.(string); ok && typeOK(s, val) {
				return true
			}
		}
		return false
	case []string:
		for _, s := range tt {
			if typeOK(s, val) {
				return true
			}
		}
		return false
	}
	return true // unknown type representation -> don't fail
}

// typeOK reports whether value matches a single JSON Schema "type" name.
func typeOK(expected string, value any) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := asNumber(value)
		return ok
	case "integer":
		f, ok := asNumber(value)
		return ok && f == math.Trunc(f) // accept whole-valued floats (1.0)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "null":
		return value == nil
	default:
		return true // unknown/absent type -> don't fail
	}
}

// asNumber reports whether v is a JSON-ish number and returns its float value.
// JSON decodes numbers to float64; the integer cases cover programmatic callers.
func asNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// typeName renders a "type" for an error message ("string", "string or null").
func typeName(t any) string {
	switch tt := t.(type) {
	case string:
		return tt
	case []any:
		parts := make([]string, 0, len(tt))
		for _, e := range tt {
			if s, ok := e.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " or ")
	case []string:
		return strings.Join(tt, " or ")
	}
	return "valid value"
}

// stringSlice coerces a "required" value into a []string. It accepts both
// []any (from decoded JSON) and []string (programmatic callers).
func stringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}
