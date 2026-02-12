package telemetry

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	startedAtUnix  = time.Now().UTC().Unix()
	requestCount   uint64
	error5xxCount  uint64
	authFailCount  uint64

	windowMu     sync.Mutex
	windowEvents []windowEvent

	historyMu          sync.Mutex
	healthHistoryItems []healthHistoryItem
)

type windowEvent struct {
	timestamp    time.Time
	statusCode   int
	isAuthFailure bool
}

type healthHistoryItem struct {
	TimestampUnix int64    `json:"timestampUnix"`
	Status        string   `json:"status"`
	Scope         string   `json:"scope"`
	RequestsTotal uint64   `json:"requestsTotal"`
	WindowRequests uint64  `json:"windowRequests"`
	ErrorRate     float64  `json:"errorRate"`
	AuthFailRate  float64  `json:"authFailRate"`
	Reasons       []string `json:"reasons"`
}

func TrackRequest(statusCode int, isAuthFailure bool) {
	atomic.AddUint64(&requestCount, 1)
	if statusCode >= 500 {
		atomic.AddUint64(&error5xxCount, 1)
	}
	if isAuthFailure {
		atomic.AddUint64(&authFailCount, 1)
	}

	trackWindowEvent(statusCode, isAuthFailure)
	recordHealthSnapshot()
}

func Snapshot() map[string]interface{} {
	requests := atomic.LoadUint64(&requestCount)
	errors5xx := atomic.LoadUint64(&error5xxCount)
	authFails := atomic.LoadUint64(&authFailCount)
	windowStats := recentWindowStats()

	var errorRate float64
	var authFailRate float64
	if requests > 0 {
		errorRate = float64(errors5xx) / float64(requests)
		authFailRate = float64(authFails) / float64(requests)
	}

	return map[string]interface{}{
		"startedAtUnix": startedAtUnix,
		"requestsTotal": requests,
		"errors5xx":     errors5xx,
		"authFailures":  authFails,
		"errorRate":     errorRate,
		"authFailRate":  authFailRate,
		"window":        windowStats,
	}
}

func Alerts() map[string]interface{} {
	snapshot := Snapshot()

	requests := snapshot["requestsTotal"].(uint64)
	errorRate := snapshot["errorRate"].(float64)
	authFailRate := snapshot["authFailRate"].(float64)
	window := snapshot["window"].(map[string]interface{})
	windowRequests := window["requests"].(uint64)
	windowErrorRate := window["errorRate"].(float64)
	windowAuthFailRate := window["authFailRate"].(float64)

	minRequests := getEnvUint64("OPS_ALERT_MIN_REQUESTS", 20)
	warn5xxRate := getEnvFloat64("OPS_WARN_5XX_RATE", 0.05)
	critical5xxRate := getEnvFloat64("OPS_CRITICAL_5XX_RATE", 0.10)
	warnAuthFailRate := getEnvFloat64("OPS_WARN_AUTH_FAIL_RATE", 0.10)
	criticalAuthFailRate := getEnvFloat64("OPS_CRITICAL_AUTH_FAIL_RATE", 0.20)

	level := "ok"
	reasons := []string{}

	activeRequests := requests
	activeErrorRate := errorRate
	activeAuthFailRate := authFailRate
	evaluationScope := "total"
	if windowRequests >= minRequests {
		activeRequests = windowRequests
		activeErrorRate = windowErrorRate
		activeAuthFailRate = windowAuthFailRate
		evaluationScope = "window"
	}

	if activeRequests < minRequests {
		reasons = append(reasons, "insufficient_data")
	} else {
		if activeErrorRate >= critical5xxRate {
			level = "critical"
			reasons = append(reasons, "high_5xx_rate")
		} else if activeErrorRate >= warn5xxRate {
			if level != "critical" {
				level = "warn"
			}
			reasons = append(reasons, "elevated_5xx_rate")
		}

		if activeAuthFailRate >= criticalAuthFailRate {
			level = "critical"
			reasons = append(reasons, "high_auth_failure_rate")
		} else if activeAuthFailRate >= warnAuthFailRate {
			if level != "critical" {
				level = "warn"
			}
			reasons = append(reasons, "elevated_auth_failure_rate")
		}
	}

	return map[string]interface{}{
		"level":            level,
		"reasons":          reasons,
		"evaluationScope":  evaluationScope,
		"thresholds": map[string]interface{}{
			"minRequests":          minRequests,
			"warn5xxRate":          warn5xxRate,
			"critical5xxRate":      critical5xxRate,
			"warnAuthFailRate":     warnAuthFailRate,
			"criticalAuthFailRate": criticalAuthFailRate,
		},
		"snapshot": snapshot,
	}
}

func Health() map[string]interface{} {
	alerts := Alerts()
	level := alerts["level"].(string)

	recommendations := []string{}
	switch level {
	case "critical":
		recommendations = append(recommendations,
			"Revisar logs por requestId y errores 5xx recientes",
			"Validar estado de dependencias externas y rutas privadas de auth",
		)
	case "warn":
		recommendations = append(recommendations,
			"Monitorear tendencia de errores y auth failures en la ventana actual",
			"Verificar si existen picos de carga o intentos invalidos recurrentes",
		)
	default:
		recommendations = append(recommendations,
			"Estado operativo estable",
			"Mantener monitoreo y revisar scorecard periodicamente",
		)
	}

	return map[string]interface{}{
		"status":          level,
		"generatedAtUnix": time.Now().UTC().Unix(),
		"alerts":          alerts,
		"recommendations": recommendations,
	}
}

