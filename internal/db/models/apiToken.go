package db

import "time"

type APIToken struct {
	ID        string    `json:"id"`
	TokenHash string    `json:"token_hash"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
