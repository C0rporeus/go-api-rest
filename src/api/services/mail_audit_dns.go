package services

import (
	"context"
	"errors"
	"net"
	"sync/atomic"

	"backend-yonathan/src/pkg/mailaudit"
)

// errQueryBudget is returned once the audit has used its DNS query allowance.
var errQueryBudget = &dnsLookupError{status: dnsStatusError, reason: "presupuesto de consultas agotado"}

// dnsLookupError carries the dnsStatus of a failed lookup through the pure
// mailaudit package, which only distinguishes "not found" from "unavailable".
type dnsLookupError struct {
	status dnsStatus
	reason string
}

func (e *dnsLookupError) Error() string {
	if e.reason != "" {
		return e.reason
	}
	return string(e.status)
}

// Is lets errors.Is(err, mailaudit.ErrNotFound) recognise a definitive "not found".
func (e *dnsLookupError) Is(target error) bool {
	return target == mailaudit.ErrNotFound && e.status == dnsStatusNotFound
}

// statusOfLookup extracts the dnsStatus of an error produced by auditDNS.
func statusOfLookup(err error, records int) dnsStatus {
	if err == nil {
		return lookupStatus(nil, records)
	}
	var lookupErr *dnsLookupError
	if errors.As(err, &lookupErr) {
		return lookupErr.status
	}
	return dnsStatusError
}

// auditDNS runs the audit's DNS queries under a shared context and budget.
type auditDNS struct {
	ctx      context.Context
	resolver dnsResolver
	used     atomic.Int32
	max      int32
}

func newAuditDNS(ctx context.Context, resolver dnsResolver, maxQueries int) *auditDNS {
	return &auditDNS{ctx: ctx, resolver: resolver, max: int32(maxQueries)}
}

func (d *auditDNS) take() error {
	if d.used.Add(1) > d.max {
		return errQueryBudget
	}
	return nil
}

func wrapLookupError(err error) error {
	if err == nil {
		return nil
	}
	lookupErr := &dnsLookupError{status: classifyDNSError(err)}
	// For unclassified failures keep Go's own message (e.g. "cannot unmarshal DNS
	// message") so the finding says what happened. It never includes the resolver
	// address: only DNSError.Err is used.
	var dnsErr *net.DNSError
	if lookupErr.status == dnsStatusError && errors.As(err, &dnsErr) && dnsErr.Err != "" {
		lookupErr.reason = "error: " + dnsErr.Err
	}
	return lookupErr
}

func (d *auditDNS) txt(name string) ([]string, error) {
	if err := d.take(); err != nil {
		return nil, err
	}
	txts, err := d.resolver.LookupTXT(d.ctx, name)
	return txts, wrapLookupError(err)
}

func (d *auditDNS) mx(name string) ([]mailaudit.MXRecord, error) {
	if err := d.take(); err != nil {
		return nil, err
	}
	records, err := d.resolver.LookupMX(d.ctx, name)
	if err != nil {
		return nil, wrapLookupError(err)
	}
	out := make([]mailaudit.MXRecord, 0, len(records))
	for _, record := range records {
		out = append(out, mailaudit.MXRecord{Host: record.Host, Pref: record.Pref})
	}
	return out, nil
}
