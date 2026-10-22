package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// getAuthUser pulls the authenticated user out of the gin context. Returns
// nil when unauthenticated; the caller must respond with 401.
func getAuthUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return nil, false
	}
	userObj := user.(*models.User)
	return userObj, true
}

// canManageRoom reports whether the caller may administer the room's
// membership. Only the super_admin or an admin of the room's unit qualifies.
func canManageRoom(c *gin.Context, userObj *models.User, roomID uuid.UUID) bool {
	if userObj.IsSuperAdmin || userObj.Role == "super_admin" {
		return true
	}

	var room models.CommunicationRoom
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return false
	}

	if userObj.UnitID == nil || *userObj.UnitID != room.UnitID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return false
	}

	if userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only unit admins can manage room members"})
		return false
	}

	return true
}

// roomAccessAllowed enforces object-level authorization on a communication
// room. Only the super_admin or a member of the room's unit may proceed; active
// room membership records are honored for unit-scoped access.
func roomAccessAllowed(c *gin.Context, userObj *models.User, roomID uuid.UUID) bool {
	if userObj.IsSuperAdmin || userObj.Role == "super_admin" {
		return true
	}

	var room models.CommunicationRoom
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return false
	}

	// Unit-scoped admins/officers may act within their unit.
	if userObj.UnitID != nil && *userObj.UnitID == room.UnitID {
		return true
	}

	// Active membership grants access to the room.
	var membership models.CommunicationRoomMember
	if err := config.DB.Where("room_id = ? AND user_id = ? AND status = ?", roomID, userObj.ID, "active").First(&membership).Error; err == nil {
		return true
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	return false
}
