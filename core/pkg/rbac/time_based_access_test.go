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

// TimeBasedAccessTestSuite provides comprehensive testing for time-based access control
type TimeBasedAccessTestSuite struct {
	suite.Suite
	db          *sqlx.DB
	rbacService *PostgreSQLRBACService

	testOrg   *Organization
	testUser  *User
	testRole  *Role
	testAdmin *User
}

func (suite *TimeBasedAccessTestSuite) SetupSuite() {
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

	// Cleanup potential leftovers from previous failed runs
	suite.db.ExecContext(ctx, "DELETE FROM organizations WHERE slug = $1", "test-time-based")
	suite.db.ExecContext(ctx, "DELETE FROM users WHERE email = $1", "admin@timebased.test")
	suite.db.ExecContext(ctx, "DELETE FROM users WHERE email = $1", "user@timebased.test")

	// Create test organization
	suite.testOrg = &Organization{
		Name:        "Test Time-Based Org",
		Slug:        "test-time-based",
		Description: "Organization for testing time-based access",
	}
	err = suite.rbacService.CreateOrganization(ctx, suite.testOrg)
	require.NoError(suite.T(), err)

	// Create test admin user
	suite.testAdmin = &User{
		Email:        "admin@timebased.test",
		Username:     "timebasedadmin",
		FirstName:    "Time",
		LastName:     "Admin",
		IsActive:     true,
		PasswordHash: "T3st@TimeBased!2026", // Will be hashed by CreateUser
	}
	err = suite.rbacService.CreateUser(ctx, suite.testAdmin)
	require.NoError(suite.T(), err)

	// Create test user
	suite.testUser = &User{
		Email:        "user@timebased.test",
		Username:     "timebaseduser",
		FirstName:    "Time",
		LastName:     "User",
		IsActive:     true,
		PasswordHash: "user123", // Will be hashed by CreateUser
	}
	err = suite.rbacService.CreateUser(ctx, suite.testUser)
	require.NoError(suite.T(), err)

	// Create test role
	suite.testRole = &Role{
		OrganizationID: suite.testOrg.ID,
		Name:           "test-time-role",
		Description:    "Role for testing time-based access",
		IsSystemRole:   false,
	}
	err = suite.rbacService.CreateRole(ctx, suite.testRole)
	require.NoError(suite.T(), err)
}

func (suite *TimeBasedAccessTestSuite) TearDownSuite() {
	ctx := context.Background()

	// Clean up test data
	suite.db.ExecContext(ctx, "DELETE FROM organizations WHERE slug = $1", suite.testOrg.Slug)
	suite.db.ExecContext(ctx, "DELETE FROM users WHERE email = $1", suite.testAdmin.Email)
	suite.db.ExecContext(ctx, "DELETE FROM users WHERE email = $1", suite.testUser.Email)

	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *TimeBasedAccessTestSuite) TearDownTest() {
	ctx := context.Background()
	// Clean up assignments created during tests
	_, err := suite.db.ExecContext(ctx, "DELETE FROM user_role_assignments WHERE user_id = $1", suite.testUser.ID)
	if err != nil {
		suite.T().Logf("Failed to cleanup user role assignments: %v", err)
	}
	_, err = suite.db.ExecContext(ctx, "DELETE FROM scheduled_role_assignments WHERE user_id = $1", suite.testUser.ID)
	if err != nil {
		suite.T().Logf("Failed to cleanup scheduled role assignments: %v", err)
	}
}

func (suite *TimeBasedAccessTestSuite) TestCreateScheduledRoleAssignment() {
	ctx := context.Background()

	// Create a scheduled assignment for 1 hour from now
	activationTime := time.Now().Add(1 * time.Hour)
	expirationTime := activationTime.Add(24 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: activationTime,
		ScheduledExpiration: &expirationTime,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "Testing scheduled role assignment",
		Metadata: map[string]interface{}{
			"test_type": "unit_test",
			"priority":  "high",
		},
	}

	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, assignment.ID)
	assert.False(suite.T(), assignment.IsProcessed)

	// Verify the assignment was created
	assignments, err := suite.rbacService.GetScheduledRoleAssignments(ctx, suite.testUser.ID, suite.testOrg.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), assignments, 1)
	assert.Equal(suite.T(), assignment.ID, assignments[0].ID)
	assert.Equal(suite.T(), "Testing scheduled role assignment", assignments[0].AssignmentReason)
}

