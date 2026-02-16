package authServices

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

func ResolveDomain(c *fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := &net.Resolver{}
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
	})
}

func CheckPropagation(c *fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}

	recordType := strings.ToUpper(strings.TrimSpace(c.Query("type", "A")))
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := &net.Resolver{}
	var records []string

	switch recordType {
	case "A", "AAAA":
		ips, err := resolver.LookupIPAddr(ctx, domain)
		if err == nil {
			for _, ip := range ips {
				isV4 := ip.IP.To4() != nil
				if (recordType == "A" && isV4) || (recordType == "AAAA" && !isV4) {
					records = append(records, ip.IP.String())
				}
			}
		}
	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, domain)
		if err == nil && cname != "" {
			records = append(records, cname)
		}
	case "MX":
		mxs, err := resolver.LookupMX(ctx, domain)
		if err == nil {
			for _, mx := range mxs {
				records = append(records, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
			}
		}
	case "NS":
		nss, err := resolver.LookupNS(ctx, domain)
		if err == nil {
			for _, ns := range nss {
				records = append(records, ns.Host)
			}
		}
	case "TXT":
		txts, err := resolver.LookupTXT(ctx, domain)
		if err == nil {
			records = txts
		}
	default:
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_record_type", "Tipo de registro no soportado. Usa: A, AAAA, CNAME, MX, NS, TXT", nil)
	}

	return apiresponse.Success(c, fiber.Map{
		"domain":     domain,
		"recordType": recordType,
		"records":    records,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func GetMailRecords(c *fiber.Ctx) error {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := &net.Resolver{}

	// MX records
	var mxRecords []string
	mxs, err := resolver.LookupMX(ctx, domain)
	if err == nil {
		for _, mx := range mxs {
			mxRecords = append(mxRecords, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
		}
	}

	// TXT records for SPF, DKIM, DMARC
	var spfRecords []string
	var dkimRecords []string
	var dmarcRecords []string

	// SPF: TXT records on the domain itself
	txts, err := resolver.LookupTXT(ctx, domain)
	if err == nil {
		for _, txt := range txts {
			if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
				spfRecords = append(spfRecords, txt)
			}
		}
	}

	// DKIM: TXT on default._domainkey.<domain>
	dkimTxts, err := resolver.LookupTXT(ctx, "default._domainkey."+domain)
	if err == nil {
		for _, txt := range dkimTxts {
			dkimRecords = append(dkimRecords, txt)
		}
	}

	// DMARC: TXT on _dmarc.<domain>
	dmarcTxts, err := resolver.LookupTXT(ctx, "_dmarc."+domain)
	if err == nil {
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
	})
}

func CheckBlacklist(c *fiber.Ctx) error {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_ip", "El parametro 'ip' es requerido", nil)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() == nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_ip", "La IP proporcionada no es una IPv4 valida", nil)
	}

	// Reverse the IP octets
	parts := strings.Split(parsedIP.To4().String(), ".")
	reversed := parts[3] + "." + parts[2] + "." + parts[1] + "." + parts[0]

	type blacklistResult struct {
		Provider string `json:"provider"`
		Listed   bool   `json:"listed"`
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := &net.Resolver{}
	results := make([]blacklistResult, len(constants.DNSBLProviders))

	var wg sync.WaitGroup
	for i, provider := range constants.DNSBLProviders {
		wg.Add(1)
		go func(idx int, prov string) {
			defer wg.Done()
			query := reversed + "." + prov
			addrs, lookupErr := resolver.LookupHost(ctx, query)
			results[idx] = blacklistResult{
				Provider: prov,
				Listed:   lookupErr == nil && len(addrs) > 0,
			}
		}(i, provider)
	}
	wg.Wait()

	return apiresponse.Success(c, fiber.Map{
		"ip":      ip,
		"results": results,
	})
}
