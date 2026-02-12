package telemetry

import (
	"os"
	"testing"
)

func TestTrackRequestAndSnapshot(t *testing.T) {
	initial := Snapshot()
	initialRequests := initial["requestsTotal"].(uint64)

	TrackRequest(200, false)
	TrackRequest(500, false)
	TrackRequest(401, true)

	next := Snapshot()
	if next["requestsTotal"].(uint64) < initialRequests+3 {
		t.Fatalf("expected requests to increase by at least 3")
	}
	if next["errors5xx"].(uint64) < 1 {
		t.Fatalf("expected at least one 5xx")
	}
	if next["authFailures"].(uint64) < 1 {
		t.Fatalf("expected at least one auth failure")
	}
}

func TestHealthAlertsSummaryAndHistory(t *testing.T) {
	alerts := Alerts()
	if alerts["level"] == "" {
		t.Fatalf("expected alert level")
	}

	health := Health()
	if health["status"] == "" {
		t.Fatalf("expected health status")
	}

	history := History()
	if history["count"] == nil {
		t.Fatalf("expected history count")
	}

	summary := Summary()
	if summary["status"] == "" {
		t.Fatalf("expected summary status")
	}
}

func TestEnvParsingFallbacks(t *testing.T) {
	t.Setenv("OPS_WARN_5XX_RATE", "0.25")
	if value := getEnvFloat64("OPS_WARN_5XX_RATE", 0.05); value != 0.25 {
		t.Fatalf("expected float env value 0.25 got %f", value)
	}

	t.Setenv("OPS_WARN_5XX_RATE", "invalid")
	if value := getEnvFloat64("OPS_WARN_5XX_RATE", 0.05); value != 0.05 {
		t.Fatalf("expected float fallback 0.05 got %f", value)
	}

	t.Setenv("OPS_ALERT_MIN_REQUESTS", "30")
	if value := getEnvUint64("OPS_ALERT_MIN_REQUESTS", 20); value != 30 {
		t.Fatalf("expected uint env value 30 got %d", value)
	}

	t.Setenv("OPS_ALERT_MIN_REQUESTS", "invalid")
	if value := getEnvUint64("OPS_ALERT_MIN_REQUESTS", 20); value != 20 {
		t.Fatalf("expected uint fallback 20 got %d", value)
	}

	t.Setenv("OPS_WINDOW_SECONDS", "120")
	if value := windowSeconds(); value != 120 {
		t.Fatalf("expected window seconds 120 got %d", value)
	}

	t.Setenv("OPS_WINDOW_SECONDS", "-1")
	if value := windowSeconds(); value != 300 {
		t.Fatalf("expected window fallback 300 got %d", value)
	}
}

func TestToStringSliceVariants(t *testing.T) {
	if value := toStringSlice(nil); len(value) != 0 {
		t.Fatalf("expected empty slice for nil input")
	}
	if value := toStringSlice([]string{"a", "b"}); len(value) != 2 {
		t.Fatalf("expected 2 values for []string input")
	}
	if value := toStringSlice([]interface{}{"x", 1, "y"}); len(value) != 2 {
		t.Fatalf("expected filtered []interface{} values")
	}
}

func TestAlertsLevelsAndScopes(t *testing.T) {
	t.Setenv("OPS_ALERT_MIN_REQUESTS", "3")
	t.Setenv("OPS_WARN_5XX_RATE", "0.20")
	t.Setenv("OPS_CRITICAL_5XX_RATE", "0.50")
	t.Setenv("OPS_WARN_AUTH_FAIL_RATE", "0.20")
	t.Setenv("OPS_CRITICAL_AUTH_FAIL_RATE", "0.50")
	t.Setenv("OPS_HEALTH_HISTORY_LIMIT", "5")

	// Isolate by forcing a fresh process state assumptions through spikes.
	for i := 0; i < 3; i++ {
		TrackRequest(200, false)
	}
	alerts := Alerts()
	if alerts["level"] == "" {
		t.Fatalf("expected level on alerts")
	}

	// Push into warning through auth failures.
	TrackRequest(401, true)
	TrackRequest(401, true)
	TrackRequest(200, false)

	alertsWarn := Alerts()
	levelWarn, _ := alertsWarn["level"].(string)
	if levelWarn != "warn" && levelWarn != "critical" {
		t.Fatalf("expected warn or critical level, got %s", levelWarn)
	}

	// Force critical with 5xx spike.
	TrackRequest(500, false)
	TrackRequest(500, false)
	TrackRequest(500, false)

	alertsCritical := Alerts()
	levelCritical, _ := alertsCritical["level"].(string)
	if levelCritical != "critical" && levelCritical != "warn" {
		t.Fatalf("expected warn or critical level, got %s", levelCritical)
	}

	// exercise history cap and summary branches
	_ = Summary()
	_ = History()

	// restore env touched directly by os package if any future test depends on unset state
	_ = os.Unsetenv("OPS_ALERT_MIN_REQUESTS")
	_ = os.Unsetenv("OPS_WARN_5XX_RATE")
	_ = os.Unsetenv("OPS_CRITICAL_5XX_RATE")
	_ = os.Unsetenv("OPS_WARN_AUTH_FAIL_RATE")
	_ = os.Unsetenv("OPS_CRITICAL_AUTH_FAIL_RATE")
	_ = os.Unsetenv("OPS_HEALTH_HISTORY_LIMIT")
}
