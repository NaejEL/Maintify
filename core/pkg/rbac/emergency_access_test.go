//go:build integration

package rbac

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// EmergencyAccessTestSuite provides comprehensive testing for emergency access system
type EmergencyAccessTestSuite struct {
	suite.Suite
	db          *sqlx.DB
	rbacService *PostgreSQLRBACService

	testOrg      *Organization
	testUser     *User
	testApprover *User
	testAdmin    *User
}

func (suite *EmergencyAccessTestSuite) SetupSuite() {
	// Database connection setup
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "postgres"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "maintify"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "maintify"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "maintify_core"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(suite.T(), err)

	suite.db = db
	suite.rbacService = NewPostgreSQLRBACService(db)

	ctx := context.Background()

	// Create test organization
	suite.testOrg = &Organization{
		Name:        "Test Emergency Org",
		Slug:        "test-emergency-" + uuid.New().String()[:8],
		Description: "Organization for testing emergency access",
	}
	err = suite.rbacService.CreateOrganization(ctx, suite.testOrg)
	require.NoError(suite.T(), err)

	// Create test admin user
	uniqueId := uuid.New().String()[:8]
	suite.testAdmin = &User{
		Email:         "admin-" + uniqueId + "@emergency.test",
		Username:      "emergencyadmin-" + uniqueId,
		FirstName:     "Emergency",
		LastName:      "Admin",
		IsSystemAdmin: true,
		IsActive:      true,
		PasswordHash:  "T3st@Emergency!" + uniqueId,
	}
	err = suite.rbacService.CreateUser(ctx, suite.testAdmin)
	require.NoError(suite.T(), err)

	// Create test user
	suite.testUser = &User{
		Email:        "user-" + uniqueId + "@emergency.test",
		Username:     "emergencyuser-" + uniqueId,
		FirstName:    "Emergency",
		LastName:     "User",
		IsActive:     true,
		PasswordHash: "user123",
	}
	err = suite.rbacService.CreateUser(ctx, suite.testUser)
	require.NoError(suite.T(), err)

	// Create test approver
	suite.testApprover = &User{
		Email:        "approver-" + uniqueId + "@emergency.test",
		Username:     "emergencyapprover-" + uniqueId,
		FirstName:    "Emergency",
		LastName:     "Approver",
		IsActive:     true,
		PasswordHash: "approver123",
	}
	err = suite.rbacService.CreateUser(ctx, suite.testApprover)
	require.NoError(suite.T(), err)
}

func (suite *EmergencyAccessTestSuite) TearDownSuite() {
	ctx := context.Background()

	// Clean up test data
	suite.db.ExecContext(ctx, "DELETE FROM organizations WHERE slug = $1", suite.testOrg.Slug)

	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *EmergencyAccessTestSuite) TestCreateEmergencyAccessRequest() {
	ctx := context.Background()

	request := &EmergencyAccessRequest{
		UserID:               suite.testUser.ID,
		OrganizationID:       suite.testOrg.ID,
		RequestedPermissions: []string{"resource.manage", "user.admin"},
		Reason:               "Critical system maintenance required",
		UrgencyLevel:         EmergencyUrgencyHigh,
		RequestedDuration:    int64(2 * time.Hour),
		BreakGlass:           false,
		RequiredApprovals:    1,
		Metadata: map[string]interface{}{
			"incident_id":  "INC-12345",
			"escalated_by": "john.doe",
		},
	}

	err := suite.rbacService.CreateEmergencyAccessRequest(ctx, request)
	require.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, request.ID)
	assert.Equal(suite.T(), EmergencyAccessRequestStatusPending, request.Status)
	assert.NotNil(suite.T(), request.ExpiresAt)
}

func (suite *EmergencyAccessTestSuite) TestApprovalWorkflow() {
	ctx := context.Background()

	// Create request
	request := &EmergencyAccessRequest{
		UserID:               suite.testUser.ID,
		OrganizationID:       suite.testOrg.ID,
		RequestedPermissions: []string{"system.admin"},
		Reason:               "Database recovery needed",
		UrgencyLevel:         EmergencyUrgencyMedium,
		RequestedDuration:    int64(1 * time.Hour),
		RequiredApprovals:    1,
	}

	err := suite.rbacService.CreateEmergencyAccessRequest(ctx, request)
	require.NoError(suite.T(), err)

	// Approve the request
	err = suite.rbacService.ApproveEmergencyAccessRequest(ctx, request.ID, suite.testApprover.ID,
		EmergencyAccessApprovalActionApprove, "Approved for database recovery")
	require.NoError(suite.T(), err)

	// Check request status
	updatedRequest, err := suite.rbacService.GetEmergencyAccessRequest(ctx, request.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), EmergencyAccessRequestStatusGranted, updatedRequest.Status)
	assert.NotNil(suite.T(), updatedRequest.EmergencyAccessID)

	// Verify emergency access was created
	accesses, err := suite.rbacService.GetActiveEmergencyAccess(ctx, suite.testUser.ID, suite.testOrg.ID)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(accesses), 1)

	// Find our specific access
	found := false
	for _, access := range accesses {
		if access.ID == *updatedRequest.EmergencyAccessID {
			found = true
			assert.Equal(suite.T(), []string{"system.admin"}, access.GrantedPermissions)
			break
		}
	}
	assert.True(suite.T(), found, "Emergency access should be created")

	// Check approvals
	approvals, err := suite.rbacService.GetEmergencyAccessApprovals(ctx, request.ID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), approvals, 1)
	assert.Equal(suite.T(), suite.testApprover.ID, approvals[0].ApproverID)
	assert.Equal(suite.T(), EmergencyAccessApprovalActionApprove, approvals[0].Action)
}

