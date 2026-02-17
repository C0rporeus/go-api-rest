package services

import (
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

// GetOpsMetrics godoc
// @Summary      Metricas operativas
// @Description  Snapshot de request count, 5xx rate, auth failures. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/ops/metrics [get]
func GetOpsMetrics(c *fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.Snapshot())
}

// GetOpsAlerts godoc
// @Summary      Alertas operativas
// @Description  Alertas activas basadas en umbrales de 5xx y auth failures. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/ops/alerts [get]
func GetOpsAlerts(c *fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.Alerts())
}

// GetOpsHealth godoc
// @Summary      Estado de salud
// @Description  Estado de salud con recomendaciones. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/ops/health [get]
func GetOpsHealth(c *fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.Health())
}

// GetOpsHistory godoc
// @Summary      Historial de estados
// @Description  Historial de snapshots de salud. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/ops/history [get]
func GetOpsHistory(c *fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.History())
}

// GetOpsSummary godoc
// @Summary      Resumen operativo
// @Description  Resumen agregado para semaforo de salud. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/ops/summary [get]
func GetOpsSummary(c *fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.Summary())
}
