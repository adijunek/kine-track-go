package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	// Remember to adjust this to your actual module path!
	"github.com/adijunek/kine-track-go/internal/spatial"
)

type MapController struct {
	DB               *gorm.DB
	TrajectoryEngine *spatial.TrajectoryEngine
}

func NewMapController(db *gorm.DB) *MapController {
	return &MapController{
		DB:               db,
		TrajectoryEngine: spatial.NewTrajectoryEngine(db),
	}
}

func (mc *MapController) GetMapDashboard(c *gin.Context) {
	// Hardcoded for the CV showcase: DLN MANDALIKA (3) on Balikpapan-Surabaya (1551)
	vesselID := 3
	routeID := 1551

	// 1. Fetch Trajectory & ETA
	eta, err := mc.TrajectoryEngine.PredictETA(vesselID, routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Fetch Latest Ship Coordinates (for the map marker)
	var ship struct {
		Lat   float64 `json:"lat"`
		Lon   float64 `json:"lon"`
		Speed float64 `json:"speed"`
	}
	mc.DB.Raw(`
		SELECT lat, lon, speed 
		FROM vessel_histories 
		WHERE vessel_id = ? 
		ORDER BY recorded_at DESC LIMIT 1
	`, vesselID).Scan(&ship)

	// 3. Fetch Cyclone Hazards (for drawing red danger zones on the map)
	var cyclones []struct {
		Lat       float64 `json:"lat"`
		Lon       float64 `json:"lon"`
		WindSpeed float64 `json:"wind_speed"`
	}
	mc.DB.Raw(`
		SELECT lat, lon, wind_speed 
		FROM cyclone_predictions 
		WHERE target_time >= NOW() - INTERVAL '24 HOURS'
	`).Scan(&cyclones)

	// Return the unified dashboard payload
	c.JSON(http.StatusOK, gin.H{
		"vessel":     ship,
		"trajectory": eta,
		"hazards":    cyclones,
	})
}
