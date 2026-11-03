package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/mailaudit"

	"github.com/gofiber/fiber/v3"
)

// zoneResolver is an offline, name-aware DNS used by the mail audit tests.
type zoneResolver struct {
	mu    sync.Mutex
	txt   map[string][]string
	mx    map[string][]*net.MX
	errs  map[string]error
	calls []string
	gen   func(name string) ([]string, bool)
}

func (z *zoneResolver) record(name string) {
	z.mu.Lock()
	z.calls = append(z.calls, name)
	z.mu.Unlock()
}

func (z *zoneResolver) LookupTXT(_ context.Context, name string) ([]string, error) {
	z.record(name)
	if err, ok := z.errs[name]; ok {
		return nil, err
	}
	if txts, ok := z.txt[name]; ok {
		return txts, nil
	}
	if z.gen != nil {
		if txts, ok := z.gen(name); ok {
			return txts, nil
		}
	}
	return nil, &net.DNSError{IsNotFound: true, Name: name}
}

func (z *zoneResolver) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	z.record(name)
	if err, ok := z.errs["mx:"+name]; ok {
		return nil, err
	}
	if mx, ok := z.mx[name]; ok {
		return mx, nil
	}
	return nil, &net.DNSError{IsNotFound: true, Name: name}
}

func (z *zoneResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) { return nil, nil }
func (z *zoneResolver) LookupCNAME(context.Context, string) (string, error)        { return "", nil }
func (z *zoneResolver) LookupNS(context.Context, string) ([]*net.NS, error)        { return nil, nil }
func (z *zoneResolver) LookupHost(context.Context, string) ([]string, error)       { return nil, nil }

func (z *zoneResolver) callCount() int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return len(z.calls)
}

func dkimKeyRecord(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	return "v=DKIM1; k=rsa; p=" + base64.StdEncoding.EncodeToString(der)
}

func callAudit(t *testing.T, resolver *zoneResolver, query string) (int, mailAuditResponse) {
	t.Helper()
	var result mailAuditResponse
	original := newDNSResolver
	newDNSResolver = func() dnsResolver { return resolver }
	defer func() { newDNSResolver = original }()

	app := fiber.New()
	app.Get("/audit", MailAudit)
	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/audit?"+query, nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(res.Body).Decode(&result)
	return res.StatusCode, result
}

func findingIDs(result mailAuditResponse) map[string]mailaudit.Severity {
	out := map[string]mailaudit.Severity{}
	for _, f := range result.Findings {
		out[f.ID] = f.Severity
	}
	return out
}

func healthyZone(t *testing.T) *zoneResolver {
	return &zoneResolver{
		mx: map[string][]*net.MX{"example.com": {{Host: "mx1.example.net.", Pref: 10}, {Host: "mx2.example.net.", Pref: 20}}},
		txt: map[string][]string{
			"example.com":                   {"v=spf1 include:_spf.provider.test -all", "google-site-verification=abc"},
			"_spf.provider.test":            {"v=spf1 ip4:192.0.2.0/24 -all"},
			"_dmarc.example.com":            {"v=DMARC1; p=reject; rua=mailto:dmarc@example.com"},
			"google._domainkey.example.com": {dkimKeyRecord(t)},
			"_mta-sts.example.com":          {"v=STSv1; id=20260101"},
			"_smtp._tls.example.com":        {"v=TLSRPTv1; rua=mailto:tls@example.com"},
		},
	}
}

func TestMailAuditWellConfiguredDomainHasNoProblems(t *testing.T) {
	status, result := callAudit(t, healthyZone(t), "domain=Example.COM")
	if status != fiber.StatusOK || result.Domain != "example.com" {
		t.Fatalf("status=%d domain=%q", status, result.Domain)
	}
	if result.Summary[mailaudit.SeverityCritical] != 0 || result.Summary[mailaudit.SeverityWarning] != 0 {
		t.Fatalf("a healthy domain must not warn: %v", findingIDs(result))
	}
	for section, want := range map[string]dnsStatus{"mx": dnsStatusOK, "spf": dnsStatusOK, "dkim": dnsStatusOK, "dmarc": dnsStatusOK, "mta-sts": dnsStatusOK, "tls-rpt": dnsStatusOK} {
		if got := result.Sections[section].Status; got != want {
			t.Errorf("section %s status %q, want %q", section, got, want)
		}
	}
	if sel := result.Sections["dkim"].Selectors; len(sel) != 1 || sel[0] != "google" {
		t.Fatalf("dkim selectors %v", sel)
	}
	if lookups := result.Sections["spf"].Lookups; lookups == nil || *lookups != 1 {
		t.Fatalf("spf lookups %v", lookups)
	}
}

