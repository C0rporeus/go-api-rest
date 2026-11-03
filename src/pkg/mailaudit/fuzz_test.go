package mailaudit

import "testing"

// Fuzz targets: parsers receive attacker-controlled TXT content and must never
// panic, whatever the input. `go test` runs the seeds; `go test -fuzz=FuzzParseSPF`
// explores further.

func FuzzParseSPF(f *testing.F) {
	for _, seed := range []string{
		"v=spf1 -all", "v=spf1 include:a.test mx a/24//64 ip4:1.2.3.4/33 ip6:::1 redirect=x.test exp=y.test ~all",
		"v=spf1 ===", "v=spf1 :", "v=spf1 /", "v=spf1 a:", "v=spf1 -", "v=spf1 =", "V=SPF1\x00 a", "v=spf1 " + string(make([]byte, 5000)),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		record, findings := ParseSPF(raw)
		if record.lookupTerms() < 0 {
			t.Fatal("negative lookup count")
		}
		for _, finding := range findings {
			if len(finding.Detail) > maxFindingText+len("…") {
				t.Fatal("finding detail not truncated")
			}
		}
		_ = AuditSPF("example.com", []string{raw}, nil, func(string) ([]string, error) { return []string{raw}, nil })
	})
}

func FuzzParseDMARC(f *testing.F) {
	for _, seed := range []string{
		"v=DMARC1; p=reject; rua=mailto:a@b.c", "v=DMARC1;;;;", "v=DMARC1; =; ==", "v=DMARC1; p", "v=DMARC1; pct=99999999999999999999",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = ParseDMARC(raw)
		_ = AuditDMARC([]string{raw}, nil)
	})
}

func FuzzParseDKIM(f *testing.F) {
	for _, seed := range []string{
		"v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC", "p=", "k=ed25519; p=AAAA", "v=DKIM1; p=====", ";;;;", "p=" + string(make([]byte, 4000)),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = ParseDKIM(raw)
		_ = AuditDKIM([]DKIMProbe{{Selector: "s", TXTs: []string{raw}}})
	})
}
