package models

import "time"

type Device struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	DeviceKey  string    `json:"deviceKey"`
	Name       string    `json:"name,omitempty"`
	UserAgent  string    `json:"userAgent,omitempty"`
	IPAddress  string    `json:"ipAddress,omitempty"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time `json:"createdAt"`
}
