package services

import (
	"context"
	"regexp"
	"strings"

	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/sanitizer"

	"github.com/gofiber/fiber/v3"
)

// dkimSelectorPattern accepts RFC 6376 selectors: labels of letters, digits and
// hyphens, optionally dot-separated (e.g. "mail", "s1", "2026-01.eu").
var dkimSelectorPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

// MailAudit godoc
// @Summary      Auditar autenticación de correo
// @Description  Evalúa MX, SPF (con conteo de lookups), DKIM, DMARC, MTA-STS y TLS-RPT de un dominio y devuelve hallazgos con severidad y remediación. Un registro que no se pudo consultar se reporta como "unavailable", nunca como ausente ni correcto
// @Tags         DNS
// @Produce      json
// @Param        domain    query  string  true   "Dominio a auditar"
// @Param        selector  query  string  false  "Selector DKIM adicional a comprobar"
// @Success      200  {object}  map[string]interface{}  "domain, summary, findings, sections"
// @Failure      400  {object}  map[string]interface{}
// @Failure      429  {object}  map[string]interface{}
// @Router       /api/tools/dns/mail-audit [get]
func MailAudit(c fiber.Ctx) error {
	domain := strings.ToLower(strings.TrimSpace(c.Query("domain")))
	if domain == "" {
		return apiresponse.Error(c, fiber.StatusBadRequest, "missing_domain", "El parametro 'domain' es requerido", nil)
	}
	if !sanitizer.IsValidDomain(domain) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_domain", "Formato de dominio invalido", nil)
	}
	selector := strings.ToLower(strings.TrimSpace(c.Query("selector")))
	if selector != "" && (len(selector) > constants.MaxDKIMSelectorLen || !dkimSelectorPattern.MatchString(selector)) {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_selector", "El selector DKIM solo admite letras, numeros, guiones y puntos", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.MailAuditTimeout)
	defer cancel()

	return apiresponse.Success(c, runMailAudit(ctx, newDNSResolver(), domain, selector))
}
