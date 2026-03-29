package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLRBACService_CreateOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		org := &Organization{
			Name:        "Test Org",
			Slug:        "test-org",
			Description: "Test Description",
		}

		mock.ExpectExec("INSERT INTO organizations").
			WithArgs(sqlmock.AnyArg(), org.Name, org.Slug, org.Description, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateOrganization(context.Background(), org)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, org.ID)
	})

	t.Run("DB Error", func(t *testing.T) {
		org := &Organization{
			Name: "Test Org",
			Slug: "test-org",
		}

		mock.ExpectExec("INSERT INTO organizations").
			WillReturnError(errors.New("db error"))

		err := service.CreateOrganization(context.Background(), org)
		assert.Error(t, err)
	})
}

func TestPostgreSQLRBACService_GetUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "first_name", "last_name", "is_active", "is_system_admin", "metadata", "created_at", "updated_at", "last_login_at"}).
			AddRow(userID, "test@example.com", "testuser", "hash", "Test", "User", true, false, []byte("{}"), time.Now(), time.Now(), nil)

		mock.ExpectQuery("SELECT .* FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnRows(rows)

		user, err := service.GetUser(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("Not Found", func(t *testing.T) {
		userID := uuid.New()
		mock.ExpectQuery("SELECT .* FROM users WHERE id = \\$1").
			WithArgs(userID).
			WillReturnError(errors.New("sql: no rows in result set"))

		user, err := service.GetUser(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestPostgreSQLRBACService_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		user := &User{
			Email:        "new@example.com",
			Username:     "newuser",
			PasswordHash: "password",
			FirstName:    "New",
			LastName:     "User",
		}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(sqlmock.AnyArg(), user.Email, user.Username, sqlmock.AnyArg(), user.FirstName, user.LastName, false, false, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateUser(context.Background(), user)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
	})
}

func TestPostgreSQLRBACService_GetOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "description", "settings", "created_at", "updated_at"}).
			AddRow(orgID, "Test Org", "test-org", "Description", []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM organizations WHERE id = \\$1").
			WithArgs(orgID).
			WillReturnRows(rows)

		org, err := service.GetOrganization(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Equal(t, orgID, org.ID)
	})
}

func TestPostgreSQLRBACService_UpdateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		user := &User{
			ID:        uuid.New(),
			Email:     "update@example.com",
			FirstName: "Updated",
			LastName:  "User",
			IsActive:  true,
		}

		mock.ExpectExec("UPDATE users SET").
			WithArgs(user.ID, user.Email, user.Username, user.FirstName, user.LastName, user.IsActive, user.IsSystemAdmin, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateUser(context.Background(), user)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_DeactivateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()

		mock.ExpectExec("UPDATE users SET is_active = false").
			WithArgs(userID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.DeactivateUser(context.Background(), userID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_ListUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "first_name", "last_name", "is_active", "is_system_admin", "metadata", "created_at", "updated_at", "last_login_at"}).
			AddRow(uuid.New(), "user1@example.com", "user1", "hash", "User", "One", true, false, []byte("{}"), time.Now(), time.Now(), nil).
			AddRow(uuid.New(), "user2@example.com", "user2", "hash", "User", "Two", true, false, []byte("{}"), time.Now(), time.Now(), nil)

		mock.ExpectQuery("SELECT .* FROM users .* ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 0).
			WillReturnRows(rows)

		users, err := service.ListUsers(context.Background(), nil, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, users, 2)
	})
}

func TestPostgreSQLRBACService_GetOrganizationBySlug(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		slug := "test-org"
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "description", "settings", "created_at", "updated_at"}).
			AddRow(orgID, "Test Org", slug, "Description", []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM organizations WHERE slug = \\$1").
			WithArgs(slug).
			WillReturnRows(rows)

		org, err := service.GetOrganizationBySlug(context.Background(), slug)
		assert.NoError(t, err)
		assert.Equal(t, slug, org.Slug)
	})
}

