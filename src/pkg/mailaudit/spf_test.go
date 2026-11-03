package mailaudit

import (
	"fmt"
	"testing"
)

func TestParseSPFValidRecordsAndLookupCount(t *testing.T) {
	cases := []struct {
		raw     string
		lookups int
	}{
		{"v=spf1 -all", 0},
		{"v=spf1 ip4:192.0.2.0/24 ip6:2001:db8::/32 -all", 0},
		{"v=spf1 include:_spf.example.com mx -all", 2},
		{"v=spf1 a a:mail.example.com/24 mx:mx.example.com//64 exists:%{i}.rbl.example.com ptr ~all", 5},
		{"V=SPF1 INCLUDE:Example.COM -ALL", 1},
		{"v=spf1 redirect=_spf.example.com", 1},
		{"v=spf1 foo=bar exp=explain.example.com -all", 0},
	}
	for _, tc := range cases {
		record, findings := ParseSPF(tc.raw)
		if len(findings) != 0 {
			t.Errorf("%q: unexpected findings %v", tc.raw, ids(findings))
		}
		if got := record.lookupTerms(); got != tc.lookups {
			t.Errorf("%q: lookups %d, want %d", tc.raw, got, tc.lookups)
		}
	}
}

func TestParseSPFSyntaxErrors(t *testing.T) {
	for _, raw := range []string{
		"v=spf1 foo",
		"v=spf1 ip4:300.1.1.1",
		"v=spf1 ip4:2001:db8::1",
		"v=spf1 ip6:192.0.2.1",
		"v=spf1 ip4:",
		"v=spf1 include:",
		"v=spf1 include:a.test/24",
		"v=spf1 all:x",
		"v=spf1 a/abc",
		"v=spf1 mx//",
		"v=spf1 redirect=",
		"v=spf1 redirect=a.test redirect=b.test",
		"v=spf2.0/pra -all",
		"",
		"   ",
	} {
		_, findings := ParseSPF(raw)
		requireFinding(t, findings, "spf.syntax", SeverityCritical)
	}
}

func TestParseSPFKeepsQualifiersAndRedirect(t *testing.T) {
	record, _ := ParseSPF("v=spf1 +a -mx ~include:x.test ?ip4:1.2.3.4 redirect=Fallback.TEST")
	want := []byte{'+', '-', '~', '?'}
	if len(record.Terms) != len(want) {
		t.Fatalf("terms: %+v", record.Terms)
	}
	for i, q := range want {
		if record.Terms[i].Qualifier != q {
			t.Errorf("term %d qualifier %q, want %q", i, record.Terms[i].Qualifier, q)
		}
	}
	if record.Redirect != "fallback.test" {
		t.Fatalf("redirect %q", record.Redirect)
	}
}

func TestAuditSPFMissingAndUnavailableAreDifferent(t *testing.T) {
	audit := AuditSPF("example.com", []string{"google-site-verification=abc"}, nil, zone{}.lookup)
	requireFinding(t, audit.Findings, "spf.missing", SeverityWarning)

	audit = AuditSPF("example.com", nil, errTimeout, zone{}.lookup)
	requireFinding(t, audit.Findings, "spf.unavailable", SeverityWarning)
	rejectFinding(t, audit.Findings, "spf.missing")
}

func TestAuditSPFMultipleRecordsIsPermerror(t *testing.T) {
	audit := AuditSPF("example.com", []string{"v=spf1 -all", "v=spf1 +all"}, nil, zone{}.lookup)
	requireFinding(t, audit.Findings, "spf.multiple", SeverityCritical)
	if len(audit.Records) != 2 {
		t.Fatalf("records: %v", audit.Records)
	}
}

func TestAuditSPFFinalPolicy(t *testing.T) {
	cases := []struct {
		raw string
		id  string
		sev Severity
	}{
		{"v=spf1 -all", "spf.all_fail", SeverityOK},
		{"v=spf1 ~all", "spf.all_softfail", SeverityInfo},
		{"v=spf1 ?all", "spf.all_neutral", SeverityWarning},
		{"v=spf1 +all", "spf.all_plus", SeverityCritical},
		{"v=spf1 all", "spf.all_plus", SeverityCritical},
		{"v=spf1 ip4:192.0.2.1", "spf.no_all", SeverityWarning},
		{"v=spf1 ptr -all", "spf.ptr", SeverityWarning},
		{"v=spf1 -all ip4:192.0.2.1", "spf.terms_after_all", SeverityWarning},
	}
	for _, tc := range cases {
		audit := AuditSPF("example.com", []string{tc.raw}, nil, zone{}.lookup)
		requireFinding(t, audit.Findings, tc.id, tc.sev)
	}
	audit := AuditSPF("example.com", []string{"v=spf1 redirect=_spf.example.com"}, nil, zone{"_spf.example.com": {"v=spf1 -all"}}.lookup)
	rejectFinding(t, audit.Findings, "spf.no_all")
}

