package services

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-yonathan/src/pkg/constants"

	"github.com/gofiber/fiber/v3"
)

func TestClassifyDNSBLAnswer(t *testing.T) {
	cases := []struct {
		name       string
		addrs      []string
		err        error
		wantStatus string
		wantErr    string
	}{
		{"listed", []string{"127.0.0.2"}, nil, dnsblListed, ""},
		{"listed with several codes", []string{"127.0.0.2", "127.0.0.10"}, nil, dnsblListed, ""},
		{"nxdomain is not listed", nil, &net.DNSError{IsNotFound: true}, dnsblNotListed, ""},
		{"empty answer is not listed", nil, nil, dnsblNotListed, ""},
		{"timeout is an error, never clean", nil, &net.DNSError{IsTimeout: true}, dnsblError, "timeout"},
		{"servfail is an error", nil, &net.DNSError{IsTemporary: true}, dnsblError, "servfail"},
		{"spamhaus open-resolver block", []string{"127.255.255.254"}, nil, dnsblError, "query_blocked"},
		{"spamhaus excessive queries", []string{"127.255.255.255"}, nil, dnsblError, "query_blocked"},
		{"non-loopback answer", []string{"203.0.113.9"}, nil, dnsblError, "unexpected_response"},
		{"garbage answer", []string{"not-an-ip"}, nil, dnsblError, "unexpected_response"},
	}
	for _, tc := range cases {
		got := classifyDNSBLAnswer(tc.addrs, tc.err)
		if got.Status != tc.wantStatus || got.Error != tc.wantErr {
			t.Errorf("%s: got status=%q error=%q want %q/%q", tc.name, got.Status, got.Error, tc.wantStatus, tc.wantErr)
		}
		if (got.Status == dnsblListed) != got.Listed {
			t.Errorf("%s: listed flag inconsistent with status", tc.name)
		}
	}
}

func TestCheckBlacklistSummaryNeverCountsFailuresAsClean(t *testing.T) {
	providers := constants.DNSBLProviders
	mock := &mockResolver{
		lookupFn: func(_ context.Context, host string) ([]string, error) {
			switch {
			case hostHasSuffix(host, providers[0]):
				return []string{"127.0.0.2"}, nil
			case hostHasSuffix(host, providers[1]):
				return nil, &net.DNSError{IsNotFound: true}
			default:
				return nil, &net.DNSError{IsTimeout: true}
			}
		},
	}
	withMockResolver(mock, func() {
		app := fiber.New()
		app.Get("/blacklist", CheckBlacklist)
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/blacklist?ip=127.0.0.2", nil))
		if err != nil || res.StatusCode != fiber.StatusOK {
			t.Fatalf("unexpected: %v status=%d", err, res.StatusCode)
		}
		var body struct {
			Results []blacklistResult `json:"results"`
			Summary map[string]int    `json:"summary"`
		}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Summary["listed"] != 1 || body.Summary["notListed"] != 1 {
			t.Fatalf("unexpected summary %v", body.Summary)
		}
		if body.Summary["failed"] != len(providers)-2 || body.Summary["checked"] != 2 {
			t.Fatalf("failures must be reported, not counted as checked: %v", body.Summary)
		}
		for _, r := range body.Results {
			if r.Status == dnsblError && r.Listed {
				t.Fatalf("error result flagged as listed: %+v", r)
			}
		}
	})
}

func hostHasSuffix(host, provider string) bool {
	return len(host) > len(provider) && host[len(host)-len(provider):] == provider
}

func TestDNSStatusFieldsInHandlers(t *testing.T) {
	timeoutMock := &mockResolver{err: &net.DNSError{IsTimeout: true}}
	withMockResolver(timeoutMock, func() {
		app := fiber.New()
		app.Get("/resolve", ResolveDomain)
		app.Get("/propagation", CheckPropagation)
		app.Get("/mail", GetMailRecords)

		for _, path := range []string{"/resolve?domain=example.com", "/propagation?domain=example.com&type=TXT", "/propagation?domain=example.com&type=MX"} {
			res, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]interface{}
			_ = json.NewDecoder(res.Body).Decode(&body)
			if body["status"] != "timeout" {
				t.Fatalf("%s: expected status timeout, got %v", path, body["status"])
			}
		}

		res, _ := app.Test(httptest.NewRequest(http.MethodGet, "/mail?domain=example.com", nil))
		var mail map[string]interface{}
		_ = json.NewDecoder(res.Body).Decode(&mail)
		statuses, ok := mail["status"].(map[string]interface{})
		if !ok || statuses["mx"] != "timeout" || statuses["dmarc"] != "timeout" {
			t.Fatalf("mail-records must expose per-record status: %v", mail["status"])
		}
	})

	notFound := &mockResolver{err: &net.DNSError{IsNotFound: true}}
	withMockResolver(notFound, func() {
		app := fiber.New()
		app.Get("/propagation", CheckPropagation)
		res, _ := app.Test(httptest.NewRequest(http.MethodGet, "/propagation?domain=example.com&type=A", nil))
		var body map[string]interface{}
		_ = json.NewDecoder(res.Body).Decode(&body)
		if body["status"] != "not_found" {
			t.Fatalf("expected not_found, got %v", body["status"])
		}
	})
}
