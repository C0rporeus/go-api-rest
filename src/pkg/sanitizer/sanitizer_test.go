package sanitizer

import "testing"

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"strips bold tag", "<b>bold</b>", "bold"},
		{"strips script tag", "<script>alert('xss')</script>", ""},
		{"strips nested tags", "<div><p>text</p></div>", "text"},
		{"empty string", "", ""},
		{"trims whitespace", "  hello  ", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripHTML(tt.input); got != tt.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		notEmpty bool
	}{
		{"allows safe tags", "<p>hello</p>", true},
		{"strips script", "<script>bad</script>", false},
		{"allows links", `<a href="https://example.com">link</a>`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeHTML(tt.input)
			if tt.notEmpty && got == "" {
				t.Errorf("SanitizeHTML(%q) returned empty, expected content", tt.input)
			}
		})
	}
}

func TestSanitizePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"strips html and trims", "  <b>hello</b>  ", 100, "hello"},
		{"truncates at max length", "abcdefghij", 5, "abcde"},
		{"zero max means no limit", "hello", 0, "hello"},
		{"handles unicode truncation", "héllo wörld", 5, "héllo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizePlainText(tt.input, tt.maxLen); got != tt.expected {
				t.Errorf("SanitizePlainText(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestSanitizeRichText(t *testing.T) {
	input := "<p>hello</p><script>bad</script>"
	got := SanitizeRichText(input, 1000)
	if got == "" {
		t.Error("SanitizeRichText returned empty for valid input")
	}
	if ContainsDangerousPatterns(got) {
		t.Error("SanitizeRichText output still contains dangerous patterns")
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"not-a-uuid", false},
		{"", false},
		{"550e8400-e29b-41d4-a716", false},
		{"550E8400-E29B-41D4-A716-446655440000", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidUUID(tt.input); got != tt.expected {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"user@example.com", true},
		{"user+tag@sub.domain.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidEmail(tt.input); got != tt.expected {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"example.com", true},
		{"sub.example.co.uk", true},
		{"invalid", false},
		{"-invalid.com", false},
		{"", false},
		{"a.b", false},
		{"a.co", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidDomain(tt.input); got != tt.expected {
				t.Errorf("IsValidDomain(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidIPv4(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"256.1.1.1", false},
		{"1.2.3", false},
		{"not-an-ip", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidIPv4(tt.input); got != tt.expected {
				t.Errorf("IsValidIPv4(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://localhost:3000", true},
		{"ftp://invalid.com", false},
		{"not-a-url", false},
		{"", false},
		{"https://example.com/path?q=1", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidURL(tt.input); got != tt.expected {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContainsDangerousPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"script tag", "<script>alert(1)</script>", true},
		{"javascript protocol", "javascript:alert(1)", true},
		{"event handler", `onload=alert(1)`, true},
		{"data uri", "data:text/html,<h1>hi</h1>", true},
		{"vbscript", "vbscript:exec", true},
		{"safe text", "hello world", false},
		{"safe html", "<p>paragraph</p>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsDangerousPatterns(tt.input); got != tt.expected {
				t.Errorf("ContainsDangerousPatterns(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Abcdef1!", true},
		{"short1A", false},
		{"alllowercase1", false},
		{"ALLUPPERCASE1", false},
		{"NoDigitsHere", false},
		{"", false},
		{"ValidPass123", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidPassword(tt.input); got != tt.expected {
				t.Errorf("IsValidPassword(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		min      int
		max      int
		expected bool
	}{
		{"within range", "hello", 1, 10, true},
		{"too short", "hi", 5, 10, false},
		{"too long", "hello world", 1, 5, false},
		{"exact min", "ab", 2, 10, true},
		{"exact max", "abcde", 1, 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateStringLength(tt.input, tt.min, tt.max); got != tt.expected {
				t.Errorf("ValidateStringLength(%q, %d, %d) = %v, want %v", tt.input, tt.min, tt.max, got, tt.expected)
			}
		})
	}
}

func TestSanitizeSlice(t *testing.T) {
	upper := func(s string) string { return s }

	t.Run("filters empty results", func(t *testing.T) {
		input := []string{"a", "", "b", ""}
		got := SanitizeSlice(input, 0, upper)
		if len(got) != 2 {
			t.Errorf("expected 2 items, got %d", len(got))
		}
	})

	t.Run("enforces max count", func(t *testing.T) {
		input := []string{"a", "b", "c", "d"}
		got := SanitizeSlice(input, 2, upper)
		if len(got) != 2 {
			t.Errorf("expected 2 items, got %d", len(got))
		}
	})
}

func TestValidateURLSlice(t *testing.T) {
	t.Run("filters invalid URLs", func(t *testing.T) {
		input := []string{"https://example.com", "not-a-url", "http://valid.com"}
		got := ValidateURLSlice(input, 0)
		if len(got) != 2 {
			t.Errorf("expected 2 valid URLs, got %d", len(got))
		}
	})

	t.Run("enforces max count", func(t *testing.T) {
		input := []string{"https://a.com", "https://b.com", "https://c.com"}
		got := ValidateURLSlice(input, 2)
		if len(got) != 2 {
			t.Errorf("expected 2 URLs, got %d", len(got))
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		input := []string{"  https://example.com  ", "  "}
		got := ValidateURLSlice(input, 0)
		if len(got) != 1 {
			t.Errorf("expected 1 URL, got %d", len(got))
		}
	})
}
