package config

import (
	"reflect"
	"testing"
)

func TestAllowedCORSOriginsProductionDropsDevelopmentOrigins(t *testing.T) {
	raw := "https://www.yonathangutierrez.dev, http://localhost:3000,https://localhost:3000," +
		"http://www.yonathangutierrez.dev,https://127.0.0.1,https://10.0.0.5,https://app.localhost,https://example.com/path"
	allowed, dropped := AllowedCORSOrigins(raw, true)

	if want := []string{"https://www.yonathangutierrez.dev"}; !reflect.DeepEqual(allowed, want) {
		t.Fatalf("allowed = %v, want %v", allowed, want)
	}
	if len(dropped) != 7 {
		t.Fatalf("dropped = %v, want 7 origins", dropped)
	}
}

func TestAllowedCORSOriginsProductionFailsClosed(t *testing.T) {
	for _, raw := range []string{"", "http://localhost:3000"} {
		if allowed, _ := AllowedCORSOrigins(raw, true); len(allowed) != 0 {
			t.Errorf("AllowedCORSOrigins(%q, true) = %v, want none", raw, allowed)
		}
	}
}

func TestAllowedCORSOriginsDevelopment(t *testing.T) {
	if allowed, _ := AllowedCORSOrigins("", false); !reflect.DeepEqual(allowed, []string{DevCORSOrigin}) {
		t.Fatalf("empty dev config = %v, want %v", allowed, []string{DevCORSOrigin})
	}
	allowed, dropped := AllowedCORSOrigins("http://localhost:3000/, http://localhost:3001", false)
	if !reflect.DeepEqual(allowed, []string{"http://localhost:3000", "http://localhost:3001"}) || len(dropped) != 0 {
		t.Fatalf("dev config = %v dropped %v", allowed, dropped)
	}
}

func TestIsProduction(t *testing.T) {
	env := func(vals map[string]string) func(string) string {
		return func(k string) string { return vals[k] }
	}
	cases := []struct {
		vals map[string]string
		want bool
	}{
		{map[string]string{}, false},
		{map[string]string{"K_SERVICE": "porfolio-api"}, true},
		{map[string]string{"APP_ENV": "Production"}, true},
		{map[string]string{"APP_ENV": "development"}, false},
	}
	for _, c := range cases {
		if got := IsProduction(env(c.vals)); got != c.want {
			t.Errorf("IsProduction(%v) = %v, want %v", c.vals, got, c.want)
		}
	}
}
