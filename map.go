package qwery

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/redhajuanda/qwery/vars"
)

// StructToMapQwery converts a struct to map[string]any using qwery tags
func StructToMap(obj any) JSONMap {
	return structToMapWithTag(obj, vars.TagKey)
}

// structToMapWithTag is the core function that handles the conversion
func structToMapWithTag(obj any, tagName string) JSONMap {
	result := make(map[string]any)

	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	// Handle pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		t = t.Elem()
	}

	// Only process structs
	if v.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get tag value
		tagValue := field.Tag.Get(tagName)
		if tagValue == "" || tagValue == "-" {
			continue
		}

		// Get the actual value
		value := fieldValue.Interface()

		// Handle different types
		switch fieldValue.Kind() {
		case reflect.Ptr:
			if fieldValue.IsNil() {
				result[tagValue] = nil
			} else {
				// Handle pointer to time.Time specially
				if fieldValue.Elem().Type() == reflect.TypeOf(time.Time{}) {
					result[tagValue] = fieldValue.Elem().Interface()
				} else if fieldValue.Elem().Kind() == reflect.Struct {
					// For other pointer to struct, convert recursively
					result[tagValue] = structToMapWithTag(value, tagName)
				} else {
					result[tagValue] = fieldValue.Elem().Interface()
				}
			}
		case reflect.Struct:
			// Handle time.Time specially (don't convert to map)
			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				result[tagValue] = value
			} else {
				// Convert nested struct recursively
				result[tagValue] = structToMapWithTag(value, tagName)
			}
		case reflect.Slice:
			if fieldValue.Len() == 0 {
				result[tagValue] = []any{}
			} else {
				// Handle slice of structs
				sliceResult := make([]any, fieldValue.Len())
				for j := 0; j < fieldValue.Len(); j++ {
					item := fieldValue.Index(j).Interface()
					itemValue := reflect.ValueOf(item)

					if itemValue.Kind() == reflect.Struct {
						sliceResult[j] = structToMapWithTag(item, tagName)
					} else {
						sliceResult[j] = item
					}
				}
				result[tagValue] = sliceResult
			}
		default:
			result[tagValue] = value
		}
	}

	return result
}

// MapToStruct converts a map[string]any to struct using qwery tags
func MapToStruct(m JSONMap, target any) error {
	return mapToStructWithTag(m, target, vars.TagKey)
}

// mapToStructWithTag is the core function that handles the conversion from map to struct
func mapToStructWithTag(m map[string]any, target any, tagName string) error {
	if m == nil {
		return nil
	}

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer to struct")
	}

	// Get the underlying struct value
	v = v.Elem()
	t := v.Type()

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	// Build tag to field index mapping for faster lookup
	tagToField := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagValue := field.Tag.Get(tagName)
		if tagValue != "" && tagValue != "-" {
			tagToField[tagValue] = i
		}
	}

	// Iterate through map and set struct fields
	for key, value := range m {
		fieldIndex, exists := tagToField[key]
		if !exists {
			continue // Skip if no corresponding struct field
		}

		field := t.Field(fieldIndex)
		fieldValue := v.Field(fieldIndex)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		if err := setFieldValue(fieldValue, field, value, tagName); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

// setFieldValue sets the value of a struct field based on its type
func setFieldValue(fieldValue reflect.Value, field reflect.StructField, value any, tagName string) error {
	if value == nil {
		// Handle nil values
		if fieldValue.Kind() == reflect.Ptr {
			fieldValue.Set(reflect.Zero(fieldValue.Type()))
		}
		return nil
	}

	switch fieldValue.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			fieldValue.SetString(str)
		} else {
			return fmt.Errorf("expected string, got %T", value)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, ok := convertToInt64(value); ok {
			fieldValue.SetInt(num)
		} else {
			return fmt.Errorf("cannot convert %T to int", value)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if num, ok := convertToUint64(value); ok {
			fieldValue.SetUint(num)
		} else {
			return fmt.Errorf("cannot convert %T to uint", value)
		}

	case reflect.Float32, reflect.Float64:
		if num, ok := convertToFloat64(value); ok {
			fieldValue.SetFloat(num)
		} else {
			return fmt.Errorf("cannot convert %T to float", value)
		}

	case reflect.Bool:
		if b, ok := value.(bool); ok {
			fieldValue.SetBool(b)
		} else {
			return fmt.Errorf("expected bool, got %T", value)
		}

	case reflect.Ptr:
		return setPointerValue(fieldValue, field, value, tagName)

	case reflect.Struct:
		return setStructValue(fieldValue, field, value, tagName)

	case reflect.Slice:
		return setSliceValue(fieldValue, field, value, tagName)

	default:
		// For other types, try direct assignment
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().AssignableTo(fieldValue.Type()) {
			fieldValue.Set(valueReflect)
		} else {
			return fmt.Errorf("cannot assign %T to %s", value, fieldValue.Type())
		}
	}

	return nil
}

// setPointerValue handles pointer field types
func setPointerValue(fieldValue reflect.Value, field reflect.StructField, value any, tagName string) error {
	if value == nil {
		fieldValue.Set(reflect.Zero(fieldValue.Type()))
		return nil
	}

	// Create new instance of the pointed-to type
	elemType := fieldValue.Type().Elem()
	newElem := reflect.New(elemType)

	if elemType.Kind() == reflect.Struct {
		// Handle time.Time specially
		if elemType == reflect.TypeOf(time.Time{}) {
			if timeVal, ok := value.(time.Time); ok {
				newElem.Elem().Set(reflect.ValueOf(timeVal))
				fieldValue.Set(newElem)
				return nil
			}
			// Try to parse time from string if needed
			if timeStr, ok := value.(string); ok {
				if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
					newElem.Elem().Set(reflect.ValueOf(parsedTime))
					fieldValue.Set(newElem)
					return nil
				}
			}
			return fmt.Errorf("cannot convert %T to *time.Time", value)
		}

		// For other structs, convert recursively
		if valueMap, ok := value.(map[string]any); ok {
			if err := mapToStructWithTag(valueMap, newElem.Interface(), tagName); err != nil {
				return err
			}
			fieldValue.Set(newElem)
			return nil
		}

		// Also try JSONMap type
		if valueMap, ok := value.(JSONMap); ok {
			if err := mapToStructWithTag(valueMap, newElem.Interface(), tagName); err != nil {
				return err
			}
			fieldValue.Set(newElem)
			return nil
		}
		return fmt.Errorf("expected map for struct pointer, got %T", value)
	} else {
		// For primitive pointers, set the value directly
		if err := setFieldValue(newElem.Elem(), field, value, tagName); err != nil {
			return err
		}
		fieldValue.Set(newElem)
	}

	return nil
}