func TestPostgreSQLRBACService_UpdateOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		org := &Organization{
			ID:   uuid.New(),
			Name: "Updated Org",
			Slug: "updated-org",
		}

		mock.ExpectExec("UPDATE organizations SET").
			WithArgs(org.ID, org.Name, org.Slug, org.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateOrganization(context.Background(), org)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_DeleteOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()

		mock.ExpectExec("DELETE FROM organizations WHERE id = \\$1").
			WithArgs(orgID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.DeleteOrganization(context.Background(), orgID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_ListOrganizations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "description", "settings", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Org 1", "org-1", "Desc 1", []byte("{}"), time.Now(), time.Now()).
			AddRow(uuid.New(), "Org 2", "org-2", "Desc 2", []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM organizations ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 0).
			WillReturnRows(rows)

		orgs, err := service.ListOrganizations(context.Background(), 10, 0)
		assert.NoError(t, err)
		assert.Len(t, orgs, 2)
	})
}

func TestPostgreSQLRBACService_CreateRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		role := &Role{
			OrganizationID: uuid.New(),
			Name:           "New Role",
			Description:    "Description",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO roles").
			WithArgs(sqlmock.AnyArg(), role.OrganizationID, role.Name, role.Description, false, false, []byte("{}"), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := service.CreateRole(context.Background(), role)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, role.ID)
	})
}

func TestPostgreSQLRBACService_GetRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		roleID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "is_system_role", "is_template", "metadata", "created_at", "updated_at"}).
			AddRow(roleID, uuid.New(), "Test Role", "Description", false, false, []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM roles WHERE id = \\$1").
			WithArgs(roleID).
			WillReturnRows(rows)

		permRows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "resource_type_id", "action", "created_at"})
		mock.ExpectQuery("SELECT .* FROM permissions p JOIN role_permissions rp").
			WithArgs(roleID).
			WillReturnRows(permRows)

		role, err := service.GetRole(context.Background(), roleID)
		assert.NoError(t, err)
		assert.Equal(t, roleID, role.ID)
	})
}

func TestPostgreSQLRBACService_UpdateRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		role := &Role{
			ID:          uuid.New(),
			Name:        "Updated Role",
			Description: "Updated Description",
		}

		mock.ExpectExec("UPDATE roles SET").
			WithArgs(role.ID, role.Name, role.Description, false, []byte("{}"), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateRole(context.Background(), role)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_DeleteRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		roleID := uuid.New()

		mock.ExpectExec("DELETE FROM roles WHERE id = \\$1").
			WithArgs(roleID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.DeleteRole(context.Background(), roleID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_ListRoles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "is_system_role", "is_template", "metadata", "created_at", "updated_at"}).
			AddRow(uuid.New(), orgID, "Role 1", "Desc 1", false, false, []byte("{}"), time.Now(), time.Now()).
			AddRow(uuid.New(), orgID, "Role 2", "Desc 2", false, false, []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM roles WHERE organization_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
			WithArgs(orgID, 10, 0).
			WillReturnRows(rows)

		roles, err := service.ListRoles(context.Background(), orgID, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, roles, 2)
	})
}

func TestPostgreSQLRBACService_CreatePermission(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		permission := &Permission{
			OrganizationID: uuid.New(),
			Name:           "user:create",
			Description:    "Create users",
			Action:         "create",
		}

		mock.ExpectExec("INSERT INTO permissions").
			WithArgs(sqlmock.AnyArg(), permission.OrganizationID, permission.Name, permission.Description, sqlmock.AnyArg(), permission.Action, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreatePermission(context.Background(), permission)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, permission.ID)
	})
}

func TestPostgreSQLRBACService_GetPermission(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		permID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "resource_type_id", "action", "created_at"}).
			AddRow(permID, uuid.New(), "user:create", "Create users", nil, "create", time.Now())

		mock.ExpectQuery("SELECT .* FROM permissions WHERE id = \\$1").
			WithArgs(permID).
			WillReturnRows(rows)

		perm, err := service.GetPermission(context.Background(), permID)
		assert.NoError(t, err)
		assert.Equal(t, permID, perm.ID)
	})
}

