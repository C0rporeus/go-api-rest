package mailaudit

import (
	"errors"
	"strings"
	"testing"
)

// zone is an in-memory DNS: name -> TXT records. Unknown names are ErrNotFound.
type zone map[string][]string

func (z zone) lookup(name string) ([]string, error) {
	if txts, ok := z[strings.ToLower(name)]; ok {
		return txts, nil
	}
	return nil, ErrNotFound
}

var errTimeout = errors.New("lookup timeout")

func ids(findings []Finding) map[string]Severity {
	out := map[string]Severity{}
	for _, f := range findings {
		out[f.ID] = f.Severity
	}
	return out
}

func requireFinding(t *testing.T, findings []Finding, id string, severity Severity) {
	t.Helper()
	got, ok := ids(findings)[id]
	if !ok {
		t.Fatalf("expected finding %q, got %v", id, ids(findings))
	}
	if got != severity {
		t.Fatalf("finding %q: severity %q, want %q", id, got, severity)
	}
}

func rejectFinding(t *testing.T, findings []Finding, id string) {
	t.Helper()
	if _, ok := ids(findings)[id]; ok {
		t.Fatalf("unexpected finding %q in %v", id, ids(findings))
	}
}
