package validator

import (
	"fmt"
	"strconv"
	"strings"

	"rsps/internal/ast"
)

type ErrorList struct {
	Errors []error
}

func (e *ErrorList) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var builder strings.Builder
	builder.WriteString("validation errors:\n")
	for _, err := range e.Errors {
		builder.WriteString(" - ")
		builder.WriteString(err.Error())
		builder.WriteByte('\n')
	}
	return builder.String()
}

func Validate(app *ast.Application) error {
	errors := make([]error, 0)

	entityNames := make(map[string]struct{}, len(app.Entities))
	for _, entity := range app.Entities {
		if entity.Name == "" {
			errors = append(errors, fmt.Errorf("entity at %d:%d is missing a name", entity.Pos.Line, entity.Pos.Column))
			continue
		}
		if _, exists := entityNames[entity.Name]; exists {
			errors = append(errors, fmt.Errorf("duplicate entity '%s' at %d:%d", entity.Name, entity.Pos.Line, entity.Pos.Column))
			continue
		}
		entityNames[entity.Name] = struct{}{}

		fieldNames := make(map[string]struct{}, len(entity.Fields))
		for _, field := range entity.Fields {
			if field.Name == "" {
				errors = append(errors, fmt.Errorf("entity '%s' has unnamed field at %d:%d", entity.Name, field.Pos.Line, field.Pos.Column))
				continue
			}
			if _, exists := fieldNames[field.Name]; exists {
				errors = append(errors, fmt.Errorf("duplicate field '%s' in entity '%s' at %d:%d", field.Name, entity.Name, field.Pos.Line, field.Pos.Column))
				continue
			}
			fieldNames[field.Name] = struct{}{}

			if field.Type != ast.TypeRef && !isSupportedType(field.Type) {
				errors = append(errors, fmt.Errorf("invalid type '%s' for field '%s' in entity '%s' at %d:%d", field.Type, field.Name, entity.Name, field.Pos.Line, field.Pos.Column))
			}

			if field.Type == ast.TypeRef {
				if field.Reference == nil || field.Reference.Entity == "" {
					errors = append(errors, fmt.Errorf("field '%s' in entity '%s' has invalid reference at %d:%d", field.Name, entity.Name, field.Pos.Line, field.Pos.Column))
				}
			}

			if err := validateDefaultValue(entity, field); err != nil {
				errors = append(errors, err)
			}

			if hasDuplicateAttributes(field.Attributes) {
				errors = append(errors, fmt.Errorf("field '%s' in entity '%s' has duplicate attributes at %d:%d", field.Name, entity.Name, field.Pos.Line, field.Pos.Column))
			}
		}
	}

	for _, entity := range app.Entities {
		for _, field := range entity.Fields {
			if field.Type != ast.TypeRef || field.Reference == nil {
				continue
			}
			if _, exists := entityNames[field.Reference.Entity]; !exists {
				errors = append(errors, fmt.Errorf("field '%s' in entity '%s' references missing entity '%s' at %d:%d", field.Name, entity.Name, field.Reference.Entity, field.Reference.Pos.Line, field.Reference.Pos.Column))
			}
		}
	}

	if len(errors) > 0 {
		return &ErrorList{Errors: errors}
	}
	return nil
}

func isSupportedType(fieldType ast.FieldType) bool {
	switch fieldType {
	case ast.TypeString, ast.TypeText, ast.TypeInt, ast.TypeFloat, ast.TypeBool, ast.TypeDate, ast.TypeDatetime, ast.TypeJSON:
		return true
	default:
		return false
	}
}

func hasDuplicateAttributes(attributes []ast.FieldAttribute) bool {
	seen := make(map[ast.FieldAttribute]struct{}, len(attributes))
	for _, attr := range attributes {
		if _, exists := seen[attr]; exists {
			return true
		}
		seen[attr] = struct{}{}
	}
	return false
}

func validateDefaultValue(entity *ast.Entity, field *ast.Field) error {
	if field.Default == nil {
		return nil
	}

	if field.Type == ast.TypeRef {
		return fmt.Errorf("field '%s' in entity '%s' cannot define default for reference type at %d:%d", field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
	}

	switch field.Type {
	case ast.TypeInt:
		if _, err := strconv.Atoi(field.Default.Value); err != nil {
			return fmt.Errorf("default value '%s' for field '%s' in entity '%s' is not a valid int at %d:%d", field.Default.Value, field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
		}
	case ast.TypeFloat:
		if _, err := strconv.ParseFloat(field.Default.Value, 64); err != nil {
			return fmt.Errorf("default value '%s' for field '%s' in entity '%s' is not a valid float at %d:%d", field.Default.Value, field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
		}
	case ast.TypeBool:
		if field.Default.Kind != "bool" {
			return fmt.Errorf("default value for field '%s' in entity '%s' must be true/false at %d:%d", field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
		}
	case ast.TypeDatetime:
		if field.Default.Kind != "string" && field.Default.Kind != "now" {
			return fmt.Errorf("default value for field '%s' in entity '%s' must be string or now at %d:%d", field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
		}
	case ast.TypeDate, ast.TypeString, ast.TypeText, ast.TypeJSON:
		if field.Default.Kind != "string" && field.Default.Kind != "identifier" {
			return fmt.Errorf("default value for field '%s' in entity '%s' must be string-compatible at %d:%d", field.Name, entity.Name, field.Default.Pos.Line, field.Default.Pos.Column)
		}
	}

	return nil
}
