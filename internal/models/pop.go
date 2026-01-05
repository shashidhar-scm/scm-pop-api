package models

import "time"

type PopInput struct {
	PosterName      string    `json:"poster_name"`
	PosterID        string    `json:"poster_id"`
	CampaignID      string    `json:"campaign_id,omitempty"`
	HostName        string    `json:"host_name"`
	KioskName       string    `json:"kiosk_name"`
	PosterType      string    `json:"poster_type"`
	PopDatetime     time.Time `json:"pop_datetime"`
	PosterCreatedBy int       `json:"poster_created_by"`
	KioskLat        float64   `json:"kiosk_lat"`
	KioskLong       float64   `json:"kiosk_long"`
	City            string    `json:"city"`
	Region          string    `json:"region"`
	PlayCount       int       `json:"play_count"`
	Value           int       `json:"value"`
	ClickX          *int      `json:"click_x,omitempty"`
	ClickY          *int      `json:"click_y,omitempty"`
	Type            string    `json:"type"`
	Url             string    `json:"url"`
}
