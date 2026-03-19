package db

import (
	"context"
	"time"

	"github.com/WahyuS002/uploy/db/sqlcgen"
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

func userFromGen(u sqlcgen.User) User {
	return User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		PlatformRole: u.PlatformRole,
		Status:       u.Status,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func CreateUserTx(ctx context.Context, tx pgx.Tx, email, passwordHash string) (User, error) {
	u, err := sqlcgen.New(tx).CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return User{}, err
	}
	return userFromGen(u), nil
}

func GetUserByEmail(ctx context.Context, email string) (User, error) {
	u, err := Queries.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	return userFromGen(u), nil
}

func GetUserByID(ctx context.Context, id string) (User, error) {
	u, err := Queries.GetUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return userFromGen(u), nil
}
