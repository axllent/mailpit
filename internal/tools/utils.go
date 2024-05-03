package tools

import (
	"fmt"
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
	k = strings.ToLower(k)
	for _, v := range arr {
		if strings.ToLower(v) == k {
			return true
		}
	}

	return false
}
