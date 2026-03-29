package rbac

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for RBAC operations
type Handler struct {
	rbacService RBACService
	authService *AuthService
}

// NewHandler creates a new RBAC HTTP handler
func NewHandler(rbacService RBACService, authService *AuthService) *Handler {
	return &Handler{
		rbacService: rbacService,
		authService: authService,
	}
}

// SetupRoutes configures the HTTP routes for RBAC
func (h *Handler) SetupRoutes(r *mux.Router, middleware *AuthMiddleware) {
	// Public routes
	r.HandleFunc("/auth/login", h.Login).Methods("POST")

	// Protected routes
	protected := r.PathPrefix("").Subrouter()
	protected.Use(middleware.RequireAuth)
	protected.Use(middleware.AuditMiddleware)

	// User management
	users := protected.PathPrefix("/users").Subrouter()
	users.HandleFunc("", h.CreateUser).Methods("POST")
	users.HandleFunc("", h.ListUsers).Methods("GET")
	users.HandleFunc("/{userId}", h.GetUser).Methods("GET")
	users.HandleFunc("/{userId}", h.UpdateUser).Methods("PUT")
	users.HandleFunc("/{userId}/deactivate", h.DeactivateUser).Methods("POST")
	users.HandleFunc("/{userId}/roles", h.GetUserRoles).Methods("GET")
	users.HandleFunc("/{userId}/permissions", h.GetUserPermissions).Methods("GET")

	// Organization management (with permission middleware)
	orgs := protected.PathPrefix("/organizations").Subrouter()
	orgs.Handle("", middleware.RequirePermission(PermissionOrgCreate)(http.HandlerFunc(h.CreateOrganization))).Methods("POST")
	orgs.HandleFunc("", h.ListOrganizations).Methods("GET")
	orgs.HandleFunc("/{orgId}", h.GetOrganization).Methods("GET")

	// Organization-specific routes
	orgSpecific := orgs.PathPrefix("/{orgId}").Subrouter()
	orgSpecific.Use(middleware.RequireOrganizationAccess)
	orgSpecific.Handle("", middleware.RequirePermission(PermissionOrgManage)(http.HandlerFunc(h.UpdateOrganization))).Methods("PUT")
	orgSpecific.Handle("", middleware.RequirePermission(PermissionOrgManage)(http.HandlerFunc(h.DeleteOrganization))).Methods("DELETE")

	// Role management
	roles := orgSpecific.PathPrefix("/roles").Subrouter()
	roles.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.CreateRole))).Methods("POST")
	roles.HandleFunc("", h.ListRoles).Methods("GET")
	roles.HandleFunc("/{roleId}", h.GetRole).Methods("GET")
	roles.Handle("/{roleId}", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.UpdateRole))).Methods("PUT")
	roles.Handle("/{roleId}", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.DeleteRole))).Methods("DELETE")
	roles.Handle("/{roleId}/permissions", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.AssignPermissionToRole))).Methods("POST")
	roles.Handle("/{roleId}/permissions/{permissionId}", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.RemovePermissionFromRole))).Methods("DELETE")

	// Permission management
	permissions := orgSpecific.PathPrefix("/permissions").Subrouter()
	permissions.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.CreatePermission))).Methods("POST")
	permissions.HandleFunc("", h.ListPermissions).Methods("GET")
	permissions.HandleFunc("/{permissionId}", h.GetPermission).Methods("GET")

	// Role assignments
	assignments := orgSpecific.PathPrefix("/assignments").Subrouter()
	assignments.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.AssignRoleToUser))).Methods("POST")
	assignments.Handle("/{userId}/{roleId}", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.RemoveRoleFromUser))).Methods("DELETE")

	// Resource management
	resourceTypes := orgSpecific.PathPrefix("/resource-types").Subrouter()
	resourceTypes.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.CreateResourceType))).Methods("POST")
	resourceTypes.HandleFunc("", h.ListResourceTypes).Methods("GET")
	resourceTypes.HandleFunc("/{resourceTypeId}", h.GetResourceType).Methods("GET")

	resources := orgSpecific.PathPrefix("/resources").Subrouter()
	resources.HandleFunc("", h.CreateResource).Methods("POST")
	resources.HandleFunc("", h.ListResources).Methods("GET")
	resources.HandleFunc("/{resourceId}", h.GetResource).Methods("GET")
	resources.HandleFunc("/{resourceId}", h.UpdateResource).Methods("PUT")
	resources.HandleFunc("/{resourceId}", h.DeleteResource).Methods("DELETE")

	// Emergency access
	emergency := orgSpecific.PathPrefix("/emergency-access").Subrouter()
	emergency.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.GrantEmergencyAccess))).Methods("POST")
	emergency.Handle("/{accessId}/revoke", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.RevokeEmergencyAccess))).Methods("POST")
	emergency.HandleFunc("/users/{userId}", h.GetUserEmergencyAccess).Methods("GET")

	// Enhanced emergency access with break-glass
	emergencyRequests := orgSpecific.PathPrefix("/emergency-requests").Subrouter()
	emergencyRequests.HandleFunc("", h.CreateEmergencyAccessRequest).Methods("POST")
	emergencyRequests.HandleFunc("", h.ListEmergencyAccessRequests).Methods("GET")
	emergencyRequests.HandleFunc("/{requestId}", h.GetEmergencyAccessRequest).Methods("GET")
	emergencyRequests.Handle("/{requestId}/approve", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.ApproveEmergencyAccessRequest))).Methods("POST")
	emergencyRequests.HandleFunc("/{requestId}/approvals", h.GetEmergencyAccessApprovals).Methods("GET")
	emergencyRequests.Handle("/{requestId}/break-glass", middleware.RequirePermission("emergency.break_glass")(http.HandlerFunc(h.ProcessBreakGlassAccess))).Methods("POST")

	// Break-glass configuration
	breakGlass := orgSpecific.PathPrefix("/break-glass-config").Subrouter()
	breakGlass.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.GetBreakGlassConfig))).Methods("GET")
	breakGlass.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.UpdateBreakGlassConfig))).Methods("PUT")

	// Audit log
	audit := orgSpecific.PathPrefix("/audit").Subrouter()
	audit.Handle("", middleware.RequirePermission(PermissionRBACManage)(http.HandlerFunc(h.GetAuditLog))).Methods("GET")

	// Time-based access control
	scheduled := orgSpecific.PathPrefix("/scheduled-assignments").Subrouter()
	scheduled.Handle("", middleware.RequirePermission("schedule.manage")(http.HandlerFunc(h.CreateScheduledRoleAssignment))).Methods("POST")
	scheduled.Handle("/{assignmentId}", middleware.RequirePermission("schedule.manage")(http.HandlerFunc(h.UpdateScheduledRoleAssignment))).Methods("PUT")
	scheduled.Handle("/{assignmentId}", middleware.RequirePermission("schedule.manage")(http.HandlerFunc(h.DeleteScheduledRoleAssignment))).Methods("DELETE")
	scheduled.Handle("/users/{userId}", middleware.RequirePermission("schedule.view")(http.HandlerFunc(h.GetUserScheduledAssignments))).Methods("GET")
	scheduled.Handle("/pending", middleware.RequirePermission("schedule.view")(http.HandlerFunc(h.ListPendingActivations))).Methods("GET")
	scheduled.Handle("/expired", middleware.RequirePermission("schedule.view")(http.HandlerFunc(h.ListExpiredAssignments))).Methods("GET")
	scheduled.Handle("/status", middleware.RequirePermission("schedule.view")(http.HandlerFunc(h.GetTimeBasedAccessStatus))).Methods("GET")
	scheduled.Handle("/process", middleware.RequirePermission("temporal.admin")(http.HandlerFunc(h.ProcessScheduledActivations))).Methods("POST")
	scheduled.Handle("/cleanup", middleware.RequirePermission("temporal.admin")(http.HandlerFunc(h.CleanupExpiredAssignments))).Methods("POST")
}

