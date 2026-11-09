package authz

import (
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"go-be/internal/models"
)

// Roles (built-in names; custom roles may exist in DB)
const (
	RoleUser      = "user"
	RoleModerator = "moderator"
	RoleAdmin     = "admin"
)

// Permissions — keep names stable; frontend mirrors these strings.
const (
	PermUsersRead          = "users.read"
	PermUsersCreate        = "users.create"
	PermUsersUpdate        = "users.update"
	PermUsersDelete        = "users.delete"
	PermUsersRestore       = "users.restore"
	PermUsersKick          = "users.kick"
	PermUsersResetPassword = "users.reset_password"
	PermUsersVerifyEmail   = "users.verify_email"
	PermUsersExport        = "users.export"
	PermAdminStats         = "admin.stats"
	PermRolesRead          = "roles.read"
	PermRolesCreate        = "roles.create"
	PermRolesUpdate        = "roles.update"
	PermRolesDelete        = "roles.delete"
)

var allPermissions = []string{
	PermUsersRead,
	PermUsersCreate,
	PermUsersUpdate,
	PermUsersDelete,
	PermUsersRestore,
	PermUsersKick,
	PermUsersResetPassword,
	PermUsersVerifyEmail,
	PermUsersExport,
	PermAdminStats,
	PermRolesRead,
	PermRolesCreate,
	PermRolesUpdate,
	PermRolesDelete,
}

func DefaultRolePermissions() map[string][]string {
	return map[string][]string{
		RoleUser: {},
		RoleModerator: {
			PermUsersRead,
			PermUsersKick,
			PermUsersVerifyEmail,
			PermAdminStats,
			PermRolesRead,
		},
		RoleAdmin: append([]string(nil), allPermissions...),
	}
}

// Store holds an in-memory Role → Permission map (loaded from DB).
type Store struct {
	mu        sync.RWMutex
	rolePerms map[string][]string
}

func NewStore(rolePerms map[string][]string) *Store {
	s := &Store{}
	s.Replace(rolePerms)
	return s
}

func NewStoreWithDefaults() *Store {
	return NewStore(DefaultRolePermissions())
}

func (s *Store) Replace(rolePerms map[string][]string) {
	cloned := make(map[string][]string, len(rolePerms))
	for role, perms := range rolePerms {
		key := NormalizeRole(role)
		copied := make([]string, len(perms))
		copy(copied, perms)
		cloned[key] = copied
	}
	s.mu.Lock()
	s.rolePerms = cloned
	s.mu.Unlock()
}

func (s *Store) PermissionsForRole(role string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	perms, ok := s.rolePerms[NormalizeRole(role)]
	if !ok {
		return nil
	}
	out := make([]string, len(perms))
	copy(out, perms)
	return out
}

func (s *Store) IsValidRole(role string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.rolePerms[NormalizeRole(role)]
	return ok
}

func (s *Store) ValidRoles() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.rolePerms))
	for role := range s.rolePerms {
		out = append(out, role)
	}
	slices.Sort(out)
	return out
}

func (s *Store) HasPermission(role, permission string) bool {
	return slices.Contains(s.PermissionsForRole(role), permission)
}

func (s *Store) HasAnyPermission(role string, permissions ...string) bool {
	if len(permissions) == 0 {
		return false
	}
	owned := s.PermissionsForRole(role)
	for _, p := range permissions {
		if slices.Contains(owned, p) {
			return true
		}
	}
	return false
}

func (s *Store) HasAllPermissions(role string, permissions ...string) bool {
	if len(permissions) == 0 {
		return true
	}
	owned := s.PermissionsForRole(role)
	for _, p := range permissions {
		if !slices.Contains(owned, p) {
			return false
		}
	}
	return true
}

func (s *Store) CanAccessAdminArea(role string) bool {
	return len(s.PermissionsForRole(role)) > 0
}

var global = NewStoreWithDefaults()

func SetStore(s *Store) {
	if s == nil {
		return
	}
	global = s
}

func GetStore() *Store {
	return global
}

func NormalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func IsValidRole(role string) bool {
	return global.IsValidRole(role)
}

func ValidRoles() []string {
	return global.ValidRoles()
}

func PermissionsForRole(role string) []string {
	return global.PermissionsForRole(role)
}

func HasPermission(role, permission string) bool {
	return global.HasPermission(role, permission)
}

func HasAnyPermission(role string, permissions ...string) bool {
	return global.HasAnyPermission(role, permissions...)
}

func HasAllPermissions(role string, permissions ...string) bool {
	return global.HasAllPermissions(role, permissions...)
}

func CanAccessAdminArea(role string) bool {
	return global.CanAccessAdminArea(role)
}

// AttachPermissions sets user.Permissions from their role for API responses.
func AttachPermissions(user *models.User) {
	if user == nil {
		return
	}
	if user.RoleID != (uuid.UUID{}) {
		user.Permissions = PermissionsForRole(user.RoleID.String())
		return
	}
	user.Permissions = PermissionsForRole(user.Role)
}

func AllPermissionCodes() []string {
	out := make([]string, len(allPermissions))
	copy(out, allPermissions)
	return out
}
