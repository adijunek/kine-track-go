package models

import (
	"time"
)

type VesselIndex struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	MMSI        int       `gorm:"uniqueIndex;not null;column:mmsi" json:"mmsi"`
	ShipName    string    `gorm:"size:100;not null" json:"ship_name"`
	VesselType  string    `gorm:"size:50" json:"vessel_type"`
	FlagCountry string    `gorm:"size:50" json:"flag_country"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type VesselHistory struct {
	ID        uint      `gorm:"primaryKey" json:"-"` // Hide ID from JSON
	MMSI      int       `gorm:"index;not null;column:mmsi" json:"mmsi"`
	Latitude  float64   `gorm:"type:numeric(10,7);not null" json:"latitude"`
	Longitude float64   `gorm:"type:numeric(10,7);not null" json:"longitude"`
	SOG       float64   `gorm:"type:numeric(5,2)" json:"sog"`
	COG       float64   `gorm:"type:numeric(5,2)" json:"cog"`
	PingTime  time.Time `gorm:"index;not null" json:"ping_time"` // The crucial fix!
	CreatedAt time.Time `gorm:"autoCreateTime" json:"-"`
}

type BatchPayload struct {
	Vessels []VesselHistory `json:"vessels" binding:"required"`
}
