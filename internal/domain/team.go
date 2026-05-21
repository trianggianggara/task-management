package domain

import "time"

type Team struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
}
