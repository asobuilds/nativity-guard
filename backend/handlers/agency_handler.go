package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

const (
	defaultDivisionAdminRank = "Division Admin"
	defaultObserverRank      = "Observer"
)

var validAgencyTypes = map[string]struct{}{
	"police": {},
	"nscdc":  {},
	"fire":   {},
	"frsc":   {},
}

type agencyTenantInput struct {
	AgencyType   string `json:"agencyType" binding:"required"`
	State        string `json:"state" binding:"required"`
	LGA          string `json:"lga" binding:"required"`
	DivisionName string `json:"divisionName" binding:"required"`
}

type assignAgencyDivisionAdminInput struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Rank   string `json:"rank"`
}

type agencyTenantAdminSummary struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone"`
	Rank      string    `json:"rank"`
}

type agencyTenantItem struct {
	ID           uuid.UUID                 `json:"id"`
	AgencyType   string                    `json:"agencyType"`
	State        string                    `json:"state"`
	LGA          string                    `json:"lga"`
	DivisionName string                    `json:"divisionName"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
	Admin        *agencyTenantAdminSummary `json:"admin,omitempty"`
}

type createAgencyObserverInput struct {
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone" binding:"required"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Password  string `json:"password" binding:"required,min=6"`
	Rank      string `json:"rank"`
	TenantID  string `json:"tenantId"`
}

type agencyObserverListItem struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	TenantID  uuid.UUID `json:"tenantId"`
	Rank      string    `json:"rank"`
	CreatedAt time.Time `json:"createdAt"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Status    string    `json:"status"`
}

// CreateAgencyTenant creates a state/LGA-scoped agency division.
func CreateAgencyTenant(c *gin.Context) {
	var input agencyTenantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agencyType := strings.ToLower(strings.TrimSpace(input.AgencyType))
	state := strings.TrimSpace(input.State)
	lga := strings.TrimSpace(input.LGA)
	divisionName := strings.TrimSpace(input.DivisionName)

	if _, ok := validAgencyTypes[agencyType]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agencyType must be one of police, nscdc, fire, or frsc"})
		return
	}
	if state == "" || lga == "" || divisionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state, lga, and divisionName are required"})
		return
	}

	tenant := models.AgencyTenant{
		AgencyType:   agencyType,
		State:        state,
		LGA:          lga,
		DivisionName: divisionName,
	}
	if err := config.DB.Create(&tenant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agency tenant"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Agency tenant created successfully",
		"tenant":  tenant,
	})
}

// ListAgencyTenants returns all agency tenants with their assigned division admins.
func ListAgencyTenants(c *gin.Context) {
	var tenants []models.AgencyTenant
	if err := config.DB.Order("created_at DESC").Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch agency tenants"})
		return
	}

	tenantIDs := make([]uuid.UUID, len(tenants))
	for i, t := range tenants {
		tenantIDs[i] = t.ID
	}

	type adminRow struct {
		TenantID  uuid.UUID `gorm:"column:tenant_id"`
		UserID    uuid.UUID `gorm:"column:user_id"`
		Email     string    `gorm:"column:email"`
		FirstName string    `gorm:"column:first_name"`
		LastName  string    `gorm:"column:last_name"`
		Phone     string    `gorm:"column:phone"`
		Rank      string    `gorm:"column:rank"`
	}

	adminMap := make(map[uuid.UUID]agencyTenantAdminSummary)
	if len(tenantIDs) > 0 {
		var rows []adminRow
		err := config.DB.Table("agency_observers AS ao").
			Select("ao.tenant_id, ao.user_id, ao.rank, u.email, u.first_name, u.last_name, u.phone").
			Joins("JOIN users AS u ON u.id = ao.user_id").
			Where("ao.tenant_id IN ? AND u.role = ? AND u.deleted_at IS NULL", tenantIDs, models.RoleAgencyDivisionAdmin).
			Scan(&rows).Error
		if err == nil {
			for _, r := range rows {
				adminMap[r.TenantID] = agencyTenantAdminSummary{
					UserID:    r.UserID,
					Email:     r.Email,
					FirstName: r.FirstName,
					LastName:  r.LastName,
					Phone:     r.Phone,
					Rank:      r.Rank,
				}
			}
		}
	}

	items := make([]agencyTenantItem, len(tenants))
	for i, t := range tenants {
		items[i] = agencyTenantItem{
			ID:           t.ID,
			AgencyType:   t.AgencyType,
			State:        t.State,
			LGA:          t.LGA,
			DivisionName: t.DivisionName,
			CreatedAt:    t.CreatedAt,
			UpdatedAt:    t.UpdatedAt,
		}
		if admin, ok := adminMap[t.ID]; ok {
			adminCopy := admin
			items[i].Admin = &adminCopy
		}
	}

	c.JSON(http.StatusOK, gin.H{"tenants": items})
}

