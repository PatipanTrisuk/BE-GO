package authz

import "testing"

func TestAdminHasAllPermissions(t *testing.T) {
	store := NewStoreWithDefaults()
	for _, p := range allPermissions {
		if !store.HasPermission(RoleAdmin, p) {
			t.Fatalf("admin missing permission %s", p)
		}
	}
}

func TestModeratorSubset(t *testing.T) {
	store := NewStoreWithDefaults()
	if !store.HasPermission(RoleModerator, PermUsersRead) {
		t.Fatal("moderator should read users")
	}
	if store.HasPermission(RoleModerator, PermUsersDelete) {
		t.Fatal("moderator should not delete users")
	}
	if !store.CanAccessAdminArea(RoleModerator) {
		t.Fatal("moderator should access admin area")
	}
}

func TestUserHasNoAdminPerms(t *testing.T) {
	store := NewStoreWithDefaults()
	if store.CanAccessAdminArea(RoleUser) {
		t.Fatal("user should not access admin area")
	}
	if store.HasPermission(RoleUser, PermUsersRead) {
		t.Fatal("user should not read users")
	}
}

func TestIsValidRole(t *testing.T) {
	store := NewStoreWithDefaults()
	SetStore(store)
	t.Cleanup(func() { SetStore(NewStoreWithDefaults()) })

	if !IsValidRole("Admin") || !IsValidRole("moderator") || !IsValidRole("user") {
		t.Fatal("expected known roles to be valid")
	}
	if IsValidRole("superuser") {
		t.Fatal("unknown role should be invalid")
	}
}

func TestReplaceCustomRole(t *testing.T) {
	store := NewStoreWithDefaults()
	store.Replace(map[string][]string{
		"editor": {PermUsersRead, PermUsersUpdate},
	})
	if !store.IsValidRole("editor") {
		t.Fatal("custom role should be valid after replace")
	}
	if !store.HasPermission("editor", PermUsersUpdate) {
		t.Fatal("editor should update users")
	}
	if store.IsValidRole("admin") {
		t.Fatal("previous roles should be gone after full replace")
	}
}
