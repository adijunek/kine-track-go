package spatial

import (
	"fmt"

	"gorm.io/gorm"
)

// HazardZone represents the data we extract from the cyclone_zones table
type HazardZone struct {
	SystemName    string
	SeverityLevel int
}

// CycloneScanner holds the database dependency
type CycloneScanner struct {
	DB *gorm.DB
}

func NewCycloneScanner(db *gorm.DB) *CycloneScanner {
	return &CycloneScanner{DB: db}
}

// CheckExposure takes a raw coordinate and asks PostGIS if it falls inside any active weather polygons.
func (cs *CycloneScanner) CheckExposure(latitude, longitude float64) ([]HazardZone, error) {
	var zones []HazardZone

	// THE POSTGIS QUERY
	// Note: ST_MakePoint always takes (Longitude, Latitude) - X before Y!
	// ST_SetSRID(..., 4326) tells the database this is GPS data (WGS 84).
	// ST_Intersects is the spatial function that checks for overlaps.
	query := `
		SELECT system_name, severity_level 
		FROM cyclone_zones 
		WHERE ST_Intersects(
			geom, 
			ST_SetSRID(ST_MakePoint(?, ?), 4326)
		)
	`

	// Execute the raw SQL using Gorm and map it to our Go struct
	result := cs.DB.Raw(query, longitude, latitude).Scan(&zones)

	if result.Error != nil {
		return nil, fmt.Errorf("spatial query failed: %w", result.Error)
	}

	return zones, nil
}
