package mailaudit

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// SPFMaxLookups is the RFC 7208 §4.6.4 limit of DNS-querying terms.
	SPFMaxLookups = 10
	// spfWalkBudget bounds the lookups performed while walking includes, so a
	// hostile record cannot make the audit run unbounded work.
	spfWalkBudget = 30
	spfMaxDepth   = 10
	spfNearLimit  = 8
)

// SPFAudit is the outcome of auditing a domain's SPF.
type SPFAudit struct {
	Records  []string // raw SPF records found on the domain
	Lookups  int      // DNS-querying terms counted across includes/redirects
	Includes []string // domains followed
	Findings []Finding
}

// AuditSPF evaluates the SPF of domain. txts are the domain's TXT records and
// txtErr the outcome of fetching them; lookup resolves included domains.
func AuditSPF(domain string, txts []string, txtErr error, lookup TXTLookup) SPFAudit {
	audit := SPFAudit{}
	if IsUnavailable(txtErr) {
		audit.Findings = []Finding{unavailable(SectionSPF, txtErr)}
		return audit
	}
	for _, txt := range txts {
		if IsSPFRecord(txt) {
			audit.Records = append(audit.Records, txt)
		}
	}
	switch len(audit.Records) {
	case 0:
		audit.Findings = []Finding{{
			ID: "spf.missing", Section: SectionSPF, Severity: SeverityWarning,
			Title:       "El dominio no publica SPF",
			Detail:      "Sin SPF los receptores no pueden verificar qué servidores están autorizados a enviar por este dominio.",
			Remediation: "Publica un TXT en la raíz del dominio, por ejemplo: v=spf1 include:<tu-proveedor> -all (o v=spf1 -all si el dominio no envía correo).",
		}}
		return audit
	case 1:
	default:
		audit.Findings = append(audit.Findings, Finding{
			ID: "spf.multiple", Section: SectionSPF, Severity: SeverityCritical,
			Title:       fmt.Sprintf("Hay %d registros SPF: es un permerror", len(audit.Records)),
			Detail:      "RFC 7208 §4.5 exige un único registro SPF; con varios, los receptores descartan todos.",
			Remediation: "Fusiona los mecanismos en un único TXT v=spf1 y elimina los demás.",
		})
	}

	walker := &spfWalker{lookup: lookup, visited: map[string]bool{strings.ToLower(domain): true}}
	root, syntax := ParseSPF(audit.Records[0])
	audit.Findings = append(audit.Findings, syntax...)
	walker.walk(root, 0)

	audit.Lookups = walker.lookups
	audit.Includes = walker.includes
	audit.Findings = append(audit.Findings, walker.findings...)
	audit.Findings = append(audit.Findings, spfPolicyFindings(root, walker.lookups)...)
	return audit
}

type spfWalker struct {
	lookup   TXTLookup
	visited  map[string]bool
	lookups  int
	includes []string
	findings []Finding
	capped   bool
}

// walk counts lookups of record and follows its includes and redirect.
func (w *spfWalker) walk(record SPFRecord, depth int) {
	w.lookups += record.lookupTerms()
	for _, term := range record.Terms {
		if term.Mechanism == "include" {
			w.follow(strings.ToLower(term.Value), depth, true)
		}
	}
	if record.Redirect != "" {
		w.follow(record.Redirect, depth, false)
	}
}

func (w *spfWalker) follow(target string, depth int, isInclude bool) {
	if w.visited[target] {
		w.add(Finding{
			ID: "spf.cycle", Section: SectionSPF, Severity: SeverityCritical,
			Title:       "Ciclo de includes en SPF",
			Detail:      fmt.Sprintf("%s ya se había evaluado: el SPF se refiere a sí mismo.", target),
			Remediation: "Elimina la referencia circular entre los dominios involucrados.",
		})
		return
	}
	if w.capped {
		return
	}
	if depth+1 > spfMaxDepth {
		w.capped = true
		w.add(Finding{
			ID: "spf.depth_limit", Section: SectionSPF, Severity: SeverityWarning,
			Title:       "Recorrido de SPF truncado por profundidad",
			Detail:      fmt.Sprintf("Se siguieron %d niveles de include/redirect; el conteo de lookups es un mínimo.", spfMaxDepth),
			Remediation: "Reduce el anidamiento de includes.",
		})
		return
	}
	if len(w.visited) > spfWalkBudget {
		w.capped = true
		w.add(Finding{
			ID: "spf.walk_limit", Section: SectionSPF, Severity: SeverityWarning,
			Title:       "Recorrido de SPF truncado",
			Detail:      fmt.Sprintf("Se alcanzó el límite de %d dominios consultados; el conteo de lookups es un mínimo.", spfWalkBudget),
			Remediation: "Simplifica la cadena de includes.",
		})
		return
	}
	w.visited[target] = true
	if isInclude {
		w.includes = append(w.includes, target)
	}
	txts, err := w.lookup(target)
	if err != nil {
		w.add(unavailableOrMissing(target, err))
		return
	}
	for _, txt := range txts {
		if IsSPFRecord(txt) {
			child, syntax := ParseSPF(txt)
			w.findings = append(w.findings, syntax...)
			w.walk(child, depth+1)
			return
		}
	}
	w.add(missingTarget(target))
}

func (w *spfWalker) add(finding Finding) { w.findings = append(w.findings, finding) }

func unavailableOrMissing(target string, err error) Finding {
	if errors.Is(err, ErrNotFound) {
		return missingTarget(target)
	}
	finding := unavailable(SectionSPF, err)
	finding.ID = "spf.include_unavailable"
	finding.Detail = truncate(fmt.Sprintf("No se pudo consultar %s: %v", target, err))
	return finding
}

func missingTarget(target string) Finding {
	return Finding{
		ID: "spf.include_missing", Section: SectionSPF, Severity: SeverityCritical,
		Title:       "Un include apunta a un dominio sin SPF",
		Detail:      fmt.Sprintf("%s no publica un registro SPF: RFC 7208 §5.2 lo trata como permerror.", target),
		Remediation: "Corrige o elimina ese include (comprueba que el proveedor siga publicando su SPF).",
	}
}

// spfPolicyFindings evaluates the root record's final policy and the lookup budget.
func spfPolicyFindings(root SPFRecord, lookups int) []Finding {
	var findings []Finding
	switch {
	case lookups > SPFMaxLookups:
		findings = append(findings, Finding{
			ID: "spf.lookups_exceeded", Section: SectionSPF, Severity: SeverityCritical,
			Title:       fmt.Sprintf("%d lookups: supera el límite de %d", lookups, SPFMaxLookups),
			Detail:      "Pasado el límite el SPF devuelve permerror y los receptores pueden rechazar o marcar como spam todo tu correo.",
			Remediation: "Reduce includes (consolida proveedores), reemplaza a/mx por ip4/ip6 o usa un servicio de aplanado de SPF.",
		})
	case lookups >= spfNearLimit:
		findings = append(findings, Finding{
			ID: "spf.lookups_near_limit", Section: SectionSPF, Severity: SeverityWarning,
			Title:       fmt.Sprintf("%d de %d lookups usados", lookups, SPFMaxLookups),
			Detail:      "Estás cerca del límite: agregar un proveedor más puede romper el SPF.",
			Remediation: "Planifica consolidar includes antes de añadir otro servicio de envío.",
		})
	}
	return append(findings, spfAllFindings(root)...)
}
