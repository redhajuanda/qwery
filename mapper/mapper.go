package mapper

import (
	"reflect"
	"time"

	"github.com/redhajuanda/qwery/vars"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// timeHook prevents time.Time from being converted to map and handles flexible mapping
func timeHook() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		// Handle time.Time preservation in all cases
		switch {
		case from == reflect.TypeOf(time.Time{}):
			return data, nil
		case from == reflect.TypeOf(&time.Time{}):
			if ptr, ok := data.(*time.Time); ok && ptr != nil {
				return *ptr, nil
			}
		case from.Kind() == reflect.Struct &&
			to.Kind() == reflect.Map &&
			to.Key().Kind() == reflect.String:
			// Special handling for struct to map conversion
			if rv := reflect.ValueOf(data); rv.IsValid() && rv.Kind() == reflect.Struct {
				result := make(map[string]any)
				rt := rv.Type()

				for i := 0; i < rv.NumField(); i++ {
					field := rt.Field(i)
					fieldValue := rv.Field(i)

					if !fieldValue.CanInterface() {
						continue
					}

					// Get the tag name
					tagName := field.Tag.Get(vars.TagKey)
					if tagName == "" {
						tagName = field.Name
					}

					// If this is a time.Time field, preserve it
					if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
						result[tagName] = fieldValue.Interface()
					} else {
						// For non-time fields, let mapstructure handle them
						// by recursively converting nested structs
						if fieldValue.Kind() == reflect.Struct {
							// Recursively convert struct to map using Decode
							nested := make(map[string]any)
							nestedCfg := &mapstructure.DecoderConfig{
								TagName:          vars.TagKey,
								Result:           &nested,
								DecodeHook:       timeHook(),
								WeaklyTypedInput: true,
							}
							nestedDec, err := mapstructure.NewDecoder(nestedCfg)
							if err == nil {
								err = nestedDec.Decode(fieldValue.Interface())
								if err == nil {
									result[tagName] = nested
									continue
								}
							}
						}
						result[tagName] = fieldValue.Interface()
					}
				}
				return result, nil
			}
		}

		return data, nil
	}
}

// preprocessMapForFlexibleMapping preprocesses a map to handle flexible field name matching
func preprocessMapForFlexibleMapping(input map[string]interface{}, targetType reflect.Type) map[string]interface{} {
	result := make(map[string]interface{})

	// Create field mapping for target struct
	fieldMap := make(map[string]string) // map[fieldName]tagName

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		tagName := field.Tag.Get(vars.TagKey)
		if tagName != "" {
			fieldMap[field.Name] = tagName
		}
	}

	// Process input map
	for key, value := range input {
		// First try direct key mapping
		if value != nil {
			// Handle nested maps (for nested structs)
			if nestedMap, ok := value.(map[string]interface{}); ok {
				// Find the corresponding struct field type
				for i := 0; i < targetType.NumField(); i++ {
					field := targetType.Field(i)
					tagName := field.Tag.Get(vars.TagKey)
					if (tagName != "" && tagName == key) || (tagName == "" && field.Name == key) {
						if field.Type.Kind() == reflect.Struct {
							result[key] = preprocessMapForFlexibleMapping(nestedMap, field.Type)
						} else {
							result[key] = value
						}
						goto nextKey
					}
				}
			}

			// Handle slices of maps (for slices of structs)
			if slice, ok := value.([]interface{}); ok {
				// Find the corresponding field type
				for i := 0; i < targetType.NumField(); i++ {
					field := targetType.Field(i)
					tagName := field.Tag.Get(vars.TagKey)
					if (tagName != "" && tagName == key) || (tagName == "" && field.Name == key) {
						if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
							processedSlice := make([]interface{}, len(slice))
							for j, item := range slice {
								if itemMap, ok := item.(map[string]interface{}); ok {
									// Preprocess each map in the slice with flexible field name mapping
									processedMap := preprocessMapForFlexibleMapping(itemMap, field.Type.Elem())
									// Also handle PascalCase to tag mapping for nested structs
									processedMap = handlePascalCaseToTagMapping(processedMap, field.Type.Elem())
									processedSlice[j] = processedMap
								} else {
									processedSlice[j] = item
								}
							}
							result[key] = processedSlice
						} else {
							result[key] = value
						}
						goto nextKey
					}
				}
			}

			// Check if key matches field name and there's a corresponding tag
			for fieldName, tagName := range fieldMap {
				if key == fieldName {
					// Use tag name instead
					result[tagName] = value
					goto nextKey
				}
			}
		}

		// Default: use original key
		result[key] = value
	nextKey:
	}

	return result
}

// handlePascalCaseToTagMapping converts PascalCase keys to their corresponding sikat tag names
func handlePascalCaseToTagMapping(input map[string]interface{}, targetType reflect.Type) map[string]interface{} {
	result := make(map[string]interface{})

	// Create mapping from field name to tag name
	fieldToTag := make(map[string]string)
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		tagName := field.Tag.Get(vars.TagKey)
		if tagName != "" {
			fieldToTag[field.Name] = tagName
		}
	}

	// Process input map
	for key, value := range input {
		// Check if key matches any field name and map to tag
		if tagName, exists := fieldToTag[key]; exists {
			result[tagName] = value
		} else {
			result[key] = value
		}
	}

	return result
}

// Deprecated: Use sikat.StructToMap or sikat.MapToStruct instead
func Decode(input any, output any) error {
	// Preprocess input if it's a map and output is a struct
	processedInput := input
	if inputMap, ok := input.(map[string]interface{}); ok {
		outputValue := reflect.ValueOf(output)
		if outputValue.Kind() == reflect.Ptr && outputValue.Elem().Kind() == reflect.Struct {
			targetType := outputValue.Elem().Type()
			// First apply general preprocessing
			processed := preprocessMapForFlexibleMapping(inputMap, targetType)
			// Then apply PascalCase to tag mapping
			processedInput = handlePascalCaseToTagMapping(processed, targetType)

		}
	}

	cfg := &mapstructure.DecoderConfig{
		TagName:          vars.TagKey,
		Result:           output,
		DecodeHook:       timeHook(),
		WeaklyTypedInput: true,
	}

	// init decoder
	dec, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return errors.Wrap(err, "cannot init decoder")
	}

	// decode
	err = dec.Decode(processedInput)
	if err != nil {
		return errors.Wrap(err, "cannot decode dest")
	}

	return nil
}