// setStructValue handles struct field types
func setStructValue(fieldValue reflect.Value, field reflect.StructField, value any, tagName string) error {
	// Handle time.Time specially
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		if timeVal, ok := value.(time.Time); ok {
			fieldValue.Set(reflect.ValueOf(timeVal))
			return nil
		}
		// Try to parse time from string
		if timeStr, ok := value.(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
				fieldValue.Set(reflect.ValueOf(parsedTime))
				return nil
			}
		}
		return fmt.Errorf("cannot convert %T to time.Time", value)
	}

	// For other structs, expect a map
	if valueMap, ok := value.(map[string]any); ok {
		// Create a pointer to the struct for the recursive call
		structPtr := reflect.New(fieldValue.Type())
		if err := mapToStructWithTag(valueMap, structPtr.Interface(), tagName); err != nil {
			return err
		}
		fieldValue.Set(structPtr.Elem())
		return nil
	}

	// Also try JSONMap type
	if valueMap, ok := value.(JSONMap); ok {
		// Create a pointer to the struct for the recursive call
		structPtr := reflect.New(fieldValue.Type())
		if err := mapToStructWithTag(valueMap, structPtr.Interface(), tagName); err != nil {
			return err
		}
		fieldValue.Set(structPtr.Elem())
		return nil
	}

	return fmt.Errorf("expected map for struct, got %T", value)
}

// setSliceValue handles slice field types
func setSliceValue(fieldValue reflect.Value, field reflect.StructField, value any, tagName string) error {
	valueSlice, ok := value.([]any)
	if !ok {
		return fmt.Errorf("expected []any for slice, got %T", value)
	}

	sliceLen := len(valueSlice)
	newSlice := reflect.MakeSlice(fieldValue.Type(), sliceLen, sliceLen)

	elemType := fieldValue.Type().Elem()

	for i, item := range valueSlice {
		elemValue := newSlice.Index(i)

		if elemType.Kind() == reflect.Struct {
			// Handle slice of structs
			if itemMap, ok := item.(map[string]any); ok {
				elemPtr := reflect.New(elemType)
				if err := mapToStructWithTag(itemMap, elemPtr.Interface(), tagName); err != nil {
					return fmt.Errorf("failed to convert slice element %d: %w", i, err)
				}
				elemValue.Set(elemPtr.Elem())
			} else if itemMap, ok := item.(JSONMap); ok {
				// Also try JSONMap type
				elemPtr := reflect.New(elemType)
				if err := mapToStructWithTag(itemMap, elemPtr.Interface(), tagName); err != nil {
					return fmt.Errorf("failed to convert slice element %d: %w", i, err)
				}
				elemValue.Set(elemPtr.Elem())
			} else {
				return fmt.Errorf("expected map for struct slice element, got %T", item)
			}
		} else {
			// Handle slice of primitives
			dummyField := reflect.StructField{Type: elemType}
			if err := setFieldValue(elemValue, dummyField, item, tagName); err != nil {
				return fmt.Errorf("failed to set slice element %d: %w", i, err)
			}
		}
	}

	fieldValue.Set(newSlice)
	return nil
}

// Helper functions for type conversion
func convertToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

func convertToUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case uint:
		return uint64(v), true
	case uint8:
		return uint64(v), true
	case uint16:
		return uint64(v), true
	case uint32:
		return uint64(v), true
	case uint64:
		return v, true
	case int:
		if v >= 0 {
			return uint64(v), true
		}
	case int8:
		if v >= 0 {
			return uint64(v), true
		}
	case int16:
		if v >= 0 {
			return uint64(v), true
		}
	case int32:
		if v >= 0 {
			return uint64(v), true
		}
	case int64:
		if v >= 0 {
			return uint64(v), true
		}
	case float32:
		if v >= 0 {
			return uint64(v), true
		}
	case float64:
		if v >= 0 {
			return uint64(v), true
		}
	case string:
		if i, err := strconv.ParseUint(v, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

func convertToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
