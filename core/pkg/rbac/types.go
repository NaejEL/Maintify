package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant in the multi-tenant system
type Organization struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Slug        string                 `json:"slug" db:"slug"`
	Description string                 `json:"description" db:"description"`
	Settings    map[string]interface{} `json:"settings" db:"settings"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// User represents a system user
type User struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	Email         string                 `json:"email" db:"email"`
	Username      string                 `json:"username" db:"username"`
	PasswordHash  string                 `json:"-" db:"password_hash"`
	FirstName     string                 `json:"first_name" db:"first_name"`
	LastName      string                 `json:"last_name" db:"last_name"`
	IsActive      bool                   `json:"is_active" db:"is_active"`
	IsSystemAdmin bool                   `json:"is_system_admin" db:"is_system_admin"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	LastLoginAt   *time.Time             `json:"last_login_at" db:"last_login_at"`
}

// ResourceType defines the types of resources that can be managed
type ResourceType struct {
	ID               uuid.UUID `json:"id" db:"id"`
	OrganizationID   uuid.UUID `json:"organization_id" db:"organization_id"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	HierarchyEnabled bool      `json:"hierarchy_enabled" db:"hierarchy_enabled"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// Resource represents a manageable resource in the system
type Resource struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	ResourceTypeID uuid.UUID              `json:"resource_type_id" db:"resource_type_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	ParentPath     string                 `json:"parent_path" db:"parent_path"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// Permission represents a granular action that can be performed
type Permission struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	ResourceTypeID *uuid.UUID `json:"resource_type_id" db:"resource_type_id"`
	Action         string     `json:"action" db:"action"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// Role represents a collection of permissions
type Role struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	IsSystemRole   bool                   `json:"is_system_role" db:"is_system_role"`
	IsTemplate     bool                   `json:"is_template" db:"is_template"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
	Permissions    []Permission           `json:"permissions,omitempty"`
}

// UserRoleAssignment represents the assignment of a role to a user
type UserRoleAssignment struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID           uuid.UUID  `json:"role_id" db:"role_id"`
	OrganizationID   uuid.UUID  `json:"organization_id" db:"organization_id"`
	ResourceScope    *string    `json:"resource_scope" db:"resource_scope"`
	ValidFrom        time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil       *time.Time `json:"valid_until" db:"valid_until"`
	AssignedBy       *uuid.UUID `json:"assigned_by" db:"assigned_by"`
	AssignmentReason string     `json:"assignment_reason" db:"assignment_reason"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// EmergencyAccess represents temporary privilege escalation
type EmergencyAccess struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	UserID             uuid.UUID  `json:"user_id" db:"user_id"`
	OrganizationID     uuid.UUID  `json:"organization_id" db:"organization_id"`
	GrantedPermissions []string   `json:"granted_permissions" db:"granted_permissions"`
	Reason             string     `json:"reason" db:"reason"`
	GrantedBy          *uuid.UUID `json:"granted_by" db:"granted_by"`
	ApprovedBy         *uuid.UUID `json:"approved_by" db:"approved_by"`
	ValidFrom          time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil         time.Time  `json:"valid_until" db:"valid_until"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	RevokedAt          *time.Time `json:"revoked_at" db:"revoked_at"`
	RevokedBy          *uuid.UUID `json:"revoked_by" db:"revoked_by"`
	RevokeReason       *string    `json:"revoke_reason" db:"revoke_reason"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

// EmergencyAccessRequest represents a request for emergency access that requires approval
type EmergencyAccessRequest struct {
	ID                   uuid.UUID                    `json:"id" db:"id"`
	UserID               uuid.UUID                    `json:"user_id" db:"user_id"`
	OrganizationID       uuid.UUID                    `json:"organization_id" db:"organization_id"`
	RequestedPermissions []string                     `json:"requested_permissions" db:"requested_permissions"`
	Reason               string                       `json:"reason" db:"reason"`
	UrgencyLevel         EmergencyUrgencyLevel        `json:"urgency_level" db:"urgency_level"`
	RequestedDuration    int64                        `json:"requested_duration" db:"requested_duration"`
	BreakGlass           bool                         `json:"break_glass" db:"break_glass"`
	RequiredApprovals    int                          `json:"required_approvals" db:"required_approvals"`
	Status               EmergencyAccessRequestStatus `json:"status" db:"status"`
	RequestedAt          time.Time                    `json:"requested_at" db:"requested_at"`
	ExpiresAt            *time.Time                   `json:"expires_at" db:"expires_at"`
	AutoApprovedAt       *time.Time                   `json:"auto_approved_at" db:"auto_approved_at"`
	EmergencyAccessID    *uuid.UUID                   `json:"emergency_access_id" db:"emergency_access_id"`
	Metadata             map[string]interface{}       `json:"metadata" db:"metadata"`
	CreatedAt            time.Time                    `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time                    `json:"updated_at" db:"updated_at"`
}

