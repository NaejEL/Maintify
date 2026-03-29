package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// contextKey is the unexported type used for all context keys in this package
// to avoid collisions with other packages.
type contextKey string

const authContextKey contextKey = "auth"

// JWTClaims represents the structure of JWT token claims
type JWTClaims struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	IsSystemAdmin  bool      `json:"is_system_admin"`
	Username       string    `json:"username"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	rbacService RBACService
	jwtSecret   []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(rbacService RBACService, jwtSecret []byte) *AuthService {
	return &AuthService{
		rbacService: rbacService,
		jwtSecret:   jwtSecret,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	OrgSlug  string `json:"org_slug,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         *User         `json:"user"`
	Organization *Organization `json:"organization,omitempty"`
	Token        string        `json:"token"`
	ExpiresAt    int64         `json:"expires_at"`
}

// AuthContext represents the authenticated user context
type AuthContext struct {
	UserID         uuid.UUID `json:"user_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	SessionID      string    `json:"session_id"`
	IsSystemAdmin  bool      `json:"is_system_admin"`
	ExpiresAt      int64     `json:"expires_at"`
}

// AuthMiddleware provides authentication and authorization middleware
type AuthMiddleware struct {
	authService *AuthService
	rbacService RBACService
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *AuthService, rbacService RBACService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		rbacService: rbacService,
	}
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Get user by email
	user, err := s.rbacService.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is deactivated")
	}

	// Get organization if specified
	var org *Organization
	var orgID uuid.UUID

	if req.OrgSlug != "" {
		org, err = s.rbacService.GetOrganizationBySlug(ctx, req.OrgSlug)
		if err != nil {
			return nil, fmt.Errorf("organization not found")
		}
		orgID = org.ID

		// Verify user has access to this organization
		if !user.IsSystemAdmin {
			roles, err := s.rbacService.GetUserRoles(ctx, user.ID, orgID)
			if err != nil || len(roles) == 0 {
				return nil, fmt.Errorf("no access to organization")
			}
		}
	} else {
		// Use system organization for system admins
		if user.IsSystemAdmin {
			org, err = s.rbacService.GetOrganizationBySlug(ctx, "system")
			if err != nil {
				return nil, fmt.Errorf("system organization not found")
			}
			orgID = org.ID
		} else {
			return nil, fmt.Errorf("organization required for non-system users")
		}
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	err = s.rbacService.UpdateUser(ctx, user)
	if err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login time: %v\n", err)
	}

	// Generate JWT token
	token, expiresAt, err := s.generateJWT(user.ID, orgID, user.IsSystemAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token")
	}

	// Clear password hash from response
	user.PasswordHash = ""

	return &LoginResponse{
		User:         user,
		Organization: org,
		Token:        token,
		ExpiresAt:    expiresAt,
	}, nil
}

// VerifyToken verifies a JWT token and returns the auth context
func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*AuthContext, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse and validate JWT token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// Return auth context
	authCtx := &AuthContext{
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		SessionID:      "session-" + uuid.New().String(),
		IsSystemAdmin:  claims.IsSystemAdmin,
		ExpiresAt:      claims.ExpiresAt.Unix(),
	}

	return authCtx, nil
}

// generateJWT generates a JWT token with proper claims
func (s *AuthService) generateJWT(userID, orgID uuid.UUID, isSystemAdmin bool) (string, int64, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	// Create JWT claims
	claims := &JWTClaims{
		UserID:         userID,
		OrganizationID: orgID,
		IsSystemAdmin:  isSystemAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "maintify-core",
			Subject:   userID.String(),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt.Unix(), nil
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		authCtx, err := m.authService.VerifyToken(r.Context(), authHeader)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add auth context to request
		ctx := context.WithValue(r.Context(), authContextKey, authCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission middleware that requires a specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r.Context())
			if authCtx == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// System admins have all permissions
			if authCtx.IsSystemAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check permission
			hasPermission, err := m.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, permission, nil)
			if err != nil {
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrganizationAccess middleware that requires access to a specific organization
func (m *AuthMiddleware) RequireOrganizationAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())
		if authCtx == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Get organization from URL path
		vars := mux.Vars(r)
		orgIDStr, exists := vars["orgId"]
		if !exists {
			http.Error(w, "Organization ID required", http.StatusBadRequest)
			return
		}

		orgID, parseErr := uuid.Parse(orgIDStr)

		// System admins have access to all organizations
		if authCtx.IsSystemAdmin {
			if parseErr != nil {
				http.Error(w, "Invalid organization ID", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Check if user has access to this organization
		if authCtx.OrganizationID != orgID {
			// Check if user has roles in this organization
			roles, err := m.rbacService.GetUserRoles(r.Context(), authCtx.UserID, orgID)
			if err != nil || len(roles) == 0 {
				http.Error(w, "No access to organization", http.StatusForbidden)
				return
			}
		}

		if parseErr != nil {
			http.Error(w, "Invalid organization ID", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireResourceAccess middleware that requires access to a specific resource
func (m *AuthMiddleware) RequireResourceAccess(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r.Context())
			if authCtx == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Get resource from URL path
			vars := mux.Vars(r)
			resourceIDStr, exists := vars["resourceId"]
			if !exists {
				http.Error(w, "Resource ID required", http.StatusBadRequest)
				return
			}

			resourceID, err := uuid.Parse(resourceIDStr)
			if err != nil {
				http.Error(w, "Invalid resource ID", http.StatusBadRequest)
				return
			}

			// Get resource to check its path
			resource, err := m.rbacService.GetResource(r.Context(), resourceID)
			if err != nil {
				http.Error(w, "Resource not found", http.StatusNotFound)
				return
			}

			// System admins have access to all resources
			if authCtx.IsSystemAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check permission with resource path
			hasPermission, err := m.rbacService.HasPermission(r.Context(), authCtx.UserID, authCtx.OrganizationID, permission, &resource.ParentPath)
			if err != nil {
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				http.Error(w, "Insufficient permissions for resource", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthContext retrieves the auth context from request context
func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(authContextKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

// AuditMiddleware logs all requests for audit purposes
func (m *AuthMiddleware) AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := GetAuthContext(r.Context())

		// Create audit event
		event := &AuditEvent{
			OrganizationID: uuid.Nil,
			Action:         r.Method + " " + r.URL.Path,
			Success:        true, // Will be updated if request fails
			IPAddress:      getClientIP(r),
			UserAgent:      r.UserAgent(),
			SessionID:      "",
		}

		if authCtx != nil {
			event.OrganizationID = authCtx.OrganizationID
			event.UserID = &authCtx.UserID
			event.SessionID = authCtx.SessionID
		}

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Update success based on status code
		event.Success = wrapped.statusCode < 400
		if !event.Success {
			event.Reason = fmt.Sprintf("HTTP %d", wrapped.statusCode)
		}

		// Log audit event (ignore errors to not fail the request)
		if authCtx != nil {
			requestCtx := r.Context()
			go func() {
				m.rbacService.LogAuditEvent(requestCtx, event)
			}()
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts the client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) >= 1 {
		return parts[0]
	}

	return r.RemoteAddr
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// WriteJSONError writes a JSON error response
func WriteJSONError(w http.ResponseWriter, statusCode int, err string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   err,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// parseLimitOffset parses limit and offset from query parameters
func parseLimitOffset(r *http.Request) (int, int) {
	limit := 50 // default limit
	offset := 0 // default offset

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
