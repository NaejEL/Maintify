package rbac

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestHandler_SetupRoutes(t *testing.T) {
	mockService := new(MockRBACService)
	authService := NewAuthService(mockService, []byte("secret"))
	handler := NewHandler(mockService, authService)
	middleware := NewAuthMiddleware(authService, mockService)

	r := mux.NewRouter()
	handler.SetupRoutes(r, middleware)

	// Helper to check if a route exists
	checkRoute := func(method, path string) {
		req, _ := http.NewRequest(method, path, nil)
		match := &mux.RouteMatch{}
		assert.True(t, r.Match(req, match), "Route not found: %s %s", method, path)
	}

	// Public routes
	checkRoute("POST", "/auth/login")

	// User management
	checkRoute("POST", "/users")
	checkRoute("GET", "/users")
	checkRoute("GET", "/users/123")
	checkRoute("PUT", "/users/123")
	checkRoute("POST", "/users/123/deactivate")
	checkRoute("GET", "/users/123/roles")
	checkRoute("GET", "/users/123/permissions")

	// Organization management
	checkRoute("POST", "/organizations")
	checkRoute("GET", "/organizations")
	checkRoute("GET", "/organizations/123")

	// Organization-specific routes
	checkRoute("PUT", "/organizations/123")
	checkRoute("DELETE", "/organizations/123")

	// Role management
	checkRoute("POST", "/organizations/123/roles")
	checkRoute("GET", "/organizations/123/roles")
	checkRoute("GET", "/organizations/123/roles/456")
	checkRoute("PUT", "/organizations/123/roles/456")
	checkRoute("DELETE", "/organizations/123/roles/456")
	checkRoute("POST", "/organizations/123/roles/456/permissions")
	checkRoute("DELETE", "/organizations/123/roles/456/permissions/789")

	// Permission management
	checkRoute("POST", "/organizations/123/permissions")
	checkRoute("GET", "/organizations/123/permissions")
	checkRoute("GET", "/organizations/123/permissions/456")

	// Role assignments
	checkRoute("POST", "/organizations/123/assignments")
	checkRoute("DELETE", "/organizations/123/assignments/456/789")

	// Resource management
	checkRoute("POST", "/organizations/123/resource-types")
	checkRoute("GET", "/organizations/123/resource-types")
	checkRoute("GET", "/organizations/123/resource-types/456")

	checkRoute("POST", "/organizations/123/resources")
	checkRoute("GET", "/organizations/123/resources")
	checkRoute("GET", "/organizations/123/resources/456")
	checkRoute("PUT", "/organizations/123/resources/456")
	checkRoute("DELETE", "/organizations/123/resources/456")

	// Emergency access
	checkRoute("POST", "/organizations/123/emergency-access")
	checkRoute("POST", "/organizations/123/emergency-access/456/revoke")
	checkRoute("GET", "/organizations/123/emergency-access/users/789")

	// Emergency requests
	checkRoute("POST", "/organizations/123/emergency-requests")
	checkRoute("GET", "/organizations/123/emergency-requests")
	checkRoute("GET", "/organizations/123/emergency-requests/456")
	checkRoute("POST", "/organizations/123/emergency-requests/456/approve")
	checkRoute("GET", "/organizations/123/emergency-requests/456/approvals")
	checkRoute("POST", "/organizations/123/emergency-requests/456/break-glass")

	// Break-glass config
	checkRoute("GET", "/organizations/123/break-glass-config")
	checkRoute("PUT", "/organizations/123/break-glass-config")

	// Audit log
	checkRoute("GET", "/organizations/123/audit")

	// Scheduled assignments
	checkRoute("POST", "/organizations/123/scheduled-assignments")
	checkRoute("PUT", "/organizations/123/scheduled-assignments/456")
	checkRoute("DELETE", "/organizations/123/scheduled-assignments/456")
	checkRoute("GET", "/organizations/123/scheduled-assignments/users/789")
	checkRoute("GET", "/organizations/123/scheduled-assignments/pending")
	checkRoute("GET", "/organizations/123/scheduled-assignments/expired")
	checkRoute("GET", "/organizations/123/scheduled-assignments/status")
	checkRoute("POST", "/organizations/123/scheduled-assignments/process")
	checkRoute("POST", "/organizations/123/scheduled-assignments/cleanup")
}
