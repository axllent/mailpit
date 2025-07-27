package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// Plural returns a singular or plural of a word together with the total
func Plural(total int, singular, plural string) string {
	if total == 1 {
		return fmt.Sprintf("%d %s", total, singular)
	}

	return fmt.Sprintf("%d %s", total, plural)
}

// InArray tests if a string is within an array. It is not case sensitive.
func InArray(k string, arr []string) bool {
	for _, v := range arr {
		if strings.EqualFold(v, k) {
			return true
		}
	}

	return false
}

// Normalize will remove any extra spaces, remove newlines, and trim leading and trailing spaces
func Normalize(s string) string {
	nlRe := regexp.MustCompile(`\r?\r`)
	re := regexp.MustCompile(`\s+`)

	s = nlRe.ReplaceAllString(s, " ")
	s = re.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// SafeUint64 converts an int or int64 to uint64, ensuring it does not exceed the maximum value for uint64.
func SafeUint64(i any) uint64 {
	switch v := i.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	default:
		// only accepts int or int64
		return 0
	}
}
