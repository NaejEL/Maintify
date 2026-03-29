package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// PostgreSQLRBACService implements the RBACService interface using PostgreSQL
type PostgreSQLRBACService struct {
	db *sqlx.DB
}

// NewPostgreSQLRBACService creates a new PostgreSQL-based RBAC service
func NewPostgreSQLRBACService(db *sqlx.DB) *PostgreSQLRBACService {
	return &PostgreSQLRBACService{db: db}
}

// Organization management

func (s *PostgreSQLRBACService) CreateOrganization(ctx context.Context, org *Organization) error {
	// Marshal settings to JSON
	var settingsJSON []byte
	var err error
	if org.Settings != nil {
		settingsJSON, err = json.Marshal(org.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
	} else {
		settingsJSON = []byte("{}")
	}

	query := `
		INSERT INTO organizations (id, name, slug, description, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	_, err = s.db.ExecContext(ctx, query, org.ID, org.Name, org.Slug, org.Description, settingsJSON, org.CreatedAt, org.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var org Organization
	var settingsJSON []byte
	query := `SELECT id, name, slug, description, settings, created_at, updated_at FROM organizations WHERE id = $1`

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &settingsJSON, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal settings if present
	if len(settingsJSON) > 0 {
		err = json.Unmarshal(settingsJSON, &org.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &org, nil
}

func (s *PostgreSQLRBACService) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	var settingsJSON []byte
	query := `SELECT id, name, slug, description, settings, created_at, updated_at FROM organizations WHERE slug = $1`

	err := s.db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description, &settingsJSON, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal settings if present
	if len(settingsJSON) > 0 {
		err = json.Unmarshal(settingsJSON, &org.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
	}

	return &org, nil
}

func (s *PostgreSQLRBACService) UpdateOrganization(ctx context.Context, org *Organization) error {
	// Marshal settings to JSON
	var settingsJSON []byte
	var err error
	if org.Settings != nil {
		settingsJSON, err = json.Marshal(org.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
	} else {
		settingsJSON = []byte("{}")
	}

	query := `
		UPDATE organizations 
		SET name = $2, slug = $3, description = $4, settings = $5, updated_at = $6
		WHERE id = $1`

	org.UpdatedAt = time.Now()
	_, err = s.db.ExecContext(ctx, query, org.ID, org.Name, org.Slug, org.Description, settingsJSON, org.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *PostgreSQLRBACService) ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error) {
	var orgs []*Organization
	query := `
		SELECT id, name, slug, description, settings, created_at, updated_at 
		FROM organizations 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2`

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var org Organization
		var settingsJSON []byte

		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description, &settingsJSON, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &org.Settings); err != nil {
				return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
			}
		}
		orgs = append(orgs, &org)
	}

	return orgs, rows.Err()
}

// User management

func (s *PostgreSQLRBACService) CreateUser(ctx context.Context, user *User) error {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Marshal metadata to JSON
	var metadataJSON []byte
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name, is_active, is_system_admin, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	user.ID = uuid.New()
	user.PasswordHash = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err = s.db.ExecContext(ctx, query, user.ID, user.Email, user.Username, user.PasswordHash,
		user.FirstName, user.LastName, user.IsActive, user.IsSystemAdmin, metadataJSON, user.CreatedAt, user.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	var metadataJSON []byte
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, is_active, is_system_admin, 
		       metadata, created_at, updated_at, last_login_at 
		FROM users WHERE id = $1`

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.IsActive, &user.IsSystemAdmin,
		&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &user, nil
}

func (s *PostgreSQLRBACService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	var metadataJSON []byte
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, is_active, is_system_admin, 
		       metadata, created_at, updated_at, last_login_at 
		FROM users WHERE email = $1`

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.IsActive, &user.IsSystemAdmin,
		&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &user, nil
}

func (s *PostgreSQLRBACService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	var metadataJSON []byte
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, is_active, is_system_admin, 
		       metadata, created_at, updated_at, last_login_at 
		FROM users WHERE username = $1`

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.IsActive, &user.IsSystemAdmin,
		&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &user.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &user, nil
}

func (s *PostgreSQLRBACService) UpdateUser(ctx context.Context, user *User) error {
	// Marshal metadata to JSON
	var metadataJSON []byte
	var err error
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		UPDATE users 
		SET email = $2, username = $3, first_name = $4, last_name = $5, is_active = $6, 
		    is_system_admin = $7, metadata = $8, updated_at = $9, last_login_at = $10
		WHERE id = $1`

	user.UpdatedAt = time.Now()
	_, err = s.db.ExecContext(ctx, query, user.ID, user.Email, user.Username, user.FirstName,
		user.LastName, user.IsActive, user.IsSystemAdmin, metadataJSON, user.UpdatedAt, user.LastLoginAt)
	return err
}

func (s *PostgreSQLRBACService) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id, time.Now())
	return err
}

func (s *PostgreSQLRBACService) ListUsers(ctx context.Context, orgID *uuid.UUID, limit, offset int) ([]*User, error) {
	var users []*User
	var query string
	var args []interface{}

	if orgID != nil {
		// Filter users by organization through role assignments
		query = `
			SELECT DISTINCT u.id, u.email, u.username, u.password_hash, u.first_name, u.last_name, 
			       u.is_active, u.is_system_admin, u.metadata, u.created_at, u.updated_at, u.last_login_at
			FROM users u
			JOIN user_role_assignments ura ON u.id = ura.user_id
			WHERE ura.organization_id = $1 AND u.is_active = true
			ORDER BY u.created_at DESC
			LIMIT $2 OFFSET $3`
		args = []interface{}{*orgID, limit, offset}
	} else {
		query = `
			SELECT id, email, username, password_hash, first_name, last_name, is_active, is_system_admin, 
			       metadata, created_at, updated_at, last_login_at
			FROM users 
			WHERE is_active = true
			ORDER BY created_at DESC 
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		var metadataJSON []byte

		err := rows.Scan(
			&user.ID, &user.Email, &user.Username, &user.PasswordHash,
			&user.FirstName, &user.LastName, &user.IsActive, &user.IsSystemAdmin,
			&metadataJSON, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

// Role management

func (s *PostgreSQLRBACService) CreateRole(ctx context.Context, role *Role) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert role
	// Marshal metadata to JSON
	var metadataJSON []byte
	if role.Metadata != nil {
		metadataJSON, err = json.Marshal(role.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO roles (id, organization_id, name, description, is_system_role, is_template, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	role.ID = uuid.New()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	_, err = tx.ExecContext(ctx, query, role.ID, role.OrganizationID, role.Name, role.Description,
		role.IsSystemRole, role.IsTemplate, metadataJSON, role.CreatedAt, role.UpdatedAt)
	if err != nil {
		return err
	}

	// Assign permissions if provided
	if len(role.Permissions) > 0 {
		for _, permission := range role.Permissions {
			err = s.assignPermissionToRoleInTx(ctx, tx, role.ID, permission.ID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *PostgreSQLRBACService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	var role Role
	var metadataJSON []byte
	query := `
		SELECT id, organization_id, name, description, is_system_role, is_template, metadata, created_at, updated_at
		FROM roles WHERE id = $1`

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID, &role.OrganizationID, &role.Name, &role.Description,
		&role.IsSystemRole, &role.IsTemplate, &metadataJSON, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &role.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	// Load permissions
	permQuery := `
		SELECT p.id, p.organization_id, p.name, p.description, p.resource_type_id, p.action, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1`

	err = s.db.SelectContext(ctx, &role.Permissions, permQuery, id)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (s *PostgreSQLRBACService) UpdateRole(ctx context.Context, role *Role) error {
	// Marshal metadata to JSON
	var metadataJSON []byte
	var err error
	if role.Metadata != nil {
		metadataJSON, err = json.Marshal(role.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		UPDATE roles 
		SET name = $2, description = $3, is_template = $4, metadata = $5, updated_at = $6
		WHERE id = $1`

	role.UpdatedAt = time.Now()
	_, err = s.db.ExecContext(ctx, query, role.ID, role.Name, role.Description, role.IsTemplate, metadataJSON, role.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM roles WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *PostgreSQLRBACService) ListRoles(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*Role, error) {
	var roles []*Role
	query := `
		SELECT id, organization_id, name, description, is_system_role, is_template, metadata, created_at, updated_at
		FROM roles 
		WHERE organization_id = $1
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var role Role
		var metadataJSON []byte

		err := rows.Scan(
			&role.ID, &role.OrganizationID, &role.Name, &role.Description,
			&role.IsSystemRole, &role.IsTemplate, &metadataJSON, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &role.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		roles = append(roles, &role)
	}

	return roles, rows.Err()
}

// Permission management

func (s *PostgreSQLRBACService) CreatePermission(ctx context.Context, permission *Permission) error {
	query := `
		INSERT INTO permissions (id, organization_id, name, description, resource_type_id, action, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	permission.ID = uuid.New()
	permission.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query, permission.ID, permission.OrganizationID, permission.Name,
		permission.Description, permission.ResourceTypeID, permission.Action, permission.CreatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetPermission(ctx context.Context, id uuid.UUID) (*Permission, error) {
	var permission Permission
	query := `
		SELECT id, organization_id, name, description, resource_type_id, action, created_at
		FROM permissions WHERE id = $1`

	err := s.db.GetContext(ctx, &permission, query, id)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (s *PostgreSQLRBACService) ListPermissions(ctx context.Context, orgID uuid.UUID) ([]*Permission, error) {
	var permissions []*Permission
	query := `
		SELECT id, organization_id, name, description, resource_type_id, action, created_at
		FROM permissions 
		WHERE organization_id = $1
		ORDER BY name`

	err := s.db.SelectContext(ctx, &permissions, query, orgID)
	return permissions, err
}

func (s *PostgreSQLRBACService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (id, role_id, permission_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := s.db.ExecContext(ctx, query, uuid.New(), roleID, permissionID, time.Now())
	return err
}

func (s *PostgreSQLRBACService) assignPermissionToRoleInTx(ctx context.Context, tx *sqlx.Tx, roleID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (id, role_id, permission_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := tx.ExecContext(ctx, query, uuid.New(), roleID, permissionID, time.Now())
	return err
}

func (s *PostgreSQLRBACService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := s.db.ExecContext(ctx, query, roleID, permissionID)
	return err
}

// Role assignments

func (s *PostgreSQLRBACService) AssignRoleToUser(ctx context.Context, assignment *UserRoleAssignment) error {
	query := `
		INSERT INTO user_role_assignments 
		(id, user_id, role_id, organization_id, resource_scope, valid_from, valid_until, assigned_by, assignment_reason, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	assignment.ID = uuid.New()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query, assignment.ID, assignment.UserID, assignment.RoleID,
		assignment.OrganizationID, assignment.ResourceScope, assignment.ValidFrom, assignment.ValidUntil,
		assignment.AssignedBy, assignment.AssignmentReason, assignment.IsActive, assignment.CreatedAt, assignment.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID, orgID uuid.UUID) error {
	query := `UPDATE user_role_assignments SET is_active = false, updated_at = $4 WHERE user_id = $1 AND role_id = $2 AND organization_id = $3`
	_, err := s.db.ExecContext(ctx, query, userID, roleID, orgID, time.Now())
	return err
}

func (s *PostgreSQLRBACService) GetUserRoles(ctx context.Context, userID, orgID uuid.UUID) ([]*Role, error) {
	query := `
		SELECT r.id, r.organization_id, r.name, r.description, r.is_system_role, r.is_template, r.metadata, r.created_at, r.updated_at
		FROM roles r
		JOIN user_role_assignments ura ON r.id = ura.role_id
		WHERE ura.user_id = $1 AND ura.organization_id = $2 AND ura.is_active = true
		  AND (ura.valid_until IS NULL OR ura.valid_until > CURRENT_TIMESTAMP)
		  AND ura.valid_from <= CURRENT_TIMESTAMP`

	rows, err := s.db.QueryContext(ctx, query, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var role Role
		var metadataJSON []byte

		err = rows.Scan(&role.ID, &role.OrganizationID, &role.Name, &role.Description,
			&role.IsSystemRole, &role.IsTemplate, &metadataJSON, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &role.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		roles = append(roles, &role)
	}

	return roles, rows.Err()
}

func (s *PostgreSQLRBACService) GetUserAssignments(ctx context.Context, userID uuid.UUID) ([]*UserRoleAssignment, error) {
	var assignments []*UserRoleAssignment
	query := `
		SELECT id, user_id, role_id, organization_id, resource_scope, valid_from, valid_until, 
		       assigned_by, assignment_reason, is_active, created_at, updated_at
		FROM user_role_assignments 
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC`

	err := s.db.SelectContext(ctx, &assignments, query, userID)
	return assignments, err
}

// Permission checking

func (s *PostgreSQLRBACService) HasPermission(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, permission string, resourcePath *string) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_role_assignments ura ON rp.role_id = ura.role_id
		WHERE ura.user_id = $1 AND ura.organization_id = $2 AND p.name = $3
		  AND ura.is_active = true
		  AND (ura.valid_until IS NULL OR ura.valid_until > CURRENT_TIMESTAMP)
		  AND ura.valid_from <= CURRENT_TIMESTAMP`

	args := []interface{}{userID, orgID, permission}

	// Add resource scope check if provided
	if resourcePath != nil {
		query += ` AND (ura.resource_scope IS NULL OR $4 <@ ura.resource_scope::ltree)`
		args = append(args, *resourcePath)
	}

	var hasPermission bool
	err := s.db.GetContext(ctx, &hasPermission, query, args...)

	// Also check emergency access
	if !hasPermission && err == nil {
		emergencyQuery := `
			SELECT COUNT(*) > 0
			FROM emergency_access ea
			WHERE ea.user_id = $1 AND ea.organization_id = $2 AND $3 = ANY(ea.granted_permissions)
			  AND ea.is_active = true
			  AND ea.valid_from <= CURRENT_TIMESTAMP
			  AND ea.valid_until > CURRENT_TIMESTAMP`

		err = s.db.GetContext(ctx, &hasPermission, emergencyQuery, userID, orgID, permission)
	}

	return hasPermission, err
}

func (s *PostgreSQLRBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, resourcePath *string) ([]string, error) {
	query := `
		SELECT DISTINCT p.name
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_role_assignments ura ON rp.role_id = ura.role_id
		WHERE ura.user_id = $1 AND ura.organization_id = $2
		  AND ura.is_active = true
		  AND (ura.valid_until IS NULL OR ura.valid_until > CURRENT_TIMESTAMP)
		  AND ura.valid_from <= CURRENT_TIMESTAMP`

	args := []interface{}{userID, orgID}

	if resourcePath != nil {
		query += ` AND (ura.resource_scope IS NULL OR $3 <@ ura.resource_scope::ltree)`
		args = append(args, *resourcePath)
	}

	var permissions []string
	err := s.db.SelectContext(ctx, &permissions, query, args...)
	if err != nil {
		return nil, err
	}

	// Add emergency access permissions
	emergencyQuery := `
		SELECT UNNEST(granted_permissions) as permission
		FROM emergency_access
		WHERE user_id = $1 AND organization_id = $2
		  AND is_active = true
		  AND valid_from <= CURRENT_TIMESTAMP
		  AND valid_until > CURRENT_TIMESTAMP`

	var emergencyPermissions []string
	err = s.db.SelectContext(ctx, &emergencyPermissions, emergencyQuery, userID, orgID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Merge permissions
	permissionSet := make(map[string]bool)
	for _, p := range permissions {
		permissionSet[p] = true
	}
	for _, p := range emergencyPermissions {
		permissionSet[p] = true
	}

	result := make([]string, 0, len(permissionSet))
	for p := range permissionSet {
		result = append(result, p)
	}

	return result, nil
}

// Emergency access

func (s *PostgreSQLRBACService) GrantEmergencyAccess(ctx context.Context, access *EmergencyAccess) error {
	query := `
		INSERT INTO emergency_access 
		(id, user_id, organization_id, granted_permissions, reason, granted_by, approved_by, valid_from, valid_until, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	access.ID = uuid.New()
	access.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query, access.ID, access.UserID, access.OrganizationID,
		pq.Array(access.GrantedPermissions), access.Reason, access.GrantedBy, access.ApprovedBy,
		access.ValidFrom, access.ValidUntil, access.IsActive, access.CreatedAt)
	return err
}

func (s *PostgreSQLRBACService) RevokeEmergencyAccess(ctx context.Context, accessID uuid.UUID, revokedBy uuid.UUID, reason string) error {
	query := `
		UPDATE emergency_access 
		SET is_active = false, revoked_at = $2, revoked_by = $3, revoke_reason = $4
		WHERE id = $1`

	_, err := s.db.ExecContext(ctx, query, accessID, time.Now(), revokedBy, reason)
	return err
}

func (s *PostgreSQLRBACService) GetActiveEmergencyAccess(ctx context.Context, userID, orgID uuid.UUID) ([]*EmergencyAccess, error) {
	var accesses []*EmergencyAccess
	query := `
		SELECT id, user_id, organization_id, granted_permissions, reason, granted_by, approved_by,
		       valid_from, valid_until, is_active, revoked_at, revoked_by, revoke_reason, created_at
		FROM emergency_access
		WHERE user_id = $1 AND organization_id = $2 AND is_active = true
		  AND valid_from <= CURRENT_TIMESTAMP AND valid_until > CURRENT_TIMESTAMP
		ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var access EmergencyAccess
		err := rows.Scan(
			&access.ID, &access.UserID, &access.OrganizationID,
			pq.Array(&access.GrantedPermissions), &access.Reason, &access.GrantedBy,
			&access.ApprovedBy, &access.ValidFrom, &access.ValidUntil, &access.IsActive,
			&access.RevokedAt, &access.RevokedBy, &access.RevokeReason, &access.CreatedAt)
		if err != nil {
			return nil, err
		}
		accesses = append(accesses, &access)
	}

	return accesses, rows.Err()
}

// Enhanced emergency access with break-glass procedures

func (s *PostgreSQLRBACService) CreateEmergencyAccessRequest(ctx context.Context, request *EmergencyAccessRequest) error {
	query := `
		INSERT INTO emergency_access_requests 
		(id, user_id, organization_id, requested_permissions, reason, urgency_level, 
		 requested_duration, break_glass, required_approvals, expires_at, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	request.ID = uuid.New()
	request.CreatedAt = time.Now()
	request.Status = EmergencyAccessRequestStatusPending
	request.RequestedAt = request.CreatedAt

	// Set expiration time based on urgency level
	if request.ExpiresAt == nil {
		switch request.UrgencyLevel {
		case EmergencyUrgencyCritical:
			expiresAt := request.CreatedAt.Add(1 * time.Hour)
			request.ExpiresAt = &expiresAt
		case EmergencyUrgencyHigh:
			expiresAt := request.CreatedAt.Add(4 * time.Hour)
			request.ExpiresAt = &expiresAt
		case EmergencyUrgencyMedium:
			expiresAt := request.CreatedAt.Add(24 * time.Hour)
			request.ExpiresAt = &expiresAt
		case EmergencyUrgencyLow:
			expiresAt := request.CreatedAt.Add(72 * time.Hour)
			request.ExpiresAt = &expiresAt
		}
	}

	metadataJSON, _ := json.Marshal(request.Metadata)

	_, err := s.db.ExecContext(ctx, query,
		request.ID, request.UserID, request.OrganizationID,
		pq.Array(request.RequestedPermissions), request.Reason, request.UrgencyLevel,
		request.RequestedDuration, request.BreakGlass, request.RequiredApprovals,
		request.ExpiresAt, metadataJSON, request.CreatedAt)

	if err != nil {
		return err
	}

	// Process break-glass if enabled
	if request.BreakGlass {
		return s.ProcessBreakGlassAccess(ctx, request.ID)
	}

	return nil
}

func (s *PostgreSQLRBACService) ApproveEmergencyAccessRequest(ctx context.Context, requestID, approverID uuid.UUID, action EmergencyAccessApprovalAction, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if request is still pending
	var status EmergencyAccessRequestStatus
	err = tx.QueryRowContext(ctx,
		"SELECT status FROM emergency_access_requests WHERE id = $1",
		requestID).Scan(&status)
	if err != nil {
		return err
	}

	if status != EmergencyAccessRequestStatusPending {
		return fmt.Errorf("request is not pending: %s", status)
	}

	// Check if approver hasn't already voted
	var count int
	err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM emergency_access_approvals WHERE request_id = $1 AND approver_id = $2",
		requestID, approverID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("approver has already voted on this request")
	}

	// Insert approval/denial
	_, err = tx.ExecContext(ctx, `
		INSERT INTO emergency_access_approvals (id, request_id, approver_id, action, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), requestID, approverID, action, reason, time.Now())

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgreSQLRBACService) GetEmergencyAccessRequest(ctx context.Context, requestID uuid.UUID) (*EmergencyAccessRequest, error) {
	var request EmergencyAccessRequest
	var metadataJSON []byte

	query := `
		SELECT id, user_id, organization_id, requested_permissions, reason, urgency_level,
		       requested_duration, break_glass, required_approvals, status, requested_at,
		       expires_at, auto_approved_at, emergency_access_id, metadata, created_at, updated_at
		FROM emergency_access_requests
		WHERE id = $1`

	err := s.db.QueryRowContext(ctx, query, requestID).Scan(
		&request.ID, &request.UserID, &request.OrganizationID,
		pq.Array(&request.RequestedPermissions), &request.Reason, &request.UrgencyLevel,
		&request.RequestedDuration, &request.BreakGlass, &request.RequiredApprovals,
		&request.Status, &request.RequestedAt, &request.ExpiresAt,
		&request.AutoApprovedAt, &request.EmergencyAccessID, &metadataJSON,
		&request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &request.Metadata)
	}

	return &request, nil
}

func (s *PostgreSQLRBACService) ListEmergencyAccessRequests(ctx context.Context, orgID uuid.UUID, status *EmergencyAccessRequestStatus, limit, offset int) ([]*EmergencyAccessRequest, error) {
	var requests []*EmergencyAccessRequest
	var args []interface{}
	argPos := 1

	query := `
		SELECT id, user_id, organization_id, requested_permissions, reason, urgency_level,
		       requested_duration, break_glass, required_approvals, status, requested_at,
		       expires_at, auto_approved_at, emergency_access_id, metadata, created_at, updated_at
		FROM emergency_access_requests
		WHERE organization_id = $1`
	args = append(args, orgID)
	argPos++

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var request EmergencyAccessRequest
		var metadataJSON []byte

		err := rows.Scan(
			&request.ID, &request.UserID, &request.OrganizationID,
			pq.Array(&request.RequestedPermissions), &request.Reason, &request.UrgencyLevel,
			&request.RequestedDuration, &request.BreakGlass, &request.RequiredApprovals,
			&request.Status, &request.RequestedAt, &request.ExpiresAt,
			&request.AutoApprovedAt, &request.EmergencyAccessID, &metadataJSON,
			&request.CreatedAt, &request.UpdatedAt)

		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &request.Metadata)
		}

		requests = append(requests, &request)
	}

	return requests, rows.Err()
}

func (s *PostgreSQLRBACService) GetEmergencyAccessApprovals(ctx context.Context, requestID uuid.UUID) ([]*EmergencyAccessApproval, error) {
	var approvals []*EmergencyAccessApproval

	query := `
		SELECT id, request_id, approver_id, action, reason, created_at
		FROM emergency_access_approvals
		WHERE request_id = $1
		ORDER BY created_at ASC`

	err := s.db.SelectContext(ctx, &approvals, query, requestID)
	return approvals, err
}

func (s *PostgreSQLRBACService) ProcessBreakGlassAccess(ctx context.Context, requestID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "SELECT process_break_glass_access($1)", requestID)
	return err
}

func (s *PostgreSQLRBACService) ProcessEmergencyAccessEscalations(ctx context.Context) error {
	// First expire old requests
	_, err := s.db.ExecContext(ctx, "SELECT expire_emergency_access_requests()")
	if err != nil {
		return err
	}

	// Process escalations (placeholder for more complex escalation logic)
	query := `
		SELECT ear.id, ear.organization_id, ear.urgency_level, ear.created_at, 
		       bgc.escalation_rules
		FROM emergency_access_requests ear
		JOIN break_glass_config bgc ON ear.organization_id = bgc.organization_id
		WHERE ear.status = 'pending' 
		  AND bgc.enabled = true
		  AND jsonb_array_length(bgc.escalation_rules) > 0`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var requestID, orgID uuid.UUID
		var urgencyLevel EmergencyUrgencyLevel
		var createdAt time.Time
		var escalationRulesJSON []byte

		err := rows.Scan(&requestID, &orgID, &urgencyLevel, &createdAt, &escalationRulesJSON)
		if err != nil {
			continue
		}

		var escalationRules []EmergencyEscalationRule
		if err := json.Unmarshal(escalationRulesJSON, &escalationRules); err != nil {
			continue
		}

		// Check if any escalation rules should be triggered
		for _, rule := range escalationRules {
			if rule.UrgencyLevel == urgencyLevel {
				if time.Since(createdAt) >= rule.EscalateAfter {
					if rule.AutoApprove {
						// Auto-approve the request
						_, err = s.db.ExecContext(ctx, `
							UPDATE emergency_access_requests 
							SET status = 'approved', auto_approved_at = CURRENT_TIMESTAMP
							WHERE id = $1 AND status = 'pending'`, requestID)
						if err != nil {
							continue
						}

						// Create emergency access grant
						_, err = s.db.ExecContext(ctx, `
							INSERT INTO emergency_access (
								user_id, organization_id, granted_permissions, reason,
								valid_from, valid_until, is_active
							) 
							SELECT user_id, organization_id, requested_permissions, 
								   'ESCALATED: ' || reason, CURRENT_TIMESTAMP,
								   CURRENT_TIMESTAMP + requested_duration, true
							FROM emergency_access_requests 
							WHERE id = $1`, requestID)
						if err != nil {
							continue
						}
					}
				}
			}
		}
	}

	return rows.Err()
}

func (s *PostgreSQLRBACService) GetBreakGlassConfig(ctx context.Context, orgID uuid.UUID) (*BreakGlassConfig, error) {
	var config BreakGlassConfig
	var approvalRequirementsJSON, escalationRulesJSON []byte

	query := `
		SELECT enabled, auto_approval_urgency, max_duration, required_permissions,
		       approval_requirements, auto_revocation_minutes, notification_channels,
		       escalation_rules
		FROM break_glass_config
		WHERE organization_id = $1`

	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
		&config.Enabled, &config.AutoApprovalUrgency, &config.MaxDuration,
		pq.Array(&config.RequiredPermissions), &approvalRequirementsJSON,
		&config.AutoRevocationMinutes, pq.Array(&config.NotificationChannels),
		&escalationRulesJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default config if none exists
			return &BreakGlassConfig{
				Enabled:             false,
				AutoApprovalUrgency: EmergencyUrgencyCritical,
				MaxDuration:         4 * time.Hour,
				RequiredPermissions: []string{},
				ApprovalRequirements: map[EmergencyUrgencyLevel]int{
					EmergencyUrgencyLow:      2,
					EmergencyUrgencyMedium:   2,
					EmergencyUrgencyHigh:     1,
					EmergencyUrgencyCritical: 0,
				},
				AutoRevocationMinutes: 240,
				NotificationChannels:  []string{},
				EscalationRules:       []EmergencyEscalationRule{},
			}, nil
		}
		return nil, err
	}

	if len(approvalRequirementsJSON) > 0 {
		json.Unmarshal(approvalRequirementsJSON, &config.ApprovalRequirements)
	}

	if len(escalationRulesJSON) > 0 {
		json.Unmarshal(escalationRulesJSON, &config.EscalationRules)
	}

	return &config, nil
}

func (s *PostgreSQLRBACService) UpdateBreakGlassConfig(ctx context.Context, orgID uuid.UUID, config *BreakGlassConfig) error {
	approvalRequirementsJSON, _ := json.Marshal(config.ApprovalRequirements)
	escalationRulesJSON, _ := json.Marshal(config.EscalationRules)

	query := `
		INSERT INTO break_glass_config 
		(organization_id, enabled, auto_approval_urgency, max_duration, required_permissions,
		 approval_requirements, auto_revocation_minutes, notification_channels, escalation_rules, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (organization_id) DO UPDATE SET
		enabled = EXCLUDED.enabled,
		auto_approval_urgency = EXCLUDED.auto_approval_urgency,
		max_duration = EXCLUDED.max_duration,
		required_permissions = EXCLUDED.required_permissions,
		approval_requirements = EXCLUDED.approval_requirements,
		auto_revocation_minutes = EXCLUDED.auto_revocation_minutes,
		notification_channels = EXCLUDED.notification_channels,
		escalation_rules = EXCLUDED.escalation_rules,
		updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query, orgID, config.Enabled, config.AutoApprovalUrgency,
		config.MaxDuration, pq.Array(config.RequiredPermissions), approvalRequirementsJSON,
		config.AutoRevocationMinutes, pq.Array(config.NotificationChannels), escalationRulesJSON,
		time.Now())

	return err
}

// Resource management

func (s *PostgreSQLRBACService) CreateResourceType(ctx context.Context, resourceType *ResourceType) error {
	query := `
		INSERT INTO resource_types (id, organization_id, name, description, hierarchy_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	resourceType.ID = uuid.New()
	resourceType.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, query, resourceType.ID, resourceType.OrganizationID, resourceType.Name,
		resourceType.Description, resourceType.HierarchyEnabled, resourceType.CreatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetResourceType(ctx context.Context, id uuid.UUID) (*ResourceType, error) {
	var resourceType ResourceType
	query := `
		SELECT id, organization_id, name, description, hierarchy_enabled, created_at
		FROM resource_types WHERE id = $1`

	err := s.db.GetContext(ctx, &resourceType, query, id)
	if err != nil {
		return nil, err
	}
	return &resourceType, nil
}

func (s *PostgreSQLRBACService) ListResourceTypes(ctx context.Context, orgID uuid.UUID) ([]*ResourceType, error) {
	var resourceTypes []*ResourceType
	query := `
		SELECT id, organization_id, name, description, hierarchy_enabled, created_at
		FROM resource_types 
		WHERE organization_id = $1
		ORDER BY name`

	err := s.db.SelectContext(ctx, &resourceTypes, query, orgID)
	return resourceTypes, err
}

func (s *PostgreSQLRBACService) CreateResource(ctx context.Context, resource *Resource) error {
	query := `
		INSERT INTO resources (id, organization_id, resource_type_id, name, description, parent_path, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	resource.ID = uuid.New()
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(resource.Metadata)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, resource.ID, resource.OrganizationID, resource.ResourceTypeID,
		resource.Name, resource.Description, resource.ParentPath, metadataJSON, resource.CreatedAt, resource.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetResource(ctx context.Context, id uuid.UUID) (*Resource, error) {
	var resource Resource
	var metadataJSON []byte

	query := `
		SELECT id, organization_id, resource_type_id, name, description, parent_path, metadata, created_at, updated_at
		FROM resources WHERE id = $1`

	row := s.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&resource.ID, &resource.OrganizationID, &resource.ResourceTypeID,
		&resource.Name, &resource.Description, &resource.ParentPath,
		&metadataJSON, &resource.CreatedAt, &resource.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &resource, nil
}

func (s *PostgreSQLRBACService) ListResources(ctx context.Context, orgID uuid.UUID, resourceTypeID *uuid.UUID, parentPath *string) ([]*Resource, error) {
	var resources []*Resource
	var query string
	var args []interface{}

	query = `
		SELECT id, organization_id, resource_type_id, name, description, parent_path, metadata, created_at, updated_at
		FROM resources 
		WHERE organization_id = $1`
	args = []interface{}{orgID}

	if resourceTypeID != nil {
		query += ` AND resource_type_id = $2`
		args = append(args, *resourceTypeID)
	}

	if parentPath != nil {
		if resourceTypeID != nil {
			query += ` AND parent_path <@ $3::ltree`
		} else {
			query += ` AND parent_path <@ $2::ltree`
		}
		args = append(args, *parentPath)
	}

	query += ` ORDER BY parent_path, name`

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var resource Resource
		var metadataJSON []byte

		err := rows.Scan(
			&resource.ID, &resource.OrganizationID, &resource.ResourceTypeID,
			&resource.Name, &resource.Description, &resource.ParentPath,
			&metadataJSON, &resource.CreatedAt, &resource.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &resource.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		resources = append(resources, &resource)
	}

	return resources, rows.Err()
}

func (s *PostgreSQLRBACService) UpdateResource(ctx context.Context, resource *Resource) error {
	query := `
		UPDATE resources 
		SET name = $2, description = $3, parent_path = $4, metadata = $5, updated_at = $6
		WHERE id = $1`

	resource.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(resource.Metadata)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, resource.ID, resource.Name, resource.Description,
		resource.ParentPath, metadataJSON, resource.UpdatedAt)
	return err
}

func (s *PostgreSQLRBACService) DeleteResource(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM resources WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// Audit

func (s *PostgreSQLRBACService) LogAuditEvent(ctx context.Context, event *AuditEvent) error {
	query := `
		INSERT INTO rbac_audit 
		(id, organization_id, user_id, action, resource_type, resource_id, permission_name, success, reason, ip_address, user_agent, session_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	metadataJSON, _ := json.Marshal(event.Metadata)

	_, err := s.db.ExecContext(ctx, query, event.ID, event.OrganizationID, event.UserID, event.Action,
		event.ResourceType, event.ResourceID, event.PermissionName, event.Success, event.Reason,
		event.IPAddress, event.UserAgent, event.SessionID, metadataJSON, event.CreatedAt)
	return err
}

func (s *PostgreSQLRBACService) GetAuditLog(ctx context.Context, orgID uuid.UUID, filters AuditFilters) ([]*AuditEvent, error) {
	var events []*AuditEvent
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Base query
	query := `
		SELECT id, organization_id, user_id, action, resource_type, resource_id, permission_name, 
		       success, reason, ip_address, user_agent, session_id, metadata, created_at
		FROM rbac_audit
		WHERE organization_id = $1`
	args = append(args, orgID)
	argIndex++

	// Add filters
	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, filters.Action)
		argIndex++
	}

	if filters.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIndex))
		args = append(args, filters.ResourceType)
		argIndex++
	}

	if filters.Success != nil {
		conditions = append(conditions, fmt.Sprintf("success = $%d", argIndex))
		args = append(args, *filters.Success)
		argIndex++
	}

	if filters.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filters.StartTime)
		argIndex++
	}

	if filters.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filters.EndTime)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event AuditEvent
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID, &event.OrganizationID, &event.UserID, &event.Action,
			&event.ResourceType, &event.ResourceID, &event.PermissionName,
			&event.Success, &event.Reason, &event.IPAddress, &event.UserAgent,
			&event.SessionID, &metadataJSON, &event.CreatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		events = append(events, &event)
	}

	return events, rows.Err()
}

// Time-based access control methods

func (s *PostgreSQLRBACService) CreateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error {
	query := `
		INSERT INTO scheduled_role_assignments 
		(id, user_id, role_id, organization_id, resource_scope, scheduled_activation, scheduled_expiration, 
		 assigned_by, assignment_reason, recurrence_pattern, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	assignment.ID = uuid.New()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(assignment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		assignment.ID, assignment.UserID, assignment.RoleID, assignment.OrganizationID,
		assignment.ResourceScope, assignment.ScheduledActivation, assignment.ScheduledExpiration,
		assignment.AssignedBy, assignment.AssignmentReason, assignment.RecurrencePattern,
		metadataJSON, assignment.CreatedAt, assignment.UpdatedAt)

	if err != nil {
		return err
	}

	// Log the scheduling action
	auditEvent := &AuditEvent{
		OrganizationID: assignment.OrganizationID,
		UserID:         assignment.AssignedBy,
		Action:         "schedule_role_assignment",
		ResourceType:   "scheduled_role_assignment",
		ResourceID:     &assignment.ID,
		Success:        true,
		Reason:         "Scheduled role assignment created",
		Metadata: map[string]interface{}{
			"target_user_id":       assignment.UserID,
			"role_id":              assignment.RoleID,
			"scheduled_activation": assignment.ScheduledActivation,
			"scheduled_expiration": assignment.ScheduledExpiration,
			"assignment_reason":    assignment.AssignmentReason,
		},
	}
	s.LogAuditEvent(ctx, auditEvent)

	return nil
}

func (s *PostgreSQLRBACService) UpdateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error {
	query := `
		UPDATE scheduled_role_assignments 
		SET scheduled_activation = $2, scheduled_expiration = $3, assignment_reason = $4,
		    recurrence_pattern = $5, metadata = $6, updated_at = $7
		WHERE id = $1 AND NOT is_processed`

	assignment.UpdatedAt = time.Now()

	metadataJSON, err := json.Marshal(assignment.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := s.db.ExecContext(ctx, query,
		assignment.ID, assignment.ScheduledActivation, assignment.ScheduledExpiration,
		assignment.AssignmentReason, assignment.RecurrencePattern, metadataJSON,
		assignment.UpdatedAt)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled assignment not found or already processed")
	}

	// Log the update action
	auditEvent := &AuditEvent{
		OrganizationID: assignment.OrganizationID,
		Action:         "update_scheduled_role_assignment",
		ResourceType:   "scheduled_role_assignment",
		ResourceID:     &assignment.ID,
		Success:        true,
		Reason:         "Scheduled role assignment updated",
		Metadata: map[string]interface{}{
			"scheduled_activation": assignment.ScheduledActivation,
			"scheduled_expiration": assignment.ScheduledExpiration,
		},
	}
	s.LogAuditEvent(ctx, auditEvent)

	return nil
}

func (s *PostgreSQLRBACService) DeleteScheduledRoleAssignment(ctx context.Context, id uuid.UUID) error {
	// First get the assignment for audit logging
	var assignment ScheduledRoleAssignment
	selectQuery := `
		SELECT id, user_id, role_id, organization_id, scheduled_activation, is_processed
		FROM scheduled_role_assignments WHERE id = $1`

	err := s.db.GetContext(ctx, &assignment, selectQuery, id)
	if err != nil {
		return err
	}

	if assignment.IsProcessed {
		return fmt.Errorf("cannot delete processed scheduled assignment")
	}

	// Delete the assignment
	deleteQuery := `DELETE FROM scheduled_role_assignments WHERE id = $1 AND NOT is_processed`
	result, err := s.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled assignment not found or already processed")
	}

	// Log the deletion
	auditEvent := &AuditEvent{
		OrganizationID: assignment.OrganizationID,
		Action:         "delete_scheduled_role_assignment",
		ResourceType:   "scheduled_role_assignment",
		ResourceID:     &id,
		Success:        true,
		Reason:         "Scheduled role assignment deleted",
		Metadata: map[string]interface{}{
			"target_user_id":       assignment.UserID,
			"role_id":              assignment.RoleID,
			"scheduled_activation": assignment.ScheduledActivation,
		},
	}
	s.LogAuditEvent(ctx, auditEvent)

	return nil
}

func (s *PostgreSQLRBACService) GetScheduledRoleAssignments(ctx context.Context, userID, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error) {
	var assignments []*ScheduledRoleAssignment
	query := `
		SELECT id, user_id, role_id, organization_id, resource_scope, scheduled_activation,
		       scheduled_expiration, assigned_by, assignment_reason, notification_sent,
		       is_processed, processed_at, processing_error, recurrence_pattern,
		       metadata, created_at, updated_at
		FROM scheduled_role_assignments
		WHERE user_id = $1 AND organization_id = $2
		ORDER BY scheduled_activation ASC`

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var assignment ScheduledRoleAssignment
		var metadataJSON []byte

		err := rows.Scan(
			&assignment.ID, &assignment.UserID, &assignment.RoleID, &assignment.OrganizationID,
			&assignment.ResourceScope, &assignment.ScheduledActivation,
			&assignment.ScheduledExpiration, &assignment.AssignedBy, &assignment.AssignmentReason,
			&assignment.NotificationSent, &assignment.IsProcessed, &assignment.ProcessedAt,
			&assignment.ProcessingError, &assignment.RecurrencePattern,
			&metadataJSON, &assignment.CreatedAt, &assignment.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &assignment.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, rows.Err()
}

func (s *PostgreSQLRBACService) ListPendingActivations(ctx context.Context, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error) {
	var assignments []*ScheduledRoleAssignment
	query := `
		SELECT sra.id, sra.user_id, sra.role_id, sra.organization_id, sra.resource_scope,
		       sra.scheduled_activation, sra.scheduled_expiration, sra.assigned_by,
		       sra.assignment_reason, sra.notification_sent, sra.is_processed,
		       sra.processed_at, sra.processing_error, sra.recurrence_pattern,
		       sra.metadata, sra.created_at, sra.updated_at
		FROM scheduled_role_assignments sra
		JOIN users u ON u.id = sra.user_id
		JOIN roles r ON r.id = sra.role_id
		WHERE sra.organization_id = $1 
		  AND NOT sra.is_processed
		  AND sra.scheduled_activation <= CURRENT_TIMESTAMP + INTERVAL '24 hours'
		ORDER BY sra.scheduled_activation ASC`

	// Use QueryxContext instead of SelectContext to handle manual scanning
	rows, err := s.db.QueryxContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var assignment ScheduledRoleAssignment
		var metadataJSON []byte

		err := rows.Scan(
			&assignment.ID, &assignment.UserID, &assignment.RoleID, &assignment.OrganizationID,
			&assignment.ResourceScope, &assignment.ScheduledActivation, &assignment.ScheduledExpiration,
			&assignment.AssignedBy, &assignment.AssignmentReason, &assignment.NotificationSent,
			&assignment.IsProcessed, &assignment.ProcessedAt, &assignment.ProcessingError,
			&assignment.RecurrencePattern, &metadataJSON, &assignment.CreatedAt, &assignment.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &assignment.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		assignments = append(assignments, &assignment)
	}

	return assignments, rows.Err()
}

func (s *PostgreSQLRBACService) ListExpiredAssignments(ctx context.Context, orgID uuid.UUID) ([]*UserRoleAssignment, error) {
	var assignments []*UserRoleAssignment
	query := `
		SELECT id, user_id, role_id, organization_id, resource_scope, valid_from, valid_until,
		       assigned_by, assignment_reason, is_active, created_at, updated_at
		FROM user_role_assignments
		WHERE organization_id = $1 
		  AND is_active = true
		  AND valid_until IS NOT NULL
		  AND valid_until < CURRENT_TIMESTAMP
		ORDER BY valid_until ASC`

	err := s.db.SelectContext(ctx, &assignments, query, orgID)
	return assignments, err
}

func (s *PostgreSQLRBACService) ProcessScheduledActivations(ctx context.Context) error {
	// Use the database function for atomic processing
	var activationCount int
	query := `SELECT process_scheduled_activations()`

	err := s.db.GetContext(ctx, &activationCount, query)
	if err != nil {
		return fmt.Errorf("failed to process scheduled activations: %w", err)
	}

	// Log the processing summary
	auditEvent := &AuditEvent{
		Action:       "process_scheduled_activations",
		ResourceType: "system",
		Success:      true,
		Reason:       "Batch processing of scheduled activations completed",
		Metadata: map[string]interface{}{
			"activations_processed": activationCount,
			"processed_at":          time.Now(),
		},
	}
	s.LogAuditEvent(ctx, auditEvent)

	return nil
}

func (s *PostgreSQLRBACService) CleanupExpiredAssignments(ctx context.Context) error {
	// Use the database function for atomic cleanup
	var expiredCount int
	query := `SELECT cleanup_expired_assignments()`

	err := s.db.GetContext(ctx, &expiredCount, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired assignments: %w", err)
	}

	// Log the cleanup summary
	auditEvent := &AuditEvent{
		Action:       "cleanup_expired_assignments",
		ResourceType: "system",
		Success:      true,
		Reason:       "Batch cleanup of expired assignments completed",
		Metadata: map[string]interface{}{
			"assignments_expired": expiredCount,
			"cleaned_at":          time.Now(),
		},
	}
	s.LogAuditEvent(ctx, auditEvent)

	return nil
}

func (s *PostgreSQLRBACService) GetTimeBasedAccessStatus(ctx context.Context, orgID uuid.UUID) (*TimeBasedAccessStatus, error) {
	status := &TimeBasedAccessStatus{
		OrganizationID: orgID,
	}

	// Get pending activations count
	pendingQuery := `
		SELECT COUNT(*) FROM scheduled_role_assignments 
		WHERE organization_id = $1 AND NOT is_processed`
	err := s.db.GetContext(ctx, &status.PendingActivations, pendingQuery, orgID)
	if err != nil {
		return nil, err
	}

	// Get active assignments count
	activeQuery := `
		SELECT COUNT(*) FROM user_role_assignments 
		WHERE organization_id = $1 AND is_active = true 
		  AND (valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP)`
	err = s.db.GetContext(ctx, &status.ActiveAssignments, activeQuery, orgID)
	if err != nil {
		return nil, err
	}

	// Get expired assignments count
	expiredQuery := `
		SELECT COUNT(*) FROM user_role_assignments 
		WHERE organization_id = $1 AND is_active = true 
		  AND valid_until IS NOT NULL AND valid_until < CURRENT_TIMESTAMP`
	err = s.db.GetContext(ctx, &status.ExpiredAssignments, expiredQuery, orgID)
	if err != nil {
		return nil, err
	}

	// Get scheduled for next 24h count
	next24hQuery := `
		SELECT COUNT(*) FROM scheduled_role_assignments 
		WHERE organization_id = $1 AND NOT is_processed
		  AND scheduled_activation <= CURRENT_TIMESTAMP + INTERVAL '24 hours'
		  AND scheduled_activation > CURRENT_TIMESTAMP`
	err = s.db.GetContext(ctx, &status.ScheduledForNext24h, next24hQuery, orgID)
	if err != nil {
		return nil, err
	}

	// Get processing errors count
	errorsQuery := `
		SELECT COUNT(*) FROM scheduled_role_assignments 
		WHERE organization_id = $1 AND processing_error IS NOT NULL`
	err = s.db.GetContext(ctx, &status.ProcessingErrors, errorsQuery, orgID)
	if err != nil {
		return nil, err
	}

	// Get last processed timestamp
	lastProcessedQuery := `
		SELECT COALESCE(MAX(processed_at), '1970-01-01'::timestamp) 
		FROM scheduled_role_assignments 
		WHERE organization_id = $1 AND is_processed = true`
	err = s.db.GetContext(ctx, &status.LastProcessedAt, lastProcessedQuery, orgID)
	if err != nil {
		return nil, err
	}

	return status, nil
}
