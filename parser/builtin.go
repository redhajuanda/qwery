package parser

import (
	reflect "reflect"
	"strings"
	"time"
)

// DerefBool dereferences a bool pointer.
func DerefBool(b *bool) bool {
	return *b
}

// IsTimeZero checks if a time.Time value is zero (i.e., equal to the zero time).
func IsTimeZero(t time.Time) bool {
	return t.IsZero()
}

// IsTimeNotZero checks if a time.Time value is not zero (i.e., not equal to the zero time).
func IsTimeNotZero(t time.Time) bool {
	return !t.IsZero()
}

func JSONOmitEmpty(input interface{}) interface{} {
	// Check if the input is nil
	if input == nil {
		return `"__null__"`
	}

	// Use reflection to check if the input is a zero value
	v := reflect.ValueOf(input)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return `"__null__"`
	}
	if v.IsValid() && v.IsZero() {
		return `"__null__"`
	}

	// Return the input as is
	return input
}

// ftsOperators are the characters MySQL treats as operators in BOOLEAN MODE.
// They must be stripped from user input so they are not interpreted as
// operators (e.g. a lone "-" would be parsed as the exclude operator and
// trigger a syntax error when followed by another operator).
const ftsOperators = "+-><()~*\"@"

// function to split every space and add prefix + to each word
// and at the last word, add * as suffix
func FullTextSearch(s string) string {
	fields := strings.Fields(s)
	words := make([]string, 0, len(fields))
	for _, w := range fields {
		// remove any reserved boolean-mode operator characters wherever
		// they appear so they cannot be interpreted as operators
		w = strings.Map(func(r rune) rune {
			if strings.ContainsRune(ftsOperators, r) {
				return -1
			}
			return r
		}, w)
		if w == "" {
			continue
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return ""
	}
	for i := range words {
		if len(words[i]) >= 3 {
			words[i] = "+" + words[i]
		}
	}
	words[len(words)-1] = words[len(words)-1] + "*"
	return strings.Join(words, " ")
}

// function to add suffix % to the string
func WildcardLikeSuffix(s string) string {
	return strings.TrimSpace(s) + "%"
}

// function to add prefix % to the string
func WildcardLikePrefix(s string) string {
	return "%" + strings.TrimSpace(s)
}
