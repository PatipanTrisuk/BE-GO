package seed

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserSeed struct {
	Username string
	Email    string
	Password string
	Role     string
}

func DefaultUsers() []UserSeed {
	return []UserSeed{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "Admin1234!",
			Role:     "admin",
		},
		{
			Username: "moderator",
			Email:    "moderator@example.com",
			Password: "Moderator123!",
			Role:     "moderator",
		},
		{
			Username: "alice",
			Email:    "alice@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "bob",
			Email:    "bob@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "carol",
			Email:    "carol@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "dave",
			Email:    "dave@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "erin",
			Email:    "erin@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "frank",
			Email:    "frank@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "grace",
			Email:    "grace@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "heidi",
			Email:    "heidi@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "ivan",
			Email:    "ivan@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "judy",
			Email:    "judy@example.com",
			Password: "Password123!",
			Role:     "user",
		},
		{
			Username: "mallory",
			Email:    "mallory@example.com",
			Password: "Password123!",
			Role:     "user",
		},
	}
}

func Users(ctx context.Context, db *pgxpool.Pool, users []UserSeed) error {
	const query = `
		INSERT INTO users (username, email, password, role_id, email_verified_at)
		VALUES (
			$1,
			$2,
			$3,
			(SELECT id FROM roles WHERE LOWER(name) = LOWER($4) LIMIT 1),
			NOW()
		)
		ON CONFLICT (email) DO NOTHING
	`

	inserted := 0
	skipped := 0

	for _, user := range users {
		hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", user.Email, err)
		}

		tag, err := db.Exec(ctx, query, user.Username, user.Email, string(hashed), user.Role)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", user.Email, err)
		}

		if tag.RowsAffected() > 0 {
			inserted++
			log.Printf("seeded %-7s %s", user.Role, user.Email)
		} else {
			skipped++
			log.Printf("skipped %-7s %s (already exists)", user.Role, user.Email)
		}
	}

	log.Printf("users seed done: inserted=%d skipped=%d", inserted, skipped)
	return nil
}
