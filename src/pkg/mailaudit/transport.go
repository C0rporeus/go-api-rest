package mailaudit

import (
	"fmt"
	"strings"
)

// MXRecord is a mail exchanger.
type MXRecord struct {
	Host string
	Pref uint16
}

// AuditMX evaluates the MX records of a domain.
func AuditMX(records []MXRecord, err error) []Finding {
	if IsUnavailable(err) {
		return []Finding{unavailable(SectionMX, err)}
	}
	switch {
	case len(records) == 0:
		return []Finding{{
			ID: "mx.missing", Section: SectionMX, Severity: SeverityWarning,
			Title:       "El dominio no tiene registros MX",
			Detail:      "Sin MX los remitentes intentarán el registro A como último recurso; lo normal es declarar los servidores de correo.",
			Remediation: "Publica los MX de tu proveedor, o un null MX (0 .) si el dominio no debe recibir correo.",
		}}
	case len(records) == 1 && strings.Trim(records[0].Host, ".") == "":
		return []Finding{{
			ID: "mx.null", Section: SectionMX, Severity: SeverityInfo,
			Title: "Null MX (RFC 7505): el dominio declara que no recibe correo",
		}}
	case len(records) == 1:
		return []Finding{{
			ID: "mx.single", Section: SectionMX, Severity: SeverityInfo,
			Title:       "Un solo servidor MX: sin redundancia",
			Detail:      fmt.Sprintf("Si %s no está disponible, el correo entrante se encola o rebota.", strings.TrimSuffix(records[0].Host, ".")),
			Remediation: "Añade un MX secundario si tu proveedor lo ofrece.",
		}}
	}
	return []Finding{{
		ID: "mx.ok", Section: SectionMX, Severity: SeverityOK,
		Title: fmt.Sprintf("%d servidores MX publicados", len(records)),
	}}
}

// AuditPolicyTXT evaluates MTA-STS (_mta-sts) or TLS-RPT (_smtp._tls) records.
// Both are optional hardening, so their absence is informational.
func AuditPolicyTXT(section string, txts []string, err error) ([]string, []Finding) {
	prefix, requiredTag, title, remediation := "v=stsv1", "id", "MTA-STS", "Publica _mta-sts TXT (v=STSv1; id=<id>) y la política en https://mta-sts.<dominio>/.well-known/mta-sts.txt para exigir TLS en el correo entrante."
	if section == SectionTLSRPT {
		prefix, requiredTag, title, remediation = "v=tlsrptv1", "rua", "TLS-RPT", "Publica _smtp._tls TXT (v=TLSRPTv1; rua=mailto:<buzón>) para recibir reportes de fallos TLS."
	}
	if IsUnavailable(err) {
		return nil, []Finding{unavailable(section, err)}
	}
	var records []string
	for _, txt := range txts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(txt)), prefix) {
			records = append(records, txt)
		}
	}
	switch {
	case len(records) == 0:
		return nil, []Finding{{ID: section + ".missing", Section: section, Severity: SeverityInfo, Title: title + " no está configurado", Remediation: remediation}}
	case len(records) > 1:
		return records, []Finding{{ID: section + ".multiple", Section: section, Severity: SeverityWarning, Title: fmt.Sprintf("Hay %d registros %s", len(records), title), Remediation: "Deja uno solo."}}
	}
	if _, value, ok := cutTag(records[0], requiredTag); !ok || value == "" {
		return records, []Finding{{ID: section + ".syntax", Section: section, Severity: SeverityCritical, Title: fmt.Sprintf("%s sin la etiqueta %s=", title, requiredTag), Remediation: "Corrige el registro."}}
	}
	return records, []Finding{{ID: section + ".ok", Section: section, Severity: SeverityOK, Title: title + " configurado"}}
}
