package handlers

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
    "security-solution/services"
)

// ApproveUnitMember — POST /units/:id/members/:membershipId/approve
func ApproveUnitMember(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    membershipID, err := uuid.Parse(c.Param("membershipId"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid membership id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var m models.UnitMembership
    if err := config.DB.First(&m, "id = ? AND unit_id = ?", membershipID, unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
        return
    }
    if m.Status != "pending" {
        c.JSON(http.StatusConflict, gin.H{"error": "membership is not pending"})
        return
    }

    now := time.Now().UTC()
    m.Status = "active"
    m.AcceptedAt = &now
    if err := config.DB.Save(&m).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve"})
        return
    }

    // Notify the applicant.
    notif := models.Notification{
        UserID:     m.UserID,
        Title:      "Membership approved",
        Message:    "You are now an active member of the unit.",
        Type:       "membership",
        Status:     "unread",
        EntityType: "unit",
        EntityID:   unitID,
        LinkTo:     "/units/" + unitID.String(),
    }
    config.DB.Create(&notif)

    // Audit.
    if userVal, ok := c.Get("user"); ok {
        if admin, ok := userVal.(*models.User); ok && admin != nil {
            _ = services.NewAuditService().LogAction(
                admin.ID, "unit.member.approve", "unit_membership", m.ID.String(),
                map[string]string{"status": "pending"},
                map[string]string{"status": "active", "userId": m.UserID.String()},
                c.ClientIP(), c.Request.UserAgent(),
            )
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "member approved", "status": "active"})
}

// RejectUnitMember — POST /units/:id/members/:membershipId/reject
// Body: { reason?: string }
func RejectUnitMember(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    membershipID, err := uuid.Parse(c.Param("membershipId"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid membership id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var input struct {
        Reason string `json:"reason"`
    }
    _ = c.ShouldBindJSON(&input)
    input.Reason = strings.TrimSpace(input.Reason)
    if len(input.Reason) > 500 {
        input.Reason = input.Reason[:500]
    }

    var m models.UnitMembership
    if err := config.DB.First(&m, "id = ? AND unit_id = ?", membershipID, unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
        return
    }
    if m.Status != "pending" {
        c.JSON(http.StatusConflict, gin.H{"error": "membership is not pending"})
        return
    }

    m.Status = "rejected"
    if input.Reason != "" {
        m.RevokedReason = &input.Reason
    }
    if err := config.DB.Save(&m).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject"})
        return
    }

    notif := models.Notification{
        UserID:     m.UserID,
        Title:      "Membership application rejected",
        Message:    "Your application to join the unit was not accepted.",
        Type:       "membership",
        Status:     "unread",
        EntityType: "unit",
        EntityID:   unitID,
    }
    config.DB.Create(&notif)

    c.JSON(http.StatusOK, gin.H{"message": "member rejected", "status": "rejected"})
}

// RevokeUnitMember — POST /units/:id/members/:membershipId/revoke
// Body: { reason: string } (required)
func RevokeUnitMember(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    membershipID, err := uuid.Parse(c.Param("membershipId"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid membership id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var input struct {
        Reason string `json:"reason" binding:"required"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "reason is required"})
        return
    }
    input.Reason = strings.TrimSpace(input.Reason)
    if input.Reason == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "reason cannot be empty"})
        return
    }
    if len(input.Reason) > 500 {
        input.Reason = input.Reason[:500]
    }

    var m models.UnitMembership
    if err := config.DB.First(&m, "id = ? AND unit_id = ?", membershipID, unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
        return
    }
    if m.Status != "active" {
        c.JSON(http.StatusConflict, gin.H{"error": "membership is not active"})
        return
    }
    if m.IsHeadAdmin {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "head admin can only be removed through a revocation cycle",
        })
        return
    }

    now := time.Now().UTC()
    m.Status = "revoked"
    m.RevokedAt = &now
    m.RevokedReason = &input.Reason
    if err := config.DB.Save(&m).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
        return
    }

    notif := models.Notification{
        UserID:     m.UserID,
        Title:      "Membership revoked",
        Message:    "Your membership in the unit was revoked: " + input.Reason,
        Type:       "membership",
        Status:     "unread",
        EntityType: "unit",
        EntityID:   unitID,
    }
    config.DB.Create(&notif)

    if userVal, ok := c.Get("user"); ok {
        if admin, ok := userVal.(*models.User); ok && admin != nil {
            _ = services.NewAuditService().LogAction(
                admin.ID, "unit.member.revoke", "unit_membership", m.ID.String(),
                map[string]string{"status": "active", "role": m.Role},
                map[string]string{"status": "revoked", "reason": input.Reason, "userId": m.UserID.String()},
                c.ClientIP(), c.Request.UserAgent(),
            )
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "member revoked", "status": "revoked"})
}

// PromoteUnitMember — POST /units/:id/members/:membershipId/promote
// Promotes an active officer to unit_admin. Head-admin designation is
// election-only and cannot be set here.
func PromoteUnitMember(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    membershipID, err := uuid.Parse(c.Param("membershipId"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid membership id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var m models.UnitMembership
    if err := config.DB.First(&m, "id = ? AND unit_id = ?", membershipID, unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
        return
    }
    if m.Status != "active" {
        c.JSON(http.StatusConflict, gin.H{"error": "member must be active to be promoted"})
        return
    }
    if m.Role == models.UnitRoleAdmin {
        c.JSON(http.StatusConflict, gin.H{"error": "member is already an admin"})
        return
    }

    oldRole := m.Role
    m.Role = models.UnitRoleAdmin
    if err := config.DB.Save(&m).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to promote"})
        return
    }

    notif := models.Notification{
        UserID:     m.UserID,
        Title:      "You are now a unit admin",
        Message:    "A unit admin promoted you to admin.",
        Type:       "membership",
        Status:     "unread",
        EntityType: "unit",
        EntityID:   unitID,
        LinkTo:     "/units/" + unitID.String(),
    }
    config.DB.Create(&notif)

    if userVal, ok := c.Get("user"); ok {
        if admin, ok := userVal.(*models.User); ok && admin != nil {
            _ = services.NewAuditService().LogAction(
                admin.ID, "unit.member.promote", "unit_membership", m.ID.String(),
                map[string]string{"role": oldRole},
                map[string]string{"role": "unit_admin", "userId": m.UserID.String()},
                c.ClientIP(), c.Request.UserAgent(),
            )
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "member promoted", "role": "unit_admin"})
}

// ListUnitInvites — GET /units/:id/invites
// Returns the metadata of every invite for the unit (never the plaintext
// code — only the hash is stored, so the code is unrecoverable by design).
func ListUnitInvites(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var invites []models.UnitInvite
    if err := config.DB.
        Where("unit_id = ?", unitID).
        Order("created_at desc").
        Limit(100).
        Find(&invites).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load invites"})
        return
    }

    out := make([]gin.H, 0, len(invites))
    now := time.Now().UTC()
    for _, inv := range invites {
        expired := inv.ExpiresAt != nil && inv.ExpiresAt.Before(now)
        exhausted := inv.UseCount >= inv.MaxUses
        effectiveStatus := inv.Status
        if inv.Status == "pending" && expired {
            effectiveStatus = "expired"
        } else if inv.Status == "pending" && exhausted {
            effectiveStatus = "exhausted"
        }
        out = append(out, gin.H{
            "id":            inv.ID,
            "scope":         inv.Scope,
            "status":        effectiveStatus,
            "maxUses":       inv.MaxUses,
            "useCount":      inv.UseCount,
            "expiresAt":     inv.ExpiresAt,
            "revokedAt":     inv.RevokedAt,
            "revokedReason": inv.RevokedReason,
            "createdBy":     inv.CreatedBy,
            "createdAt":     inv.CreatedAt,
        })
    }

    c.JSON(http.StatusOK, gin.H{"invites": out})
}

// CreateUnitInvite — POST /units/:id/invites
// Body: { expiresInHours?: number, maxUses?: number }
// Returns the plaintext code ONCE. It cannot be retrieved again.
func CreateUnitInvite(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
        return
    }
    if !unit.IsVerified {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit must be verified before inviting members"})
        return
    }

    var input struct {
        ExpiresInHours int `json:"expiresInHours"`
        MaxUses        int `json:"maxUses"`
    }
    _ = c.ShouldBindJSON(&input)

    maxUses := input.MaxUses
    if maxUses < 1 {
        maxUses = 1
    }
    if maxUses > 500 {
        maxUses = 500
    }

    var expiresAt *time.Time
    if input.ExpiresInHours > 0 {
        t := time.Now().UTC().Add(time.Duration(input.ExpiresInHours) * time.Hour)
        expiresAt = &t
    }

    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate code"})
        return
    }
    code := hex.EncodeToString(raw)
    hash := sha256.Sum256([]byte(code))
    codeHash := hex.EncodeToString(hash[:])

    userVal, _ := c.Get("user")
    admin := userVal.(*models.User)

    uid := unitID
    invite := models.UnitInvite{
        UnitID:    &uid,
        Scope:     "unit",
        CodeHash:  codeHash,
        Status:    "pending",
        ExpiresAt: expiresAt,
        MaxUses:   maxUses,
        UseCount:  0,
        CreatedBy: admin.ID,
    }
    if err := config.DB.Create(&invite).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
        return
    }

    _ = services.NewAuditService().LogAction(
        admin.ID, "unit.invite.create", "unit_invite", invite.ID.String(),
        nil,
        map[string]interface{}{"unitId": unitID.String(), "maxUses": maxUses, "expiresAt": expiresAt},
        c.ClientIP(), c.Request.UserAgent(),
    )

    c.JSON(http.StatusCreated, gin.H{
        "invite": gin.H{
            "id":        invite.ID,
            "scope":     invite.Scope,
            "unitId":    unitID,
            "code":      code,
            "expiresAt": invite.ExpiresAt,
            "maxUses":   invite.MaxUses,
        },
        "warning": "Save this code now. It cannot be retrieved again.",
    })
}

// RevokeUnitInvite — POST /units/:id/invites/:inviteId/revoke
func RevokeUnitInvite(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit id"})
        return
    }
    inviteID, err := uuid.Parse(c.Param("inviteId"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
        return
    }
    if !callerIsUnitAdminOf(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "unit admin access required"})
        return
    }

    var inv models.UnitInvite
    if err := config.DB.First(&inv, "id = ? AND unit_id = ?", inviteID, unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "invite not found"})
        return
    }
    if inv.Status == "revoked" {
        c.JSON(http.StatusOK, gin.H{"message": "already revoked"})
        return
    }

    now := time.Now().UTC()
    inv.Status = "revoked"
    inv.RevokedAt = &now
    if err := config.DB.Save(&inv).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "invite revoked"})
}
