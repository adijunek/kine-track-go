package spatial

import (
	"fmt"

	"gorm.io/gorm"
)

// TrajectoryPrediction maps directly to your CTE output
type TrajectoryPrediction struct {
	RouteName           string  `json:"route_name"`
	ReportedDestination string  `json:"reported_destination"`
	TotalDistanceNM     float64 `json:"total_distance_nm"`
	TruePercentComplete float64 `json:"true_percent_complete"`
	RemainingNM         float64 `json:"remaining_nm"`
	EstimatedHours      float64 `json:"estimated_hours"`

	// ADDED GORM TAGS HERE
	CompletedGeoJSON string `gorm:"column:completed_geojson" json:"completed_geojson"`
	RemainingGeoJSON string `gorm:"column:remaining_geojson" json:"remaining_geojson"`
}

type TrajectoryEngine struct {
	DB *gorm.DB
}

func NewTrajectoryEngine(db *gorm.DB) *TrajectoryEngine {
	return &TrajectoryEngine{DB: db}
}

// PredictETA executes the bi-directional spatial CTE
func (te *TrajectoryEngine) PredictETA(vesselID int, routeID int) (*TrajectoryPrediction, error) {
	// We use a temporary struct to capture the DB response, including the ship's current speed
	type dbResult struct {
		TrajectoryPrediction
		CurrentSpeed float64 `gorm:"column:current_speed"`
	}
	var res dbResult

	query := `
		SELECT 
			r.name AS route_name,
			v.destination AS reported_destination,
			ROUND((ST_Length(r.geom::geography) / 1852.0)::numeric, 2) AS total_distance_nm,
			
			CASE WHEN v.destination ILIKE split_part(r.name, ' - ', 1) || '%' THEN ROUND(((1 - ST_LineLocatePoint(r.geom, v.geom)) * 100)::numeric, 2)
			ELSE ROUND((ST_LineLocatePoint(r.geom, v.geom) * 100)::numeric, 2) END AS true_percent_complete,
			
			CASE WHEN v.destination ILIKE split_part(r.name, ' - ', 1) || '%' THEN ROUND((ST_LineLocatePoint(r.geom, v.geom) * (ST_Length(r.geom::geography) / 1852.0))::numeric, 2)
			ELSE ROUND(((1 - ST_LineLocatePoint(r.geom, v.geom)) * (ST_Length(r.geom::geography) / 1852.0))::numeric, 2) END AS remaining_nm,
			
			-- THE MAGIC: Slicing the geometry based on the ship's directional heading
			CASE WHEN v.destination ILIKE split_part(r.name, ' - ', 1) || '%' THEN ST_AsGeoJSON(ST_LineSubstring(r.geom, ST_LineLocatePoint(r.geom, v.geom), 1))
			ELSE ST_AsGeoJSON(ST_LineSubstring(r.geom, 0, ST_LineLocatePoint(r.geom, v.geom))) END AS completed_geojson,
			
			CASE WHEN v.destination ILIKE split_part(r.name, ' - ', 1) || '%' THEN ST_AsGeoJSON(ST_LineSubstring(r.geom, 0, ST_LineLocatePoint(r.geom, v.geom)))
			ELSE ST_AsGeoJSON(ST_LineSubstring(r.geom, ST_LineLocatePoint(r.geom, v.geom), 1)) END AS remaining_geojson,

			v.speed AS current_speed
		FROM vessel_histories v
		CROSS JOIN osi_routes r
		WHERE v.vessel_id = ? AND r.id = ?
		ORDER BY v.recorded_at DESC 
		LIMIT 1;
	`

	result := te.DB.Raw(query, vesselID, routeID).Scan(&res)

	if result.Error != nil {
		return nil, fmt.Errorf("trajectory query failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("no telemetry or route found for the given IDs")
	}

	pred := res.TrajectoryPrediction

	// Calculate ETA. Prevent division by zero if the ship is anchored/stationary.
	if res.CurrentSpeed > 0 {
		pred.EstimatedHours = pred.RemainingNM / res.CurrentSpeed
	} else {
		pred.EstimatedHours = -1 // -1 indicates stationary
	}

	return &pred, nil
}
