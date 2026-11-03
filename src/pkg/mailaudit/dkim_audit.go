package mailaudit

import (
	"fmt"
	"strings"
)

// AuditDKIM evaluates the probed selectors. Not finding a key among common
// selectors is informational only: DKIM selectors are arbitrary names.
func AuditDKIM(probes []DKIMProbe) DKIMAudit {
	audit := DKIMAudit{Keys: map[string]DKIMKey{}}
	unavailableCount := 0
	var revoked []string
	for _, probe := range probes {
		if IsUnavailable(probe.Err) {
			unavailableCount++
			continue
		}
		for _, txt := range probe.TXTs {
			if !looksLikeDKIM(txt) {
				continue
			}
			key, syntax := ParseDKIM(txt)
			if key.Revoked {
				// A revoked selector (empty p=) publishes no key. Domains with a wildcard
				// revocation answer it for every selector, so they are reported together.
				revoked = append(revoked, probe.Selector)
				break
			}
			audit.Found = append(audit.Found, probe.Selector)
			audit.Keys[probe.Selector] = key
			for _, finding := range syntax {
				finding.Title = fmt.Sprintf("[%s] %s", probe.Selector, finding.Title)
				audit.Findings = append(audit.Findings, finding)
			}
			if len(syntax) == 0 {
				audit.Findings = append(audit.Findings, dkimKeyFindings(probe.Selector, key)...)
			}
			break
		}
	}
	if len(revoked) > 0 {
		audit.Findings = append(audit.Findings, Finding{
			ID: "dkim.revoked", Section: SectionDKIM, Severity: SeverityInfo,
			Title:  fmt.Sprintf("%d selector(es) revocado(s) (p= vacío): %s", len(revoked), strings.Join(revoked, ", ")),
			Detail: "Un p= vacío declara que el selector no debe usarse; un dominio con revocación comodín (*._domainkey) lo responde para cualquier selector.",
		})
	}
	if len(audit.Found) == 0 {
		audit.Findings = append(audit.Findings, noSelectorFinding(len(probes), unavailableCount))
	}
	return audit
}

func looksLikeDKIM(txt string) bool {
	_, _, hasP := cutTag(txt, "p")
	return hasP || containsFold(txt, "v=dkim1")
}

func noSelectorFinding(probed, unavailable int) Finding {
	if unavailable == probed && probed > 0 {
		return Finding{
			ID: "dkim.unavailable", Section: SectionDKIM, Severity: SeverityWarning,
			Title:       "No se pudieron consultar los selectores DKIM",
			Remediation: "Reintenta en unos minutos.",
		}
	}
	return Finding{
		ID: "dkim.none_found", Section: SectionDKIM, Severity: SeverityInfo,
		Title:       "No se encontró DKIM en los selectores comunes",
		Detail:      fmt.Sprintf("Se probaron %d selectores habituales. Esto no significa que no exista: cada proveedor usa su propio selector.", probed),
		Remediation: "Indica el selector exacto de tu proveedor (aparece en el encabezado DKIM-Signature: s=) para auditarlo.",
	}
}

func dkimKeyFindings(selector string, key DKIMKey) []Finding {
	prefix := "[" + selector + "] "
	var findings []Finding
	switch {
	case key.Type == "rsa" && key.Bits < dkimWeakBits:
		findings = append(findings, Finding{
			ID: "dkim.key_weak", Section: SectionDKIM, Severity: SeverityCritical,
			Title:       fmt.Sprintf("%sClave RSA de %d bits: insegura", prefix, key.Bits),
			Remediation: "Rota a una clave RSA de 2048 bits (o ed25519).",
		})
	case key.Type == "rsa" && key.Bits < dkimMinGoodBits:
		findings = append(findings, Finding{
			ID: "dkim.key_short", Section: SectionDKIM, Severity: SeverityWarning,
			Title:       fmt.Sprintf("%sClave RSA de %d bits: se recomienda 2048", prefix, key.Bits),
			Remediation: "Rota a 2048 bits; los proveedores grandes ya lo recomiendan.",
		})
	default:
		findings = append(findings, Finding{
			ID: "dkim.key_ok", Section: SectionDKIM, Severity: SeverityOK,
			Title: fmt.Sprintf("%sClave %s de %d bits", prefix, key.Type, key.Bits),
		})
	}
	if key.Testing {
		findings = append(findings, Finding{
			ID: "dkim.testing", Section: SectionDKIM, Severity: SeverityWarning,
			Title:       prefix + "Modo de pruebas activo (t=y)",
			Detail:      "Los receptores no deben tratar los fallos de este selector como definitivos.",
			Remediation: "Quita t=y cuando termines las pruebas.",
		})
	}
	return findings
}