// EmergencyAccessApproval represents an approval for an emergency access request
type EmergencyAccessApproval struct {
	ID         uuid.UUID                     `json:"id" db:"id"`
	RequestID  uuid.UUID                     `json:"request_id" db:"request_id"`
	ApproverID uuid.UUID                     `json:"approver_id" db:"approver_id"`
	Action     EmergencyAccessApprovalAction `json:"action" db:"action"`
	Reason     string                        `json:"reason" db:"reason"`
	CreatedAt  time.Time                     `json:"created_at" db:"created_at"`
}

// EmergencyUrgencyLevel represents the urgency level of an emergency access request
type EmergencyUrgencyLevel string

const (
	EmergencyUrgencyLow      EmergencyUrgencyLevel = "low"
	EmergencyUrgencyMedium   EmergencyUrgencyLevel = "medium"
	EmergencyUrgencyHigh     EmergencyUrgencyLevel = "high"
	EmergencyUrgencyCritical EmergencyUrgencyLevel = "critical"
)

// EmergencyAccessRequestStatus represents the status of an emergency access request
type EmergencyAccessRequestStatus string

const (
	EmergencyAccessRequestStatusPending  EmergencyAccessRequestStatus = "pending"
	EmergencyAccessRequestStatusApproved EmergencyAccessRequestStatus = "approved"
	EmergencyAccessRequestStatusDenied   EmergencyAccessRequestStatus = "denied"
	EmergencyAccessRequestStatusExpired  EmergencyAccessRequestStatus = "expired"
	EmergencyAccessRequestStatusRevoked  EmergencyAccessRequestStatus = "revoked"
	EmergencyAccessRequestStatusGranted  EmergencyAccessRequestStatus = "granted"
)

// EmergencyAccessApprovalAction represents an approval action
type EmergencyAccessApprovalAction string

const (
	EmergencyAccessApprovalActionApprove EmergencyAccessApprovalAction = "approve"
	EmergencyAccessApprovalActionDeny    EmergencyAccessApprovalAction = "deny"
)

// BreakGlassConfig represents the configuration for break-glass emergency access
type BreakGlassConfig struct {
	Enabled               bool                          `json:"enabled"`
	AutoApprovalUrgency   EmergencyUrgencyLevel         `json:"auto_approval_urgency"`
	MaxDuration           time.Duration                 `json:"max_duration"`
	RequiredPermissions   []string                      `json:"required_permissions"`
	ApprovalRequirements  map[EmergencyUrgencyLevel]int `json:"approval_requirements"`
	AutoRevocationMinutes int                           `json:"auto_revocation_minutes"`
	NotificationChannels  []string                      `json:"notification_channels"`
	EscalationRules       []EmergencyEscalationRule     `json:"escalation_rules"`
}

// EmergencyEscalationRule represents escalation rules for emergency access
type EmergencyEscalationRule struct {
	UrgencyLevel  EmergencyUrgencyLevel `json:"urgency_level"`
	EscalateAfter time.Duration         `json:"escalate_after"`
	EscalateTo    []uuid.UUID           `json:"escalate_to"`
	AutoApprove   bool                  `json:"auto_approve"`
}

// RBACService defines the interface for RBAC operations
type RBACService interface {
	// Organization management
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error)

	// User management
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, orgID *uuid.UUID, limit, offset int) ([]*User, error)

	// Role management
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*Role, error)

	// Permission management
	CreatePermission(ctx context.Context, permission *Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*Permission, error)
	ListPermissions(ctx context.Context, orgID uuid.UUID) ([]*Permission, error)
	AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error

	// Role assignments
	AssignRoleToUser(ctx context.Context, assignment *UserRoleAssignment) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID, orgID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID, orgID uuid.UUID) ([]*Role, error)
	GetUserAssignments(ctx context.Context, userID uuid.UUID) ([]*UserRoleAssignment, error)

	// Permission checking
	HasPermission(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, permission string, resourcePath *string) (bool, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, resourcePath *string) ([]string, error)

	// Emergency access
	GrantEmergencyAccess(ctx context.Context, access *EmergencyAccess) error
	RevokeEmergencyAccess(ctx context.Context, accessID uuid.UUID, revokedBy uuid.UUID, reason string) error
	GetActiveEmergencyAccess(ctx context.Context, userID, orgID uuid.UUID) ([]*EmergencyAccess, error)

	// Enhanced emergency access with break-glass procedures
	CreateEmergencyAccessRequest(ctx context.Context, request *EmergencyAccessRequest) error
	ApproveEmergencyAccessRequest(ctx context.Context, requestID, approverID uuid.UUID, action EmergencyAccessApprovalAction, reason string) error
	GetEmergencyAccessRequest(ctx context.Context, requestID uuid.UUID) (*EmergencyAccessRequest, error)
	ListEmergencyAccessRequests(ctx context.Context, orgID uuid.UUID, status *EmergencyAccessRequestStatus, limit, offset int) ([]*EmergencyAccessRequest, error)
	GetEmergencyAccessApprovals(ctx context.Context, requestID uuid.UUID) ([]*EmergencyAccessApproval, error)
	ProcessBreakGlassAccess(ctx context.Context, requestID uuid.UUID) error
	ProcessEmergencyAccessEscalations(ctx context.Context) error
	GetBreakGlassConfig(ctx context.Context, orgID uuid.UUID) (*BreakGlassConfig, error)
	UpdateBreakGlassConfig(ctx context.Context, orgID uuid.UUID, config *BreakGlassConfig) error

	// Resource management
	CreateResourceType(ctx context.Context, resourceType *ResourceType) error
	GetResourceType(ctx context.Context, id uuid.UUID) (*ResourceType, error)
	ListResourceTypes(ctx context.Context, orgID uuid.UUID) ([]*ResourceType, error)

	CreateResource(ctx context.Context, resource *Resource) error
	GetResource(ctx context.Context, id uuid.UUID) (*Resource, error)
	ListResources(ctx context.Context, orgID uuid.UUID, resourceTypeID *uuid.UUID, parentPath *string) ([]*Resource, error)
	UpdateResource(ctx context.Context, resource *Resource) error
	DeleteResource(ctx context.Context, id uuid.UUID) error

	// Audit
	LogAuditEvent(ctx context.Context, event *AuditEvent) error
	GetAuditLog(ctx context.Context, orgID uuid.UUID, filters AuditFilters) ([]*AuditEvent, error)

	// Time-based access control
	CreateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error
	UpdateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error
	DeleteScheduledRoleAssignment(ctx context.Context, id uuid.UUID) error
	GetScheduledRoleAssignments(ctx context.Context, userID, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error)
	ListPendingActivations(ctx context.Context, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error)
	ListExpiredAssignments(ctx context.Context, orgID uuid.UUID) ([]*UserRoleAssignment, error)
	ProcessScheduledActivations(ctx context.Context) error
	CleanupExpiredAssignments(ctx context.Context) error
	GetTimeBasedAccessStatus(ctx context.Context, orgID uuid.UUID) (*TimeBasedAccessStatus, error)
}

