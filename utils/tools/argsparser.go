package tools

import "strings"

// ArgsParser will split a string by new words and quotes phrases
func ArgsParser(s string) []string {
	args := []string{}
	sb := &strings.Builder{}
	quoted := false
	for _, r := range s {
		if r == '"' {
			quoted = !quoted
			sb.WriteRune(r) // keep '"' otherwise comment this line
		} else if !quoted && r == ' ' {
			v := strings.TrimSpace(strings.ReplaceAll(sb.String(), "\"", ""))
			if v != "" {
				args = append(args, v)
			}
			sb.Reset()
		} else {
			sb.WriteRune(r)
		}
	}
	if sb.Len() > 0 {
		v := strings.TrimSpace(strings.ReplaceAll(sb.String(), "\"", ""))
		if v != "" {
			args = append(args, v)
		}
	}

	return args
}
