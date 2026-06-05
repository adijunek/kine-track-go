package haversine

import (
	"math"
	"time"
)

const (
	// EarthRadiusNM is the volumetric mean radius of the Earth in Nautical Miles
	EarthRadiusNM = 3440.065

	// MaxPlausibleSpeed is the absolute physical limit for commercial/fishing vessels (knots)
	MaxPlausibleSpeed = 45.0
)

// degreesToRadians converts standard GPS coordinates into radians for trigonometry
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// Distance calculates the shortest distance between two points on the sphere.
// Returns the distance strictly in Nautical Miles (NM).
func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)

	rLat1 := degreesToRadians(lat1)
	rLat2 := degreesToRadians(lat2)

	// The core Haversine mathematics
	a := math.Pow(math.Sin(dLat/2), 2) +
		math.Pow(math.Sin(dLon/2), 2)*math.Cos(rLat1)*math.Cos(rLat2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusNM * c
}

// IsKinematicallyValid acts as the bouncer for your database.
// It checks if the required speed to travel between two pings is physically possible.
func IsKinematicallyValid(lat1, lon1, lat2, lon2 float64, t1, t2 time.Time) (bool, float64) {
	// Protect against division by zero if two pings arrive with identical timestamps
	if t1.Equal(t2) {
		return false, 0
	}

	distance := Distance(lat1, lon1, lat2, lon2)

	// Convert the time difference into absolute hours
	hours := math.Abs(t2.Sub(t1).Hours())

	// True Speed Over Ground (Distance / Time)
	calculatedSpeedKnots := distance / hours

	// If the calculated speed exceeds 45 knots, the ping is a physically impossible spoof.
	isValid := calculatedSpeedKnots <= MaxPlausibleSpeed

	return isValid, calculatedSpeedKnots
}
