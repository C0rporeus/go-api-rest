package mailaudit

import (
	"fmt"
	"strconv"
	"strings"
)

// DMARCAudit is the outcome of auditing a domain's DMARC.
type DMARCAudit struct {
	Records  []string
	Findings []Finding
}

// AuditDMARC evaluates the TXT records found at _dmarc.<domain>.
func AuditDMARC(txts []string, txtErr error) DMARCAudit {
	audit := DMARCAudit{}
	if IsUnavailable(txtErr) {
		audit.Findings = []Finding{unavailable(SectionDMARC, txtErr)}
		return audit
	}
	for _, txt := range txts {
		if IsDMARCRecord(txt) {
			audit.Records = append(audit.Records, txt)
		}
	}
	switch len(audit.Records) {
	case 0:
		audit.Findings = []Finding{{
			ID: "dmarc.missing", Section: SectionDMARC, Severity: SeverityWarning,
			Title:       "El dominio no publica DMARC",
			Detail:      "Sin DMARC nadie te avisa de suplantaciones y los receptores no tienen política para fallos de SPF/DKIM. Los grandes proveedores ya lo exigen a los remitentes masivos. Si es un subdominio, hereda la política del dominio organizacional.",
			Remediation: "Publica un TXT en _dmarc.<dominio>, p. ej.: v=DMARC1; p=none; rua=mailto:dmarc@<dominio> y endurece la política gradualmente.",
		}}
		return audit
	case 1:
	default:
		audit.Findings = append(audit.Findings, Finding{
			ID: "dmarc.multiple", Section: SectionDMARC, Severity: SeverityCritical,
			Title:       fmt.Sprintf("Hay %d registros DMARC: se ignoran todos", len(audit.Records)),
			Detail:      "RFC 7489 §6.6.3: con más de un registro DMARC los receptores lo descartan.",
			Remediation: "Deja un único TXT en _dmarc.<dominio>.",
		})
		return audit
	}
	record, syntax := ParseDMARC(audit.Records[0])
	audit.Findings = append(audit.Findings, syntax...)
	if len(syntax) == 0 {
		audit.Findings = append(audit.Findings, dmarcPolicyFindings(record.Tags)...)
	}
	return audit
}

func dmarcPolicyFindings(tags map[string]string) []Finding {
	var findings []Finding
	policy := strings.ToLower(tags["p"])
	switch policy {
	case "none":
		findings = append(findings, Finding{
			ID: "dmarc.policy_none", Section: SectionDMARC, Severity: SeverityWarning,
			Title:       "p=none: solo monitoreo, no protege",
			Detail:      "Los correos que fallan SPF/DKIM se entregan igualmente: no impide la suplantación.",
			Remediation: "Revisa los reportes y avanza a p=quarantine y luego p=reject.",
		})
	case "quarantine":
		findings = append(findings, Finding{
			ID: "dmarc.policy_quarantine", Section: SectionDMARC, Severity: SeverityOK,
			Title: "p=quarantine: los fallos van a spam",
		})
	case "reject":
		findings = append(findings, Finding{
			ID: "dmarc.policy_reject", Section: SectionDMARC, Severity: SeverityOK,
			Title: "p=reject: los fallos se rechazan (máxima protección)",
		})
	}
	if pct, ok := tags["pct"]; ok && policy != "none" {
		if n, err := strconv.Atoi(pct); err == nil && n < 100 {
			findings = append(findings, Finding{
				ID: "dmarc.pct", Section: SectionDMARC, Severity: SeverityWarning,
				Title:       fmt.Sprintf("pct=%d: la política solo aplica al %d%% del correo", n, n),
				Remediation: "Sube pct a 100 cuando termines el despliegue gradual.",
			})
		}
	}
	if _, ok := tags["rua"]; !ok {
		findings = append(findings, Finding{
			ID: "dmarc.no_rua", Section: SectionDMARC, Severity: SeverityWarning,
			Title:       "Sin reportes agregados (rua)",
			Detail:      "Sin rua no sabrás quién envía en nombre de tu dominio ni si tu propio correo está fallando.",
			Remediation: "Añade rua=mailto:<buzón> (o un servicio de reportes DMARC).",
		})
	}
	if sp, ok := tags["sp"]; ok && rank(strings.ToLower(sp)) < rank(policy) {
		findings = append(findings, Finding{
			ID: "dmarc.subdomain_weaker", Section: SectionDMARC, Severity: SeverityInfo,
			Title:       fmt.Sprintf("Los subdominios usan sp=%s, más débil que p=%s", sp, policy),
			Remediation: "Alinea sp con p salvo que haya subdominios que aún no puedas proteger.",
		})
	}
	return findings
}

// rank orders DMARC policies by strictness.
func rank(policy string) int {
	switch policy {
	case "reject":
		return 2
	case "quarantine":
		return 1
	}
	return 0
}
