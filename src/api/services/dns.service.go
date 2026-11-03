package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// dnsResolver abstracts net.Resolver for testability.
type dnsResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupNS(ctx context.Context, name string) ([]*net.NS, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// newDNSResolver is the factory for DNS resolvers. Override in tests.
var newDNSResolver = func() dnsResolver {
	return fqdnResolver{inner: &net.Resolver{}}
}

// ResolveDomain godoc
// @Summary      Resolver dominio
// @Description  Resuelve un dominio a direcciones IPv4 e IPv6
// @Tags         DNS
// @Produce      json
// @Param        domain  query  string  true  "Dominio a resolver"
// @Success      200  {object}  map[string]interface{}  "domain, ipv4, ipv6, resolved"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/dns/resolve [get]
func ResolveDomain(c fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}
	if !sanitizer.IsValidDomain(domain) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_domain", "Formato de dominio invalido", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := newDNSResolver()
	ips, err := resolver.LookupIPAddr(ctx, domain)

	var ipv4List []string
	var ipv6List []string

	if err == nil {
		for _, ip := range ips {
			if ip.IP.To4() != nil {
				ipv4List = append(ipv4List, ip.IP.String())
			} else {
				ipv6List = append(ipv6List, ip.IP.String())
			}
		}
	}

	return apiresponse.Success(c, fiber.Map{
		"domain":   domain,
		"ipv4":     ipv4List,
		"ipv6":     ipv6List,
		"resolved": len(ipv4List)+len(ipv6List) > 0,
		"status":   lookupStatus(err, len(ipv4List)+len(ipv6List)),
	})
}

// CheckPropagation godoc
// @Summary      Verificar propagacion DNS
// @Description  Consulta registros DNS por tipo (A, AAAA, CNAME, MX, NS, TXT)
// @Tags         DNS
// @Produce      json
// @Param        domain  query  string  true   "Dominio"
// @Param        type    query  string  false  "Tipo de registro (default: A)"
// @Success      200  {object}  map[string]interface{}  "domain, recordType, records, timestamp"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/dns/propagation [get]
func CheckPropagation(c fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}
	if !sanitizer.IsValidDomain(domain) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_domain", "Formato de dominio invalido", nil)
	}

	recordType := strings.ToUpper(strings.TrimSpace(c.Query("type", "A")))
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := newDNSResolver()
	var records []string
	var lookupErr error

	switch recordType {
	case "A", "AAAA":
		var ips []net.IPAddr
		ips, lookupErr = resolver.LookupIPAddr(ctx, domain)
		if lookupErr == nil {
			for _, ip := range ips {
				isV4 := ip.IP.To4() != nil
				if (recordType == "A" && isV4) || (recordType == "AAAA" && !isV4) {
					records = append(records, ip.IP.String())
				}
			}
		}
	case "CNAME":
		var cname string
		cname, lookupErr = resolver.LookupCNAME(ctx, domain)
		if lookupErr == nil && cname != "" {
			records = append(records, cname)
		}
	case "MX":
		var mxs []*net.MX
		mxs, lookupErr = resolver.LookupMX(ctx, domain)
		if lookupErr == nil {
			for _, mx := range mxs {
				records = append(records, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
			}
		}
	case "NS":
		var nss []*net.NS
		nss, lookupErr = resolver.LookupNS(ctx, domain)
		if lookupErr == nil {
			for _, ns := range nss {
				records = append(records, ns.Host)
			}
		}
	case "TXT":
		var txts []string
		txts, lookupErr = resolver.LookupTXT(ctx, domain)
		if lookupErr == nil {
			records = txts
		}
	default:
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_record_type", "Tipo de registro no soportado. Usa: A, AAAA, CNAME, MX, NS, TXT", nil)
	}

	return apiresponse.Success(c, fiber.Map{
		"domain":     domain,
		"recordType": recordType,
		"records":    records,
		"status":     lookupStatus(lookupErr, len(records)),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// GetMailRecords godoc
// @Summary      Registros de correo
// @Description  Obsoleto: usa /api/tools/dns/mail-audit. Consulta registros MX, SPF, DKIM y DMARC de un dominio sin evaluarlos
// @Tags         DNS
// @Produce      json
// @Param        domain  query  string  true  "Dominio"
// @Success      200  {object}  map[string]interface{}  "domain, mx, spf, dkim, dmarc"
// @Failure      400  {object}  map[string]interface{}
// @Deprecated
// @Router       /api/tools/dns/mail-records [get]
func GetMailRecords(c fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}
	if !sanitizer.IsValidDomain(domain) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_domain", "Formato de dominio invalido", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := newDNSResolver()

	// MX records
	var mxRecords []string
	mxs, mxErr := resolver.LookupMX(ctx, domain)
	if mxErr == nil {
		for _, mx := range mxs {
			mxRecords = append(mxRecords, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
		}
	}

	// TXT records for SPF, DKIM, DMARC
	var spfRecords []string
	var dkimRecords []string
	var dmarcRecords []string

	// SPF: TXT records on the domain itself
	txts, spfErr := resolver.LookupTXT(ctx, domain)
	if spfErr == nil {
		for _, txt := range txts {
			if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
				spfRecords = append(spfRecords, txt)
			}
		}
	}

	// DKIM: TXT on default._domainkey.<domain>
	dkimTxts, dkimErr := resolver.LookupTXT(ctx, "default._domainkey."+domain)
	if dkimErr == nil {
		for _, txt := range dkimTxts {
			dkimRecords = append(dkimRecords, txt)
		}
	}

	// DMARC: TXT on _dmarc.<domain>
	dmarcTxts, dmarcErr := resolver.LookupTXT(ctx, "_dmarc."+domain)
	if dmarcErr == nil {
		for _, txt := range dmarcTxts {
			if strings.HasPrefix(strings.ToLower(txt), "v=dmarc1") {
				dmarcRecords = append(dmarcRecords, txt)
			}
		}
	}

	return apiresponse.Success(c, fiber.Map{
		"domain": domain,
		"mx":     mxRecords,
		"spf":    spfRecords,
		"dkim":   dkimRecords,
		"dmarc":  dmarcRecords,
		"status": fiber.Map{
			"mx":    lookupStatus(mxErr, len(mxRecords)),
			"spf":   lookupStatus(spfErr, len(spfRecords)),
			"dkim":  lookupStatus(dkimErr, len(dkimRecords)),
			"dmarc": lookupStatus(dmarcErr, len(dmarcRecords)),
		},
	})
}
