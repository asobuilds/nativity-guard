package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/models"
)

// RequireRole checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}
		userObj, ok := user.(*models.User)
		if !ok || userObj == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}
		if userObj.Status == "banned" || userObj.Status == "suspended" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
			c.Abort()
			return
		}

		for _, role := range roles {
			if userObj.Role == role {
				c.Next()
				return
			}
			if role == models.RoleSuperAdmin && userObj.IsSuperAdmin {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// CanAccessCase enforces the Nativity Guard core rule: unit access is NOT case access.
//
// Access levels (stored in context as "case_access_level"):
//   - super_admin:     all cases
//   - head_admin:      all cases in the user's unit
//   - assigned_admin:  cases submitted to this admin via CaseAdminAssignment
//   - officer_primary: full access (Case.AssignedTo == user.ID)
//   - officer_paired:  full access (CaseOfficer.Role in {primary, paired, investigator})
//   - officer_support: limited access (CaseOfficer.Role == support)
//   - reporter:        public-safe view only (Case.ReportedBy == user.ID)
//
// Accepts either :id or :caseId route params.
func CanAccessCase(c *gin.Context) {
	caseIDStr := c.Param("id")
	if caseIDStr == "" {
		caseIDStr = c.Param("caseId")
	}
	id, err := uuid.Parse(caseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid case ID"})
		c.Abort()
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		c.Abort()
		return
	}
	userObj := user.(*models.User)

	// Super Admin bypass
	if userObj.IsSuperAdmin || userObj.Role == "super_admin" {
		c.Set("case_access_level", "super_admin")
		c.Next()
		return
	}

	var caseObj models.Case
	if err := config.DB.First(&caseObj, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Case not found"})
		c.Abort()
		return
	}

	// Linked agency division admins may oversee cases in their exact verified unit.
	if userObj.Role == models.RoleAgencyDivisionAdmin {
		var observer models.AgencyObserver
		if err := config.DB.Where("user_id = ?", userObj.ID).First(&observer).Error; err == nil {
			var tenant models.AgencyTenant
			if err := config.DB.First(&tenant, "id = ?", observer.TenantID).Error; err == nil && tenant.UnitID != nil && *tenant.UnitID == caseObj.UnitID {
				c.Set("case_access_level", "agency_admin")
				c.Next()
				return
			}
		}
	}

	// Reporter (public-safe view only)
	if caseObj.ReportedBy == userObj.ID {
		c.Set("case_access_level", "reporter")
		c.Next()
		return
	}

	// Authoritative membership lookup (UnitMembership is the source of truth)
	var membership models.UnitMembership
	hasMembership := config.DB.
		Where("unit_id = ? AND user_id = ? AND status = ?", caseObj.UnitID, userObj.ID, "active").
		First(&membership).Error == nil

	if !hasMembership {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		c.Abort()
		return
	}

	// Head Admin: all cases in their unit
	if membership.IsHeadAdmin {
		c.Set("case_access_level", "head_admin")
		c.Next()
		return
	}

	// Regular Admin: only cases submitted to them
	if membership.Role == "unit_admin" {
		var assignment models.CaseAdminAssignment
		if err := config.DB.
			Where("case_id = ? AND admin_id = ? AND status IN ?", caseObj.ID, userObj.ID, []string{"pending", "approved"}).
			First(&assignment).Error; err == nil {
			c.Set("case_access_level", "assigned_admin")
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Case not submitted to you"})
		c.Abort()
		return
	}

	// Officer tiers
	if membership.Role == "officer" {
		// Primary: Case.AssignedTo == user.ID
		if caseObj.AssignedTo != nil && *caseObj.AssignedTo == userObj.ID {
			c.Set("case_access_level", "officer_primary")
			c.Next()
			return
		}

		// Paired / Support via CaseOfficer
		var caseOfficer models.CaseOfficer
		if err := config.DB.
			Where("case_id = ? AND officer_id = ?", caseObj.ID, userObj.ID).
			First(&caseOfficer).Error; err == nil {
			switch caseOfficer.Role {
			case "primary", "paired", "investigator":
				// "investigator" is the legacy default — treated as paired for compatibility
				c.Set("case_access_level", "officer_paired")
				c.Next()
				return
			case "support":
				c.Set("case_access_level", "officer_support")
				c.Next()
				return
			}
		}
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	c.Abort()
}

// CanAccessUnit checks if user can access a specific unit
func CanAccessUnit(c *gin.Context) {
	unitID := c.Param("unitId")
	if unitID == "" {
		unitID = c.Param("id")
	}
	id, err := uuid.Parse(unitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
		c.Abort()
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		c.Abort()
		return
	}
	userObj, ok := user.(*models.User)
	if !ok || userObj == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		c.Abort()
		return
	}
	if userObj.Status == "banned" || userObj.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
		c.Abort()
		return
	}

	if userObj.IsSuperAdmin || userObj.Role == models.RoleSuperAdmin {
		c.Next()
		return
	}

	if userObj.UnitID == nil || *userObj.UnitID != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this unit"})
		c.Abort()
		return
	}

	c.Next()
}


// DenyAgencyAdminCaseMutation keeps agency division oversight read-only until
// an explicit, audited agency-to-officer assignment workflow is implemented.
func DenyAgencyAdminCaseMutation(c *gin.Context) {
	userValue, exists := c.Get("user")
	if exists {
		if user, ok := userValue.(*models.User); ok && user.Role == models.RoleAgencyDivisionAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agency division access is read-only for this operation"})
			c.Abort()
			return
		}
	}
	if c.GetString("case_access_level") == "agency_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agency division access is read-only for this operation"})
		c.Abort()
		return
	}
	c.Next()
}