func TestAuditSPFFollowsIncludesAndCountsLookups(t *testing.T) {
	dns := zone{
		"_spf.a.test": {"v=spf1 include:_spf.b.test mx -all"},
		"_spf.b.test": {"v=spf1 a ip4:192.0.2.0/24 -all"},
	}
	audit := AuditSPF("example.com", []string{"v=spf1 include:_spf.a.test -all"}, nil, dns.lookup)
	if audit.Lookups != 4 { // include a + include b + mx + a
		t.Fatalf("lookups %d, want 4", audit.Lookups)
	}
	if len(audit.Includes) != 2 {
		t.Fatalf("includes %v", audit.Includes)
	}
	rejectFinding(t, audit.Findings, "spf.lookups_exceeded")
}

func TestAuditSPFLookupLimits(t *testing.T) {
	record := func(n int) string {
		out := "v=spf1"
		for i := 0; i < n; i++ {
			out += " a"
		}
		return out + " -all"
	}
	requireFinding(t, AuditSPF("e.com", []string{record(10)}, nil, zone{}.lookup).Findings, "spf.lookups_near_limit", SeverityWarning)
	audit := AuditSPF("e.com", []string{record(11)}, nil, zone{}.lookup)
	requireFinding(t, audit.Findings, "spf.lookups_exceeded", SeverityCritical)
	rejectFinding(t, AuditSPF("e.com", []string{record(7)}, nil, zone{}.lookup).Findings, "spf.lookups_near_limit")
}

func TestAuditSPFDetectsCycles(t *testing.T) {
	dns := zone{
		"a.test": {"v=spf1 include:b.test -all"},
		"b.test": {"v=spf1 include:a.test -all"},
	}
	audit := AuditSPF("example.com", []string{"v=spf1 include:a.test -all"}, nil, dns.lookup)
	requireFinding(t, audit.Findings, "spf.cycle", SeverityCritical)

	self := AuditSPF("example.com", []string{"v=spf1 include:example.com -all"}, nil, zone{}.lookup)
	requireFinding(t, self.Findings, "spf.cycle", SeverityCritical)
}

func TestAuditSPFIncludeProblems(t *testing.T) {
	audit := AuditSPF("example.com", []string{"v=spf1 include:gone.test -all"}, nil, zone{}.lookup)
	requireFinding(t, audit.Findings, "spf.include_missing", SeverityCritical)

	noSPF := zone{"txt-only.test": {"google-site-verification=x"}}
	audit = AuditSPF("example.com", []string{"v=spf1 include:txt-only.test -all"}, nil, noSPF.lookup)
	requireFinding(t, audit.Findings, "spf.include_missing", SeverityCritical)

	failing := func(string) ([]string, error) { return nil, errTimeout }
	audit = AuditSPF("example.com", []string{"v=spf1 include:slow.test -all"}, nil, failing)
	requireFinding(t, audit.Findings, "spf.include_unavailable", SeverityWarning)
	rejectFinding(t, audit.Findings, "spf.include_missing")
}

func TestAuditSPFWalkIsBounded(t *testing.T) {
	calls := 0
	fanout := func(name string) ([]string, error) {
		calls++
		return []string{fmt.Sprintf("v=spf1 include:x%d.%s include:y%d.%s include:z%d.%s -all", calls, name, calls, name, calls, name)}, nil
	}
	audit := AuditSPF("example.com", []string{"v=spf1 include:seed.test -all"}, nil, fanout)
	if calls > spfWalkBudget+1 {
		t.Fatalf("walk performed %d lookups, budget is %d", calls, spfWalkBudget)
	}
	requireFinding(t, audit.Findings, "spf.lookups_exceeded", SeverityCritical)
	if _, ok := ids(audit.Findings)["spf.walk_limit"]; !ok {
		if _, ok := ids(audit.Findings)["spf.depth_limit"]; !ok {
			t.Fatalf("a truncated walk must be reported: %v", ids(audit.Findings))
		}
	}
}

func TestAuditSPFDeepChainIsTruncatedAndReported(t *testing.T) {
	chain := func(name string) ([]string, error) {
		return []string{"v=spf1 include:next." + name + " -all"}, nil
	}
	audit := AuditSPF("example.com", []string{"v=spf1 include:start.test -all"}, nil, chain)
	requireFinding(t, audit.Findings, "spf.depth_limit", SeverityWarning)
}

func TestFindingsHaveSectionAndRemediationWhereActionable(t *testing.T) {
	audit := AuditSPF("example.com", []string{"v=spf1 +all"}, nil, zone{}.lookup)
	for _, f := range audit.Findings {
		if f.Section != SectionSPF || f.Title == "" {
			t.Fatalf("incomplete finding %+v", f)
		}
		if f.Severity == SeverityCritical && f.Remediation == "" {
			t.Fatalf("critical finding without remediation: %+v", f)
		}
	}
}