func History() map[string]interface{} {
	historyMu.Lock()
	defer historyMu.Unlock()

	items := make([]healthHistoryItem, len(healthHistoryItems))
	copy(items, healthHistoryItems)

	return map[string]interface{}{
		"items": items,
		"count": len(items),
	}
}

func Summary() map[string]interface{} {
	historyMu.Lock()
	items := make([]healthHistoryItem, len(healthHistoryItems))
	copy(items, healthHistoryItems)
	historyMu.Unlock()

	size := int(getEnvUint64("OPS_SUMMARY_SAMPLE_SIZE", 50))
	if size <= 0 {
		size = 50
	}
	if len(items) > size {
		items = items[len(items)-size:]
	}

	total := len(items)
	okCount := 0
	warnCount := 0
	criticalCount := 0

	var errorRateSum float64
	var authFailRateSum float64

	for _, item := range items {
		switch item.Status {
		case "critical":
			criticalCount++
		case "warn":
			warnCount++
		default:
			okCount++
		}
		errorRateSum += item.ErrorRate
		authFailRateSum += item.AuthFailRate
	}

	avgErrorRate := 0.0
	avgAuthFailRate := 0.0
	if total > 0 {
		avgErrorRate = errorRateSum / float64(total)
		avgAuthFailRate = authFailRateSum / float64(total)
	}

	level := "ok"
	if criticalCount > 0 {
		level = "critical"
	} else if warnCount > 0 {
		level = "warn"
	}

	currentHealth := Health()

	return map[string]interface{}{
		"status": level,
		"samples": map[string]interface{}{
			"size":  size,
			"count": total,
		},
		"distribution": map[string]interface{}{
			"ok":       okCount,
			"warn":     warnCount,
			"critical": criticalCount,
		},
		"averages": map[string]interface{}{
			"errorRate":    avgErrorRate,
			"authFailRate": avgAuthFailRate,
		},
		"currentHealth": currentHealth,
	}
}

func getEnvFloat64(name string, fallback float64) float64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvUint64(name string, fallback uint64) uint64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func windowSeconds() int64 {
	value := os.Getenv("OPS_WINDOW_SECONDS")
	if value == "" {
		return 300
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 300
	}
	return parsed
}

func trackWindowEvent(statusCode int, isAuthFailure bool) {
	windowMu.Lock()
	defer windowMu.Unlock()

	now := time.Now().UTC()
	windowEvents = append(windowEvents, windowEvent{
		timestamp:    now,
		statusCode:   statusCode,
		isAuthFailure: isAuthFailure,
	})
	pruneWindow(now)
}

func pruneWindow(now time.Time) {
	cutoff := now.Add(-time.Duration(windowSeconds()) * time.Second)
	firstValid := 0
	for i, event := range windowEvents {
		if event.timestamp.After(cutoff) {
			firstValid = i
			break
		}
		firstValid = i + 1
	}
	if firstValid > 0 {
		windowEvents = append([]windowEvent{}, windowEvents[firstValid:]...)
	}
}

func recentWindowStats() map[string]interface{} {
	windowMu.Lock()
	defer windowMu.Unlock()

	now := time.Now().UTC()
	pruneWindow(now)

	var requests uint64
	var errors5xx uint64
	var authFails uint64
	for _, event := range windowEvents {
		requests++
		if event.statusCode >= 500 {
			errors5xx++
		}
		if event.isAuthFailure {
			authFails++
		}
	}

	var errorRate float64
	var authFailRate float64
	if requests > 0 {
		errorRate = float64(errors5xx) / float64(requests)
		authFailRate = float64(authFails) / float64(requests)
	}

	return map[string]interface{}{
		"seconds":      windowSeconds(),
		"requests":     requests,
		"errors5xx":    errors5xx,
		"authFailures": authFails,
		"errorRate":    errorRate,
		"authFailRate": authFailRate,
	}
}

func recordHealthSnapshot() {
	alerts := Alerts()
	snapshot := alerts["snapshot"].(map[string]interface{})
	window := snapshot["window"].(map[string]interface{})

	item := healthHistoryItem{
		TimestampUnix:  time.Now().UTC().Unix(),
		Status:         alerts["level"].(string),
		Scope:          alerts["evaluationScope"].(string),
		RequestsTotal:  snapshot["requestsTotal"].(uint64),
		WindowRequests: window["requests"].(uint64),
		ErrorRate:      snapshot["errorRate"].(float64),
		AuthFailRate:   snapshot["authFailRate"].(float64),
		Reasons:        toStringSlice(alerts["reasons"]),
	}

	historyMu.Lock()
	defer historyMu.Unlock()

	healthHistoryItems = append(healthHistoryItems, item)
	limit := int(getEnvUint64("OPS_HEALTH_HISTORY_LIMIT", 100))
	if limit <= 0 {
		limit = 100
	}
	if len(healthHistoryItems) > limit {
		healthHistoryItems = append([]healthHistoryItem{}, healthHistoryItems[len(healthHistoryItems)-limit:]...)
	}
}

func toStringSlice(input interface{}) []string {
	if input == nil {
		return []string{}
	}
	if direct, ok := input.([]string); ok {
		return direct
	}
	generic, ok := input.([]interface{})
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(generic))
	for _, item := range generic {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
