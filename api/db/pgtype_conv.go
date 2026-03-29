package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func stringPtrFromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

func timePtrFromPgTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