func TestPostgreSQLRBACService_ListPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "resource_type_id", "action", "created_at"}).
			AddRow(uuid.New(), orgID, "perm1", "desc1", nil, "read", time.Now()).
			AddRow(uuid.New(), orgID, "perm2", "desc2", nil, "write", time.Now())

		mock.ExpectQuery("SELECT .* FROM permissions WHERE organization_id = \\$1").
			WithArgs(orgID).
			WillReturnRows(rows)

		perms, err := service.ListPermissions(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Len(t, perms, 2)
	})
}

func TestPostgreSQLRBACService_AssignPermissionToRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		roleID := uuid.New()
		permID := uuid.New()

		mock.ExpectExec("INSERT INTO role_permissions").
			WithArgs(sqlmock.AnyArg(), roleID, permID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.AssignPermissionToRole(context.Background(), roleID, permID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_RemovePermissionFromRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		roleID := uuid.New()
		permID := uuid.New()

		mock.ExpectExec("DELETE FROM role_permissions").
			WithArgs(roleID, permID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.RemovePermissionFromRole(context.Background(), roleID, permID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_AssignRoleToUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		assignment := &UserRoleAssignment{
			UserID:         uuid.New(),
			RoleID:         uuid.New(),
			OrganizationID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO user_role_assignments").
			WithArgs(sqlmock.AnyArg(), assignment.UserID, assignment.RoleID, assignment.OrganizationID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.AssignRoleToUser(context.Background(), assignment)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_RemoveRoleFromUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		roleID := uuid.New()
		orgID := uuid.New()

		mock.ExpectExec("UPDATE user_role_assignments SET is_active = false").
			WithArgs(userID, roleID, orgID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.RemoveRoleFromUser(context.Background(), userID, roleID, orgID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_CreateResourceType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resourceType := &ResourceType{
			OrganizationID:   uuid.New(),
			Name:             "Test Resource Type",
			Description:      "Description",
			HierarchyEnabled: true,
		}

		mock.ExpectExec("INSERT INTO resource_types").
			WithArgs(sqlmock.AnyArg(), resourceType.OrganizationID, resourceType.Name, resourceType.Description, resourceType.HierarchyEnabled, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateResourceType(context.Background(), resourceType)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resourceType.ID)
	})
}

func TestPostgreSQLRBACService_GetResourceType(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resourceTypeID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "hierarchy_enabled", "created_at"}).
			AddRow(resourceTypeID, uuid.New(), "Test Resource Type", "Description", true, time.Now())

		mock.ExpectQuery("SELECT .* FROM resource_types WHERE id = \\$1").
			WithArgs(resourceTypeID).
			WillReturnRows(rows)

		rt, err := service.GetResourceType(context.Background(), resourceTypeID)
		assert.NoError(t, err)
		assert.Equal(t, resourceTypeID, rt.ID)
	})
}

func TestPostgreSQLRBACService_ListResourceTypes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "hierarchy_enabled", "created_at"}).
			AddRow(uuid.New(), orgID, "RT1", "Desc 1", true, time.Now()).
			AddRow(uuid.New(), orgID, "RT2", "Desc 2", false, time.Now())

		mock.ExpectQuery("SELECT .* FROM resource_types WHERE organization_id = \\$1 ORDER BY name").
			WithArgs(orgID).
			WillReturnRows(rows)

		rts, err := service.ListResourceTypes(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Len(t, rts, 2)
	})
}

func TestPostgreSQLRBACService_CreateResource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resource := &Resource{
			OrganizationID: uuid.New(),
			ResourceTypeID: uuid.New(),
			Name:           "Test Resource",
			Description:    "Description",
			Metadata:       map[string]interface{}{"key": "value"},
		}

		metadataJSON, _ := json.Marshal(resource.Metadata)

		mock.ExpectExec("INSERT INTO resources").
			WithArgs(sqlmock.AnyArg(), resource.OrganizationID, resource.ResourceTypeID, resource.Name, resource.Description, sqlmock.AnyArg(), metadataJSON, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateResource(context.Background(), resource)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resource.ID)
	})
}

