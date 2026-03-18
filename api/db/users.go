package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	PlatformRole string    `json:"platform_role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func CreateUserTx(ctx context.Context, tx pgx.Tx, email, passwordHash string) (User, error) {
	id := fmt.Sprintf("usr-%d", time.Now().UnixNano())
	var u User
	err := tx.QueryRow(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, platform_role, status, created_at, updated_at`,
		id, email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.PlatformRole, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, platform_role, status, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.PlatformRole, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, platform_role, status, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.PlatformRole, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
