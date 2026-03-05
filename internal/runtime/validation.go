package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"rsps/internal/ast"
)

func ValidatePayload(entity *EntityMeta, payload map[string]any, create bool) error {
	for key := range payload {
		if _, ok := entity.Field(key); !ok {
			return fmt.Errorf("unknown field '%s' for entity '%s'", key, entity.Name)
		}
	}

	if create {
		for _, field := range entity.Fields {
			if field.Nullable || field.Default != nil {
				continue
			}
			if _, ok := payload[field.Name]; !ok {
				return fmt.Errorf("missing required field '%s' for entity '%s'", field.Name, entity.Name)
			}
		}
	}

	for key, value := range payload {
		field, _ := entity.Field(key)
		if value == nil {
			if !field.Nullable {
				return fmt.Errorf("field '%s' in entity '%s' cannot be null", field.Name, entity.Name)
			}
			continue
		}
		if err := validateType(entity.Name, field, value); err != nil {
			return err
		}
	}

	return nil
}

func ParseStringValue(field *FieldMeta, raw string) (any, error) {
	if raw == "" {
		if field.Nullable {
			return nil, nil
		}
		return "", nil
	}

	switch field.Type {
	case ast.TypeString, ast.TypeText, ast.TypeDate, ast.TypeDatetime, ast.TypeJSON:
		return raw, nil
	case ast.TypeInt, ast.TypeRef:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("field '%s' expects integer", field.Name)
		}
		return value, nil
	case ast.TypeFloat:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("field '%s' expects float", field.Name)
		}
		return value, nil
	case ast.TypeBool:
		lower := strings.ToLower(raw)
		switch lower {
		case "true", "1", "on", "yes":
			return true, nil
		case "false", "0", "off", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("field '%s' expects bool", field.Name)
		}
	default:
		return raw, nil
	}
}

func validateType(entityName string, field *FieldMeta, value any) error {
	switch field.Type {
	case ast.TypeString, ast.TypeText, ast.TypeDate, ast.TypeDatetime, ast.TypeJSON:
		if _, ok := value.(string); ok {
			return nil
		}
		return fmt.Errorf("field '%s' in entity '%s' expects string value", field.Name, entityName)
	case ast.TypeInt, ast.TypeRef:
		if isIntegerLike(value) {
			return nil
		}
		return fmt.Errorf("field '%s' in entity '%s' expects integer value", field.Name, entityName)
	case ast.TypeFloat:
		if isFloatLike(value) {
			return nil
		}
		return fmt.Errorf("field '%s' in entity '%s' expects float value", field.Name, entityName)
	case ast.TypeBool:
		if _, ok := value.(bool); ok {
			return nil
		}
		return fmt.Errorf("field '%s' in entity '%s' expects boolean value", field.Name, entityName)
	default:
		return nil
	}
}

func isIntegerLike(value any) bool {
	switch candidate := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return candidate == float64(int64(candidate))
	case float32:
		return float64(candidate) == float64(int64(candidate))
	case string:
		_, err := strconv.ParseInt(candidate, 10, 64)
		return err == nil
	default:
		return false
	}
}

func isFloatLike(value any) bool {
	switch candidate := value.(type) {
	case float32, float64:
		return true
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseFloat(candidate, 64)
		return err == nil
	default:
		return false
	}
}