// Auth handlers

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		WriteJSONError(w, http.StatusUnauthorized, "login_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, response)
}

// User handlers

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionUserManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if err := h.rbacService.CreateUser(r.Context(), &user); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	// Clear password hash from response
	user.PasswordHash = ""
	WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	// Check permission
	// We check permission using the authenticated user's organization context
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionUserView, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	userID, parseErr := uuid.Parse(userIDStr)

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	user, err := h.rbacService.GetUser(r.Context(), userID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	// Clear password hash from response
	user.PasswordHash = ""
	WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionUserManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	userID, parseErr := uuid.Parse(userIDStr)

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	user.ID = userID

	if err := h.rbacService.UpdateUser(r.Context(), &user); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	// Clear password hash from response
	user.PasswordHash = ""
	WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionUserManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	userID, parseErr := uuid.Parse(userIDStr)

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	if err := h.rbacService.DeactivateUser(r.Context(), userID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "deactivation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionUserView, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	// Get pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// Get organization filter if provided
	var orgID *uuid.UUID
	if orgIDStr := r.URL.Query().Get("org_id"); orgIDStr != "" {
		if id, err := uuid.Parse(orgIDStr); err == nil {
			orgID = &id
		}
	}

	// Check permission
	// Re-checking permission here is redundant if we checked at top of handler, removing duplicates

	users, err := h.rbacService.ListUsers(r.Context(), orgID, limit, offset)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	// Clear password hashes from response
	for _, user := range users {
		user.PasswordHash = ""
	}

	WriteJSON(w, http.StatusOK, users)
}

func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	roles, err := h.rbacService.GetUserRoles(r.Context(), userID, authCtx.OrganizationID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, roles)
}

