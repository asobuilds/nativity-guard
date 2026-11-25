package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidCitizenChatScope(t *testing.T) {
	for _, scope := range []string{"locality", "LGA", " State "} {
		if !validCitizenChatScope(scope) { t.Errorf("expected %q to be a valid chat scope", scope) }
	}
	for _, scope := range []string{"", "country", "world", "national", "ward"} {
		if validCitizenChatScope(scope) { t.Errorf("expected %q to be rejected as a chat scope", scope) }
	}
}

func TestCitizenChatCursorRoundTrip(t *testing.T) {
	id := uuid.New()
	at := time.Date(2026, 11, 13, 15, 30, 0, 123456789, time.UTC)
	cursor := encodeCitizenChatCursor(at, id)
	gotAt, gotID, err := decodeCitizenChatCursor(cursor)
	if err != nil { t.Fatalf("decode cursor: %v", err) }
	if !gotAt.Equal(at) || gotID != id { t.Fatalf("round trip mismatch: got %s %s", gotAt, gotID) }
}

func TestCitizenChatCursorRejectsMalformedValues(t *testing.T) {
	for _, raw := range []string{"!", "", "bm90LXJlYWw", "MjAyNi0xMS0xM1QxNTowMDowMFp8bm90LXV1aWQ"} {
		if _, _, err := decodeCitizenChatCursor(raw); err == nil { t.Errorf("expected malformed cursor %q to fail", raw) }
	}
}
