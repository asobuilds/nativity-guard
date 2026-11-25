package handlers

import (
	"strings"
	"testing"
)

func TestSanitizeObserverDescription(t *testing.T) {
	t.Run("redacts emails phones and coordinates", func(t *testing.T) {
		raw := "phone 08031234567 email ada@example.test at 9.0765, 7.3986"
		got := sanitizeObserverDescription(raw)

		for _, secret := range []string{"08031234567", "ada@example.test", "9.0765", "7.3986"} {
			if strings.Contains(got, secret) {
				t.Errorf("sanitized description leaked %q: %s", secret, got)
			}
		}
		if !strings.Contains(got, "[redacted]") {
			t.Errorf("sanitized description did not mark redactions: %s", got)
		}
		if got == "Incident details withheld" {
			t.Errorf("direct identifier redaction should preserve non-identifying text: %s", got)
		}
	})

	t.Run("withholds suspected personal names", func(t *testing.T) {
		got := sanitizeObserverDescription("Call Adaeze Obi at the market")
		if got != "Incident details withheld" {
			t.Fatalf("description = %q, want Incident details withheld", got)
		}
	})

	t.Run("keeps ordinary lower-case descriptions", func(t *testing.T) {
		raw := "fire reported near the market"
		if got := sanitizeObserverDescription(raw); got != raw {
			t.Fatalf("description = %q, want %q", got, raw)
		}
	})
}
