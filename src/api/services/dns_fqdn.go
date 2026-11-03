package services

import (
	"context"
	"net"
	"strings"
)

// fqdnResolver makes every query absolute (trailing dot). Go's resolver tries
// the search domains of resolv.conf first for relative names; on Cloud Run
// those internal zones answer SERVFAIL for unknown external names, and Go lets
// that temporary error override the real NXDOMAIN. The audit then reported
// "could not be checked" for names that simply do not exist. Rooted names skip
// the search list entirely.
type fqdnResolver struct {
	inner dnsResolver
}

func rooted(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

func (r fqdnResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return r.inner.LookupIPAddr(ctx, rooted(host))
}

func (r fqdnResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	return r.inner.LookupCNAME(ctx, rooted(host))
}

func (r fqdnResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return r.inner.LookupMX(ctx, rooted(name))
}

func (r fqdnResolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	return r.inner.LookupNS(ctx, rooted(name))
}

func (r fqdnResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.inner.LookupTXT(ctx, rooted(name))
}

func (r fqdnResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.inner.LookupHost(ctx, rooted(host))
}
