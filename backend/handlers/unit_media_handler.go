package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "security-solution/config"
    "security-solution/models"
    "security-solution/services"
)

// canManageUnit returns true when the caller is a super_admin or a
// unit_admin whose UnitID matches the target unit.
func canManageUnit(c *gin.Context, unitID uuid.UUID) bool {
    v, ok := c.Get("user")
    if !ok {
        return false
    }
    u, ok := v.(*models.User)
    if !ok {
        return false
    }
    if u.Role == "super_admin" {
        return true
    }
    return u.Role == "unit_admin" && u.UnitID != nil && *u.UnitID == unitID
}

// UploadUnitLogo — POST /units/:id/logo
func UploadUnitLogo(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
        return
    }
    if !canManageUnit(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to manage this unit"})
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
        return
    }

    storage := services.NewFileStorageService()
    stored, err := storage.Save(file, "unit_logo")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        _ = storage.Delete(stored.RelativePath)
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }

    old := unit.BrandLogoURL
    if err := config.DB.Model(&unit).Update("brand_logo_url", stored.RelativePath).Error; err != nil {
        _ = storage.Delete(stored.RelativePath)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save logo"})
        return
    }
    if old != "" && old != stored.RelativePath {
        _ = storage.Delete(old)
    }

    c.JSON(http.StatusOK, gin.H{
        "message":      "Logo uploaded",
        "brandLogoUrl": stored.RelativePath,
        "hash":         stored.Hash,
        "size":         stored.Size,
    })
}

// UploadUnitCover — POST /units/:id/cover
func UploadUnitCover(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
        return
    }
    if !canManageUnit(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to manage this unit"})
        return
    }

    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
        return
    }

    storage := services.NewFileStorageService()
    stored, err := storage.Save(file, "unit_cover")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        _ = storage.Delete(stored.RelativePath)
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }

    old := unit.BrandCoverURL
    if err := config.DB.Model(&unit).Update("brand_cover_url", stored.RelativePath).Error; err != nil {
        _ = storage.Delete(stored.RelativePath)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cover"})
        return
    }
    if old != "" && old != stored.RelativePath {
        _ = storage.Delete(old)
    }

    c.JSON(http.StatusOK, gin.H{
        "message":       "Cover uploaded",
        "brandCoverUrl": stored.RelativePath,
        "hash":          stored.Hash,
        "size":          stored.Size,
    })
}

// DeleteUnitLogo — DELETE /units/:id/logo
func DeleteUnitLogo(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
        return
    }
    if !canManageUnit(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to manage this unit"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }
    if unit.BrandLogoURL != "" {
        _ = services.NewFileStorageService().Delete(unit.BrandLogoURL)
    }
    config.DB.Model(&unit).Update("brand_logo_url", "")
    c.JSON(http.StatusOK, gin.H{"message": "Logo removed"})
}

// DeleteUnitCover — DELETE /units/:id/cover
func DeleteUnitCover(c *gin.Context) {
    unitID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
        return
    }
    if !canManageUnit(c, unitID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed to manage this unit"})
        return
    }

    var unit models.SecurityUnit
    if err := config.DB.First(&unit, "id = ?", unitID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Unit not found"})
        return
    }
    if unit.BrandCoverURL != "" {
        _ = services.NewFileStorageService().Delete(unit.BrandCoverURL)
    }
    config.DB.Model(&unit).Update("brand_cover_url", "")
    c.JSON(http.StatusOK, gin.H{"message": "Cover removed"})
}