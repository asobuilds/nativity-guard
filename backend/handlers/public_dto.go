package handlers

import (
	"time"

	"github.com/google/uuid"

	"security-solution/models"
)

type PublicUnitDTO struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	State             string    `json:"state"`
	LGA               string    `json:"lga"`
	Description       string    `json:"description,omitempty"`
	Status            string    `json:"status"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	TotalMembers      int       `json:"totalMembers"`
	OperationalRadius float64   `json:"operationalRadius"`
	IsVerified        bool      `json:"isVerified"`
}

type PublicCaseDTO struct {
	ID              uuid.UUID `json:"id"`
	UnitID          uuid.UUID `json:"unitId"`
	TrackingID      string    `json:"trackingId"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Location        string    `json:"location"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	GISLatitude     float64   `json:"gisLatitude"`
	GISLongitude    float64   `json:"gisLongitude"`
	LocationGeohash string    `json:"locationGeohash,omitempty"`
	Status          string    `json:"status"`
	PriorityLevel   string    `json:"priorityLevel"`
	IsPublic        bool      `json:"isPublic"`
	IsAnonymous     bool      `json:"isAnonymous"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func publicUnitDTOs(units []models.SecurityUnit) []PublicUnitDTO {
	out := make([]PublicUnitDTO, len(units))
	for i, unit := range units {
		out[i] = PublicUnitDTO{
			ID:                unit.ID,
			Name:              unit.Name,
			Type:              unit.Type,
			State:             unit.State,
			LGA:               unit.LGA,
			Description:       unit.CoverageArea,
			Status:            unit.Status,
			Latitude:          unit.Latitude,
			Longitude:         unit.Longitude,
			TotalMembers:      unit.TotalMembers,
			OperationalRadius: unit.OperationalRadius,
			IsVerified:        unit.IsVerified,
		}
	}
	return out
}

func publicCaseDTOs(cases []models.Case) []PublicCaseDTO {
	out := make([]PublicCaseDTO, len(cases))
	for i, c := range cases {
		out[i] = PublicCaseDTO{
			ID:              c.ID,
			UnitID:          c.UnitID,
			TrackingID:      c.TrackingID,
			Title:           c.Title,
			Description:     c.Description,
			Location:        c.Location,
			Latitude:        c.Latitude,
			Longitude:       c.Longitude,
			GISLatitude:     c.GISLatitude,
			GISLongitude:    c.GISLongitude,
			LocationGeohash: c.LocationGeohash,
			Status:          c.Status,
			PriorityLevel:   c.PriorityLevel,
			IsPublic:        c.IsPublic,
			IsAnonymous:     c.IsAnonymous,
			CreatedAt:       c.CreatedAt,
			UpdatedAt:       c.UpdatedAt,
		}
	}
	return out
}
