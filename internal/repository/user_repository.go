package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	var emailVerifiedAt sql.NullTime
	var deletedAt sql.NullTime

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
		&emailVerifiedAt,
		&deletedAt,
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
	if emailVerifiedAt.Valid {
		t := emailVerifiedAt.Time
		user.EmailVerifiedAt = &t
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		user.DeletedAt = &t
	}
	if user.AuthProvider == "" {
		user.AuthProvider = "local"
	}
	return nil
}

const userSelectWithPassword = `
	id,
	username,
	email,
	password,
	role,
	auth_provider,
	discord_id,
	email_verified_at,
	deleted_at,
	created_at,
	updated_at
`

const userSelectPublic = `
	id,
	username,
	email,
	role,
	auth_provider,
	discord_id,
	email_verified_at,
	deleted_at,
	created_at,
	updated_at
`

// Create User
func (r *UserRepository) Create(
	ctx context.Context,
	user *models.User,
) error {
	if user.AuthProvider == "" {
		user.AuthProvider = "local"
	}

	query := `
		INSERT INTO users (username, email, password, role, auth_provider, discord_id, email_verified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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

	var emailVerifiedAt any
	if user.EmailVerifiedAt != nil {
		emailVerifiedAt = *user.EmailVerifiedAt
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
		emailVerifiedAt,
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
        SELECT ` + userSelectWithPassword + `
        FROM users
        WHERE email = $1 AND deleted_at IS NULL
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
        SELECT ` + userSelectWithPassword + `
        FROM users
        WHERE discord_id = $1 AND deleted_at IS NULL
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
        SELECT ` + userSelectWithPassword + `
        FROM users
        WHERE id = $1 AND deleted_at IS NULL
    `

	err := scanUser(r.DB.QueryRow(ctx, query, id), user, true)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByIDAny(
	ctx context.Context,
	id string,
) (*models.User, error) {
	user := &models.User{}

	query := `
        SELECT ` + userSelectWithPassword + `
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
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`,
		username,
	).Scan(&exists)
	return exists, err
}

type ListParams struct {
	Page          int
	Limit         int
	Query         string
	Role          string
	AuthProvider  string
	DeletedStatus string // active | deleted | all (default active)
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	UpdatedFrom   *time.Time
	UpdatedTo     *time.Time
	SortBy        string
	SortDir       string
}

var userSortColumns = map[string]string{
	"username":          "username",
	"email":             "email",
	"role":              "role",
	"auth_provider":     "auth_provider",
	"email_verified_at": "email_verified_at",
	"created_at":        "created_at",
	"updated_at":        "updated_at",
}

func orderByClause(sortBy, sortDir string) string {
	column, ok := userSortColumns[strings.ToLower(strings.TrimSpace(sortBy))]
	if !ok {
		column = "created_at"
	}
	dir := "DESC"
	switch strings.ToLower(strings.TrimSpace(sortDir)) {
	case "asc", "ascending":
		dir = "ASC"
	case "desc", "descending":
		dir = "DESC"
	}
	return " ORDER BY " + column + " " + dir + ", id ASC"
}

func appendUserListFilters(params ListParams, args []any) (string, []any) {
	var clauses []string

	switch strings.ToLower(strings.TrimSpace(params.DeletedStatus)) {
	case "deleted":
		clauses = append(clauses, "deleted_at IS NOT NULL")
	case "all":
		// no deleted_at filter
	default:
		clauses = append(clauses, "deleted_at IS NULL")
	}

	search := strings.TrimSpace(params.Query)
	if search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(email ILIKE $%d OR username ILIKE $%d)", len(args), len(args)))
	}

	role := strings.ToLower(strings.TrimSpace(params.Role))
	if role != "" {
		args = append(args, role)
		clauses = append(clauses, fmt.Sprintf("LOWER(role) = $%d", len(args)))
	}

	provider := strings.ToLower(strings.TrimSpace(params.AuthProvider))
	if provider != "" {
		args = append(args, provider)
		clauses = append(clauses, fmt.Sprintf("LOWER(auth_provider) = $%d", len(args)))
	}

	if params.CreatedFrom != nil {
		args = append(args, *params.CreatedFrom)
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if params.CreatedTo != nil {
		args = append(args, *params.CreatedTo)
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if params.UpdatedFrom != nil {
		args = append(args, *params.UpdatedFrom)
		clauses = append(clauses, fmt.Sprintf("updated_at >= $%d", len(args)))
	}
	if params.UpdatedTo != nil {
		args = append(args, *params.UpdatedTo)
		clauses = append(clauses, fmt.Sprintf("updated_at <= $%d", len(args)))
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}
	return where, args
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

	where, args := appendUserListFilters(params, nil)
	args = append(args, limit, offset)
	limitIdx := len(args) - 1
	offsetIdx := len(args)

	query := `
        SELECT ` + userSelectPublic + `
        FROM users` + where + orderByClause(params.SortBy, params.SortDir) + `
        LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx)

	rows, err := r.DB.Query(ctx, query, args...)
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

func (r *UserRepository) Count(ctx context.Context, params ListParams) (int, error) {
	where, args := appendUserListFilters(params, nil)
	var count int
	err := r.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users`+where, args...).Scan(&count)
	return count, err
}

type UserProfileUpdate struct {
	Username string
	Email    string
	Role     string
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID string, input UserProfileUpdate) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET username = $2,
			    email = $3,
			    role = $4,
			    updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`,
		userID,
		input.Username,
		input.Email,
		input.Role,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) UpdateUsername(ctx context.Context, userID, username string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET username = $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`,
		userID,
		username,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET password = $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
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

func (r *UserRepository) SetEmailVerified(ctx context.Context, userID string, verifiedAt time.Time) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET email_verified_at = $2, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`,
		userID,
		verifiedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, userID string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) Restore(ctx context.Context, userID string) error {
	tag, err := r.DB.Exec(
		ctx,
		`
			UPDATE users
			SET deleted_at = NULL, updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NOT NULL
		`,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) ListAll(ctx context.Context, params ListParams) ([]models.User, error) {
	where, args := appendUserListFilters(params, nil)
	query := `
        SELECT ` + userSelectPublic + `
        FROM users` + where + orderByClause(params.SortBy, params.SortDir)

	rows, err := r.DB.Query(ctx, query, args...)
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
