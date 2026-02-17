package sanitizer

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

// Compiled patterns for input validation.
var (
	uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	emailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	ipv4Regex   = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)

	// Patterns that indicate injection attempts in text fields.
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[\s>]`),
		regexp.MustCompile(`(?i)javascript\s*:`),
		regexp.MustCompile(`(?i)on\w+\s*=`),
		regexp.MustCompile(`(?i)data\s*:\s*text/html`),
		regexp.MustCompile(`(?i)vbscript\s*:`),
	}

	// Strict text policy strips all HTML tags.
	strictPolicy = bluemonday.StrictPolicy()

	// UGC policy allows safe HTML for rich content.
	ugcPolicy = func() *bluemonday.Policy {
		p := bluemonday.UGCPolicy()
		p.AllowAttrs("class").Globally()
		p.AllowAttrs("target", "rel").OnElements("a")
		return p
	}()
)

// StripHTML removes all HTML tags from input, returning plain text.
func StripHTML(input string) string {
	return strings.TrimSpace(strictPolicy.Sanitize(input))
}

// SanitizeHTML allows safe HTML tags (UGC policy) and strips dangerous ones.
func SanitizeHTML(input string) string {
	return ugcPolicy.Sanitize(input)
}

// SanitizePlainText trims, strips HTML, and enforces a max length on plain text fields.
func SanitizePlainText(input string, maxLen int) string {
	cleaned := StripHTML(strings.TrimSpace(input))
	if maxLen > 0 && utf8.RuneCountInString(cleaned) > maxLen {
		runes := []rune(cleaned)
		cleaned = string(runes[:maxLen])
	}
	return cleaned
}

// SanitizeRichText sanitizes user-generated HTML content and enforces a max length.
func SanitizeRichText(input string, maxLen int) string {
	sanitized := SanitizeHTML(input)
	if maxLen > 0 && utf8.RuneCountInString(sanitized) > maxLen {
		runes := []rune(sanitized)
		sanitized = string(runes[:maxLen])
	}
	return sanitized
}

// IsValidUUID checks if the string is a valid UUID v4 format.
func IsValidUUID(id string) bool {
	return uuidRegex.MatchString(id)
}

// IsValidEmail checks email format with a strict regex.
func IsValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidDomain checks if the string is a valid domain name.
func IsValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

// IsValidIPv4 checks if the string is a valid IPv4 address (0-255 per octet).
func IsValidIPv4(ip string) bool {
	matches := ipv4Regex.FindStringSubmatch(ip)
	if matches == nil {
		return false
	}
	for i := 1; i <= 4; i++ {
		octet := 0
		for _, ch := range matches[i] {
			octet = octet*10 + int(ch-'0')
		}
		if octet > 255 {
			return false
		}
	}
	return true
}

// IsValidURL validates that a string is a well-formed HTTP/HTTPS URL.
func IsValidURL(rawURL string) bool {
	if len(rawURL) > 2048 {
		return false
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// ContainsDangerousPatterns checks input for common XSS/injection vectors.
func ContainsDangerousPatterns(input string) bool {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// IsValidPassword checks minimum password requirements.
func IsValidPassword(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// ValidateStringLength checks if a string is within the allowed length range.
func ValidateStringLength(s string, minLen, maxLen int) bool {
	length := utf8.RuneCountInString(s)
	return length >= minLen && length <= maxLen
}

// SanitizeSlice applies a sanitizer function to each element of a string slice,
// filtering out empty results and enforcing a max count.
func SanitizeSlice(items []string, maxCount int, sanitizeFn func(string) string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		cleaned := sanitizeFn(item)
		if cleaned != "" {
			result = append(result, cleaned)
		}
		if maxCount > 0 && len(result) >= maxCount {
			break
		}
	}
	return result
}

// ValidateURLSlice validates each URL in a slice, returning only valid ones.
func ValidateURLSlice(urls []string, maxCount int) []string {
	result := make([]string, 0, len(urls))
	for _, u := range urls {
		trimmed := strings.TrimSpace(u)
		if trimmed != "" && IsValidURL(trimmed) {
			result = append(result, trimmed)
		}
		if maxCount > 0 && len(result) >= maxCount {
			break
		}
	}
	return result
}