func TestPostgreSQLRBACService_GetResource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resourceID := uuid.New()
		parentPath := "root"
		rows := sqlmock.NewRows([]string{"id", "organization_id", "resource_type_id", "name", "description", "parent_path", "metadata", "created_at", "updated_at"}).
			AddRow(resourceID, uuid.New(), uuid.New(), "Test Resource", "Description", parentPath, []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM resources WHERE id = \\$1").
			WithArgs(resourceID).
			WillReturnRows(rows)

		resource, err := service.GetResource(context.Background(), resourceID)
		assert.NoError(t, err)
		if assert.NotNil(t, resource) {
			assert.Equal(t, resourceID, resource.ID)
			assert.Equal(t, parentPath, resource.ParentPath)
		}
	})
}

func TestPostgreSQLRBACService_UpdateResource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resource := &Resource{
			ID:          uuid.New(),
			Name:        "Updated Resource",
			Description: "Updated Description",
			Metadata:    map[string]interface{}{"key": "value"},
		}

		metadataJSON, _ := json.Marshal(resource.Metadata)

		mock.ExpectExec("UPDATE resources SET").
			WithArgs(resource.ID, resource.Name, resource.Description, sqlmock.AnyArg(), metadataJSON, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateResource(context.Background(), resource)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_DeleteResource(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		resourceID := uuid.New()

		mock.ExpectExec("DELETE FROM resources WHERE id = \\$1").
			WithArgs(resourceID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.DeleteResource(context.Background(), resourceID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_ListResources(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		parentPath := "root"
		rows := sqlmock.NewRows([]string{"id", "organization_id", "resource_type_id", "name", "description", "parent_path", "metadata", "created_at", "updated_at"}).
			AddRow(uuid.New(), orgID, uuid.New(), "R1", "Desc 1", parentPath, []byte("{}"), time.Now(), time.Now()).
			AddRow(uuid.New(), orgID, uuid.New(), "R2", "Desc 2", parentPath, []byte("{}"), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM resources WHERE organization_id = \\$1 ORDER BY parent_path, name").
			WithArgs(orgID).
			WillReturnRows(rows)

		resources, err := service.ListResources(context.Background(), orgID, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, resources, 2)
	})
}

func TestPostgreSQLRBACService_GrantEmergencyAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		access := &EmergencyAccess{
			UserID:             uuid.New(),
			OrganizationID:     uuid.New(),
			GrantedPermissions: []string{"admin:access"},
			Reason:             "Emergency",
			ValidFrom:          time.Now(),
			ValidUntil:         time.Now().Add(1 * time.Hour),
			IsActive:           true,
		}

		mock.ExpectExec("INSERT INTO emergency_access").
			WithArgs(sqlmock.AnyArg(), access.UserID, access.OrganizationID, sqlmock.AnyArg(), access.Reason, sqlmock.AnyArg(), sqlmock.AnyArg(), access.ValidFrom, access.ValidUntil, access.IsActive, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.GrantEmergencyAccess(context.Background(), access)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, access.ID)
	})
}

func TestPostgreSQLRBACService_RevokeEmergencyAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		accessID := uuid.New()
		revokedBy := uuid.New()
		reason := "Done"

		mock.ExpectExec("UPDATE emergency_access SET is_active = false").
			WithArgs(accessID, sqlmock.AnyArg(), revokedBy, reason).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.RevokeEmergencyAccess(context.Background(), accessID, revokedBy, reason)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_GetActiveEmergencyAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "user_id", "organization_id", "granted_permissions", "reason", "granted_by", "approved_by", "valid_from", "valid_until", "is_active", "revoked_at", "revoked_by", "revoke_reason", "created_at"}).
			AddRow(uuid.New(), userID, orgID, []byte("{admin:access}"), "Reason", nil, nil, time.Now(), time.Now().Add(time.Hour), true, nil, nil, nil, time.Now())

		mock.ExpectQuery("SELECT .* FROM emergency_access WHERE user_id = \\$1 AND organization_id = \\$2 AND is_active = true").
			WithArgs(userID, orgID).
			WillReturnRows(rows)

		accesses, err := service.GetActiveEmergencyAccess(context.Background(), userID, orgID)
		assert.NoError(t, err)
		assert.Len(t, accesses, 1)
	})
}

