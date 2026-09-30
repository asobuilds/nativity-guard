package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "security-solution/config"
    "security-solution/content"
    "security-solution/models"
)

// GetTermsForRole — GET /terms/:role
// Public. Returns the currently-active document for the given role and kind.
// Query: ?kind=terms|privacy (default: terms)
func GetTermsForRole(c *gin.Context) {
    role := c.Param("role")
    kind := c.DefaultQuery("kind", "terms")

    if kind != "terms" && kind != "privacy" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be terms or privacy"})
        return
    }
    if kind == "privacy" {
        role = "all"
    }

    var doc models.TermsDocument
    err := config.DB.
        Where("kind = ? AND role = ? AND is_active = ?", kind, role, true).
        Order("effective_at desc").
        First(&doc).Error
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "No active document for that role"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "document": doc,
        "version":  content.TermsVersion,
    })
}

// GetMyAcceptances — GET /terms/my-acceptance
// Authenticated. Returns what this user has accepted (latest per kind).
func GetMyAcceptances(c *gin.Context) {
    userVal, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    userObj := userVal.(*models.User)

    var acceptances []models.TermsAcceptance
    if err := config.DB.
        Where("user_id = ?", userObj.ID).
        Order("accepted_at desc").
        Find(&acceptances).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load acceptances"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "acceptances": acceptances,
        "currentVersion": content.TermsVersion,
    })
}