package jsonschema

import (
	"fmt"
	"reflect"
	"strings"
)

// SchemaTag represents the struct tag for JSON schema configuration
type SchemaTag struct {
	Description string
	Required    bool
}

// ParseSchemaTag parses the jsonschema tag and returns SchemaTag
func ParseSchemaTag(tag reflect.StructTag) SchemaTag {
	t := tag.Get("jsonschema")
	if t == "" {
		return SchemaTag{}
	}

	parts := strings.Split(t, ",")
	st := SchemaTag{}
	for _, p := range parts {
		if p == "required" {
			st.Required = true
			continue
		}
		if strings.HasPrefix(p, "description=") {
			st.Description = strings.TrimPrefix(p, "description=")
		}
	}
	return st
}

// FromStruct generates a JSON schema Object from a struct type
func FromStruct(v any) (Object, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Object{}, fmt.Errorf("input must be a struct or pointer to struct")
	}

	obj := Object{
		Properties: make(map[string]Schema),
	}

	var required []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}

		// Parse schema tag
		schemaTag := ParseSchemaTag(field.Tag)
		if schemaTag.Required {
			required = append(required, name)
		}

		// Generate schema for the field
		schema, err := generateSchema(field.Type, schemaTag)
		if err != nil {
			return Object{}, fmt.Errorf("field %s: %w", field.Name, err)
		}

		obj.Properties[name] = schema
	}

	if len(required) > 0 {
		obj.Required = required
	}

	return obj, nil
}

// generateSchema generates a Schema for the given type
func generateSchema(t reflect.Type, tag SchemaTag) (Schema, error) {
	switch t.Kind() {
	case reflect.String:
		return String{Description: tag.Description}, nil
	case reflect.Bool:
		return Boolean{Description: tag.Description}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return Integer{Description: tag.Description}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Integer{Description: tag.Description}, nil
	case reflect.Float32, reflect.Float64:
		return Number{Description: tag.Description}, nil
	case reflect.Slice, reflect.Array:
		items, err := generateSchema(t.Elem(), SchemaTag{})
		if err != nil {
			return nil, fmt.Errorf("array items: %w", err)
		}
		return Array{Items: items, Description: tag.Description}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string")
		}
		additionalProperties, err := generateSchema(t.Elem(), SchemaTag{})
		if err != nil {
			return nil, fmt.Errorf("map values: %w", err)
		}
		return Map{AdditionalProperties: additionalProperties, Description: tag.Description}, nil
	case reflect.Struct:
		obj, err := FromStruct(reflect.New(t).Interface())
		if err != nil {
			return nil, fmt.Errorf("nested struct: %w", err)
		}
		obj.Description = tag.Description
		return obj, nil
	case reflect.Ptr:
		return generateSchema(t.Elem(), tag)
	default:
		return nil, fmt.Errorf("unsupported type: %s", t.Kind())
	}
}
