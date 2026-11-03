// Package mailaudit evaluates email-authentication DNS records (SPF, DKIM,
// DMARC, MX, MTA-STS, TLS-RPT). It is pure logic: every DNS query is delegated
// to a lookup function supplied by the caller, so the rules are testable
// offline and the caller controls budgets, timeouts and network policy.
package mailaudit

import "errors"

// Severity ranks a finding.
type Severity string

const (
	SeverityCritical Severity = "critical" // breaks delivery or authentication
	SeverityWarning  Severity = "warning"  // weakens protection or could not be checked
	SeverityInfo     Severity = "info"     // worth knowing, no action required
	SeverityOK       Severity = "ok"       // correct configuration
)

// Section names used in findings.
const (
	SectionMX      = "mx"
	SectionSPF     = "spf"
	SectionDKIM    = "dkim"
	SectionDMARC   = "dmarc"
	SectionMTASTS  = "mta-sts"
	SectionTLSRPT  = "tls-rpt"
	maxFindingText = 300
)

// Finding is one observation about a record, with how to fix it.
type Finding struct {
	ID          string   `json:"id"`
	Section     string   `json:"section"`
	Severity    Severity `json:"severity"`
	Title       string   `json:"title"`
	Detail      string   `json:"detail,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// ErrNotFound is returned by lookups when the name does not exist or has no
// record of the requested type. Any other error means "could not be checked".
var ErrNotFound = errors.New("mailaudit: record not found")

// TXTLookup resolves TXT records for a name.
type TXTLookup func(name string) ([]string, error)

// IsUnavailable reports whether err means the lookup failed (timeout, SERVFAIL,
// budget exhausted...) as opposed to a definitive "no such record".
func IsUnavailable(err error) bool {
	return err != nil && !errors.Is(err, ErrNotFound)
}

// unavailable builds the finding used when a record could not be checked. A
// check that could not run is never reported as "missing" or "correct".
func unavailable(section string, err error) Finding {
	return Finding{
		ID:          section + ".unavailable",
		Section:     section,
		Severity:    SeverityWarning,
		Title:       "No se pudo consultar este registro",
		Detail:      truncate("La consulta DNS no se completó: " + err.Error()),
		Remediation: "Reintenta en unos minutos; si persiste, revisa que los servidores autoritativos del dominio respondan.",
	}
}

func truncate(text string) string {
	if len(text) <= maxFindingText {
		return text
	}
	return text[:maxFindingText] + "…"
}

// Summarize counts findings per severity.
func Summarize(findings []Finding) map[Severity]int {
	counts := map[Severity]int{SeverityCritical: 0, SeverityWarning: 0, SeverityInfo: 0, SeverityOK: 0}
	for _, finding := range findings {
		counts[finding.Severity]++
	}
	return counts
}
