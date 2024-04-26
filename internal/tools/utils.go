package tools

import "fmt"

// Plural returns a singular or plural of a word together with the total
func Plural(total int, singular, plural string) string {
	if total == 1 {
		return fmt.Sprintf("%d %s", total, singular)
	}
	return fmt.Sprintf("%d %s", total, plural)
}
