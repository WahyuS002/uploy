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

func int32PtrFromPgInt4(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	i := v.Int32
	return &i
}

func pgInt4FromInt32Ptr(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func timePtrFromPgTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
