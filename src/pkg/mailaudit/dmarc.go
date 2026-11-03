package mailaudit

import (
	"fmt"
	"strconv"
	"strings"
)

// DMARCRecord is a parsed DMARC record: tag -> value (lower-case tag names).
type DMARCRecord struct {
	Raw  string
	Tags map[string]string
}

// IsDMARCRecord reports whether a TXT record is a DMARC record.
func IsDMARCRecord(txt string) bool {
	text := strings.ToLower(strings.TrimSpace(txt))
	return strings.HasPrefix(text, "v=dmarc1")
}

// ParseDMARC parses a DMARC record and returns syntax findings. It never
// panics on arbitrary input (covered by a fuzz test).
func ParseDMARC(raw string) (DMARCRecord, []Finding) {
	record := DMARCRecord{Raw: raw, Tags: map[string]string{}}
	if !IsDMARCRecord(raw) {
		return record, []Finding{dmarcSyntax("El registro no empieza con v=DMARC1")}
	}
	var findings []Finding
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		name = strings.ToLower(strings.TrimSpace(name))
		if !ok || name == "" {
			findings = append(findings, dmarcSyntax(fmt.Sprintf("etiqueta mal formada: %q", truncate(part))))
			continue
		}
		if _, duplicated := record.Tags[name]; duplicated {
			findings = append(findings, dmarcSyntax(fmt.Sprintf("etiqueta repetida: %q", name)))
			continue
		}
		record.Tags[name] = strings.TrimSpace(value)
	}
	return record, append(findings, validateDMARCTags(record.Tags)...)
}

func dmarcSyntax(detail string) Finding {
	return Finding{
		ID: "dmarc.syntax", Section: SectionDMARC, Severity: SeverityCritical,
		Title:       "Sintaxis DMARC inválida",
		Detail:      truncate(detail),
		Remediation: "Corrige el registro: un DMARC inválido se ignora por completo y el dominio queda sin política.",
	}
}

func validateDMARCTags(tags map[string]string) []Finding {
	var findings []Finding
	check := func(ok bool, detail string) {
		if !ok {
			findings = append(findings, dmarcSyntax(detail))
		}
	}
	policy, hasPolicy := tags["p"]
	check(hasPolicy, "falta la etiqueta obligatoria p=")
	if hasPolicy {
		check(isDMARCPolicy(policy), fmt.Sprintf("p=%q no es none, quarantine ni reject", truncate(policy)))
	}
	if sp, ok := tags["sp"]; ok {
		check(isDMARCPolicy(sp), fmt.Sprintf("sp=%q no es none, quarantine ni reject", truncate(sp)))
	}
	for _, tag := range []string{"adkim", "aspf"} {
		if value, ok := tags[tag]; ok {
			check(value == "r" || value == "s", fmt.Sprintf("%s=%q debe ser r o s", tag, truncate(value)))
		}
	}
	if pct, ok := tags["pct"]; ok {
		n, err := strconv.Atoi(pct)
		check(err == nil && n >= 0 && n <= 100, fmt.Sprintf("pct=%q debe estar entre 0 y 100", truncate(pct)))
	}
	for _, tag := range []string{"rua", "ruf"} {
		if value, ok := tags[tag]; ok {
			check(validDMARCURIs(value), fmt.Sprintf("%s debe ser una lista de URIs mailto:", tag))
		}
	}
	return findings
}

func isDMARCPolicy(value string) bool {
	switch strings.ToLower(value) {
	case "none", "quarantine", "reject":
		return true
	}
	return false
}

func validDMARCURIs(value string) bool {
	for _, uri := range strings.Split(value, ",") {
		uri = strings.TrimSpace(uri)
		if !strings.HasPrefix(strings.ToLower(uri), "mailto:") || len(uri) <= len("mailto:") {
			return false
		}
	}
	return true
}