func (h *Handler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["userId"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	permissions, err := h.rbacService.GetUserPermissions(r.Context(), userID, authCtx.OrganizationID, nil)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string][]string{"permissions": permissions})
}

// Organization handlers

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if err := h.rbacService.CreateOrganization(r.Context(), &org); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, org)
}

func (h *Handler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, parseErr := uuid.Parse(orgIDStr)

	// Permission check
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionOrgView, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	org, err := h.rbacService.GetOrganization(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "Organization not found")
		return
	}

	WriteJSON(w, http.StatusOK, org)
}

func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, parseErr := uuid.Parse(orgIDStr)

	// Permission check
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionOrgManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var org Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	org.ID = orgID
	if err := h.rbacService.UpdateOrganization(r.Context(), &org); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, org)
}

func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, parseErr := uuid.Parse(orgIDStr)

	// Permission check
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionOrgManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	if err := h.rbacService.DeleteOrganization(r.Context(), orgID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "deletion_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Get pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	orgs, err := h.rbacService.ListOrganizations(r.Context(), limit, offset)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, orgs)
}

// Role handlers

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var role Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	role.OrganizationID = orgID
	if err := h.rbacService.CreateRole(r.Context(), &role); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, role)
}

func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	roleIDStr := vars["roleId"]

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	role, err := h.rbacService.GetRole(r.Context(), roleID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "Role not found")
		return
	}

	WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	roleIDStr := vars["roleId"]

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	var role Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	role.ID = roleID
	if err := h.rbacService.UpdateRole(r.Context(), &role); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	roleIDStr := vars["roleId"]

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	if err := h.rbacService.DeleteRole(r.Context(), roleID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "deletion_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	// Get pagination parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	roles, err := h.rbacService.ListRoles(r.Context(), orgID, limit, offset)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, roles)
}

// Permission handlers

func (h *Handler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var permission Permission
	if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	permission.OrganizationID = orgID
	if err := h.rbacService.CreatePermission(r.Context(), &permission); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, permission)
}

