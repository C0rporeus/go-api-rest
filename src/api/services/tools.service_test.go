package services

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestEncodeBase64(t *testing.T) {
	app := fiber.New()
	app.Post("/encode", EncodeBase64)

	body, _ := json.Marshal(map[string]string{"value": "hola"})
	req := httptest.NewRequest(http.MethodPost, "/encode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestDecodeBase64InvalidInput(t *testing.T) {
	app := fiber.New()
	app.Post("/decode", DecodeBase64)

	body, _ := json.Marshal(map[string]string{"value": "%%%invalid%%%"})
	req := httptest.NewRequest(http.MethodPost, "/decode", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestGenerateUUIDv4(t *testing.T) {
	app := fiber.New()
	app.Get("/uuid", GenerateUUIDv4)

	req := httptest.NewRequest(http.MethodGet, "/uuid", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	app := fiber.New()
	app.Post("/cert", GenerateSelfSignedCert)

	body, _ := json.Marshal(map[string]any{
		"commonName":   "localhost",
		"organization": "test-org",
		"validDays":    30,
		"password":     "changeit",
	})
	req := httptest.NewRequest(http.MethodPost, "/cert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}

func postBase64(t *testing.T, handler fiber.Handler, payload map[string]string) (int, map[string]interface{}) {
	t.Helper()
	app := fiber.New()
	app.Post("/b64", handler)
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/b64", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	var out map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestEncodeBase64Variants(t *testing.T) {
	cases := []struct {
		name, value, variant, want string
	}{
		{"std default", "???>>>", "", "Pz8/Pj4+"},
		{"std explicit", "???>>>", "std", "Pz8/Pj4+"},
		{"url alphabet", "???>>>", "url", "Pz8_Pj4-"},
		{"url has no padding", "f", "url", "Zg"},
		{"std keeps padding", "f", "std", "Zg=="},
		{"RFC 4648 foo", "foo", "url", "Zm9v"},
		{"RFC 4648 fooba", "fooba", "url", "Zm9vYmE"},
	}
	for _, tc := range cases {
		status, out := postBase64(t, EncodeBase64, map[string]string{"value": tc.value, "variant": tc.variant})
		if status != fiber.StatusOK || out["encoded"] != tc.want {
			t.Errorf("%s: status=%d encoded=%v want %q", tc.name, status, out["encoded"], tc.want)
		}
	}
}

func TestDecodeBase64Variants(t *testing.T) {
	cases := []struct {
		name, value, variant, want string
		wantStatus                 int
	}{
		{"std", "Pz8/Pj4+", "std", "???>>>", 200},
		{"url unpadded", "Pz8_Pj4-", "url", "???>>>", 200},
		{"url accepts padding too", "Zg==", "url", "f", 200},
		{"std rejects url alphabet", "Pz8_Pj4-", "std", "", 400},
		{"std rejects missing padding", "Zg", "std", "", 400},
		{"url rejects std alphabet", "Pz8/Pj4+", "url", "", 400},
	}
	for _, tc := range cases {
		status, out := postBase64(t, DecodeBase64, map[string]string{"value": tc.value, "variant": tc.variant})
		if status != tc.wantStatus {
			t.Errorf("%s: status=%d want %d", tc.name, status, tc.wantStatus)
			continue
		}
		if tc.wantStatus == 200 && out["decoded"] != tc.want {
			t.Errorf("%s: decoded=%v want %q", tc.name, out["decoded"], tc.want)
		}
	}
}

func TestBase64InvalidVariantRejected(t *testing.T) {
	for _, handler := range []fiber.Handler{EncodeBase64, DecodeBase64} {
		if status, _ := postBase64(t, handler, map[string]string{"value": "x", "variant": "rot13"}); status != fiber.StatusBadRequest {
			t.Fatalf("invalid variant must be 400, got %d", status)
		}
	}
}
