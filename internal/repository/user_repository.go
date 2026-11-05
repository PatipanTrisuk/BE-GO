package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jackc/pgx/v5"
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

func scanUser(
	scanner interface {
		Scan(dest ...any) error
	},
	user *models.User,
	includePassword bool,
) error {
	var password sql.NullString
	var discordID sql.NullString

	dest := []any{
		&user.ID,
		&user.Username,
		&user.Email,
	}
	if includePassword {
		dest = append(dest, &password)
	}
	dest = append(dest,
		&user.Role,
		&user.AuthProvider,
		&discordID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err := scanner.Scan(dest...); err != nil {
		return err
	}

	if includePassword && password.Valid {
		user.Password = password.String
	}
	if discordID.Valid {
		id := discordID.String
		user.DiscordID = &id
	}
	if user.AuthProvider == "" {
		user.AuthProvider = "local"
	}
	return nil
}

// Create User
func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {
	if user.AuthProvider == "" {
		user.AuthProvider = "local"
	}

	query := `
		INSERT INTO users (username, email, password, role, auth_provider, discord_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	var password any
	if strings.TrimSpace(user.Password) == "" {
		password = nil
	} else {
		password = user.Password
	}

	var discordID any
	if user.DiscordID != nil && strings.TrimSpace(*user.DiscordID) != "" {
		discordID = strings.TrimSpace(*user.DiscordID)
	}

	err := r.DB.QueryRow(
		ctx,
		query,
		user.Username,
		user.Email,
		password,
		user.Role,
		user.AuthProvider,
		discordID,
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
            auth_provider,
            discord_id,
            created_at,
            updated_at
        FROM users
        WHERE email = $1
    `

	err := scanUser(r.DB.QueryRow(ctx, query, email), user, true)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByDiscordID(
	ctx context.Context,
	discordID string,
) (*models.User, error) {
	user := &models.User{}

	query := `
        SELECT
            id,
            username,
            email,
            password,
            role,
            auth_provider,
            discord_id,
            created_at,
            updated_at
        FROM users
        WHERE discord_id = $1
    `

	err := scanUser(r.DB.QueryRow(ctx, query, discordID), user, true)
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
            auth_provider,
            discord_id,
            created_at,
            updated_at
        FROM users
        WHERE id = $1
    `

	err := scanUser(r.DB.QueryRow(ctx, query, id), user, true)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) LinkDiscordID(ctx context.Context, userID, discordID string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET discord_id = $2, updated_at = NOW()
			WHERE id = $1
		`,
		userID,
		discordID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`,
		username,
	).Scan(&exists)
	return exists, err
}

type ListParams struct {
	Page  int
	Limit int
	Query string
}

func (r *UserRepository) List(ctx context.Context, params ListParams) ([]models.User, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	search := strings.TrimSpace(params.Query)

	var (
		rows pgx.Rows
		err  error
	)

	if search == "" {
		query := `
        SELECT
            id,
            username,
            email,
            role,
            auth_provider,
            discord_id,
            created_at,
            updated_at
        FROM users
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
		rows, err = r.DB.Query(ctx, query, limit, offset)
	} else {
		query := `
        SELECT
            id,
            username,
            email,
            role,
            auth_provider,
            discord_id,
            created_at,
            updated_at
        FROM users
        WHERE email ILIKE $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
		rows, err = r.DB.Query(ctx, query, "%"+search+"%", limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := scanUser(rows, &user, false); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) Count(ctx context.Context, query string) (int, error) {
	var count int
	search := strings.TrimSpace(query)
	if search == "" {
		err := r.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
		return count, err
	}

	err := r.DB.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM users WHERE email ILIKE $1`,
		"%"+search+"%",
	).Scan(&count)
	return count, err
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET password = $2, updated_at = NOW()
			WHERE id = $1
		`,
		userID,
		passwordHash,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
