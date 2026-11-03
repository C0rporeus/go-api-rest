package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestClassifyDNSError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want dnsStatus
	}{
		{"nil", nil, dnsStatusOK},
		{"not found", &net.DNSError{IsNotFound: true}, dnsStatusNotFound},
		{"timeout", &net.DNSError{IsTimeout: true, IsTemporary: true}, dnsStatusTimeout},
		{"servfail", &net.DNSError{IsTemporary: true}, dnsStatusServFail},
		{"wrapped not found", fmt.Errorf("lookup: %w", &net.DNSError{IsNotFound: true}), dnsStatusNotFound},
		{"context deadline", context.DeadlineExceeded, dnsStatusTimeout},
		{"unknown", errors.New("boom"), dnsStatusError},
		{"dns error without flags", &net.DNSError{}, dnsStatusError},
	}
	for _, tc := range cases {
		if got := classifyDNSError(tc.err); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestLookupStatusDistinguishesEmptyFromFailure(t *testing.T) {
	if got := lookupStatus(nil, 2); got != dnsStatusOK {
		t.Fatalf("records present: %q", got)
	}
	if got := lookupStatus(nil, 0); got != dnsStatusNotFound {
		t.Fatalf("no records: %q", got)
	}
	if got := lookupStatus(&net.DNSError{IsTimeout: true}, 0); got != dnsStatusTimeout {
		t.Fatalf("timeout must not look like 'no records': %q", got)
	}
}
