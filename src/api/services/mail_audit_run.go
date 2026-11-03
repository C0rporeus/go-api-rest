package services

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/mailaudit"
)

// mailAuditSection is the raw DNS view of one audited record family.
type mailAuditSection struct {
	Status    dnsStatus `json:"status"`
	Records   []string  `json:"records"`
	Lookups   *int      `json:"lookups,omitempty"`   // SPF
	Includes  []string  `json:"includes,omitempty"`  // SPF
	Selectors []string  `json:"selectors,omitempty"` // DKIM selectors that publish a key
}

type mailAuditResponse struct {
	Domain   string                      `json:"domain"`
	Summary  map[mailaudit.Severity]int  `json:"summary"`
	Findings []mailaudit.Finding         `json:"findings"`
	Sections map[string]mailAuditSection `json:"sections"`
}

var sectionOrder = map[string]int{
	mailaudit.SectionMX: 0, mailaudit.SectionSPF: 1, mailaudit.SectionDKIM: 2,
	mailaudit.SectionDMARC: 3, mailaudit.SectionMTASTS: 4, mailaudit.SectionTLSRPT: 5,
}

var severityOrder = map[mailaudit.Severity]int{
	mailaudit.SeverityCritical: 0, mailaudit.SeverityWarning: 1, mailaudit.SeverityInfo: 2, mailaudit.SeverityOK: 3,
}

// runMailAudit queries every record family concurrently (bounded by the shared
// query budget and context) and evaluates them with the pure mailaudit rules.
func runMailAudit(ctx context.Context, resolver dnsResolver, domain, extraSelector string) mailAuditResponse {
	dns := newAuditDNS(ctx, resolver, constants.MailAuditMaxQueries)
	selectors := dkimSelectors(extraSelector)

	var (
		wg                       sync.WaitGroup
		mxFindings, mtaFindings  []mailaudit.Finding
		tlsFindings              []mailaudit.Finding
		mxSection, mtaSection    mailAuditSection
		tlsSection               mailAuditSection
		spf                      mailaudit.SPFAudit
		spfSection, dmarcSection mailAuditSection
		dmarc                    mailaudit.DMARCAudit
		dkim                     mailaudit.DKIMAudit
		probes                   = make([]mailaudit.DKIMProbe, len(selectors))
	)
	run := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(func() {
		records, err := dns.mx(domain)
		mxFindings = mailaudit.AuditMX(records, err)
		mxSection = mailAuditSection{Status: statusOfLookup(err, len(records)), Records: mxStrings(records)}
	})
	run(func() {
		txts, err := dns.txt(domain)
		spf = mailaudit.AuditSPF(domain, txts, err, dns.txt)
		lookups := spf.Lookups
		spfSection = mailAuditSection{Status: statusOfLookup(err, len(spf.Records)), Records: spf.Records, Lookups: &lookups, Includes: spf.Includes}
	})
	run(func() {
		txts, err := dns.txt("_dmarc." + domain)
		dmarc = mailaudit.AuditDMARC(txts, err)
		dmarcSection = mailAuditSection{Status: statusOfLookup(err, len(dmarc.Records)), Records: dmarc.Records}
	})
	run(func() {
		txts, err := dns.txt("_mta-sts." + domain)
		var records []string
		records, mtaFindings = mailaudit.AuditPolicyTXT(mailaudit.SectionMTASTS, txts, err)
		mtaSection = mailAuditSection{Status: statusOfLookup(err, len(records)), Records: records}
	})
	run(func() {
		txts, err := dns.txt("_smtp._tls." + domain)
		var records []string
		records, tlsFindings = mailaudit.AuditPolicyTXT(mailaudit.SectionTLSRPT, txts, err)
		tlsSection = mailAuditSection{Status: statusOfLookup(err, len(records)), Records: records}
	})
	for i, selector := range selectors {
		i, selector := i, selector
		run(func() {
			txts, err := dns.txt(selector + "._domainkey." + domain)
			probes[i] = mailaudit.DKIMProbe{Selector: selector, TXTs: txts, Err: err}
		})
	}
	wg.Wait()

	dkim = mailaudit.AuditDKIM(probes)
	dkimRecords := []string{}
	for _, selector := range dkim.Found {
		dkimRecords = append(dkimRecords, selector+": "+dkim.Keys[selector].Raw)
	}
	dkimSection := mailAuditSection{Status: dkimStatus(dkim, probes), Records: dkimRecords, Selectors: dkim.Found}

	findings := make([]mailaudit.Finding, 0, 16)
	for _, group := range [][]mailaudit.Finding{mxFindings, spf.Findings, dkim.Findings, dmarc.Findings, mtaFindings, tlsFindings} {
		findings = append(findings, group...)
	}
	sortFindings(findings)

	return mailAuditResponse{
		Domain:   domain,
		Summary:  mailaudit.Summarize(findings),
		Findings: findings,
		Sections: map[string]mailAuditSection{
			mailaudit.SectionMX: mxSection, mailaudit.SectionSPF: spfSection, mailaudit.SectionDKIM: dkimSection,
			mailaudit.SectionDMARC: dmarcSection, mailaudit.SectionMTASTS: mtaSection, mailaudit.SectionTLSRPT: tlsSection,
		},
	}
}

// dkimSelectors returns the common selectors plus the caller's (deduplicated).
func dkimSelectors(extra string) []string {
	selectors := append([]string(nil), constants.DKIMCommonSelectors...)
	if extra != "" {
		for _, selector := range selectors {
			if selector == extra {
				return selectors
			}
		}
		selectors = append([]string{extra}, selectors...)
	}
	return selectors
}

// dkimStatus is ok if any selector published a key, unavailable if every probe
// failed, and not_found otherwise.
func dkimStatus(audit mailaudit.DKIMAudit, probes []mailaudit.DKIMProbe) dnsStatus {
	if len(audit.Found) > 0 {
		return dnsStatusOK
	}
	for _, probe := range probes {
		if mailaudit.IsUnavailable(probe.Err) {
			return statusOfLookup(probe.Err, 0)
		}
	}
	return dnsStatusNotFound
}

func mxStrings(records []mailaudit.MXRecord) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, strings.TrimSuffix(record.Host, ".")+" (priority "+strconv.Itoa(int(record.Pref))+")")
	}
	return out
}

func sortFindings(findings []mailaudit.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if sectionOrder[a.Section] != sectionOrder[b.Section] {
			return sectionOrder[a.Section] < sectionOrder[b.Section]
		}
		return severityOrder[a.Severity] < severityOrder[b.Severity]
	})
}
