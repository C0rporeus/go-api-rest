//go:build live

package services

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"

	"backend-yonathan/src/pkg/constants"
)

// Smoke test against real DNS (not part of CI):
//
//	MAIL_AUDIT_DOMAINS=gmail.com,github.com go test -tags live -run TestMailAuditLive -v ./src/api/services
func TestMailAuditLive(t *testing.T) {
	domains := strings.Split(os.Getenv("MAIL_AUDIT_DOMAINS"), ",")
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), constants.MailAuditTimeout)
		result := runMailAudit(ctx, fqdnResolver{inner: &net.Resolver{PreferGo: true}}, domain, "")
		cancel()
		t.Logf("== %s  summary=%v", domain, result.Summary)
		for section, data := range result.Sections {
			t.Logf("   [%s] status=%s records=%d lookups=%v", section, data.Status, len(data.Records), data.Lookups)
		}
		for _, f := range result.Findings {
			t.Logf("   %-8s %-26s %s", f.Severity, f.ID, f.Title)
		}
	}
}
