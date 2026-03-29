package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockRBACService is a mock implementation of RBACService
type MockRBACService struct {
	mock.Mock
}

func (m *MockRBACService) CreateOrganization(ctx context.Context, org *Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockRBACService) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Organization), args.Error(1)
}

func (m *MockRBACService) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Organization), args.Error(1)
}

func (m *MockRBACService) UpdateOrganization(ctx context.Context, org *Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockRBACService) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRBACService) ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Organization), args.Error(1)
}

func (m *MockRBACService) CreateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRBACService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRBACService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRBACService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRBACService) UpdateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRBACService) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRBACService) ListUsers(ctx context.Context, orgID *uuid.UUID, limit, offset int) ([]*User, error) {
	args := m.Called(ctx, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*User), args.Error(1)
}

func (m *MockRBACService) CreateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRBACService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Role), args.Error(1)
}

func (m *MockRBACService) UpdateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRBACService) DeleteRole(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRBACService) ListRoles(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*Role, error) {
	args := m.Called(ctx, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Role), args.Error(1)
}

func (m *MockRBACService) CreatePermission(ctx context.Context, permission *Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *MockRBACService) GetPermission(ctx context.Context, id uuid.UUID) (*Permission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Permission), args.Error(1)
}

func (m *MockRBACService) ListPermissions(ctx context.Context, orgID uuid.UUID) ([]*Permission, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Permission), args.Error(1)
}

func (m *MockRBACService) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRBACService) RemovePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRBACService) AssignRoleToUser(ctx context.Context, assignment *UserRoleAssignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockRBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID, orgID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID, orgID)
	return args.Error(0)
}

func (m *MockRBACService) GetUserRoles(ctx context.Context, userID, orgID uuid.UUID) ([]*Role, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Role), args.Error(1)
}

func (m *MockRBACService) GetUserAssignments(ctx context.Context, userID uuid.UUID) ([]*UserRoleAssignment, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserRoleAssignment), args.Error(1)
}

func (m *MockRBACService) HasPermission(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, permission string, resourcePath *string) (bool, error) {
	args := m.Called(ctx, userID, orgID, permission, resourcePath)
	return args.Bool(0), args.Error(1)
}

func (m *MockRBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID, orgID uuid.UUID, resourcePath *string) ([]string, error) {
	args := m.Called(ctx, userID, orgID, resourcePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRBACService) GrantEmergencyAccess(ctx context.Context, access *EmergencyAccess) error {
	args := m.Called(ctx, access)
	return args.Error(0)
}

func (m *MockRBACService) RevokeEmergencyAccess(ctx context.Context, accessID uuid.UUID, revokedBy uuid.UUID, reason string) error {
	args := m.Called(ctx, accessID, revokedBy, reason)
	return args.Error(0)
}

func (m *MockRBACService) GetActiveEmergencyAccess(ctx context.Context, userID, orgID uuid.UUID) ([]*EmergencyAccess, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAccess), args.Error(1)
}

func (m *MockRBACService) CreateEmergencyAccessRequest(ctx context.Context, request *EmergencyAccessRequest) error {
	args := m.Called(ctx, request)
	return args.Error(0)
}

func (m *MockRBACService) ApproveEmergencyAccessRequest(ctx context.Context, requestID, approverID uuid.UUID, action EmergencyAccessApprovalAction, reason string) error {
	args := m.Called(ctx, requestID, approverID, action, reason)
	return args.Error(0)
}

func (m *MockRBACService) GetEmergencyAccessRequest(ctx context.Context, requestID uuid.UUID) (*EmergencyAccessRequest, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EmergencyAccessRequest), args.Error(1)
}

func (m *MockRBACService) ListEmergencyAccessRequests(ctx context.Context, orgID uuid.UUID, status *EmergencyAccessRequestStatus, limit, offset int) ([]*EmergencyAccessRequest, error) {
	args := m.Called(ctx, orgID, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAccessRequest), args.Error(1)
}

func (m *MockRBACService) GetEmergencyAccessApprovals(ctx context.Context, requestID uuid.UUID) ([]*EmergencyAccessApproval, error) {
	args := m.Called(ctx, requestID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*EmergencyAccessApproval), args.Error(1)
}

func (m *MockRBACService) ProcessBreakGlassAccess(ctx context.Context, requestID uuid.UUID) error {
	args := m.Called(ctx, requestID)
	return args.Error(0)
}

func (m *MockRBACService) ProcessEmergencyAccessEscalations(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRBACService) GetBreakGlassConfig(ctx context.Context, orgID uuid.UUID) (*BreakGlassConfig, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BreakGlassConfig), args.Error(1)
}

func (m *MockRBACService) UpdateBreakGlassConfig(ctx context.Context, orgID uuid.UUID, config *BreakGlassConfig) error {
	args := m.Called(ctx, orgID, config)
	return args.Error(0)
}

func (m *MockRBACService) CreateResourceType(ctx context.Context, resourceType *ResourceType) error {
	args := m.Called(ctx, resourceType)
	return args.Error(0)
}

func (m *MockRBACService) GetResourceType(ctx context.Context, id uuid.UUID) (*ResourceType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ResourceType), args.Error(1)
}

func (m *MockRBACService) ListResourceTypes(ctx context.Context, orgID uuid.UUID) ([]*ResourceType, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ResourceType), args.Error(1)
}

func (m *MockRBACService) CreateResource(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockRBACService) GetResource(ctx context.Context, id uuid.UUID) (*Resource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Resource), args.Error(1)
}

func (m *MockRBACService) ListResources(ctx context.Context, orgID uuid.UUID, resourceTypeID *uuid.UUID, parentPath *string) ([]*Resource, error) {
	args := m.Called(ctx, orgID, resourceTypeID, parentPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Resource), args.Error(1)
}

func (m *MockRBACService) UpdateResource(ctx context.Context, resource *Resource) error {
	args := m.Called(ctx, resource)
	return args.Error(0)
}

func (m *MockRBACService) DeleteResource(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRBACService) LogAuditEvent(ctx context.Context, event *AuditEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRBACService) GetAuditLog(ctx context.Context, orgID uuid.UUID, filters AuditFilters) ([]*AuditEvent, error) {
	args := m.Called(ctx, orgID, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AuditEvent), args.Error(1)
}

func (m *MockRBACService) CreateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockRBACService) UpdateScheduledRoleAssignment(ctx context.Context, assignment *ScheduledRoleAssignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockRBACService) DeleteScheduledRoleAssignment(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRBACService) GetScheduledRoleAssignments(ctx context.Context, userID, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ScheduledRoleAssignment), args.Error(1)
}

func (m *MockRBACService) ListPendingActivations(ctx context.Context, orgID uuid.UUID) ([]*ScheduledRoleAssignment, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ScheduledRoleAssignment), args.Error(1)
}

func (m *MockRBACService) ListExpiredAssignments(ctx context.Context, orgID uuid.UUID) ([]*UserRoleAssignment, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*UserRoleAssignment), args.Error(1)
}

func (m *MockRBACService) ProcessScheduledActivations(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRBACService) CleanupExpiredAssignments(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRBACService) GetTimeBasedAccessStatus(ctx context.Context, orgID uuid.UUID) (*TimeBasedAccessStatus, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TimeBasedAccessStatus), args.Error(1)
}
