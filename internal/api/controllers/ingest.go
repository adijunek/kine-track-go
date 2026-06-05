package controllers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	// Make sure these match your module name!
	"github.com/adijunek/kine-track-go/internal/models"
	"github.com/adijunek/kine-track-go/internal/spatial"
	"github.com/adijunek/kine-track-go/pkg/haversine"
)

type IngestController struct {
	DB *gorm.DB
}

func NewIngestController(db *gorm.DB) *IngestController {
	return &IngestController{DB: db}
}

func (ic *IngestController) HandleBatch(c *gin.Context) {
	var payload models.BatchPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed JSON payload"})
		return
	}

	if len(payload.Vessels) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Batch payload is empty"})
		return
	}

	// 1. Gather all unique MMSIs from the incoming batch
	mmsiList := make([]int, 0)
	for _, v := range payload.Vessels {
		mmsiList = append(mmsiList, v.MMSI)
	}

	// 2. The N+1 Killer: Fetch the absolute latest ping for all these ships in ONE query.
	// Postgres 'DISTINCT ON' grabs only the most recent row per MMSI.
	var lastKnownList []models.VesselHistory
	ic.DB.Where("mmsi IN ?", mmsiList).
		Select("DISTINCT ON (mmsi) *").
		Order("mmsi, ping_time DESC").
		Find(&lastKnownList)

	// Map them for O(1) lightning-fast memory lookups
	lastKnownMap := make(map[int]models.VesselHistory)
	for _, ping := range lastKnownList {
		lastKnownMap[ping.MMSI] = ping
	}

	// 3. The Validation Loop
	var cleanBatch []models.VesselHistory
	spoofedCount := 0

	scanner := spatial.NewCycloneScanner(ic.DB)
	shipsInDanger := 0

	for _, currentPing := range payload.Vessels {
		if lastPing, exists := lastKnownMap[currentPing.MMSI]; exists {

			// Fire the Anti-Spoofing Math Engine
			isValid, calculatedSpeed := haversine.IsKinematicallyValid(
				lastPing.Latitude, lastPing.Longitude,
				currentPing.Latitude, currentPing.Longitude,
				lastPing.PingTime, currentPing.PingTime,
			)

			if !isValid {
				log.Printf("🚨 SPOOF DETECTED [MMSI: %d]: Impossible speed of %.1f knots. Dropping payload.", currentPing.MMSI, calculatedSpeed)
				spoofedCount++
				continue // Skip appending this to the clean batch
			}
		}

		// The ping is physically possible (or it's a brand new ship)
		cleanBatch = append(cleanBatch, currentPing)
		// CRITICAL: Update the map in-memory!
		// If the batch contains 10 sequential pings for the same ship,
		// ping #2 must be validated against ping #1, not the old database value.
		lastKnownMap[currentPing.MMSI] = currentPing

		hazards, err := scanner.CheckExposure(currentPing.Latitude, currentPing.Longitude)
		if err == nil && len(hazards) > 0 {
			shipsInDanger++
			for _, hazard := range hazards {
				log.Printf("⚠️ HAZARD ALERT [MMSI: %d]: Sailed into %s (Category %d)",
					currentPing.MMSI, hazard.SystemName, hazard.SeverityLevel)
			}
		}
	}

	// 4. Secure Database Insertion
	if len(cleanBatch) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Entire batch was rejected by kinematic validation.",
			"spoofed_dropped": spoofedCount,
		})
		return
	}

	result := ic.DB.CreateInBatches(&cleanBatch, 500)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database ingestion failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":          "success",
		"inserted_rows":   result.RowsAffected,
		"spoofed_dropped": spoofedCount,
		"hazards_active":  shipsInDanger,
		"message":         "Validated telemetry secured and spatial geometries checked",
	})
}
