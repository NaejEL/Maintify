package rbac

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestHandlers_NoAuthContext(t *testing.T) {
	handler, _ := setupHandlerTest(t)

	// Helper to replace path variables with valid UUIDs
	replacePathVars := func(path string) string {
		parts := strings.Split(path, "/")
		for i, part := range parts {
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				parts[i] = uuid.New().String()
			}
		}
		return strings.Join(parts, "/")
	}

	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		// User handlers
		{"CreateUser", "POST", "/users", "{}", handler.CreateUser},
		{"GetUser", "GET", "/users/{userId}", "", handler.GetUser},
		{"UpdateUser", "PUT", "/users/{userId}", "{}", handler.UpdateUser},
		{"DeactivateUser", "DELETE", "/users/{userId}", "", handler.DeactivateUser},
		{"ListUsers", "GET", "/users", "", handler.ListUsers},

		// Organization handlers
		// CreateOrganization does not check auth (public/system)
		{"GetOrganization", "GET", "/organizations/{orgId}", "", handler.GetOrganization},
		{"UpdateOrganization", "PUT", "/organizations/{orgId}", "{}", handler.UpdateOrganization},
		{"DeleteOrganization", "DELETE", "/organizations/{orgId}", "", handler.DeleteOrganization},
		{"ListOrganizations", "GET", "/organizations", "", handler.ListOrganizations},

		// Role handlers
		{"CreateRole", "POST", "/roles", "{}", handler.CreateRole},
		{"GetRole", "GET", "/roles/{roleId}", "", handler.GetRole},
		{"UpdateRole", "PUT", "/roles/{roleId}", "{}", handler.UpdateRole},
		{"DeleteRole", "DELETE", "/roles/{roleId}", "", handler.DeleteRole},
		{"ListRoles", "GET", "/roles", "", handler.ListRoles},

		// Permission handlers
		{"CreatePermission", "POST", "/permissions", "{}", handler.CreatePermission},
		{"GetPermission", "GET", "/permissions/{permissionId}", "", handler.GetPermission},
		{"ListPermissions", "GET", "/permissions", "", handler.ListPermissions},

		// Role assignment handlers
		{"AssignPermissionToRole", "POST", "/roles/{roleId}/permissions", "{}", handler.AssignPermissionToRole},
		{"RemovePermissionFromRole", "DELETE", "/roles/{roleId}/permissions/{permissionId}", "", handler.RemovePermissionFromRole},
		{"AssignRoleToUser", "POST", "/users/{userId}/roles", "{}", handler.AssignRoleToUser},
		{"RemoveRoleFromUser", "DELETE", "/users/{userId}/roles/{roleId}", "", handler.RemoveRoleFromUser},

		// Resource Type handlers
		{"CreateResourceType", "POST", "/resource-types", "{}", handler.CreateResourceType},
		{"GetResourceType", "GET", "/resource-types/{typeId}", "", handler.GetResourceType},
		{"ListResourceTypes", "GET", "/resource-types", "", handler.ListResourceTypes},

		// Resource handlers
		{"CreateResource", "POST", "/resources", "{}", handler.CreateResource},
		{"GetResource", "GET", "/resources/{resourceId}", "", handler.GetResource},
		{"UpdateResource", "PUT", "/resources/{resourceId}", "{}", handler.UpdateResource},
		{"DeleteResource", "DELETE", "/resources/{resourceId}", "", handler.DeleteResource},
		{"ListResources", "GET", "/resources", "", handler.ListResources},

		// Audit Log
		{"GetAuditLog", "GET", "/audit-log", "", handler.GetAuditLog},

		// Emergency Access
		{"GrantEmergencyAccess", "POST", "/emergency-access/grant", "{}", handler.GrantEmergencyAccess},
		{"RevokeEmergencyAccess", "POST", "/emergency-access/revoke", "{}", handler.RevokeEmergencyAccess},
		{"GetUserEmergencyAccess", "GET", "/users/{userId}/emergency-access", "", handler.GetUserEmergencyAccess},
		{"CreateEmergencyAccessRequest", "POST", "/emergency-access/requests", "{}", handler.CreateEmergencyAccessRequest},
		{"ListEmergencyAccessRequests", "GET", "/emergency-access/requests", "", handler.ListEmergencyAccessRequests},
		{"GetEmergencyAccessRequest", "GET", "/emergency-access/requests/{requestId}", "", handler.GetEmergencyAccessRequest},
		{"ApproveEmergencyAccessRequest", "POST", "/emergency-access/requests/{requestId}/approve", "{}", handler.ApproveEmergencyAccessRequest},
		{"GetEmergencyAccessApprovals", "GET", "/emergency-access/requests/{requestId}/approvals", "", handler.GetEmergencyAccessApprovals},
		{"ProcessBreakGlassAccess", "POST", "/emergency-access/requests/{requestId}/break-glass", "", handler.ProcessBreakGlassAccess},
		{"GetBreakGlassConfig", "GET", "/emergency-access/break-glass/config", "", handler.GetBreakGlassConfig},
		{"UpdateBreakGlassConfig", "PUT", "/emergency-access/break-glass/config", "{}", handler.UpdateBreakGlassConfig},

		// Scheduled Access
		{"CreateScheduledRoleAssignment", "POST", "/scheduled-access", "{}", handler.CreateScheduledRoleAssignment},
		{"UpdateScheduledRoleAssignment", "PUT", "/scheduled-access/{assignmentId}", "{}", handler.UpdateScheduledRoleAssignment},
		{"DeleteScheduledRoleAssignment", "DELETE", "/scheduled-access/{assignmentId}", "", handler.DeleteScheduledRoleAssignment},
		{"GetUserScheduledAssignments", "GET", "/users/{userId}/scheduled-access", "", handler.GetUserScheduledAssignments},
		{"ListPendingActivations", "GET", "/scheduled-access/pending", "", handler.ListPendingActivations},
		{"ListExpiredAssignments", "GET", "/scheduled-access/expired", "", handler.ListExpiredAssignments},
		{"GetTimeBasedAccessStatus", "GET", "/scheduled-access/status", "", handler.GetTimeBasedAccessStatus},
		{"ProcessScheduledActivations", "POST", "/scheduled-access/process", "", handler.ProcessScheduledActivations},
		{"CleanupExpiredAssignments", "POST", "/scheduled-access/cleanup", "", handler.CleanupExpiredAssignments},

		// User Roles/Permissions
		{"GetUserRoles", "GET", "/users/{userId}/roles", "", handler.GetUserRoles},
		{"GetUserPermissions", "GET", "/users/{userId}/permissions", "", handler.GetUserPermissions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqPath := replacePathVars(tt.path)
			var req *http.Request
			if tt.body != "" {
				req, _ = http.NewRequest(tt.method, reqPath, bytes.NewBufferString(tt.body))
			} else {
				req, _ = http.NewRequest(tt.method, reqPath, nil)
			}

			// Do NOT inject AuthContext

			rr := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc(tt.path, tt.handler)
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code, "Handler %s should return 401 when no auth context is present", tt.name)
		})
	}
}