func TestPostgreSQLRBACService_CreateEmergencyAccessRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		req := &EmergencyAccessRequest{
			UserID:               uuid.New(),
			OrganizationID:       uuid.New(),
			RequestedPermissions: []string{"admin:access"},
			Reason:               "Reason",
			UrgencyLevel:         EmergencyUrgencyHigh,
			Status:               EmergencyAccessRequestStatusPending,
		}

		mock.ExpectExec("INSERT INTO emergency_access_requests").
			WithArgs(sqlmock.AnyArg(), req.UserID, req.OrganizationID, sqlmock.AnyArg(), req.Reason, req.UrgencyLevel, req.RequestedDuration, req.BreakGlass, req.RequiredApprovals, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateEmergencyAccessRequest(context.Background(), req)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, req.ID)
	})
}

func TestPostgreSQLRBACService_GetBreakGlassConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		rows := sqlmock.NewRows([]string{"enabled", "auto_approval_urgency", "max_duration", "required_permissions", "approval_requirements", "auto_revocation_minutes", "notification_channels", "escalation_rules"}).
			AddRow(true, "high", 3600, []byte("{}"), []byte("{}"), 60, []byte("{}"), []byte("[]"))

		mock.ExpectQuery("SELECT .* FROM break_glass_config WHERE organization_id = \\$1").
			WithArgs(orgID).
			WillReturnRows(rows)

		config, err := service.GetBreakGlassConfig(context.Background(), orgID)
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.True(t, config.Enabled)
	})
}

func TestPostgreSQLRBACService_UpdateBreakGlassConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		config := &BreakGlassConfig{
			Enabled: true,
		}

		mock.ExpectExec("INSERT INTO break_glass_config .* ON CONFLICT .* DO UPDATE SET").
			WithArgs(orgID, config.Enabled, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateBreakGlassConfig(context.Background(), orgID, config)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_LogAuditEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		event := &AuditEvent{
			OrganizationID: uuid.New(),
			UserID:         func() *uuid.UUID { u := uuid.New(); return &u }(),
			Action:         "test_action",
			ResourceType:   "test_resource",
			ResourceID:     func() *uuid.UUID { u := uuid.New(); return &u }(),
			PermissionName: "test_permission",
			Success:        true,
			Reason:         "test_reason",
			IPAddress:      "127.0.0.1",
			UserAgent:      "test_agent",
			SessionID:      "test_session",
			Metadata:       map[string]interface{}{"key": "value"},
		}

		metadataJSON, _ := json.Marshal(event.Metadata)

		mock.ExpectExec("INSERT INTO rbac_audit").
			WithArgs(
				sqlmock.AnyArg(), // ID
				event.OrganizationID,
				event.UserID,
				event.Action,
				event.ResourceType,
				event.ResourceID,
				event.PermissionName,
				event.Success,
				event.Reason,
				event.IPAddress,
				event.UserAgent,
				event.SessionID,
				metadataJSON,
				sqlmock.AnyArg(), // CreatedAt
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.LogAuditEvent(context.Background(), event)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_GetAuditLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()
		userID := uuid.New()
		filters := AuditFilters{
			UserID: &userID,
			Limit:  10,
			Offset: 0,
		}

		metadata := map[string]interface{}{"key": "value"}
		metadataJSON, _ := json.Marshal(metadata)

		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "action", "resource_type", "resource_id", "permission_name",
			"success", "reason", "ip_address", "user_agent", "session_id", "metadata", "created_at",
		}).AddRow(
			uuid.New(), orgID, userID, "test_action", "test_resource", uuid.New(), "test_permission",
			true, "test_reason", "127.0.0.1", "test_agent", "test_session", metadataJSON, time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM rbac_audit").
			WithArgs(orgID, userID, filters.Limit).
			WillReturnRows(rows)

		events, err := service.GetAuditLog(context.Background(), orgID, filters)
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "test_action", events[0].Action)
	})
}

