package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// callerContext builds a short factual paragraph about the caller that goes
// into the assistant's system prompt so replies can reference their area
// without inventing anything. Deliberately excludes email and phone.
func callerContext(user *models.User) string {
	if user == nil {
		return ""
	}
	lines := []string{
		"- Name: " + user.FirstName + " " + user.LastName,
		"- Role: " + user.Role,
	}

	if user.UnitID != nil {
		var unit models.SecurityUnit
		if err := config.DB.First(&unit, "id = ?", user.UnitID).Error; err == nil {
			area := unit.LGA
			if area == "" {
				area = unit.State
			}
			lines = append(lines, "- Unit: "+unit.Name+" ("+area+")")
		}
	}

	var recent []models.Case
	recentCutoff := time.Now().AddDate(0, 0, -30)
	q := config.DB.Model(&models.Case{}).Where("created_at > ?", recentCutoff)
	if user.Role == "officer" || user.Role == "unit_admin" {
		if user.UnitID != nil {
			q = q.Where("unit_id = ?", user.UnitID)
		}
	}
	_ = q.Order("created_at desc").Limit(5).Find(&recent).Error
	if len(recent) > 0 {
		lines = append(lines, "- Recent local reports:")
		for _, c := range recent {
			lines = append(lines, fmt.Sprintf("    · %s (%s)", c.Title, c.Status))
		}
	}

	var unread int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND status = ?", user.ID, "unread").
		Count(&unread)
	if unread > 0 {
		lines = append(lines, fmt.Sprintf("- Unread notifications: %d", unread))
	}

	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// AIChatbot — AI-powered assistant for citizens.
func AIChatbot(c *gin.Context) {
	var input struct {
		Question string `json:"question" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	aiService := services.NewAIService()
	response, err := aiService.Chatbot(input.Question, userObj.Role, callerContext(userObj))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI chatbot failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"question": input.Question,
		"response": response,
	})
}

// AIStatus reports which provider and model are live, the last call's
// latency, and whether a fallback is configured. Super-admin only.
func AIStatus(c *gin.Context) {
	admin, ok := callerIsSuperAdmin(c)
	if !ok {
		return
	}

	svc := services.NewAIService()
	last := services.Status()

	payload := gin.H{
		"provider":    svc.ProviderName(),
		"model":       svc.Model(),
		"hasFallback": svc.HasFallback(),
		"requestedBy": admin.Email,
	}

	if !last.At.IsZero() {
		payload["lastCall"] = gin.H{
			"provider":  last.Provider,
			"model":     last.Model,
			"latencyMs": last.LatencyMs,
			"at":        last.At,
			"error":     last.Err,
		}
	} else {
		payload["lastCall"] = nil
	}

	c.JSON(http.StatusOK, payload)
}

// AIAnalyzeImage — AI image analysis for crime scene photos.
func AIAnalyzeImage(c *gin.Context) {
	var input struct {
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role == "citizen" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only officers and admins can analyze images"})
		return
	}

	aiService := services.NewAIService()
	analysis, err := aiService.AnalyzeImage(input.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI image analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analysis": analysis})
}

// AIAnalyzeLocation analyzes security risk for a location.
func AIAnalyzeLocation(c *gin.Context) {
	var input struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
		Location  string  `json:"location"`
		Radius    float64 `json:"radius"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var incidents []models.Case
	query := config.DB.Model(&models.Case{}).Where("status != ?", "closed")
	if userObj.Role == "officer" || userObj.Role == "unit_admin" {
		if userObj.UnitID != nil {
			query = query.Where("unit_id = ?", userObj.UnitID)
		}
	}
	if err := query.Order("created_at desc").Limit(20).Find(&incidents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incidents"})
		return
	}

	recentIncidents := ""
	for i, inc := range incidents {
		if i >= 10 {
			break
		}
		recentIncidents += fmt.Sprintf("Title: %s\nStatus: %s\nLocation: %s\n---\n", inc.Title, inc.Status, inc.Location)
	}

	aiService := services.NewAIService()
	analysis, err := aiService.AnalyzeLocationRisk(input.Latitude, input.Longitude, input.Location, recentIncidents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"location":      input.Location,
		"latitude":      input.Latitude,
		"longitude":     input.Longitude,
		"analysis":      analysis,
		"incidentCount": len(incidents),
	})
}

// AIGetMapInsights provides insights for map markers.
func AIGetMapInsights(c *gin.Context) {
	var input struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var cases []models.Case
	query := config.DB.Model(&models.Case{}).Where("status != ?", "closed")
	if userObj.Role == "officer" || userObj.Role == "unit_admin" {
		if userObj.UnitID != nil {
			query = query.Where("unit_id = ?", userObj.UnitID)
		}
	}
	if err := query.Order("created_at desc").Limit(30).Find(&cases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cases"})
		return
	}

	var incidents []string
	for _, c := range cases {
		incidents = append(incidents, fmt.Sprintf("Title: %s | Location: %s | Status: %s", c.Title, c.Location, c.Status))
	}

	aiService := services.NewAIService()
	analysis, err := aiService.AnalyzeIncidentPatterns("Current View", incidents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis":  analysis,
		"caseCount": len(cases),
		"viewLat":   input.Latitude,
		"viewLng":   input.Longitude,
	})
}

