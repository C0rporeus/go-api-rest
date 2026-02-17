package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// mockResolver implements dnsResolver for deterministic, offline tests.
type mockResolver struct {
	ipAddrs  []net.IPAddr
	cname    string
	mx       []*net.MX
	ns       []*net.NS
	txt      []string
	hosts    []string
	err      error
	lookupFn func(ctx context.Context, host string) ([]string, error)
}

func (m *mockResolver) LookupIPAddr(_ context.Context, _ string) ([]net.IPAddr, error) {
	return m.ipAddrs, m.err
}
func (m *mockResolver) LookupCNAME(_ context.Context, _ string) (string, error) {
	return m.cname, m.err
}
func (m *mockResolver) LookupMX(_ context.Context, _ string) ([]*net.MX, error) {
	return m.mx, m.err
}
func (m *mockResolver) LookupNS(_ context.Context, _ string) ([]*net.NS, error) {
	return m.ns, m.err
}
func (m *mockResolver) LookupTXT(_ context.Context, _ string) ([]string, error) {
	return m.txt, m.err
}
func (m *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if m.lookupFn != nil {
		return m.lookupFn(ctx, host)
	}
	return m.hosts, m.err
}

func withMockResolver(mock *mockResolver, fn func()) {
	original := newDNSResolver
	newDNSResolver = func() dnsResolver { return mock }
	defer func() { newDNSResolver = original }()
	fn()
}

// --- ResolveDomain tests ---

func TestResolveDomainMissingParam(t *testing.T) {
	app := fiber.New()
	app.Get("/resolve", ResolveDomain)

	req := httptest.NewRequest(http.MethodGet, "/resolve", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestResolveDomainValid(t *testing.T) {
	mock := &mockResolver{
		ipAddrs: []net.IPAddr{
			{IP: net.ParseIP("93.184.216.34")},
			{IP: net.ParseIP("2606:2800:220:1:248:1893:25c8:1946")},
		},
	}

	withMockResolver(mock, func() {
		app := fiber.New()
		app.Get("/resolve", ResolveDomain)

		req := httptest.NewRequest(http.MethodGet, "/resolve?domain=example.com", nil)
		res, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("unexpected test error: %v", err)
		}
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if body["domain"] != "example.com" {
			t.Fatalf("expected domain example.com, got %v", body["domain"])
		}
		if body["resolved"] != true {
			t.Fatal("expected resolved=true")
		}
	})
}

// --- CheckPropagation tests ---

func TestCheckPropagationMissingDomain(t *testing.T) {
	app := fiber.New()
	app.Get("/propagation", CheckPropagation)

	req := httptest.NewRequest(http.MethodGet, "/propagation", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckPropagationInvalidType(t *testing.T) {
	app := fiber.New()
	app.Get("/propagation", CheckPropagation)

	req := httptest.NewRequest(http.MethodGet, "/propagation?domain=example.com&type=INVALID", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckPropagationValidA(t *testing.T) {
	mock := &mockResolver{
		ipAddrs: []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}},
	}

	withMockResolver(mock, func() {
		app := fiber.New()
		app.Get("/propagation", CheckPropagation)

		req := httptest.NewRequest(http.MethodGet, "/propagation?domain=example.com&type=A", nil)
		res, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("unexpected test error: %v", err)
		}
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		records, ok := body["records"].([]interface{})
		if !ok || len(records) == 0 {
			t.Fatal("expected at least one A record")
		}
	})
}

// --- GetMailRecords tests ---

func TestGetMailRecordsMissingDomain(t *testing.T) {
	app := fiber.New()
	app.Get("/mail", GetMailRecords)

	req := httptest.NewRequest(http.MethodGet, "/mail", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestGetMailRecordsValid(t *testing.T) {
	mock := &mockResolver{
		mx:  []*net.MX{{Host: "mail.example.com.", Pref: 10}},
		txt: []string{"v=spf1 include:example.com ~all"},
	}

	withMockResolver(mock, func() {
		app := fiber.New()
		app.Get("/mail", GetMailRecords)

		req := httptest.NewRequest(http.MethodGet, "/mail?domain=example.com", nil)
		res, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("unexpected test error: %v", err)
		}
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		mx, ok := body["mx"].([]interface{})
		if !ok || len(mx) == 0 {
			t.Fatal("expected at least one MX record")
		}
	})
}

// --- CheckBlacklist tests ---

func TestCheckBlacklistMissingIP(t *testing.T) {
	app := fiber.New()
	app.Get("/blacklist", CheckBlacklist)

	req := httptest.NewRequest(http.MethodGet, "/blacklist", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckBlacklistInvalidIP(t *testing.T) {
	app := fiber.New()
	app.Get("/blacklist", CheckBlacklist)

	req := httptest.NewRequest(http.MethodGet, "/blacklist?ip=not-an-ip", nil)
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("unexpected test error: %v", err)
	}
	if res.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestCheckBlacklistValidIP(t *testing.T) {
	mock := &mockResolver{
		lookupFn: func(_ context.Context, host string) ([]string, error) {
			return nil, fmt.Errorf("nxdomain: %s", host)
		},
	}

	withMockResolver(mock, func() {
		app := fiber.New()
		app.Get("/blacklist", CheckBlacklist)

		req := httptest.NewRequest(http.MethodGet, "/blacklist?ip=8.8.8.8", nil)
		res, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("unexpected test error: %v", err)
		}
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		results, ok := body["results"].([]interface{})
		if !ok {
			t.Fatal("expected results array")
		}
		for _, r := range results {
			entry, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			if entry["listed"] == true {
				t.Fatalf("mock should return not-listed for all providers, but %s is listed", entry["provider"])
			}
		}
	})
}