func (h *Handler) GetPermission(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	permissionIDStr := vars["permissionId"]

	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid permission ID")
		return
	}

	permission, err := h.rbacService.GetPermission(r.Context(), permissionID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "Permission not found")
		return
	}

	WriteJSON(w, http.StatusOK, permission)
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	permissions, err := h.rbacService.ListPermissions(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, permissions)
}

// Assignment handlers

func (h *Handler) AssignPermissionToRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	roleIDStr := vars["roleId"]

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	var req struct {
		PermissionID uuid.UUID `json:"permission_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if err := h.rbacService.AssignPermissionToRole(r.Context(), roleID, req.PermissionID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "assignment_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (h *Handler) RemovePermissionFromRole(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	roleIDStr := vars["roleId"]
	permissionIDStr := vars["permissionId"]

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	permissionID, err := uuid.Parse(permissionIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid permission ID")
		return
	}

	if err := h.rbacService.RemovePermissionFromRole(r.Context(), roleID, permissionID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "removal_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var assignment UserRoleAssignment
	if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	assignment.OrganizationID = orgID
	assignment.IsActive = true
	assignment.ValidFrom = time.Now()

	if err := h.rbacService.AssignRoleToUser(r.Context(), &assignment); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "assignment_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, assignment)
}

func (h *Handler) RemoveRoleFromUser(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]
	userIDStr := vars["userId"]
	roleIDStr := vars["roleId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid role ID")
		return
	}

	if err := h.rbacService.RemoveRoleFromUser(r.Context(), userID, roleID, orgID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "removal_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// Resource handlers

func (h *Handler) CreateResourceType(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var resourceType ResourceType
	if err := json.NewDecoder(r.Body).Decode(&resourceType); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	resourceType.OrganizationID = orgID
	if err := h.rbacService.CreateResourceType(r.Context(), &resourceType); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, resourceType)
}

func (h *Handler) GetResourceType(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	resourceTypeIDStr := vars["resourceTypeId"]

	resourceTypeID, err := uuid.Parse(resourceTypeIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid resource type ID")
		return
	}

	resourceType, err := h.rbacService.GetResourceType(r.Context(), resourceTypeID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "Resource type not found")
		return
	}

	WriteJSON(w, http.StatusOK, resourceType)
}

func (h *Handler) ListResourceTypes(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	resourceTypes, err := h.rbacService.ListResourceTypes(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, resourceTypes)
}

func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var resource Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	resource.OrganizationID = orgID
	if err := h.rbacService.CreateResource(r.Context(), &resource); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, resource)
}

func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	resourceIDStr := vars["resourceId"]

	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid resource ID")
		return
	}

	resource, err := h.rbacService.GetResource(r.Context(), resourceID)
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "not_found", "Resource not found")
		return
	}

	WriteJSON(w, http.StatusOK, resource)
}

func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	resourceIDStr := vars["resourceId"]

	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid resource ID")
		return
	}

	var resource Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	resource.ID = resourceID
	if err := h.rbacService.UpdateResource(r.Context(), &resource); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, resource)
}

func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	resourceIDStr := vars["resourceId"]

	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid resource ID")
		return
	}

	if err := h.rbacService.DeleteResource(r.Context(), resourceID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "deletion_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	// Get optional filters
	var resourceTypeID *uuid.UUID
	if rtIDStr := r.URL.Query().Get("resource_type_id"); rtIDStr != "" {
		if id, err := uuid.Parse(rtIDStr); err == nil {
			resourceTypeID = &id
		}
	}

	var parentPath *string
	if path := r.URL.Query().Get("parent_path"); path != "" {
		parentPath = &path
	}

	resources, err := h.rbacService.ListResources(r.Context(), orgID, resourceTypeID, parentPath)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, resources)
}

// Emergency access handlers

func (h *Handler) GrantEmergencyAccess(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, parseErr := uuid.Parse(orgIDStr)

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionRBACManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var access EmergencyAccess
	if err := json.NewDecoder(r.Body).Decode(&access); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	access.OrganizationID = orgID
	access.IsActive = true
	access.ValidFrom = time.Now()

	access.GrantedBy = &authCtx.UserID

	if err := h.rbacService.GrantEmergencyAccess(r.Context(), &access); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "grant_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, access)
}

func (h *Handler) RevokeEmergencyAccess(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	accessIDStr := vars["accessId"]

	accessID, parseErr := uuid.Parse(accessIDStr)

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionRBACManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid access ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if err := h.rbacService.RevokeEmergencyAccess(r.Context(), accessID, authCtx.UserID, req.Reason); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "revoke_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) GetUserEmergencyAccess(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]
	userIDStr := vars["userId"]

	orgID, parseOrgErr := uuid.Parse(orgIDStr)
	userID, parseUserErr := uuid.Parse(userIDStr)

	// Check permission
	hasPermission, err := h.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, PermissionRBACManage, nil)
	if err != nil || (!hasPermission && !authCtx.IsSystemAdmin) {
		WriteJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return
	}

	if parseOrgErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	if parseUserErr != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid user ID")
		return
	}

	accesses, err := h.rbacService.GetActiveEmergencyAccess(r.Context(), userID, orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, accesses)
}

// Audit handlers

func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	// Parse filters from query parameters
	filters := AuditFilters{
		Limit:  50,
		Offset: 0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filters.UserID = &userID
		}
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filters.Action = action
	}

	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filters.ResourceType = resourceType
	}

	if successStr := r.URL.Query().Get("success"); successStr != "" {
		if success, err := strconv.ParseBool(successStr); err == nil {
			filters.Success = &success
		}
	}

	if startTimeStr := r.URL.Query().Get("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filters.StartTime = &startTime
		}
	}

	if endTimeStr := r.URL.Query().Get("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filters.EndTime = &endTime
		}
	}

	events, err := h.rbacService.GetAuditLog(r.Context(), orgID, filters)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, events)
}

// Time-based access control handlers

func (h *Handler) CreateScheduledRoleAssignment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	var req struct {
		UserID              uuid.UUID              `json:"user_id"`
		RoleID              uuid.UUID              `json:"role_id"`
		ResourceScope       *string                `json:"resource_scope"`
		ScheduledActivation time.Time              `json:"scheduled_activation"`
		ScheduledExpiration *time.Time             `json:"scheduled_expiration"`
		AssignmentReason    string                 `json:"assignment_reason"`
		RecurrencePattern   *string                `json:"recurrence_pattern"`
		Metadata            map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	// Validate required fields
	if req.UserID == uuid.Nil || req.RoleID == uuid.Nil {
		WriteJSONError(w, http.StatusBadRequest, "missing_fields", "user_id and role_id are required")
		return
	}

	if req.AssignmentReason == "" {
		WriteJSONError(w, http.StatusBadRequest, "missing_fields", "assignment_reason is required")
		return
	}

	// Validate scheduled activation is in the future
	if req.ScheduledActivation.Before(time.Now()) {
		WriteJSONError(w, http.StatusBadRequest, "invalid_time", "scheduled_activation must be in the future")
		return
	}

	// Validate expiration is after activation if provided
	if req.ScheduledExpiration != nil && req.ScheduledExpiration.Before(req.ScheduledActivation) {
		WriteJSONError(w, http.StatusBadRequest, "invalid_time", "scheduled_expiration must be after scheduled_activation")
		return
	}

	// Get current user for audit trail

	assignment := &ScheduledRoleAssignment{
		UserID:              req.UserID,
		RoleID:              req.RoleID,
		OrganizationID:      orgID,
		ResourceScope:       req.ResourceScope,
		ScheduledActivation: req.ScheduledActivation,
		ScheduledExpiration: req.ScheduledExpiration,
		AssignedBy:          &authCtx.UserID,
		AssignmentReason:    req.AssignmentReason,
		RecurrencePattern:   req.RecurrencePattern,
		Metadata:            req.Metadata,
	}

	if assignment.Metadata == nil {
		assignment.Metadata = make(map[string]interface{})
	}

	err = h.rbacService.CreateScheduledRoleAssignment(r.Context(), assignment)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "creation_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, assignment)
}

func (h *Handler) UpdateScheduledRoleAssignment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	assignmentID, err := uuid.Parse(vars["assignmentId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_assignment_id", "Invalid assignment ID")
		return
	}

	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	var req struct {
		ScheduledActivation *time.Time             `json:"scheduled_activation"`
		ScheduledExpiration *time.Time             `json:"scheduled_expiration"`
		AssignmentReason    *string                `json:"assignment_reason"`
		RecurrencePattern   *string                `json:"recurrence_pattern"`
		Metadata            map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	// Build assignment struct with only updateable fields
	assignment := &ScheduledRoleAssignment{
		ID:             assignmentID,
		OrganizationID: orgID,
	}

	if req.ScheduledActivation != nil {
		if req.ScheduledActivation.Before(time.Now()) {
			WriteJSONError(w, http.StatusBadRequest, "invalid_time", "scheduled_activation must be in the future")
			return
		}
		assignment.ScheduledActivation = *req.ScheduledActivation
	}

	if req.ScheduledExpiration != nil {
		assignment.ScheduledExpiration = req.ScheduledExpiration
	}

	if req.AssignmentReason != nil {
		assignment.AssignmentReason = *req.AssignmentReason
	}

	if req.RecurrencePattern != nil {
		assignment.RecurrencePattern = req.RecurrencePattern
	}

	if req.Metadata != nil {
		assignment.Metadata = req.Metadata
	}

	err = h.rbacService.UpdateScheduledRoleAssignment(r.Context(), assignment)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"message": "Scheduled assignment updated successfully"})
}

func (h *Handler) DeleteScheduledRoleAssignment(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	assignmentID, err := uuid.Parse(vars["assignmentId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_assignment_id", "Invalid assignment ID")
		return
	}

	err = h.rbacService.DeleteScheduledRoleAssignment(r.Context(), assignmentID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "deletion_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"message": "Scheduled assignment deleted successfully"})
}

func (h *Handler) GetUserScheduledAssignments(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	userID, err := uuid.Parse(vars["userId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID")
		return
	}

	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	assignments, err := h.rbacService.GetScheduledRoleAssignments(r.Context(), userID, orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assignments)
}

func (h *Handler) ListPendingActivations(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	assignments, err := h.rbacService.ListPendingActivations(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assignments)
}

func (h *Handler) ListExpiredAssignments(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	assignments, err := h.rbacService.ListExpiredAssignments(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "listing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assignments)
}

func (h *Handler) GetTimeBasedAccessStatus(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgID, err := uuid.Parse(vars["orgId"])
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_org_id", "Invalid organization ID")
		return
	}

	status, err := h.rbacService.GetTimeBasedAccessStatus(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "status_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, status)
}

func (h *Handler) ProcessScheduledActivations(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	err := h.rbacService.ProcessScheduledActivations(r.Context())
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "processing_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Scheduled activations processed successfully",
		"processed_at": time.Now(),
	})
}

func (h *Handler) CleanupExpiredAssignments(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	err := h.rbacService.CleanupExpiredAssignments(r.Context())
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "cleanup_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Expired assignments cleaned up successfully",
		"cleaned_at": time.Now(),
	})
}

// Enhanced Emergency Access Handlers

func (h *Handler) CreateEmergencyAccessRequest(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var request EmergencyAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	// Validate request
	if len(request.RequestedPermissions) == 0 {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Requested permissions cannot be empty")
		return
	}

	if request.Reason == "" {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Reason is required")
		return
	}

	if request.RequestedDuration <= 0 {
		request.RequestedDuration = int64(2 * time.Hour) // Default 2 hours
	}

	if request.UrgencyLevel == "" {
		request.UrgencyLevel = EmergencyUrgencyMedium
	}

	request.UserID = authCtx.UserID
	request.OrganizationID = orgID

	// Initialize metadata if nil
	if request.Metadata == nil {
		request.Metadata = make(map[string]interface{})
	}

	if err := h.rbacService.CreateEmergencyAccessRequest(r.Context(), &request); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, request)
}

func (h *Handler) ListEmergencyAccessRequests(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	// Parse query parameters
	statusStr := r.URL.Query().Get("status")
	var status *EmergencyAccessRequestStatus
	if statusStr != "" {
		s := EmergencyAccessRequestStatus(statusStr)
		status = &s
	}

	limit, offset := parseLimitOffset(r)

	requests, err := h.rbacService.ListEmergencyAccessRequests(r.Context(), orgID, status, limit, offset)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"requests": requests,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *Handler) GetEmergencyAccessRequest(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	requestIDStr := vars["requestId"]

	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid request ID")
		return
	}

	request, err := h.rbacService.GetEmergencyAccessRequest(r.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteJSONError(w, http.StatusNotFound, "not_found", "Emergency access request not found")
		} else {
			WriteJSONError(w, http.StatusInternalServerError, "get_failed", err.Error())
		}
		return
	}

	WriteJSON(w, http.StatusOK, request)
}

func (h *Handler) ApproveEmergencyAccessRequest(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	requestIDStr := vars["requestId"]

	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid request ID")
		return
	}

	var req struct {
		Action EmergencyAccessApprovalAction `json:"action"`
		Reason string                        `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if req.Action == "" {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Action is required")
		return
	}

	if err := h.rbacService.ApproveEmergencyAccessRequest(r.Context(), requestID, authCtx.UserID, req.Action, req.Reason); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "approval_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "processed",
		"action":       req.Action,
		"reason":       req.Reason,
		"processed_at": time.Now(),
	})
}