func (suite *EmergencyAccessTestSuite) TestBreakGlassAccess() {
	ctx := context.Background()

	// Configure break-glass for the organization
	config := &BreakGlassConfig{
		Enabled:             true,
		AutoApprovalUrgency: EmergencyUrgencyCritical,
		MaxDuration:         4 * time.Hour,
		RequiredPermissions: []string{"emergency.break_glass"},
		ApprovalRequirements: map[EmergencyUrgencyLevel]int{
			EmergencyUrgencyCritical: 0, // Auto-approve critical
			EmergencyUrgencyHigh:     1,
			EmergencyUrgencyMedium:   2,
			EmergencyUrgencyLow:      2,
		},
		AutoRevocationMinutes: 60,
		NotificationChannels:  []string{"slack", "email"},
	}

	err := suite.rbacService.UpdateBreakGlassConfig(ctx, suite.testOrg.ID, config)
	require.NoError(suite.T(), err)

	// Verify config was saved
	savedConfig, err := suite.rbacService.GetBreakGlassConfig(ctx, suite.testOrg.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), config.Enabled, savedConfig.Enabled)
	assert.Equal(suite.T(), config.AutoApprovalUrgency, savedConfig.AutoApprovalUrgency)

	// Create critical break-glass request
	request := &EmergencyAccessRequest{
		UserID:               suite.testUser.ID,
		OrganizationID:       suite.testOrg.ID,
		RequestedPermissions: []string{"system.root", "security.override"},
		Reason:               "CRITICAL: Production system compromised, immediate access needed",
		UrgencyLevel:         EmergencyUrgencyCritical,
		RequestedDuration:    int64(1 * time.Hour),
		BreakGlass:           true,
	}

	err = suite.rbacService.CreateEmergencyAccessRequest(ctx, request)
	require.NoError(suite.T(), err)

	// Should be auto-approved due to critical urgency
	updatedRequest, err := suite.rbacService.GetEmergencyAccessRequest(ctx, request.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), EmergencyAccessRequestStatusGranted, updatedRequest.Status)
	assert.NotNil(suite.T(), updatedRequest.AutoApprovedAt)
	assert.NotNil(suite.T(), updatedRequest.EmergencyAccessID)

	// Verify emergency access was granted
	accesses, err := suite.rbacService.GetActiveEmergencyAccess(ctx, suite.testUser.ID, suite.testOrg.ID)
	require.NoError(suite.T(), err)
	found := false
	for _, access := range accesses {
		if access.ID == *updatedRequest.EmergencyAccessID {
			found = true
			assert.Contains(suite.T(), access.Reason, "BREAK-GLASS:")
			assert.Equal(suite.T(), request.RequestedPermissions, access.GrantedPermissions)
			break
		}
	}
	assert.True(suite.T(), found, "Break-glass emergency access should be created")
}

func (suite *EmergencyAccessTestSuite) TestListEmergencyAccessRequests() {
	ctx := context.Background()

	// Create multiple requests with different statuses
	requests := []*EmergencyAccessRequest{
		{
			UserID:               suite.testUser.ID,
			OrganizationID:       suite.testOrg.ID,
			RequestedPermissions: []string{"test.permission1"},
			Reason:               "Test request 1",
			UrgencyLevel:         EmergencyUrgencyLow,
			RequestedDuration:    int64(1 * time.Hour),
		},
		{
			UserID:               suite.testUser.ID,
			OrganizationID:       suite.testOrg.ID,
			RequestedPermissions: []string{"test.permission2"},
			Reason:               "Test request 2",
			UrgencyLevel:         EmergencyUrgencyHigh,
			RequestedDuration:    int64(30 * time.Minute),
		},
	}

	for _, req := range requests {
		err := suite.rbacService.CreateEmergencyAccessRequest(ctx, req)
		require.NoError(suite.T(), err)
	}

	// List all requests
	allRequests, err := suite.rbacService.ListEmergencyAccessRequests(ctx, suite.testOrg.ID, nil, 10, 0)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(allRequests), 2)

	// List only pending requests
	pendingStatus := EmergencyAccessRequestStatusPending
	pendingRequests, err := suite.rbacService.ListEmergencyAccessRequests(ctx, suite.testOrg.ID, &pendingStatus, 10, 0)
	require.NoError(suite.T(), err)

	for _, req := range pendingRequests {
		assert.Equal(suite.T(), EmergencyAccessRequestStatusPending, req.Status)
	}
}

// Test function that runs the suite
func TestEmergencyAccessWorkflows(t *testing.T) {
	suite.Run(t, new(EmergencyAccessTestSuite))
}
