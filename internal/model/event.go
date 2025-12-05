package model

import "time"

type Event struct {
	ID    int64     `db:"id"`
	Start time.Time `db:"start"`
}
