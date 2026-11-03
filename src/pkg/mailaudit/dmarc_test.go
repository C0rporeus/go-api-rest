package mailaudit

import "testing"

func TestParseDMARCValid(t *testing.T) {
	record, findings := ParseDMARC("v=DMARC1; p=reject; sp=quarantine; pct=100; rua=mailto:a@example.com, mailto:b@example.com; adkim=s; aspf=r; fo=1")
	if len(findings) != 0 {
		t.Fatalf("unexpected findings %v", ids(findings))
	}
	if record.Tags["p"] != "reject" || record.Tags["sp"] != "quarantine" || record.Tags["fo"] != "1" {
		t.Fatalf("tags %v", record.Tags)
	}
}

func TestParseDMARCSyntaxErrors(t *testing.T) {
	for _, raw := range []string{
		"v=DMARC1",
		"v=DMARC1; p=block",
		"v=DMARC1; p=reject; sp=maybe",
		"v=DMARC1; p=reject; pct=150",
		"v=DMARC1; p=reject; pct=-1",
		"v=DMARC1; p=reject; pct=abc",
		"v=DMARC1; p=reject; adkim=x",
		"v=DMARC1; p=reject; aspf=strict",
		"v=DMARC1; p=reject; rua=admin@example.com",
		"v=DMARC1; p=reject; rua=mailto:",
		"v=DMARC1; p=reject; p=none",
		"v=DMARC1; p=reject; ???",
		"p=reject",
		"",
	} {
		_, findings := ParseDMARC(raw)
		requireFinding(t, findings, "dmarc.syntax", SeverityCritical)
	}
}

func TestAuditDMARCMissingAndUnavailableAreDifferent(t *testing.T) {
	requireFinding(t, AuditDMARC([]string{"unrelated"}, nil).Findings, "dmarc.missing", SeverityWarning)
	audit := AuditDMARC(nil, errTimeout)
	requireFinding(t, audit.Findings, "dmarc.unavailable", SeverityWarning)
	rejectFinding(t, audit.Findings, "dmarc.missing")
}

func TestAuditDMARCMultipleRecordsAreDiscarded(t *testing.T) {
	audit := AuditDMARC([]string{"v=DMARC1; p=none", "v=DMARC1; p=reject"}, nil)
	requireFinding(t, audit.Findings, "dmarc.multiple", SeverityCritical)
}

func TestAuditDMARCPolicies(t *testing.T) {
	cases := []struct {
		raw string
		id  string
		sev Severity
	}{
		{"v=DMARC1; p=none; rua=mailto:r@example.com", "dmarc.policy_none", SeverityWarning},
		{"v=DMARC1; p=quarantine; rua=mailto:r@example.com", "dmarc.policy_quarantine", SeverityOK},
		{"v=DMARC1; p=reject; rua=mailto:r@example.com", "dmarc.policy_reject", SeverityOK},
		{"v=DMARC1; p=reject", "dmarc.no_rua", SeverityWarning},
		{"v=DMARC1; p=reject; pct=25; rua=mailto:r@example.com", "dmarc.pct", SeverityWarning},
		{"v=DMARC1; p=reject; sp=none; rua=mailto:r@example.com", "dmarc.subdomain_weaker", SeverityInfo},
	}
	for _, tc := range cases {
		requireFinding(t, AuditDMARC([]string{tc.raw}, nil).Findings, tc.id, tc.sev)
	}
}

func TestAuditDMARCPctIsIgnoredWhenPolicyIsNone(t *testing.T) {
	rejectFinding(t, AuditDMARC([]string{"v=DMARC1; p=none; pct=10; rua=mailto:r@example.com"}, nil).Findings, "dmarc.pct")
}

func TestAuditDMARCSyntaxErrorSkipsPolicyFindings(t *testing.T) {
	audit := AuditDMARC([]string{"v=DMARC1; p=bogus"}, nil)
	requireFinding(t, audit.Findings, "dmarc.syntax", SeverityCritical)
	rejectFinding(t, audit.Findings, "dmarc.no_rua")
}

func TestAuditDMARCWellConfiguredHasNoProblems(t *testing.T) {
	audit := AuditDMARC([]string{"v=DMARC1; p=reject; rua=mailto:r@example.com; adkim=s; aspf=s"}, nil)
	for _, f := range audit.Findings {
		if f.Severity == SeverityCritical || f.Severity == SeverityWarning {
			t.Fatalf("well configured DMARC must not warn: %+v", f)
		}
	}
}
