package handlers

import (
	"fmt"
	"regexp"
	"strings"
)

const maxUserTextLength = 2000

// sqlInjectionPatterns are conservative signatures for obvious SQL injection
// attempts. Queries are already parameterized through GORM, so this is
// defence-in-depth and abuse control rather than the primary protection.
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(or|and)\b\s+['"]?\d+['"]?\s*=\s*['"]?\d+`),
	regexp.MustCompile(`(?i)\bunion\s+(all\s+)?select\b`),
	regexp.MustCompile(`(?i)\b(drop|truncate)\s+(table|database)\b`),
	regexp.MustCompile(`(?i)\binsert\s+into\b`),
	regexp.MustCompile(`(?i)\bdelete\s+from\b`),
	regexp.MustCompile(`(?i)\bupdate\s+\w+\s+set\b`),
	regexp.MustCompile(`(?i)\bexec(ute)?\s*\(`),
	regexp.MustCompile(`(?i)\bxp_\w+`),
	regexp.MustCompile(`--`),
	regexp.MustCompile(`/\*`),
	regexp.MustCompile(`\*/`),
}

// validateText rejects over-long input and obvious SQL injection signatures.
// The field name is echoed back so clients know which value to fix.
func validateText(field, value string, max int) error {
	if len([]rune(value)) > max {
		return fmt.Errorf("%s must be %d characters or fewer", field, max)
	}
	for _, re := range sqlInjectionPatterns {
		if re.MatchString(strings.TrimSpace(value)) {
			return fmt.Errorf("%s contains disallowed content", field)
		}
	}
	return nil
}
