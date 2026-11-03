package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"context"
	"net"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// DNSBL result states. "error" means the list could not be consulted and must
// never be counted as "not_listed".
const (
	dnsblListed    = "listed"
	dnsblNotListed = "not_listed"
	dnsblError     = "error"
)

// blockedAnswerPrefix: DNSBLs answer 127.255.255.252/254/255 (Spamhaus and
// others) when they refuse the query (typo, open resolver, rate limit).
var blockedAnswerNet = &net.IPNet{IP: net.IPv4(127, 255, 255, 0), Mask: net.CIDRMask(24, 32)}

type blacklistResult struct {
	Provider string   `json:"provider"`
	Status   string   `json:"status"`
	Listed   bool     `json:"listed"`
	Codes    []string `json:"codes,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// classifyDNSBLAnswer turns a DNSBL lookup outcome into a result (without provider).
func classifyDNSBLAnswer(addrs []string, err error) blacklistResult {
	if err != nil {
		switch status := classifyDNSError(err); status {
		case dnsStatusNotFound:
			return blacklistResult{Status: dnsblNotListed}
		default:
			return blacklistResult{Status: dnsblError, Error: string(status)}
		}
	}
	if len(addrs) == 0 {
		return blacklistResult{Status: dnsblNotListed}
	}
	codes := make([]string, 0, len(addrs))
	for _, raw := range addrs {
		ip := net.ParseIP(raw)
		if ip == nil || ip.To4() == nil || ip.To4()[0] != 127 {
			return blacklistResult{Status: dnsblError, Error: "unexpected_response"}
		}
		if blockedAnswerNet.Contains(ip) {
			return blacklistResult{Status: dnsblError, Error: "query_blocked"}
		}
		codes = append(codes, raw)
	}
	return blacklistResult{Status: dnsblListed, Listed: true, Codes: codes}
}

// CheckBlacklist godoc
// @Summary      Verificar blacklist DNSBL
// @Description  Verifica una IPv4 contra varios proveedores DNSBL en paralelo. Cada proveedor responde listed, not_listed o error (no se pudo consultar); un error nunca cuenta como "limpio"
// @Tags         DNS
// @Produce      json
// @Param        ip  query  string  true  "Direccion IPv4"
// @Success      200  {object}  map[string]interface{}  "ip, results, summary"
// @Failure      400  {object}  map[string]interface{}
// @Router       /api/tools/dns/blacklist [get]
func CheckBlacklist(c fiber.Ctx) error {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_ip", "El parametro 'ip' es requerido", nil)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() == nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_ip", "La IP proporcionada no es una IPv4 valida", nil)
	}

	parts := strings.Split(parsedIP.To4().String(), ".")
	reversed := parts[3] + "." + parts[2] + "." + parts[1] + "." + parts[0]

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultDNSTimeout)
	defer cancel()

	resolver := newDNSResolver()
	results := make([]blacklistResult, len(constants.DNSBLProviders))

	var wg sync.WaitGroup
	for i, provider := range constants.DNSBLProviders {
		wg.Add(1)
		go func(idx int, prov string) {
			defer wg.Done()
			addrs, lookupErr := resolver.LookupHost(ctx, reversed+"."+prov)
			result := classifyDNSBLAnswer(addrs, lookupErr)
			result.Provider = prov
			results[idx] = result
		}(i, provider)
	}
	wg.Wait()

	return apiresponse.Success(c, fiber.Map{
		"ip":      ip,
		"results": results,
		"summary": summarizeBlacklist(results),
	})
}

// summarizeBlacklist counts outcomes so the UI can warn when lists failed.
func summarizeBlacklist(results []blacklistResult) fiber.Map {
	var listed, notListed, failed int
	for _, r := range results {
		switch r.Status {
		case dnsblListed:
			listed++
		case dnsblNotListed:
			notListed++
		default:
			failed++
		}
	}
	return fiber.Map{"total": len(results), "listed": listed, "notListed": notListed, "failed": failed, "checked": listed + notListed}
}
