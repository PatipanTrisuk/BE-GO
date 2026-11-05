package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

// Create User
func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,

) error {
	query := `INSERT INTO users (username, email, password, role) values ($1 , $2, $3, $4) RETURNING id, created_at, updated_at`

	err := r.DB.QueryRow(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	return err
}

// Find User By Email
func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {

	user := &models.User{}

	query := `
        SELECT
            id,
            username,
            email,
            password,
            role,
            created_at,
            updated_at
        FROM users
        WHERE email = $1
    `

	err := r.DB.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// Find User By ID
func (r *UserRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.User, error) {

	user := &models.User{}

	query := `
        SELECT
            id,
            username,
            email,
            password,
            role,
            created_at,
            updated_at
        FROM users
        WHERE id = $1
    `

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
