package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-be/internal/models"
)

type RoleRepository struct {
	DB *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{DB: db}
}

type RoleCreateInput struct {
	Name        string
	Description string
	Permissions []string
}

type RoleUpdateInput struct {
	Name        string
	Description string
	Permissions []string
}

func (r *RoleRepository) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, code, description, created_at
		FROM permissions
		ORDER BY code ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Permission
	for rows.Next() {
		var p models.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *RoleRepository) ListRolePermissionMap(ctx context.Context) (map[string][]string, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT r.id, r.name, COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		GROUP BY r.id, r.name
		ORDER BY r.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string][]string)
	for rows.Next() {
		var id uuid.UUID
		var name string
		var codes []string
		if err := rows.Scan(&id, &name, &codes); err != nil {
			return nil, err
		}
		if codes == nil {
			codes = []string{}
		}
		copied := append([]string(nil), codes...)
		out[id.String()] = copied
		out[strings.ToLower(name)] = append([]string(nil), codes...)
	}
	return out, rows.Err()
}

func (r *RoleRepository) ListRoles(ctx context.Context) ([]models.Role, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT
			r.id,
			r.name,
			r.description,
			r.is_system,
			r.created_at,
			r.updated_at,
			COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		GROUP BY r.id
		ORDER BY r.is_system DESC, r.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

func (r *RoleRepository) FindByID(ctx context.Context, id string) (*models.Role, error) {
	row := r.DB.QueryRow(ctx, `
		SELECT
			r.id,
			r.name,
			r.description,
			r.is_system,
			r.created_at,
			r.updated_at,
			COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		WHERE r.id = $1
		GROUP BY r.id
	`, id)

	role, err := scanRole(row)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	row := r.DB.QueryRow(ctx, `
		SELECT
			r.id,
			r.name,
			r.description,
			r.is_system,
			r.created_at,
			r.updated_at,
			COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		WHERE LOWER(r.name) = LOWER($1)
		GROUP BY r.id
	`, name)

	role, err := scanRole(row)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Create(ctx context.Context, input RoleCreateInput) (*models.Role, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO roles (name, description, is_system)
		VALUES ($1, $2, FALSE)
		RETURNING id, created_at, updated_at
	`, input.Name, input.Description).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	if err := replaceRolePermissionsTx(ctx, tx, id, input.Permissions); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id.String())
}

func (r *RoleRepository) Update(ctx context.Context, id string, input RoleUpdateInput, allowRename bool) (*models.Role, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	roleUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, pgx.ErrNoRows
	}

	var updatedAt time.Time
	if allowRename {
		err = tx.QueryRow(ctx, `
			UPDATE roles
			SET name = $2, description = $3, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING updated_at
		`, roleUUID, input.Name, input.Description).Scan(&updatedAt)
	} else {
		err = tx.QueryRow(ctx, `
			UPDATE roles
			SET description = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1
			RETURNING updated_at
		`, roleUUID, input.Description).Scan(&updatedAt)
	}
	if err != nil {
		return nil, err
	}

	if err := replaceRolePermissionsTx(ctx, tx, roleUUID, input.Permissions); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM roles WHERE id = $1 AND is_system = FALSE`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *RoleRepository) CountUsersWithRole(ctx context.Context, roleID string) (int, error) {
	var count int
	err := r.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE role_id = $1 AND deleted_at IS NULL
	`, roleID).Scan(&count)
	return count, err
}

func (r *RoleRepository) ResolveRoleID(ctx context.Context, roleIDOrName string) (uuid.UUID, error) {
	roleIDOrName = strings.TrimSpace(roleIDOrName)
	if roleIDOrName == "" {
		return uuid.Nil, pgx.ErrNoRows
	}
	if id, err := uuid.Parse(roleIDOrName); err == nil {
		role, err := r.FindByID(ctx, id.String())
		if err != nil {
			return uuid.Nil, err
		}
		return role.ID, nil
	}
	role, err := r.FindByName(ctx, roleIDOrName)
	if err != nil {
		return uuid.Nil, err
	}
	return role.ID, nil
}

func (r *RoleRepository) PermissionCodesExist(ctx context.Context, codes []string) ([]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	rows, err := r.DB.Query(ctx, `
		SELECT code FROM permissions WHERE code = ANY($1)
	`, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[string]struct{}, len(codes))
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		found[code] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var missing []string
	for _, code := range codes {
		if _, ok := found[code]; !ok {
			missing = append(missing, code)
		}
	}
	return missing, nil
}

func replaceRolePermissionsTx(ctx context.Context, tx pgx.Tx, roleID uuid.UUID, codes []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	if len(codes) == 0 {
		return nil
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT $1, p.id
		FROM permissions p
		WHERE p.code = ANY($2)
	`, roleID, codes)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(uniqueStrings(codes)) {
		return fmt.Errorf("one or more permissions are invalid")
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

type roleScanner interface {
	Scan(dest ...any) error
}

func scanRole(row roleScanner) (models.Role, error) {
	var role models.Role
	var perms []string
	err := row.Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.IsSystem,
		&role.CreatedAt,
		&role.UpdatedAt,
		&perms,
	)
	if err != nil {
		return role, err
	}
	if perms == nil {
		perms = []string{}
	}
	role.Permissions = perms
	return role, nil
}
