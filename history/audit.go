// Package history seals and verifies value-based administrative audit hash
// chains. Seal creates an unkeyed SHA-256 link; VerifyFrom additionally checks
// contiguous sequences. Trusted anchors and tenant separation are external
// responsibilities: digests are not authentication or confidentiality, and a
// party able to replace the complete chain/root can recompute them. Inputs are
// borrowed synchronously without size validation; no persistence is owned.
package history

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrAuditTampered indicates that an entry's content or chain position does
// not match its stored digest.
var ErrAuditTampered = errors.New("history: audit chain verification failed")

// Hash is an audit-chain SHA-256 digest.
type Hash [sha256.Size]byte

// Event is the canonical content of one administrative audit record.
type Event struct {
	// Sequence is the chain position; VerifyFrom checks contiguous values,
	// while Seal does not validate it.
	Sequence uint64
	// HashVersion selects the package encoding: 2 includes CommandID, while 0/1
	// retain v1 compatibility; Verify rejects values above 2.
	HashVersion uint16
	// OccurredAt is hashed as Unix nanoseconds; persistence normalizes
	// timestamps before sealing.
	OccurredAt time.Time
	// CommandID is hashed only in version 2; it is not independently
	// authenticated.
	CommandID string
	// IdempotencyKey links command audit events; privileged-access events may
	// leave it empty.
	IdempotencyKey string
	// Actor is attributed actor text; callers must bind identity and exclude
	// secrets.
	Actor string
	// Action is the attributed operation or sensitive-access permission text.
	Action string
	// Target is attributed resource text; tenant scope is external to this
	// event digest.
	Target string
	// Result is the attributed public operation outcome.
	Result string
}

// Entry links an event to the preceding retained or anchored audit digest.
type Entry struct {
	// PreviousHash links to the preceding entry or externally trusted anchor.
	PreviousHash Hash
	// Hash is an unkeyed SHA-256 content/link digest, not a signature or MAC.
	Hash Hash
	// Event is copied value content; no size or identity validation occurs in
	// Seal.
	Event Event
}

// HashBytes creates an anchor hash for a retained prefix or external root.
func HashBytes(value []byte) Hash {
	return sha256.Sum256(value)
}

// Seal returns an immutable-value audit entry linked to previous.
func Seal(previous Hash, event Event) Entry {
	return Entry{
		PreviousHash: previous,
		Hash:         digest(previous, event),
		Event:        event,
	}
}

// Verify checks content and chain-link order after anchor, not contiguous event
// sequences; use VerifyFrom for that additional check. An empty page is valid
// but establishes no authenticity of its supplied anchor.
func Verify(anchor Hash, entries []Entry) error {
	return verifyHashes(anchor, entries)
}

// VerifyFrom checks hash and contiguous sequence integrity after an anchor.
func VerifyFrom(anchorSequence uint64, anchor Hash, entries []Entry) error {
	expected := anchorSequence
	for _, entry := range entries {
		if expected == math.MaxUint64 {
			return fmt.Errorf("%w after sequence %d", ErrAuditTampered, expected)
		}
		expected++
		if entry.Event.Sequence != expected {
			return fmt.Errorf("%w at sequence %d", ErrAuditTampered, entry.Event.Sequence)
		}
	}

	return verifyHashes(anchor, entries)
}

func verifyHashes(anchor Hash, entries []Entry) error {
	previous := anchor
	for _, entry := range entries {
		if entry.Event.HashVersion > 2 {
			return fmt.Errorf("%w at sequence %d", ErrAuditTampered, entry.Event.Sequence)
		}
		if entry.PreviousHash != previous || entry.Hash != digest(previous, entry.Event) {
			return fmt.Errorf("%w at sequence %d", ErrAuditTampered, entry.Event.Sequence)
		}

		previous = entry.Hash
	}

	return nil
}

func digest(previous Hash, event Event) Hash {
	// Domain tags preserve v1 compatibility (versions 0/1) and v2 CommandID
	// binding. Length prefixes avoid concatenation ambiguity, not oversized
	// input: append can grow beyond this scratch buffer for caller strings.
	var storage [2_048]byte
	encoded := storage[:0]
	if event.HashVersion == 2 {
		encoded = append(encoded, "go-queue-control-plane/audit/v2"...)
	} else {
		encoded = append(encoded, "go-queue-control-plane/audit/v1"...)
	}
	encoded = append(encoded, previous[:]...)
	encoded = binary.BigEndian.AppendUint64(encoded, event.Sequence)
	encoded = binary.BigEndian.AppendUint64(encoded, uint64(event.OccurredAt.UnixNano()))
	if event.HashVersion == 2 {
		encoded = appendString(encoded, event.CommandID)
	}
	encoded = appendString(encoded, event.IdempotencyKey)
	encoded = appendString(encoded, event.Actor)
	encoded = appendString(encoded, event.Action)
	encoded = appendString(encoded, event.Target)
	encoded = appendString(encoded, event.Result)

	return sha256.Sum256(encoded)
}

func appendString(target []byte, value string) []byte {
	target = binary.BigEndian.AppendUint64(target, uint64(len(value)))

	return append(target, value...)
}
