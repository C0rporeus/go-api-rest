package mailaudit

import (
	"fmt"
	"net"
	"strings"
)

// Term is one SPF mechanism with its qualifier.
type Term struct {
	Qualifier byte   // '+', '-', '~' or '?'
	Mechanism string // all, include, a, mx, ptr, ip4, ip6, exists
	Value     string
}

// SPFRecord is a parsed SPF record.
type SPFRecord struct {
	Raw      string
	Terms    []Term
	Redirect string
	Exp      string
}

// IsSPFRecord reports whether a TXT record is an SPF record (RFC 7208 §4.5).
func IsSPFRecord(txt string) bool {
	text := strings.ToLower(strings.TrimSpace(txt))
	return text == "v=spf1" || strings.HasPrefix(text, "v=spf1 ")
}

// ParseSPF parses an SPF record and returns syntax findings. It never panics on
// arbitrary input (covered by a fuzz test).
func ParseSPF(raw string) (SPFRecord, []Finding) {
	record := SPFRecord{Raw: raw}
	fields := strings.Fields(raw)
	if len(fields) == 0 || !IsSPFRecord(raw) {
		return record, []Finding{spfSyntax("El registro no empieza con v=spf1")}
	}
	var findings []Finding
	for _, field := range fields[1:] {
		if message := parseSPFTerm(field, &record); message != "" {
			findings = append(findings, spfSyntax(message))
		}
	}
	return record, findings
}

func spfSyntax(detail string) Finding {
	return Finding{
		ID: "spf.syntax", Section: SectionSPF, Severity: SeverityCritical,
		Title:       "Sintaxis SPF inválida",
		Detail:      truncate(detail),
		Remediation: "Corrige el término: un SPF con errores de sintaxis se evalúa como permerror y el correo puede rechazarse.",
	}
}

// parseSPFTerm parses one term into record and returns an error message ("" if valid).
func parseSPFTerm(field string, record *SPFRecord) string {
	if name, value, ok := splitSPFModifier(field); ok {
		switch name {
		case "redirect":
			if value == "" || record.Redirect != "" {
				return fmt.Sprintf("redirect vacío o repetido: %q", field)
			}
			record.Redirect = strings.ToLower(value)
		case "exp":
			record.Exp = value
		}
		return "" // los modificadores desconocidos se ignoran (RFC 7208 §6)
	}
	term := Term{Qualifier: '+'}
	if strings.ContainsRune("+-~?", rune(field[0])) {
		term.Qualifier = field[0]
		field = field[1:]
	}
	term.Mechanism, term.Value = splitSPFMechanism(field)
	if message := validateSPFMechanism(term); message != "" {
		return message
	}
	record.Terms = append(record.Terms, term)
	return ""
}

// splitSPFModifier recognises "name=value" where '=' comes before any ':' or '/'.
func splitSPFModifier(field string) (string, string, bool) {
	eq := strings.IndexByte(field, '=')
	if eq <= 0 {
		return "", "", false
	}
	if sep := strings.IndexAny(field, ":/"); sep >= 0 && sep < eq {
		return "", "", false
	}
	name := strings.ToLower(field[:eq])
	for i, char := range name {
		letter := char >= 'a' && char <= 'z'
		other := (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.'
		if !letter && !(other && i > 0) {
			return "", "", false
		}
	}
	return name, field[eq+1:], true
}

func splitSPFMechanism(field string) (string, string) {
	sep := strings.IndexAny(field, ":/")
	if sep < 0 {
		return strings.ToLower(field), ""
	}
	name := strings.ToLower(field[:sep])
	if field[sep] == ':' {
		return name, field[sep+1:]
	}
	return name, field[sep:]
}

func validateSPFMechanism(term Term) string {
	switch term.Mechanism {
	case "all":
		if term.Value != "" {
			return "'all' no admite valor"
		}
	case "include", "exists":
		if term.Value == "" || strings.Contains(term.Value, "/") {
			return fmt.Sprintf("'%s' requiere un dominio", term.Mechanism)
		}
	case "a", "mx", "ptr":
		return validateSPFDomainCIDR(term)
	case "ip4":
		return validateSPFIP(term.Value, false)
	case "ip6":
		return validateSPFIP(term.Value, true)
	default:
		return fmt.Sprintf("mecanismo desconocido: %q", term.Mechanism)
	}
	return ""
}

func validateSPFDomainCIDR(term Term) string {
	_, cidr, hasCIDR := strings.Cut(term.Value, "/")
	if !hasCIDR {
		return ""
	}
	for _, part := range strings.Split(strings.TrimPrefix(cidr, "/"), "//") {
		if part == "" || len(part) > 3 || strings.Trim(part, "0123456789") != "" {
			return fmt.Sprintf("longitud CIDR inválida en '%s'", term.Mechanism)
		}
	}
	return ""
}

func validateSPFIP(value string, v6 bool) string {
	ip := net.ParseIP(strings.SplitN(value, "/", 2)[0])
	if _, _, err := net.ParseCIDR(value); err == nil {
		ip, _, _ = net.ParseCIDR(value)
	}
	if ip == nil || (ip.To4() == nil) != v6 {
		kind := "IPv4"
		if v6 {
			kind = "IPv6"
		}
		return fmt.Sprintf("%s inválida: %q", kind, value)
	}
	return ""
}

// lookupTerms counts the terms that cost a DNS lookup (RFC 7208 §4.6.4).
func (record SPFRecord) lookupTerms() int {
	count := 0
	for _, term := range record.Terms {
		switch term.Mechanism {
		case "include", "a", "mx", "ptr", "exists":
			count++
		}
	}
	if record.Redirect != "" {
		count++
	}
	return count
}
