package mailaudit

import "testing"

func TestAuditMX(t *testing.T) {
	requireFinding(t, AuditMX(nil, ErrNotFound), "mx.missing", SeverityWarning)
	requireFinding(t, AuditMX([]MXRecord{{Host: ".", Pref: 0}}, nil), "mx.null", SeverityInfo)
	requireFinding(t, AuditMX([]MXRecord{{Host: "mx.example.com.", Pref: 10}}, nil), "mx.single", SeverityInfo)
	requireFinding(t, AuditMX([]MXRecord{{Host: "a.example.com.", Pref: 10}, {Host: "b.example.com.", Pref: 20}}, nil), "mx.ok", SeverityOK)

	findings := AuditMX(nil, errTimeout)
	requireFinding(t, findings, "mx.unavailable", SeverityWarning)
	rejectFinding(t, findings, "mx.missing")
}

func TestAuditPolicyTXT(t *testing.T) {
	cases := []struct {
		name    string
		section string
		txts    []string
		err     error
		id      string
		sev     Severity
	}{
		{"mta-sts ok", SectionMTASTS, []string{"v=STSv1; id=20260101T000000"}, nil, "mta-sts.ok", SeverityOK},
		{"mta-sts missing is informational", SectionMTASTS, []string{"other"}, nil, "mta-sts.missing", SeverityInfo},
		{"mta-sts without id", SectionMTASTS, []string{"v=STSv1"}, nil, "mta-sts.syntax", SeverityCritical},
		{"mta-sts duplicated", SectionMTASTS, []string{"v=STSv1; id=1", "v=STSv1; id=2"}, nil, "mta-sts.multiple", SeverityWarning},
		{"mta-sts unavailable", SectionMTASTS, nil, errTimeout, "mta-sts.unavailable", SeverityWarning},
		{"tls-rpt ok", SectionTLSRPT, []string{"v=TLSRPTv1; rua=mailto:t@example.com"}, nil, "tls-rpt.ok", SeverityOK},
		{"tls-rpt missing", SectionTLSRPT, nil, ErrNotFound, "tls-rpt.missing", SeverityInfo},
		{"tls-rpt without rua", SectionTLSRPT, []string{"v=TLSRPTv1"}, nil, "tls-rpt.syntax", SeverityCritical},
	}
	for _, tc := range cases {
		_, findings := AuditPolicyTXT(tc.section, tc.txts, tc.err)
		if got, ok := ids(findings)[tc.id]; !ok || got != tc.sev {
			t.Errorf("%s: %v, want %s=%s", tc.name, ids(findings), tc.id, tc.sev)
		}
	}
}

func TestSummarizeAndTruncate(t *testing.T) {
	counts := Summarize([]Finding{{Severity: SeverityCritical}, {Severity: SeverityOK}, {Severity: SeverityOK}})
	if counts[SeverityCritical] != 1 || counts[SeverityOK] != 2 || counts[SeverityWarning] != 0 {
		t.Fatalf("%v", counts)
	}
	long := make([]byte, 1000)
	for i := range long {
		long[i] = 'x'
	}
	if got := truncate(string(long)); len(got) > maxFindingText+len("…") {
		t.Fatalf("not truncated: %d", len(got))
	}
}