// AuditEvent represents an RBAC audit log entry
type AuditEvent struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	UserID         *uuid.UUID             `json:"user_id" db:"user_id"`
	Action         string                 `json:"action" db:"action"`
	ResourceType   string                 `json:"resource_type" db:"resource_type"`
	ResourceID     *uuid.UUID             `json:"resource_id" db:"resource_id"`
	PermissionName string                 `json:"permission_name" db:"permission_name"`
	Success        bool                   `json:"success" db:"success"`
	Reason         string                 `json:"reason" db:"reason"`
	IPAddress      string                 `json:"ip_address" db:"ip_address"`
	UserAgent      string                 `json:"user_agent" db:"user_agent"`
	SessionID      string                 `json:"session_id" db:"session_id"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// ScheduledRoleAssignment represents a future role assignment with time-based activation
type ScheduledRoleAssignment struct {
	ID                  uuid.UUID              `json:"id" db:"id"`
	UserID              uuid.UUID              `json:"user_id" db:"user_id"`
	RoleID              uuid.UUID              `json:"role_id" db:"role_id"`
	OrganizationID      uuid.UUID              `json:"organization_id" db:"organization_id"`
	ResourceScope       *string                `json:"resource_scope" db:"resource_scope"`
	ScheduledActivation time.Time              `json:"scheduled_activation" db:"scheduled_activation"`
	ScheduledExpiration *time.Time             `json:"scheduled_expiration" db:"scheduled_expiration"`
	AssignedBy          *uuid.UUID             `json:"assigned_by" db:"assigned_by"`
	AssignmentReason    string                 `json:"assignment_reason" db:"assignment_reason"`
	NotificationSent    bool                   `json:"notification_sent" db:"notification_sent"`
	IsProcessed         bool                   `json:"is_processed" db:"is_processed"`
	ProcessedAt         *time.Time             `json:"processed_at" db:"processed_at"`
	ProcessingError     *string                `json:"processing_error" db:"processing_error"`
	RecurrencePattern   *string                `json:"recurrence_pattern" db:"recurrence_pattern"` // CRON-like pattern for recurring access
	Metadata            map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
}

// TimeBasedAccessStatus represents the status of time-based access controls
type TimeBasedAccessStatus struct {
	OrganizationID      uuid.UUID `json:"organization_id"`
	PendingActivations  int       `json:"pending_activations"`
	ActiveAssignments   int       `json:"active_assignments"`
	ExpiredAssignments  int       `json:"expired_assignments"`
	ScheduledForNext24h int       `json:"scheduled_for_next_24h"`
	ProcessingErrors    int       `json:"processing_errors"`
	LastProcessedAt     time.Time `json:"last_processed_at"`
}

// AuditFilters represents filters for audit log queries
type AuditFilters struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	Success      *bool
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Common permission actions
const (
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionExecute = "execute"
	ActionManage  = "manage"
)

// Common system permissions
const (
	PermissionSystemAdmin = "system.admin"
	PermissionOrgCreate   = "org.create"
	PermissionOrgManage   = "org.manage"
	PermissionUserManage  = "user.manage"
	PermissionUserView    = "user.view"
	PermissionOrgView     = "org.view"
	PermissionRoleManage  = "role.manage"
	PermissionRoleView    = "role.view"
	PermissionRBACManage  = "rbac.manage"
)