func TestMailAuditEmptyDomainReportsEachGap(t *testing.T) {
	_, result := callAudit(t, &zoneResolver{}, "domain=empty.example")
	ids := findingIDs(result)
	for id, sev := range map[string]mailaudit.Severity{
		"mx.missing": mailaudit.SeverityWarning, "spf.missing": mailaudit.SeverityWarning, "dmarc.missing": mailaudit.SeverityWarning,
		"dkim.none_found": mailaudit.SeverityInfo, "mta-sts.missing": mailaudit.SeverityInfo, "tls-rpt.missing": mailaudit.SeverityInfo,
	} {
		if ids[id] != sev {
			t.Errorf("%s: %q want %q (all: %v)", id, ids[id], sev, ids)
		}
	}
}

func TestMailAuditTimeoutIsReportedAsUnavailableNotMissing(t *testing.T) {
	zone := healthyZone(t)
	zone.errs = map[string]error{"_dmarc.example.com": &net.DNSError{IsTimeout: true, IsTemporary: true}}
	_, result := callAudit(t, zone, "domain=example.com")
	ids := findingIDs(result)
	if ids["dmarc.unavailable"] != mailaudit.SeverityWarning {
		t.Fatalf("expected dmarc.unavailable, got %v", ids)
	}
	if _, wrong := ids["dmarc.missing"]; wrong {
		t.Fatal("a timeout must never be reported as 'missing'")
	}
	if got := result.Sections["dmarc"].Status; got != dnsStatusTimeout {
		t.Fatalf("dmarc section status %q", got)
	}
	if result.Sections["spf"].Status != dnsStatusOK {
		t.Fatal("other sections must not be affected")
	}
}

func TestMailAuditQueryBudgetBoundsTheWork(t *testing.T) {
	zone := &zoneResolver{gen: func(name string) ([]string, bool) {
		if strings.HasPrefix(name, "_dmarc.") || strings.Contains(name, "_domainkey") || strings.HasPrefix(name, "_mta-sts.") || strings.HasPrefix(name, "_smtp.") {
			return nil, false
		}
		// every domain includes three new ones: an unbounded tree
		return []string{fmt.Sprintf("v=spf1 include:a.%s include:b.%s include:c.%s -all", name, name, name)}, true
	}}
	status, result := callAudit(t, zone, "domain=hostile.example")
	if status != fiber.StatusOK {
		t.Fatalf("status %d", status)
	}
	if calls := zone.callCount(); calls > constants.MailAuditMaxQueries {
		t.Fatalf("performed %d DNS queries, budget is %d", calls, constants.MailAuditMaxQueries)
	}
	if result.Summary[mailaudit.SeverityCritical] == 0 {
		t.Fatalf("a hostile SPF must be flagged: %v", findingIDs(result))
	}
}

func TestMailAuditBudgetExhaustionIsAnUnavailableFindingNotAFalseOK(t *testing.T) {
	dns := newAuditDNS(context.Background(), &zoneResolver{}, 2)
	for i := 0; i < 2; i++ {
		_, _ = dns.txt("x.test")
	}
	_, err := dns.txt("y.test")
	if !mailaudit.IsUnavailable(err) {
		t.Fatalf("exhausted budget must be 'unavailable', got %v", err)
	}
	if statusOfLookup(err, 0) != dnsStatusError {
		t.Fatalf("status %q", statusOfLookup(err, 0))
	}
}

