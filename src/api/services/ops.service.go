package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/pkg/telemetry"

	"github.com/gofiber/fiber/v3"
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
func GetOpsMetrics(c fiber.Ctx) error {
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
func GetOpsAlerts(c fiber.Ctx) error {
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
func GetOpsHealth(c fiber.Ctx) error {
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
func GetOpsHistory(c fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.History())
}

// TriggerRedeploy godoc
// @Summary      Disparar redespliegue del frontend
// @Description  Llama al workflow de GitHub Actions para reconstruir y publicar el sitio. Requiere JWT.
// @Tags         Ops
// @Produce      json
// @Security     BearerAuth
// @Success      202  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      503  {object}  map[string]interface{}
// @Router       /api/private/ops/redeploy [post]
func TriggerRedeploy(c fiber.Ctx) error {
	pat := os.Getenv(constants.GitHubPATEnv)
	if pat == "" {
		return apiresponse.Error(c, fiber.StatusServiceUnavailable,
			"redeploy_not_configured",
			"El token de GitHub no está configurado en el servidor.",
			nil)
	}

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches",
		constants.GitHubOwner, constants.GitHubRepo, constants.GitHubWorkflow,
	)
	body, _ := json.Marshal(map[string]string{"ref": constants.GitHubRef})

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError,
			"redeploy_request_failed", "No se pudo construir la solicitud.", nil)
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusBadGateway,
			"redeploy_github_unreachable", "No se pudo contactar con GitHub.", nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return apiresponse.Error(c, fiber.StatusBadGateway,
			"redeploy_github_error",
			fmt.Sprintf("GitHub respondió con status %d.", resp.StatusCode),
			nil)
	}

	return apiresponse.Success(c, map[string]string{"status": "dispatched"})
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
func GetOpsSummary(c fiber.Ctx) error {
	return apiresponse.Success(c, telemetry.Summary())
}