func TestPostgreSQLRBACService_CreateScheduledRoleAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		assignment := &ScheduledRoleAssignment{
			UserID:              uuid.New(),
			RoleID:              uuid.New(),
			OrganizationID:      uuid.New(),
			ScheduledActivation: time.Now().Add(1 * time.Hour),
			ScheduledExpiration: func() *time.Time { t := time.Now().Add(2 * time.Hour); return &t }(),
			AssignmentReason:    "Temporary access",
			AssignedBy:          func() *uuid.UUID { u := uuid.New(); return &u }(),
			Metadata:            map[string]interface{}{"key": "value"},
		}

		metadataJSON, _ := json.Marshal(assignment.Metadata)

		mock.ExpectExec("INSERT INTO scheduled_role_assignments").
			WithArgs(
				sqlmock.AnyArg(), // ID
				assignment.UserID,
				assignment.RoleID,
				assignment.OrganizationID,
				assignment.ResourceScope,
				assignment.ScheduledActivation,
				assignment.ScheduledExpiration,
				assignment.AssignedBy,
				assignment.AssignmentReason,
				assignment.RecurrencePattern,
				metadataJSON,
				sqlmock.AnyArg(), // CreatedAt
				sqlmock.AnyArg(), // UpdatedAt
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateScheduledRoleAssignment(context.Background(), assignment)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, assignment.ID)
	})
}

func TestPostgreSQLRBACService_UpdateScheduledRoleAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		assignment := &ScheduledRoleAssignment{
			ID:                  uuid.New(),
			ScheduledActivation: time.Now().Add(1 * time.Hour),
			AssignmentReason:    "Updated reason",
			Metadata:            map[string]interface{}{"key": "value"},
		}

		metadataJSON, _ := json.Marshal(assignment.Metadata)

		mock.ExpectExec("UPDATE scheduled_role_assignments").
			WithArgs(
				assignment.ID,
				assignment.ScheduledActivation,
				assignment.ScheduledExpiration,
				assignment.AssignmentReason,
				assignment.RecurrencePattern,
				metadataJSON,
				sqlmock.AnyArg(), // UpdatedAt
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.UpdateScheduledRoleAssignment(context.Background(), assignment)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_DeleteScheduledRoleAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		assignmentID := uuid.New()

		mock.ExpectQuery("SELECT id, user_id, role_id, organization_id, scheduled_activation, is_processed FROM scheduled_role_assignments WHERE id = \\$1").
			WithArgs(assignmentID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "role_id", "organization_id", "scheduled_activation", "is_processed"}).
				AddRow(assignmentID, uuid.New(), uuid.New(), uuid.New(), time.Now(), false))

		mock.ExpectExec("DELETE FROM scheduled_role_assignments WHERE id = \\$1").
			WithArgs(assignmentID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.DeleteScheduledRoleAssignment(context.Background(), assignmentID)
		assert.NoError(t, err)
	})
}

func TestPostgreSQLRBACService_GetScheduledRoleAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "role_id", "organization_id", "resource_scope", "scheduled_activation",
			"scheduled_expiration", "assigned_by", "assignment_reason", "notification_sent",
			"is_processed", "processed_at", "processing_error", "recurrence_pattern", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), userID, uuid.New(), orgID, nil, time.Now().Add(time.Hour),
			nil, uuid.New(), "reason", false,
			false, nil, nil, nil, []byte("{}"),
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM scheduled_role_assignments WHERE user_id = \\$1 AND organization_id = \\$2").
			WithArgs(userID, orgID).
			WillReturnRows(rows)

		assignments, err := service.GetScheduledRoleAssignments(context.Background(), userID, orgID)
		assert.NoError(t, err)
		assert.Len(t, assignments, 1)
	})
}