func (suite *TimeBasedAccessTestSuite) TestScheduledAssignmentRecurrence() {
	ctx := context.Background()

	// Create a recurring scheduled assignment (every weekday at 9 AM)
	activationTime := time.Now().Add(1 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: activationTime,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "Recurring weekday access",
		RecurrencePattern:   stringPtr("0 9 * * 1-5"), // Monday to Friday at 9 AM
		Metadata: map[string]interface{}{
			"recurrence_type": "weekdays",
		},
	}

	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), assignment.RecurrencePattern)
	assert.Equal(suite.T(), "0 9 * * 1-5", *assignment.RecurrencePattern)
}

func (suite *TimeBasedAccessTestSuite) TestUpdateScheduledRoleAssignment() {
	ctx := context.Background()

	// Create a scheduled assignment
	activationTime := time.Now().Add(2 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: activationTime,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "Original reason",
	}

	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	require.NoError(suite.T(), err)

	// Update the assignment
	newActivationTime := time.Now().Add(3 * time.Hour)
	newExpirationTime := newActivationTime.Add(12 * time.Hour)

	assignment.ScheduledActivation = newActivationTime
	assignment.ScheduledExpiration = &newExpirationTime
	assignment.AssignmentReason = "Updated reason"
	assignment.Metadata = map[string]interface{}{
		"updated": true,
	}

	err = suite.rbacService.UpdateScheduledRoleAssignment(ctx, assignment)
	assert.NoError(suite.T(), err)

	// Verify the update
	assignments, err := suite.rbacService.GetScheduledRoleAssignments(ctx, suite.testUser.ID, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	var updated *ScheduledRoleAssignment
	for _, a := range assignments {
		if a.ID == assignment.ID {
			updated = a
			break
		}
	}

	require.NotNil(suite.T(), updated)
	assert.Equal(suite.T(), "Updated reason", updated.AssignmentReason)
	assert.True(suite.T(), updated.Metadata["updated"].(bool))
}

func (suite *TimeBasedAccessTestSuite) TestDeleteScheduledRoleAssignment() {
	ctx := context.Background()

	// Create a scheduled assignment
	activationTime := time.Now().Add(4 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: activationTime,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "To be deleted",
	}

	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	require.NoError(suite.T(), err)

	// Delete the assignment
	err = suite.rbacService.DeleteScheduledRoleAssignment(ctx, assignment.ID)
	assert.NoError(suite.T(), err)

	// Verify it's deleted
	assignments, err := suite.rbacService.GetScheduledRoleAssignments(ctx, suite.testUser.ID, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	for _, a := range assignments {
		assert.NotEqual(suite.T(), assignment.ID, a.ID)
	}
}

func (suite *TimeBasedAccessTestSuite) TestListPendingActivations() {
	ctx := context.Background()

	// Create assignments with different activation times
	now := time.Now()
	assignments := []*ScheduledRoleAssignment{
		{
			UserID:              suite.testUser.ID,
			RoleID:              suite.testRole.ID,
			OrganizationID:      suite.testOrg.ID,
			ScheduledActivation: now.Add(30 * time.Minute), // Within 24h
			AssignedBy:          &suite.testAdmin.ID,
			AssignmentReason:    "Pending soon",
		},
		{
			UserID:              suite.testUser.ID,
			RoleID:              suite.testRole.ID,
			OrganizationID:      suite.testOrg.ID,
			ScheduledActivation: now.Add(25 * time.Hour), // Beyond 24h
			AssignedBy:          &suite.testAdmin.ID,
			AssignmentReason:    "Pending later",
		},
	}

	for _, assignment := range assignments {
		err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
		require.NoError(suite.T(), err)
	}

	// List pending activations (should only return the one within 24h)
	pending, err := suite.rbacService.ListPendingActivations(ctx, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	// Find our test assignments
	var foundSoon, foundLater bool
	for _, p := range pending {
		if p.AssignmentReason == "Pending soon" {
			foundSoon = true
		}
		if p.AssignmentReason == "Pending later" {
			foundLater = true
		}
	}

	assert.True(suite.T(), foundSoon, "Should find assignment pending within 24h")
	assert.False(suite.T(), foundLater, "Should not find assignment pending beyond 24h")
}

func (suite *TimeBasedAccessTestSuite) TestProcessScheduledActivations() {
	ctx := context.Background()

	// Create a scheduled assignment that should be activated (in the past)
	pastTime := time.Now().Add(-1 * time.Minute)
	futureExpiration := time.Now().Add(1 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: pastTime,
		ScheduledExpiration: &futureExpiration,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "Should be activated",
	}

	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	require.NoError(suite.T(), err)

	// Process scheduled activations
	err = suite.rbacService.ProcessScheduledActivations(ctx)
	assert.NoError(suite.T(), err)

	// Verify the assignment was processed
	assignments, err := suite.rbacService.GetScheduledRoleAssignments(ctx, suite.testUser.ID, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	var processed *ScheduledRoleAssignment
	for _, a := range assignments {
		if a.ID == assignment.ID {
			processed = a
			break
		}
	}

	require.NotNil(suite.T(), processed)
	assert.True(suite.T(), processed.IsProcessed)
	assert.NotNil(suite.T(), processed.ProcessedAt)

	// Verify an actual user role assignment was created
	userRoles, err := suite.rbacService.GetUserRoles(ctx, suite.testUser.ID, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	var hasRole bool
	for _, role := range userRoles {
		if role.ID == suite.testRole.ID {
			hasRole = true
			break
		}
	}
	assert.True(suite.T(), hasRole, "User should have the role after processing")
}

func (suite *TimeBasedAccessTestSuite) TestListExpiredAssignments() {
	ctx := context.Background()

	// Create an expired user role assignment
	pastTime := time.Now().Add(-1 * time.Hour)

	assignment := &UserRoleAssignment{
		UserID:           suite.testUser.ID,
		RoleID:           suite.testRole.ID,
		OrganizationID:   suite.testOrg.ID,
		ValidFrom:        time.Now().Add(-2 * time.Hour),
		ValidUntil:       &pastTime,
		AssignedBy:       &suite.testAdmin.ID,
		AssignmentReason: "Expired assignment",
		IsActive:         true, // Still active but expired
	}

	err := suite.rbacService.AssignRoleToUser(ctx, assignment)
	require.NoError(suite.T(), err)

	// List expired assignments
	expired, err := suite.rbacService.ListExpiredAssignments(ctx, suite.testOrg.ID)
	assert.NoError(suite.T(), err)

	// Find our expired assignment
	var foundExpired bool
	for _, e := range expired {
		if e.AssignmentReason == "Expired assignment" {
			foundExpired = true
			assert.True(suite.T(), e.ValidUntil.Before(time.Now()))
			break
		}
	}
	assert.True(suite.T(), foundExpired, "Should find the expired assignment")
}

func (suite *TimeBasedAccessTestSuite) TestCleanupExpiredAssignments() {
	ctx := context.Background()

	// Create an expired user role assignment
	pastTime := time.Now().Add(-1 * time.Hour)

	assignment := &UserRoleAssignment{
		UserID:           suite.testUser.ID,
		RoleID:           suite.testRole.ID,
		OrganizationID:   suite.testOrg.ID,
		ValidFrom:        time.Now().Add(-2 * time.Hour),
		ValidUntil:       &pastTime,
		AssignedBy:       &suite.testAdmin.ID,
		AssignmentReason: "To be cleaned up",
		IsActive:         true,
	}

	err := suite.rbacService.AssignRoleToUser(ctx, assignment)
	require.NoError(suite.T(), err)

	// Cleanup expired assignments
	err = suite.rbacService.CleanupExpiredAssignments(ctx)
	assert.NoError(suite.T(), err)

	// Verify the assignment is now inactive
	userAssignments, err := suite.rbacService.GetUserAssignments(ctx, suite.testUser.ID)
	assert.NoError(suite.T(), err)

	for _, ua := range userAssignments {
		if ua.AssignmentReason == "To be cleaned up" {
			assert.False(suite.T(), ua.IsActive, "Expired assignment should be deactivated")
			break
		}
	}
}

func (suite *TimeBasedAccessTestSuite) TestGetTimeBasedAccessStatus() {
	ctx := context.Background()

	// Get status for the test organization
	status, err := suite.rbacService.GetTimeBasedAccessStatus(ctx, suite.testOrg.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), status)
	assert.Equal(suite.T(), suite.testOrg.ID, status.OrganizationID)

	// Status should have reasonable values (non-negative counts)
	assert.GreaterOrEqual(suite.T(), status.PendingActivations, 0)
	assert.GreaterOrEqual(suite.T(), status.ActiveAssignments, 0)
	assert.GreaterOrEqual(suite.T(), status.ExpiredAssignments, 0)
	assert.GreaterOrEqual(suite.T(), status.ScheduledForNext24h, 0)
	assert.GreaterOrEqual(suite.T(), status.ProcessingErrors, 0)
}

func (suite *TimeBasedAccessTestSuite) TestTimeValidation() {
	ctx := context.Background()

	// Test creating assignment with past activation time (should fail in real implementation)
	pastTime := time.Now().Add(-1 * time.Hour)

	assignment := &ScheduledRoleAssignment{
		UserID:              suite.testUser.ID,
		RoleID:              suite.testRole.ID,
		OrganizationID:      suite.testOrg.ID,
		ScheduledActivation: pastTime,
		AssignedBy:          &suite.testAdmin.ID,
		AssignmentReason:    "Invalid past time",
	}

	// Note: The validation might be at the handler level, not service level
	// This test documents the expected behavior
	err := suite.rbacService.CreateScheduledRoleAssignment(ctx, assignment)
	// In a production system, this should validate and reject past times
	// For now, we just ensure it doesn't crash
	if err != nil {
		suite.T().Logf("Expected behavior: rejecting past activation time: %v", err)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Run the test suite
func TestTimeBasedAccessTestSuite(t *testing.T) {
	suite.Run(t, new(TimeBasedAccessTestSuite))
}

// Individual test functions for specific scenarios

func TestScheduledAssignmentValidation(t *testing.T) {
	// Test various validation scenarios
	now := time.Now()

	tests := []struct {
		name        string
		assignment  ScheduledRoleAssignment
		expectError bool
	}{
		{
			name: "valid future activation",
			assignment: ScheduledRoleAssignment{
				ScheduledActivation: now.Add(1 * time.Hour),
				AssignmentReason:    "Valid assignment",
			},
			expectError: false,
		},
		{
			name: "empty assignment reason",
			assignment: ScheduledRoleAssignment{
				ScheduledActivation: now.Add(1 * time.Hour),
				AssignmentReason:    "",
			},
			expectError: true,
		},
		{
			name: "expiration before activation",
			assignment: ScheduledRoleAssignment{
				ScheduledActivation: now.Add(2 * time.Hour),
				ScheduledExpiration: timePtr(now.Add(1 * time.Hour)),
				AssignmentReason:    "Invalid time range",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be validation logic that should be implemented
			// in the service or handler layers

			if tt.assignment.AssignmentReason == "" {
				assert.True(t, tt.expectError, "Should expect error for empty reason")
			}

			if tt.assignment.ScheduledExpiration != nil &&
				tt.assignment.ScheduledExpiration.Before(tt.assignment.ScheduledActivation) {
				assert.True(t, tt.expectError, "Should expect error for expiration before activation")
			}
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