func TestMailAuditCustomSelectorIsProbedAndDeduplicated(t *testing.T) {
	zone := &zoneResolver{txt: map[string][]string{"mail2026._domainkey.example.com": {dkimKeyRecord(t)}}}
	_, result := callAudit(t, zone, "domain=example.com&selector=Mail2026")
	if sel := result.Sections["dkim"].Selectors; len(sel) != 1 || sel[0] != "mail2026" {
		t.Fatalf("custom selector not audited: %v", sel)
	}
	fresh := &zoneResolver{}
	_, _ = callAudit(t, fresh, "domain=example.com&selector=google")
	count := 0
	for _, name := range fresh.calls {
		if name == "google._domainkey.example.com" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("a selector already in the common list must be probed once, got %d", count)
	}
}

func TestMailAuditValidatesInput(t *testing.T) {
	cases := map[string]string{
		"":                                    "missing_domain",
		"domain=not%20a%20domain":             "invalid_domain",
		"domain=-bad.example.com":             "invalid_domain",
		"domain=example.com&selector=bad%20s": "invalid_selector",
		"domain=example.com&selector=a..b":    "invalid_selector",
		"domain=example.com&selector=-x":      "invalid_selector",
		"domain=example.com&selector=" + strings.Repeat("a", 64): "invalid_selector",
		"domain=example.com&selector=x_y":                        "invalid_selector",
	}
	for query, code := range cases {
		zone := &zoneResolver{}
		app := fiber.New()
		original := newDNSResolver
		newDNSResolver = func() dnsResolver { return zone }
		app.Get("/audit", MailAudit)
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/audit?"+query, nil))
		newDNSResolver = original
		if err != nil || res.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%q: status=%v err=%v", query, res.StatusCode, err)
			continue
		}
		var body map[string]interface{}
		_ = json.NewDecoder(res.Body).Decode(&body)
		if body["code"] != code {
			t.Errorf("%q: code %v want %s", query, body["code"], code)
		}
		if zone.callCount() != 0 {
			t.Errorf("%q: invalid input must not trigger DNS queries", query)
		}
	}
}

func TestMailAuditAcceptsValidSelectors(t *testing.T) {
	for _, selector := range []string{"s1", "mail-2026", "2026-01.eu", "a"} {
		if !dkimSelectorPattern.MatchString(selector) {
			t.Errorf("%q should be valid", selector)
		}
	}
}

func TestMailAuditFindingsAreOrderedBySectionThenSeverity(t *testing.T) {
	_, result := callAudit(t, &zoneResolver{}, "domain=empty.example")
	lastSection, lastSeverity := -1, -1
	for _, f := range result.Findings {
		section, severity := sectionOrder[f.Section], severityOrder[f.Severity]
		if section < lastSection || (section == lastSection && severity < lastSeverity) {
			t.Fatalf("findings out of order at %s/%s", f.Section, f.Severity)
		}
		if section != lastSection {
			lastSeverity = -1
		}
		lastSection, lastSeverity = section, severity
	}
}

func TestDNSLookupErrorBehavesWithErrorsIs(t *testing.T) {
	notFound := &dnsLookupError{status: dnsStatusNotFound}
	if !mailaudit.IsUnavailable(&dnsLookupError{status: dnsStatusTimeout}) || mailaudit.IsUnavailable(notFound) {
		t.Fatal("only non-not_found statuses are unavailable")
	}
	if statusOfLookup(fmt.Errorf("unrelated"), 0) != dnsStatusError {
		t.Fatal("foreign errors map to error")
	}
}

func TestWrapLookupErrorKeepsGoMessageForUnclassifiedFailuresOnly(t *testing.T) {
	odd := wrapLookupError(&net.DNSError{Err: "cannot unmarshal DNS message", Name: "x.test", Server: "169.254.169.254:53"})
	if got := odd.Error(); got != "error: cannot unmarshal DNS message" {
		t.Fatalf("unclassified failures must say what happened, got %q", got)
	}
	if strings.Contains(odd.Error(), "169.254") {
		t.Fatal("the resolver address must never leak into findings")
	}
	if got := wrapLookupError(&net.DNSError{IsTimeout: true, Err: "i/o timeout"}).Error(); got != "timeout" {
		t.Fatalf("classified failures keep their status, got %q", got)
	}
	if wrapLookupError(nil) != nil {
		t.Fatal("nil stays nil")
	}
}