func TestPostgreSQLRBACService_ListPendingActivations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "role_id", "organization_id", "resource_scope", "scheduled_activation",
			"scheduled_expiration", "assigned_by", "assignment_reason", "notification_sent",
			"is_processed", "processed_at", "processing_error", "recurrence_pattern", "metadata",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), uuid.New(), orgID, nil, time.Now().Add(time.Hour),
			nil, uuid.New(), "reason", false,
			false, nil, nil, nil, []byte("{}"),
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT sra.* FROM scheduled_role_assignments sra .* WHERE sra.organization_id = \\$1 AND NOT sra.is_processed").
			WithArgs(orgID).
			WillReturnRows(rows)

		assignments, err := service.ListPendingActivations(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Len(t, assignments, 1)
	})
}

func TestPostgreSQLRBACService_ListExpiredAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "role_id", "organization_id", "resource_scope", "valid_from", "valid_until",
			"assigned_by", "assignment_reason", "is_active", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), uuid.New(), orgID, nil, time.Now(), time.Now().Add(-time.Hour),
			uuid.New(), "reason", true, time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT .* FROM user_role_assignments WHERE organization_id = \\$1 AND is_active = true AND valid_until IS NOT NULL AND valid_until < CURRENT_TIMESTAMP").
			WithArgs(orgID).
			WillReturnRows(rows)

		assignments, err := service.ListExpiredAssignments(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Len(t, assignments, 1)
	})
}

func TestPostgreSQLRBACService_GetTimeBasedAccessStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		orgID := uuid.New()

		// 1. Pending activations
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM scheduled_role_assignments WHERE organization_id = \\$1 AND NOT is_processed").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		// 2. Active assignments
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_role_assignments WHERE organization_id = \\$1 AND is_active = true AND \\(valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP\\)").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// 3. Expired assignments
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM user_role_assignments WHERE organization_id = \\$1 AND is_active = true AND valid_until IS NOT NULL AND valid_until < CURRENT_TIMESTAMP").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 4. Scheduled for next 24h
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM scheduled_role_assignments WHERE organization_id = \\$1 AND NOT is_processed AND scheduled_activation <= CURRENT_TIMESTAMP \\+ INTERVAL '24 hours' AND scheduled_activation > CURRENT_TIMESTAMP").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// 5. Processing errors
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM scheduled_role_assignments WHERE organization_id = \\$1 AND processing_error IS NOT NULL").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 6. Last processed at
		mock.ExpectQuery("SELECT COALESCE\\(MAX\\(processed_at\\), '1970-01-01'::timestamp\\) FROM scheduled_role_assignments WHERE organization_id = \\$1 AND is_processed = true").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Now()))

		status, err := service.GetTimeBasedAccessStatus(context.Background(), orgID)
		assert.NoError(t, err)
		assert.Equal(t, 5, status.ActiveAssignments)
		assert.Equal(t, 2, status.PendingActivations)
	})
}

func TestPostgreSQLRBACService_ProcessScheduledActivations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		// Mock the stored procedure call
		mock.ExpectQuery("SELECT process_scheduled_activations()").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// Mock the audit log insertion
		mock.ExpectExec("INSERT INTO rbac_audit").
			WithArgs(
				sqlmock.AnyArg(), // id
				sqlmock.AnyArg(), // organization_id (nil/zero)
				sqlmock.AnyArg(), // user_id (nil/zero)
				"process_scheduled_activations",
				"system",
				sqlmock.AnyArg(), // resource_id (nil)
				sqlmock.AnyArg(), // permission_name (nil)
				true,             // success
				"Batch processing of scheduled activations completed",
				sqlmock.AnyArg(), // ip_address
				sqlmock.AnyArg(), // user_agent
				sqlmock.AnyArg(), // session_id
				sqlmock.AnyArg(), // metadata
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.ProcessScheduledActivations(context.Background())
		assert.NoError(t, err)
	})

	t.Run("DB_Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT process_scheduled_activations()").
			WillReturnError(errors.New("db error"))

		err := service.ProcessScheduledActivations(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to process scheduled activations")
	})
}