func (h *Handler) GetEmergencyAccessApprovals(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	requestIDStr := vars["requestId"]

	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid request ID")
		return
	}

	approvals, err := h.rbacService.GetEmergencyAccessApprovals(r.Context(), requestID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"approvals": approvals,
	})
}

func (h *Handler) ProcessBreakGlassAccess(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	requestIDStr := vars["requestId"]

	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid request ID")
		return
	}

	if err := h.rbacService.ProcessBreakGlassAccess(r.Context(), requestID); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "process_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "processed",
		"message":      "Break-glass access processed",
		"processed_at": time.Now(),
	})
}

func (h *Handler) GetBreakGlassConfig(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	config, err := h.rbacService.GetBreakGlassConfig(r.Context(), orgID)
	if err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, config)
}

func (h *Handler) UpdateBreakGlassConfig(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		WriteJSONError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	vars := mux.Vars(r)
	orgIDStr := vars["orgId"]

	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_id", "Invalid organization ID")
		return
	}

	var config BreakGlassConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	// Validate config
	if config.MaxDuration <= 0 {
		config.MaxDuration = 4 * time.Hour
	}

	if config.AutoRevocationMinutes <= 0 {
		config.AutoRevocationMinutes = 240
	}

	if config.ApprovalRequirements == nil {
		config.ApprovalRequirements = map[EmergencyUrgencyLevel]int{
			EmergencyUrgencyLow:      2,
			EmergencyUrgencyMedium:   2,
			EmergencyUrgencyHigh:     1,
			EmergencyUrgencyCritical: 0,
		}
	}

	if config.EscalationRules == nil {
		config.EscalationRules = []EmergencyEscalationRule{}
	}

	if err := h.rbacService.UpdateBreakGlassConfig(r.Context(), orgID, &config); err != nil {
		WriteJSONError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, config)
}
