package domain

import "time"

type TaskLog struct {
	ID        string
	TaskID    string
	Action    string
	OldValue  *string
	NewValue  *string
	ChangedBy string
	CreatedAt time.Time
}
