package mailaudit

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"
)

func rsaRecord(t *testing.T, bits int, extra string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return "v=DKIM1; k=rsa; " + extra + "p=" + base64.StdEncoding.EncodeToString(der)
}

func TestParseDKIMKeySizes(t *testing.T) {
	key, findings := ParseDKIM(rsaRecord(t, 2048, ""))
	if len(findings) != 0 || key.Bits != 2048 || key.Type != "rsa" {
		t.Fatalf("2048: %+v %v", key, ids(findings))
	}
	key, _ = ParseDKIM(rsaRecord(t, 1024, ""))
	if key.Bits != 1024 {
		t.Fatalf("1024: %+v", key)
	}
}

func TestParseDKIMDefaultsToRSAAndToleratesWhitespaceInKey(t *testing.T) {
	rec := rsaRecord(t, 2048, "")
	spaced := ""
	for i, r := range rec {
		spaced += string(r)
		if i > 30 && i%60 == 0 {
			spaced += "  "
		}
	}
	key, findings := ParseDKIM(spaced)
	if len(findings) != 0 || key.Bits != 2048 {
		t.Fatalf("whitespace inside p= must be ignored: %+v %v", key, ids(findings))
	}
	noK, _ := ParseDKIM("p=" + rec[len("v=DKIM1; k=rsa; p="):])
	if noK.Type != "rsa" || noK.Bits != 2048 {
		t.Fatalf("k defaults to rsa: %+v", noK)
	}
}

func TestParseDKIMEd25519(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	key, findings := ParseDKIM("v=DKIM1; k=ed25519; p=" + base64.StdEncoding.EncodeToString(pub))
	if len(findings) != 0 || key.Type != "ed25519" || key.Bits != 256 {
		t.Fatalf("%+v %v", key, ids(findings))
	}
	_, findings = ParseDKIM("k=ed25519; p=" + base64.StdEncoding.EncodeToString([]byte("short")))
	requireFinding(t, findings, "dkim.syntax", SeverityCritical)
}

func TestParseDKIMRevokedAndErrors(t *testing.T) {
	key, findings := ParseDKIM("v=DKIM1; p=")
	if !key.Revoked || len(findings) != 0 {
		t.Fatalf("empty p= means revoked: %+v %v", key, ids(findings))
	}
	for _, raw := range []string{
		"v=DKIM1; k=rsa",
		"v=DKIM2; p=AAAA",
		"v=DKIM1; p=%%%not-base64%%%",
		"v=DKIM1; k=rsa; p=AAAA",
		"v=DKIM1; k=dsa; p=AAAA",
		"",
	} {
		_, findings := ParseDKIM(raw)
		requireFinding(t, findings, "dkim.syntax", SeverityCritical)
	}
}

func TestAuditDKIMKeyQuality(t *testing.T) {
	good := rsaRecord(t, 2048, "")
	audit := AuditDKIM([]DKIMProbe{{Selector: "google", TXTs: []string{good}}})
	requireFinding(t, audit.Findings, "dkim.key_ok", SeverityOK)
	if len(audit.Found) != 1 || audit.Found[0] != "google" {
		t.Fatalf("found %v", audit.Found)
	}

	short := AuditDKIM([]DKIMProbe{{Selector: "old", TXTs: []string{rsaRecord(t, 1024, "")}}})
	requireFinding(t, short.Findings, "dkim.key_short", SeverityWarning)

	testing := AuditDKIM([]DKIMProbe{{Selector: "s1", TXTs: []string{rsaRecord(t, 2048, "t=y; ")}}})
	requireFinding(t, testing.Findings, "dkim.testing", SeverityWarning)
}

func TestDKIMKeyFindingsWeakKey(t *testing.T) {
	findings := dkimKeyFindings("legacy", DKIMKey{Type: "rsa", Bits: 768})
	requireFinding(t, findings, "dkim.key_weak", SeverityCritical)
}

func TestAuditDKIMRevokedSelectorsAreGroupedAndNotCountedAsKeys(t *testing.T) {
	audit := AuditDKIM([]DKIMProbe{
		{Selector: "old", TXTs: []string{"v=DKIM1; p="}},
		{Selector: "legacy", TXTs: []string{"v=DKIM1; p="}},
		{Selector: "google", Err: ErrNotFound},
	})
	requireFinding(t, audit.Findings, "dkim.revoked", SeverityInfo)
	revokedFindings := 0
	for _, f := range audit.Findings {
		if f.ID == "dkim.revoked" {
			revokedFindings++
			if !strings.Contains(f.Title, "old") || !strings.Contains(f.Title, "legacy") || !strings.Contains(f.Title, "2 selector") {
				t.Fatalf("grouped finding must name every selector: %q", f.Title)
			}
		}
	}
	if revokedFindings != 1 {
		t.Fatalf("revoked selectors must be reported once, got %d", revokedFindings)
	}
	if len(audit.Found) != 0 {
		t.Fatalf("a revoked selector publishes no key: %v", audit.Found)
	}
	requireFinding(t, audit.Findings, "dkim.none_found", SeverityInfo)
}

func TestAuditDKIMNothingFoundIsInformationalNotAFailure(t *testing.T) {
	probes := []DKIMProbe{{Selector: "default", Err: ErrNotFound}, {Selector: "google", Err: ErrNotFound}}
	audit := AuditDKIM(probes)
	requireFinding(t, audit.Findings, "dkim.none_found", SeverityInfo)
	for _, f := range audit.Findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityWarning {
			t.Fatalf("absence among common selectors must not warn: %+v", f)
		}
	}
}

func TestAuditDKIMAllFailedIsUnavailableNotMissing(t *testing.T) {
	audit := AuditDKIM([]DKIMProbe{{Selector: "default", Err: errTimeout}, {Selector: "google", Err: errTimeout}})
	requireFinding(t, audit.Findings, "dkim.unavailable", SeverityWarning)
	rejectFinding(t, audit.Findings, "dkim.none_found")
}

func TestAuditDKIMIgnoresNonDKIMTxtAndKeepsGoing(t *testing.T) {
	audit := AuditDKIM([]DKIMProbe{
		{Selector: "a", Err: errTimeout},
		{Selector: "b", TXTs: []string{"some other record"}},
		{Selector: "c", TXTs: []string{rsaRecord(t, 2048, "")}},
	})
	if len(audit.Found) != 1 || audit.Found[0] != "c" {
		t.Fatalf("found %v", audit.Found)
	}
}
