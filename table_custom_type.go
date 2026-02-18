package qwery

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Custom type for JSON handling using map
type JSONMap map[string]interface{}

// Implement the sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap) // Handle NULL values by initializing an empty map
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to type assert JSON value to []byte")
	}

	return json.Unmarshal(bytes, j)
}

// Implement the driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Parse the JSON map to a struct
func (j JSONMap) Parse(dest interface{}) error {

	return mapstructure.Decode(j, dest)
}

type Time time.Time

func (t *Time) MarshalJSON() ([]byte, error) {
	// Format the time to a custom format, without the 'Z' at the end
	return json.Marshal(time.Time(*t).Format("2006-01-02T15:04:05.000"))
}

func (t *Time) UnmarshalText(text []byte) error {
	// Parse the time from the custom format
	parsedTime, err := time.Parse("2006-01-02 15:04:05", string(text))
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}
	*t = Time(parsedTime)
	return nil
}

func (t *Time) MarshalText() ([]byte, error) {
	// Format the time to a custom format, without the 'Z' at the end
	return []byte(time.Time(*t).Format("2006-01-02 15:04:05")), nil
}

// func (c *CustomType) UnmarshalText(text []byte) error {
// 	// Custom logic to handle different input formats
// 	if len(text) > 0 && text[0] == '{' {
// 		var obj map[string]interface{}
// 		if err := json.Unmarshal(text, &obj); err != nil {
// 			return fmt.Errorf("failed to unmarshal CustomType: %w", err)
// 		}
// 		if val, ok := obj["value"].(string); ok {
// 			c.Value = val
// 			return nil
// 		}
// 		return fmt.Errorf("missing 'value' field in JSON")

// 	}
// 	c.Value = string(text)
// 	return nil
// }
