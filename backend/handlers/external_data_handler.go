package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/models"
	"security-solution/services"
)

// queryFloatRequired reads a query param as float64, returning ok=false on
// failure. Named distinctly from the existing `parseCoordinate` in
// geo_handler.go, which has a different signature and purpose.
func queryFloatRequired(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func queryFloat(raw string, fallback float64) float64 {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}

func queryInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return v
}

// ------------------------------------------------ weather

// GetWeatherNow — GET /weather/current?lat=&lng=&label=&approximate=
func GetWeatherNow(c *gin.Context) {
	lat, ok1 := queryFloatRequired(c.Query("lat"))
	lng, ok2 := queryFloatRequired(c.Query("lng"))
	if !ok1 || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng are required as numbers"})
		return
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat must be within [-90,90] and lng within [-180,180]"})
		return
	}

	label := strings.TrimSpace(c.Query("label"))
	if label == "" {
		label = "Your area"
	}
	approximate := strings.EqualFold(c.Query("approximate"), "true")

	bundle, err := services.NewWeatherService().GetWeather(lat, lng, label, approximate)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bundle)
}

// ------------------------------------------------ news

// GetLiveNews — GET /news/live?categories=security,community&limit=10
//
// When `categories` is absent, falls back to the caller's active alert
// subscription. When that is absent too, uses "general".
func GetLiveNews(c *gin.Context) {
	categories := []string{}
	if raw := strings.TrimSpace(c.Query("categories")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				categories = append(categories, p)
			}
		}
	} else if userValue, exists := c.Get("user"); exists {
		if user, ok := userValue.(*models.User); ok && user != nil {
			var sub models.AlertSubscription
			if err := config.DB.
				Where("user_id = ? AND is_active = ?", user.ID, true).
				Order("created_at DESC").
				First(&sub).Error; err == nil && sub.Type != "" {
				for _, p := range strings.Split(sub.Type, ",") {
					p = strings.TrimSpace(strings.ToLower(p))
					if p != "" {
						categories = append(categories, p)
					}
				}
			}
		}
	}
	if len(categories) == 0 {
		categories = []string{"general"}
	}

	limit := queryInt(c.Query("limit"), 10)

	items, err := services.NewNewsService().GetForCategories(categories, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"categories": categories,
	})
}

// ------------------------------------------------ directions

// PostDirections — POST /directions
// Body: { fromLat, fromLng, toLat, toLng, profile? }
func PostDirections(c *gin.Context) {
	var input struct {
		FromLat float64 `json:"fromLat" binding:"required"`
		FromLng float64 `json:"fromLng" binding:"required"`
		ToLat   float64 `json:"toLat" binding:"required"`
		ToLng   float64 `json:"toLng" binding:"required"`
		Profile string  `json:"profile"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fromLat, fromLng, toLat, toLng are required"})
		return
	}

	route, err := services.NewDirectionsService().GetRoute(
		input.FromLat, input.FromLng, input.ToLat, input.ToLng, input.Profile,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"route": route})
}

// ------------------------------------------------ POIs

// GetMapPOIs — GET /map/pois?lat=&lng=&radius=&categories=hospital,police
func GetMapPOIs(c *gin.Context) {
	lat, ok1 := queryFloatRequired(c.Query("lat"))
	lng, ok2 := queryFloatRequired(c.Query("lng"))
	if !ok1 || !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng are required as numbers"})
		return
	}
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat or lng out of range"})
		return
	}

	radius := queryInt(c.Query("radius"), 3000)

	var categories []string
	if raw := strings.TrimSpace(c.Query("categories")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				categories = append(categories, p)
			}
		}
	}

	items, err := services.NewPOIService().GetPOIs(lat, lng, radius, categories)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"count":    len(items),
		"radius":   radius,
		"location": gin.H{"latitude": lat, "longitude": lng},
	})
}