// AIGenerateSecurityWarning generates security warnings.
func AIGenerateSecurityWarning(c *gin.Context) {
	var input struct {
		Location    string   `json:"location"`
		IncidentIDs []string `json:"incidentIds"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can generate security warnings"})
		return
	}

	var incidentsData string
	if len(input.IncidentIDs) > 0 {
		var cases []models.Case
		if err := config.DB.Where("id IN (?)", input.IncidentIDs).Find(&cases).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incidents"})
			return
		}
		for _, c := range cases {
			incidentsData += fmt.Sprintf("Title: %s\nDescription: %s\nLocation: %s\nStatus: %s\n---\n", c.Title, c.Description, c.Location, c.Status)
		}
	} else {
		var cases []models.Case
		query := config.DB.Model(&models.Case{}).Where("status != ?", "closed")
		if input.Location != "" {
			query = query.Where("location ILIKE ?", "%"+input.Location+"%")
		}
		if userObj.UnitID != nil {
			query = query.Where("unit_id = ?", userObj.UnitID)
		}
		if err := query.Order("created_at desc").Limit(10).Find(&cases).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incidents"})
			return
		}
		for _, c := range cases {
			incidentsData += fmt.Sprintf("Title: %s\nLocation: %s\nStatus: %s\n---\n", c.Title, c.Location, c.Status)
		}
	}

	if incidentsData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No incident data found to generate warning"})
		return
	}

	aiService := services.NewAIService()
	warning, err := aiService.GenerateSecurityWarning(incidentsData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate warning: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"warning":     warning,
		"location":    input.Location,
		"generatedAt": time.Now(),
	})
}

// AIAnalyzeNews analyzes news for security implications.
func AIAnalyzeNews(c *gin.Context) {
	var input struct {
		Content string `json:"content" binding:"required"`
		Source  string `json:"source"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	aiService := services.NewAIService()
	analysis, err := aiService.AnalyzeNewsSentiment(input.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "News analysis failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis":   analysis,
		"source":     input.Source,
		"analyzedAt": time.Now(),
	})
}

// AIGetSmartTips provides context-aware safety tips.
func AIGetSmartTips(c *gin.Context) {
	var input struct {
		Location      string `json:"location"`
		TimeOfDay     string `json:"timeOfDay"`
		RecentThreats string `json:"recentThreats"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if input.TimeOfDay == "" {
		hour := time.Now().Hour()
		switch {
		case hour < 6:
			input.TimeOfDay = "Night (Late)"
		case hour < 12:
			input.TimeOfDay = "Morning"
		case hour < 18:
			input.TimeOfDay = "Afternoon"
		default:
			input.TimeOfDay = "Evening"
		}
	}

	var recentIncidents []models.Case
	query := config.DB.Model(&models.Case{}).Where("created_at > ?", time.Now().Add(-7*24*time.Hour))
	if userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID)
	}
	if input.Location != "" {
		query = query.Where("location ILIKE ?", "%"+input.Location+"%")
	}
	query.Order("created_at desc").Limit(5).Find(&recentIncidents)

	recentThreats := input.RecentThreats
	if recentThreats == "" && len(recentIncidents) > 0 {
		for _, inc := range recentIncidents {
			recentThreats += inc.Title + ", "
		}
	}

	aiService := services.NewAIService()
	tips, err := aiService.GetSmartSafetyTips(input.Location, userObj.Role, input.TimeOfDay, recentThreats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get safety tips: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tips":      tips,
		"location":  input.Location,
		"timeOfDay": input.TimeOfDay,
		"userRole":  userObj.Role,
	})
}

// AIPredictHotspots predicts risk hotspots.
func AIPredictHotspots(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can predict hotspots"})
		return
	}

	var cases []models.Case
	query := config.DB.Model(&models.Case{}).Where("created_at > ?", time.Now().Add(-30*24*time.Hour))
	if userObj.UnitID != nil {
		query = query.Where("unit_id = ?", userObj.UnitID)
	}
	if err := query.Find(&cases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cases"})
		return
	}
	if len(cases) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Not enough data for prediction"})
		return
	}

	historicalData := ""
	for _, c := range cases {
		historicalData += fmt.Sprintf("Title: %s\nLocation: %s\nStatus: %s\n---\n", c.Title, c.Location, c.Status)
	}

	aiService := services.NewAIService()
	prediction, err := aiService.PredictRiskHotspots(historicalData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Prediction failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prediction":  prediction,
		"caseCount":   len(cases),
		"generatedAt": time.Now(),
	})
}