package mailaudit

import "strings"

// cutTag finds a tag in a "k=v; k=v" record (case-insensitive tag name).
func cutTag(record, tag string) (string, string, bool) {
	for _, part := range strings.Split(record, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(strings.TrimSpace(name), tag) {
			return name, strings.TrimSpace(value), true
		}
	}
	return "", "", false
}

func containsFold(text, fragment string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(fragment))
}
