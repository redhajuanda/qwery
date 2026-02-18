package qwery

import (
	"fmt"
	"reflect"
	"strconv"
)

// isStruct checks if the given interface is a struct or not
func isStruct(i any) bool {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem() // Dereference the pointer
	}
	return t.Kind() == reflect.Struct
}

func StringValue(v interface{}) string {
	if v == nil {
		return ""
	}

	// Use reflection to examine the value
	val := reflect.ValueOf(v)

	// Handle different kinds of values
	switch val.Kind() {
	case reflect.Ptr:
		// Check if pointer is nil
		if val.IsNil() {
			return ""
		}
		// Recursively call with the dereferenced value
		return StringValue(val.Elem().Interface())

	case reflect.Interface:
		// Check if interface is nil
		if val.IsNil() {
			return ""
		}
		// Recursively call with the actual value in the interface
		return StringValue(val.Elem().Interface())

	case reflect.String:
		return val.String()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10)

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64)

	case reflect.Bool:
		return strconv.FormatBool(val.Bool())

	default:
		// For complex types (structs, maps, etc.), use fmt.Sprint
		return fmt.Sprint(v)
	}
}

// ToPointer converts a value T to a pointer.
func ToPointer[T any](v T) *T {
	return &v
}

// ToPointerUnsafe is a helper function to convert any value to pointer.
// This function is unsafe because it will return nil if the value is empty/zero value.
func ToPointerUnsafe[T any](v T) *T {
	if reflect.ValueOf(v).IsZero() {
		return nil
	}
	return &v
}

// ToPointerUnsafeInterface is a helper function to convert any value to pointer.
// This function is unsafe because it will return nil if the value is empty/zero value.
// This function is used when the value is an interface, but the type is known.
func ToPointerUnsafeInterface[T any](v interface{}) *T {
	if v == nil {
		return nil
	}

	// Use reflection to handle zero value checks
	val, ok := v.(T)
	if !ok {
		return nil
	}

	// Create a zero value of type T using reflection
	if reflect.ValueOf(val).IsZero() {
		return nil
	}

	return &val
}

// FromPointer converts a pointer to a value T.
func FromPointer[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

func FromPointerUnsafe[T any](v *T) T {
	return *v
}