func TestPostgreSQLRBACService_CleanupExpiredAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		// Mock the stored procedure call
		mock.ExpectQuery("SELECT cleanup_expired_assignments()").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		// Mock the audit log insertion
		mock.ExpectExec("INSERT INTO rbac_audit").
			WithArgs(
				sqlmock.AnyArg(), // id
				sqlmock.AnyArg(), // organization_id (nil/zero)
				sqlmock.AnyArg(), // user_id (nil/zero)
				"cleanup_expired_assignments",
				"system",
				sqlmock.AnyArg(), // resource_id (nil)
				sqlmock.AnyArg(), // permission_name (nil)
				true,             // success
				"Batch cleanup of expired assignments completed",
				sqlmock.AnyArg(), // ip_address
				sqlmock.AnyArg(), // user_agent
				sqlmock.AnyArg(), // session_id
				sqlmock.AnyArg(), // metadata
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CleanupExpiredAssignments(context.Background())
		assert.NoError(t, err)
	})

	t.Run("DB_Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT cleanup_expired_assignments()").
			WillReturnError(errors.New("db error"))

		err := service.CleanupExpiredAssignments(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cleanup expired assignments")
	})
}

func TestPostgreSQLRBACService_GetUserByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "first_name", "last_name", "is_active", "is_system_admin", "metadata", "created_at", "updated_at", "last_login_at"}).
			AddRow(uuid.New(), email, "username", "hash", "First", "Last", true, false, []byte("{}"), time.Now(), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnRows(rows)

		user, err := service.GetUserByEmail(context.Background(), email)
		assert.NoError(t, err)
		if assert.NotNil(t, user) {
			assert.Equal(t, email, user.Email)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		email := "nonexistent@example.com"
		mock.ExpectQuery("SELECT .* FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		user, err := service.GetUserByEmail(context.Background(), email)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestPostgreSQLRBACService_GetUserByUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		username := "testuser"
		rows := sqlmock.NewRows([]string{"id", "email", "username", "password_hash", "first_name", "last_name", "is_active", "is_system_admin", "metadata", "created_at", "updated_at", "last_login_at"}).
			AddRow(uuid.New(), "test@example.com", username, "hash", "First", "Last", true, false, []byte("{}"), time.Now(), time.Now(), time.Now())

		mock.ExpectQuery("SELECT .* FROM users WHERE username = \\$1").
			WithArgs(username).
			WillReturnRows(rows)

		user, err := service.GetUserByUsername(context.Background(), username)
		assert.NoError(t, err)
		if assert.NotNil(t, user) {
			assert.Equal(t, username, user.Username)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		username := "nonexistent"
		mock.ExpectQuery("SELECT .* FROM users WHERE username = \\$1").
			WithArgs(username).
			WillReturnError(sql.ErrNoRows)

		user, err := service.GetUserByUsername(context.Background(), username)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestPostgreSQLRBACService_GetUserRoles(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	service := NewPostgreSQLRBACService(sqlxDB)

	t.Run("Success", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "description", "is_system_role", "is_template", "metadata", "created_at", "updated_at"}).
			AddRow(uuid.New(), orgID, "Role 1", "Desc 1", false, false, "{}", time.Now(), time.Now()).
			AddRow(uuid.New(), orgID, "Role 2", "Desc 2", false, false, "{}", time.Now(), time.Now())

		mock.ExpectQuery("SELECT r.* FROM roles r JOIN user_role_assignments ura ON r.id = ura.role_id").
			WithArgs(userID, orgID).
			WillReturnRows(rows)

		roles, err := service.GetUserRoles(context.Background(), userID, orgID)
		assert.NoError(t, err)
		assert.Len(t, roles, 2)
	})

	t.Run("DB Error", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()

		mock.ExpectQuery("SELECT r.* FROM roles r JOIN user_role_assignments ura ON r.id = ura.role_id").
			WithArgs(userID, orgID).
			WillReturnError(errors.New("db error"))

		roles, err := service.GetUserRoles(context.Background(), userID, orgID)
		assert.Error(t, err)
		assert.Nil(t, roles)
	})
}
