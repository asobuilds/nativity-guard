package services

import (
    "log"
    "strings"
    "time"

    "security-solution/config"
    "security-solution/content"
    "security-solution/models"
)

// EnsureTermsSeeded inserts one TermsDocument per (kind, role) if missing.
// Called at boot. Idempotent — safe to run on every startup.
func EnsureTermsSeeded() {
    roles := []string{"citizen", "officer", "unit_admin", "super_admin"}

    for _, role := range roles {
        termsBody := content.UniversalTerms
        if extra, ok := content.RoleTerms[role]; ok {
            termsBody = termsBody + "\n" + extra
        }
        ensureDoc("terms", role, "Nativity Guard Terms — "+humanRole(role), termsBody)
    }
    ensureDoc("privacy", "all", "Nativity Guard Privacy Notice", content.PrivacyContent)
}

func humanRole(role string) string {
    switch role {
    case "citizen":
        return "Citizen"
    case "officer":
        return "Officer"
    case "unit_admin":
        return "Unit Administrator"
    case "super_admin":
        return "Platform Administrator"
    }
    return role
}

func ensureDoc(kind, role, title, body string) {
    var existing models.TermsDocument
    err := config.DB.
        Where("kind = ? AND role = ? AND version = ?", kind, role, content.TermsVersion).
        First(&existing).Error
    if err == nil {
        return
    }
    doc := models.TermsDocument{
        Kind:        kind,
        Role:        role,
        Version:     content.TermsVersion,
        Title:       title,
        Content:     body,
        Summary:     strings.TrimSpace(firstLines(body, 3)),
        EffectiveAt: time.Now().UTC(),
        IsActive:    true,
    }
    if err := config.DB.Create(&doc).Error; err != nil {
        log.Printf("terms seed failed for %s/%s: %v", kind, role, err)
        return
    }
    log.Printf("terms: seeded %s for role %s @ %s", kind, role, content.TermsVersion)
}

func firstLines(s string, n int) string {
    lines := strings.SplitN(s, "\n", n+1)
    out := []string{}
    for _, l := range lines {
        l = strings.TrimSpace(l)
        if l == "" || strings.HasPrefix(l, "#") {
            continue
        }
        out = append(out, l)
        if len(out) >= n {
            break
        }
    }
    return strings.Join(out, " ")
}