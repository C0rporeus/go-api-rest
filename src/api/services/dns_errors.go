package services

import (
	"context"
	"errors"
	"net"
)

// dnsStatus describes the outcome of a DNS lookup. A lookup that could not be
// completed (timeout, SERVFAIL...) must never be reported as "no records":
// the caller needs to tell "does not exist" apart from "could not be checked".
type dnsStatus string

const (
	dnsStatusOK       dnsStatus = "ok"
	dnsStatusNotFound dnsStatus = "not_found" // NXDOMAIN or name without data of that type
	dnsStatusTimeout  dnsStatus = "timeout"
	dnsStatusServFail dnsStatus = "servfail"
	dnsStatusError    dnsStatus = "error"
)

// classifyDNSError maps a resolver error to a dnsStatus (nil -> ok).
func classifyDNSError(err error) dnsStatus {
	if err == nil {
		return dnsStatusOK
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return dnsStatusTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		switch {
		case dnsErr.IsNotFound:
			return dnsStatusNotFound
		case dnsErr.IsTimeout:
			return dnsStatusTimeout
		case dnsErr.IsTemporary:
			return dnsStatusServFail
		}
	}
	return dnsStatusError
}

// lookupStatus is the status for a lookup that returned err and n records.
// A successful lookup with no records is "not_found" (name exists, no data).
func lookupStatus(err error, n int) dnsStatus {
	if err != nil {
		return classifyDNSError(err)
	}
	if n == 0 {
		return dnsStatusNotFound
	}
	return dnsStatusOK
}
