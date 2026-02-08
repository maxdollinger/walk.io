package db

import "time"

type VMSettings struct {
	AppID       string
	AppPath     string
	BaseVersion string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
