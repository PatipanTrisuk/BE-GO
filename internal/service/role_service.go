package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"go-be/internal/authz"
	"go-be/internal/models"
	"go-be/internal/repository"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleExists        = errors.New("role already exists")
	ErrRoleSystem        = errors.New("system role cannot be modified that way")
	ErrRoleInUse         = errors.New("role is assigned to users")
	ErrInvalidPermission = errors.New("invalid permission")
)

var roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

type RoleService struct {
	RoleRepo *repository.RoleRepository
	Store    *authz.Store
}

func NewRoleService(roleRepo *repository.RoleRepository, store *authz.Store) *RoleService {
	if store == nil {
		store = authz.GetStore()
	}
	return &RoleService{RoleRepo: roleRepo, Store: store}
}

func (s *RoleService) ReloadStore(ctx context.Context) error {
	m, err := s.RoleRepo.ListRolePermissionMap(ctx)
	if err != nil {
		return err
	}
	if len(m) == 0 {
		m = authz.DefaultRolePermissions()
	}
	s.Store.Replace(m)
	return nil
}

func (s *RoleService) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	return s.RoleRepo.ListPermissions(ctx)
}

func (s *RoleService) ListRoles(ctx context.Context) ([]models.Role, error) {
	roles, err := s.RoleRepo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	if roles == nil {
		roles = []models.Role{}
	}
	return roles, nil
}

func (s *RoleService) GetRole(ctx context.Context, id string) (*models.Role, error) {
	role, err := s.RoleRepo.FindByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	return role, err
}

type RoleInput struct {
	Name        string
	Description string
	Permissions []string
}

func normalizeRoleInput(input RoleInput, requireName bool) (RoleInput, error) {
	input.Name = authz.NormalizeRole(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Permissions = uniqueNonEmpty(input.Permissions)

	if requireName {
		if !roleNamePattern.MatchString(input.Name) {
			return input, ErrInvalidInput
		}
	}
	if len(input.Description) > 255 {
		return input, ErrInvalidInput
	}
	return input, nil
}

func uniqueNonEmpty(values []string) []string {
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

func (s *RoleService) validatePermissions(ctx context.Context, codes []string) error {
	missing, err := s.RoleRepo.PermissionCodesExist(ctx, codes)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return ErrInvalidPermission
	}
	return nil
}

func (s *RoleService) CreateRole(ctx context.Context, input RoleInput) (*models.Role, error) {
	input, err := normalizeRoleInput(input, true)
	if err != nil {
		return nil, err
	}
	if err := s.validatePermissions(ctx, input.Permissions); err != nil {
		return nil, err
	}

	role, err := s.RoleRepo.Create(ctx, repository.RoleCreateInput{
		Name:        input.Name,
		Description: input.Description,
		Permissions: input.Permissions,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrRoleExists
		}
		return nil, err
	}
	_ = s.ReloadStore(ctx)
	return role, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, id string, input RoleInput) (*models.Role, error) {
	existing, err := s.GetRole(ctx, id)
	if err != nil {
		return nil, err
	}

	input, err = normalizeRoleInput(input, !existing.IsSystem)
	if err != nil {
		return nil, err
	}
	if existing.IsSystem {
		// Keep system role name stable.
		input.Name = existing.Name
	} else if input.Name == "" {
		return nil, ErrInvalidInput
	}

	if err := s.validatePermissions(ctx, input.Permissions); err != nil {
		return nil, err
	}

	role, err := s.RoleRepo.Update(ctx, id, repository.RoleUpdateInput{
		Name:        input.Name,
		Description: input.Description,
		Permissions: input.Permissions,
	}, !existing.IsSystem)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrRoleExists
		}
		return nil, err
	}
	_ = s.ReloadStore(ctx)
	return role, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, id string) error {
	existing, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return ErrRoleSystem
	}

	count, err := s.RoleRepo.CountUsersWithRole(ctx, existing.ID.String())
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleInUse
	}

	if err := s.RoleRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRoleNotFound
		}
		return err
	}
	_ = s.ReloadStore(ctx)
	return nil
}
