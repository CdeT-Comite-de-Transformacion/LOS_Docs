package naming

import (
	"regexp"
	"strings"
	"unicode"
)

var separators = regexp.MustCompile(`[-_\s]+`)

func PackageName(input string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(input, "-", ""), "_", ""))
}

func DirectoryName(input string) string {
	name := strings.ToLower(strings.TrimSpace(input))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.Join(strings.Fields(name), "-")
	return name
}

func Pascal(input string) string {
	parts := separators.Split(strings.TrimSpace(input), -1)
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, "")
}
