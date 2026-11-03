package services

import (
	"context"
	"net"
	"testing"
)

// recordingResolver captures the exact names that reach the network layer.
type recordingResolver struct{ names []string }

func (r *recordingResolver) LookupIPAddr(_ context.Context, h string) ([]net.IPAddr, error) {
	r.names = append(r.names, h)
	return nil, nil
}
func (r *recordingResolver) LookupCNAME(_ context.Context, h string) (string, error) {
	r.names = append(r.names, h)
	return "", nil
}
func (r *recordingResolver) LookupMX(_ context.Context, h string) ([]*net.MX, error) {
	r.names = append(r.names, h)
	return nil, nil
}
func (r *recordingResolver) LookupNS(_ context.Context, h string) ([]*net.NS, error) {
	r.names = append(r.names, h)
	return nil, nil
}
func (r *recordingResolver) LookupTXT(_ context.Context, h string) ([]string, error) {
	r.names = append(r.names, h)
	return nil, nil
}
func (r *recordingResolver) LookupHost(_ context.Context, h string) ([]string, error) {
	r.names = append(r.names, h)
	return nil, nil
}

func TestFQDNResolverRootsEveryQueryExactlyOnce(t *testing.T) {
	inner := &recordingResolver{}
	r := fqdnResolver{inner: inner}
	ctx := context.Background()

	_, _ = r.LookupIPAddr(ctx, "example.com")
	_, _ = r.LookupCNAME(ctx, "example.com")
	_, _ = r.LookupMX(ctx, "example.com")
	_, _ = r.LookupNS(ctx, "example.com")
	_, _ = r.LookupTXT(ctx, "_dmarc.example.com")
	_, _ = r.LookupHost(ctx, "1.0.0.127.bl.example.org")
	_, _ = r.LookupTXT(ctx, "already.rooted.example.")

	want := []string{
		"example.com.", "example.com.", "example.com.", "example.com.",
		"_dmarc.example.com.", "1.0.0.127.bl.example.org.", "already.rooted.example.",
	}
	if len(inner.names) != len(want) {
		t.Fatalf("got %v", inner.names)
	}
	for i := range want {
		if inner.names[i] != want[i] {
			t.Errorf("query %d: %q, want %q", i, inner.names[i], want[i])
		}
	}
}

func TestDefaultResolverIsRooted(t *testing.T) {
	if _, ok := newDNSResolver().(fqdnResolver); !ok {
		t.Fatal("the production resolver must root names to avoid search-domain SERVFAIL")
	}
}