// AssignAgencyDivisionAdmin assigns a user as the division admin for a tenant.
func AssignAgencyDivisionAdmin(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agency tenant ID"})
		return
	}

	var input assignAgencyDivisionAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	rawUserID := strings.TrimSpace(input.UserID)

	if email == "" && rawUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user email or userId is required"})
		return
	}

	rank := strings.TrimSpace(input.Rank)
	if rank == "" {
		rank = defaultDivisionAdminRank
	}

	var tenant models.AgencyTenant
	if err := config.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agency tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agency tenant"})
		return
	}

	var user models.User
	var userErr error
	if email != "" {
		userErr = config.DB.First(&user, "LOWER(email) = ?", email).Error
	} else {
		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		userErr = config.DB.First(&user, "id = ?", userID).Error
	}

	if userErr != nil {
		if errors.Is(userErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	if user.IsSuperAdmin || user.Role == models.RoleSuperAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Super admin accounts cannot be assigned as agency division admins"})
		return
	}

	var observer models.AgencyObserver
	observerErr := config.DB.Where("user_id = ?", user.ID).First(&observer).Error
	switch {
	case observerErr == nil:
		if observer.TenantID != tenant.ID {
			c.JSON(http.StatusConflict, gin.H{"error": "User is already assigned to another agency tenant"})
			return
		}
	case errors.Is(observerErr, gorm.ErrRecordNotFound):
		observer = models.AgencyObserver{}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agency assignment"})
		return
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("role", models.RoleAgencyDivisionAdmin).Error; err != nil {
			return err
		}

		if observer.ID == uuid.Nil {
			observer = models.AgencyObserver{
				UserID:   user.ID,
				TenantID: tenant.ID,
				Rank:     rank,
			}
			return tx.Create(&observer).Error
		}

		observer.Rank = rank
		return tx.Model(&models.AgencyObserver{}).
			Where("id = ?", observer.ID).
			Update("rank", observer.Rank).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign agency division admin"})
		return
	}

	user.Role = models.RoleAgencyDivisionAdmin

	c.JSON(http.StatusOK, gin.H{
		"message":  "Agency division admin assigned successfully",
		"user":     agencyUserSummary(user),
		"observer": observer,
	})
}

// CreateAgencyObserver creates an observer account inside the caller's tenant.
func CreateAgencyObserver(c *gin.Context) {
	_, adminObserver, ok := agencyDivisionAdminContext(c)
	if !ok {
		return
	}

	var input createAgencyObserverInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if requestedTenantID := strings.TrimSpace(input.TenantID); requestedTenantID != "" {
		parsedTenantID, err := uuid.Parse(requestedTenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
			return
		}
		if parsedTenantID != adminObserver.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Observer tenant does not match your agency tenant"})
			return
		}
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	phone := strings.TrimSpace(input.Phone)
	firstName := strings.TrimSpace(input.FirstName)
	lastName := strings.TrimSpace(input.LastName)
	if email == "" || phone == "" || firstName == "" || lastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, phone, firstName, and lastName are required"})
		return
	}

	rank := strings.TrimSpace(input.Rank)
	if rank == "" {
		rank = defaultObserverRank
	}

	var duplicateCount int64
	if err := config.DB.Model(&models.User{}).
		Where("LOWER(email) = ? OR phone = ?", email, phone).
		Count(&duplicateCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account uniqueness"})
		return
	}
	if duplicateCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "An account with that email or phone number already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create observer account"})
		return
	}

	var user models.User
	var observer models.AgencyObserver
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:     email,
			Phone:     phone,
			FirstName: firstName,
			LastName:  lastName,
			Password:  string(hashedPassword),
			Role:      models.RoleAgencyObserver,
			Status:    "active",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		observer = models.AgencyObserver{
			UserID:   user.ID,
			TenantID: adminObserver.TenantID,
			Rank:     rank,
		}
		return tx.Create(&observer).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create observer account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Agency observer created successfully",
		"user":     agencyUserSummary(user),
		"observer": observer,
	})
}

// ListAgencyObservers returns observers in the caller's agency tenant.
func ListAgencyObservers(c *gin.Context) {
	_, adminObserver, ok := agencyDivisionAdminContext(c)
	if !ok {
		return
	}

	var observers []agencyObserverListItem
	if err := config.DB.
		Table("agency_observers AS ao").
		Select(`ao.id, ao.user_id, ao.tenant_id, ao.rank, ao.created_at,
			u.email, u.phone, u.first_name, u.last_name, u.status`).
		Joins("JOIN users AS u ON u.id = ao.user_id").
		Where("ao.tenant_id = ? AND u.deleted_at IS NULL", adminObserver.TenantID).
		Order("ao.created_at ASC").
		Scan(&observers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch agency observers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"observers": observers})
}

func agencyDivisionAdminContext(c *gin.Context) (*models.User, *models.AgencyObserver, bool) {
	user, ok := getAuthUser(c)
	if !ok {
		return nil, nil, false
	}

	var observer models.AgencyObserver
	if err := config.DB.Where("user_id = ?", user.ID).First(&observer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Agency division admin is not assigned to a tenant"})
			return nil, nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agency assignment"})
		return nil, nil, false
	}

	return user, &observer, true
}

func agencyUserSummary(user models.User) gin.H {
	return gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"phone":     user.Phone,
		"firstName": user.FirstName,
		"lastName":  user.LastName,
		"role":      user.Role,
		"status":    user.Status,
	}
}
