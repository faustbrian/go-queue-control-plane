package history_test

import (
	"errors"
	"testing"
	"time"

	"github.com/faustbrian/go-queue-control-plane/v4/history"
)

// Unkeyed chain integrity is distinct from a protected externally recorded head.
func TestR30AuditPrefixPageAndExternalHeadTrustAreDistinct(t *testing.T) {
	anchor := history.HashBytes([]byte("protected retained predecessor"))
	event := history.Event{Sequence: 8, HashVersion: 2, OccurredAt: time.Unix(1, 0), CommandID: "command", Actor: "original", Action: "drain", Target: "worker", Result: "accepted"}
	original := history.Seal(anchor, event)
	if err := history.VerifyFrom(7, anchor, []history.Entry{original}); err != nil {
		t.Fatal(err)
	}
	changed := original
	changed.Event.Actor = "changed"
	if err := history.VerifyFrom(7, anchor, []history.Entry{changed}); !errors.Is(err, history.ErrAuditTampered) {
		t.Fatalf("unsealed content mutation accepted: %v", err)
	}
	forgedAnchor := history.HashBytes([]byte("replacement predecessor"))
	replacedPage := history.Seal(forgedAnchor, changed.Event)
	if err := history.VerifyFrom(7, replacedPage.PreviousHash, []history.Entry{replacedPage}); err != nil {
		t.Fatalf("relative page integrity changed: %v", err)
	}
	if err := history.VerifyFrom(7, anchor, []history.Entry{replacedPage}); !errors.Is(err, history.ErrAuditTampered) {
		t.Fatalf("protected predecessor mismatch accepted: %v", err)
	}
	// An attacker need not replace a public predecessor hash to reseal a tail.
	// Verification succeeds, but the independently retained expected head differs.
	resealed := history.Seal(anchor, changed.Event)
	if err := history.VerifyFrom(7, anchor, []history.Entry{resealed}); err != nil {
		t.Fatalf("unkeyed resealing characterization changed: %v", err)
	}
	if resealed.Hash == original.Hash {
		t.Fatal("changed tail matched externally protected original head")
	}
}
