package config

import (
	"net"
	"net/url"
	"strings"
)

// DevCORSOrigin is the only origin allowed when CORS_ALLOWED_ORIGINS is unset
// outside production (the Next.js dev server).
const DevCORSOrigin = "http://localhost:3000"

// IsProduction reports whether the API runs in production. Cloud Run always sets
// K_SERVICE; APP_ENV=production forces it elsewhere.
func IsProduction(getenv func(string) string) bool {
	return getenv("K_SERVICE") != "" || strings.EqualFold(getenv("APP_ENV"), "production")
}

// AllowedCORSOrigins parses the comma-separated CORS_ALLOWED_ORIGINS value.
//
// In production it keeps only https origins whose host is not loopback or
// localhost, so a development origin left in the deployment configuration can
// never be trusted with credentials (AllowCredentials is on). It returns the
// dropped origins so the caller can log them. When nothing valid remains in
// production the result is empty: the browser then blocks every cross-origin
// read (fail closed). Outside production an empty value falls back to
// DevCORSOrigin.
func AllowedCORSOrigins(raw string, production bool) (allowed, dropped []string) {
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}
		if production && !isPublicHTTPSOrigin(origin) {
			dropped = append(dropped, origin)
			continue
		}
		allowed = append(allowed, origin)
	}
	if len(allowed) == 0 && !production {
		allowed = []string{DevCORSOrigin}
	}
	return allowed, dropped
}

func isPublicHTTPSOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified()) {
		return false
	}
	return true
}
