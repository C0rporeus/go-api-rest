package mailaudit

// spfAllFindings evaluates the terminal "all" mechanism, ptr and dead terms.
func spfAllFindings(root SPFRecord) []Finding {
	var findings []Finding
	allIndex := -1
	for i, term := range root.Terms {
		if term.Mechanism == "ptr" {
			findings = append(findings, Finding{
				ID: "spf.ptr", Section: SectionSPF, Severity: SeverityWarning,
				Title:       "El mecanismo ptr está desaconsejado",
				Detail:      "RFC 7208 §5.5 lo desaconseja: es lento, poco fiable y consume lookups.",
				Remediation: "Sustitúyelo por ip4/ip6 o include.",
			})
		}
		if term.Mechanism == "all" && allIndex < 0 {
			allIndex = i
		}
	}
	if allIndex < 0 {
		if root.Redirect == "" {
			findings = append(findings, Finding{
				ID: "spf.no_all", Section: SectionSPF, Severity: SeverityWarning,
				Title:       "El SPF no termina con 'all' ni 'redirect'",
				Detail:      "Sin política final, cualquier remitente no listado obtiene resultado neutral.",
				Remediation: "Termina con -all (o ~all mientras validas).",
			})
		}
		return findings
	}
	if allIndex < len(root.Terms)-1 {
		findings = append(findings, Finding{
			ID: "spf.terms_after_all", Section: SectionSPF, Severity: SeverityWarning,
			Title:       "Hay términos después de 'all'",
			Detail:      "Todo lo que sigue a 'all' se ignora (RFC 7208 §5.1).",
			Remediation: "Mueve 'all' al final o elimina los términos inertes.",
		})
	}
	return append(findings, allQualifierFinding(root.Terms[allIndex].Qualifier))
}

func allQualifierFinding(qualifier byte) Finding {
	switch qualifier {
	case '+':
		return Finding{
			ID: "spf.all_plus", Section: SectionSPF, Severity: SeverityCritical,
			Title:       "+all autoriza a cualquier servidor del mundo",
			Detail:      "Con +all (o 'all' sin calificador) el SPF aprueba a todos: permite suplantar el dominio.",
			Remediation: "Cambia a -all (o ~all si aún estás validando remitentes).",
		}
	case '?':
		return Finding{
			ID: "spf.all_neutral", Section: SectionSPF, Severity: SeverityWarning,
			Title:       "?all no ofrece protección",
			Detail:      "El resultado neutral es equivalente a no tener política.",
			Remediation: "Usa -all o ~all.",
		}
	case '~':
		return Finding{
			ID: "spf.all_softfail", Section: SectionSPF, Severity: SeverityInfo,
			Title:       "~all (softfail): aceptable, mejorable",
			Detail:      "Los no autorizados se marcan sospechosos pero no se rechazan. Es una buena postura si DMARC está en quarantine/reject.",
			Remediation: "Pasa a -all cuando hayas confirmado todos tus remitentes legítimos.",
		}
	default:
		return Finding{
			ID: "spf.all_fail", Section: SectionSPF, Severity: SeverityOK,
			Title: "El SPF termina en -all (política estricta)",
		}
	}
}
