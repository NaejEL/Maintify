package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	mockService := new(MockRBACService)
	jwtSecret := []byte("secret")
	authService := NewAuthService(mockService, jwtSecret)
	middleware := NewAuthMiddleware(authService, mockService)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("No Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		middleware.RequireAuth(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Valid Token", func(t *testing.T) {
		userID := uuid.New()
		claims := JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtSecret)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rr := httptest.NewRecorder()

		middleware.RequireAuth(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		middleware.RequireAuth(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthMiddleware_RequirePermission(t *testing.T) {
	mockService := new(MockRBACService)
	authService := NewAuthService(mockService, []byte("secret"))
	middleware := NewAuthMiddleware(authService, mockService)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Has Permission", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		authCtx := &AuthContext{
			UserID:         userID,
			OrganizationID: orgID,
		}

		mockService.On("HasPermission", mock.Anything, userID, orgID, "test:read", mock.Anything).Return(true, nil)

		req, _ := http.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware.RequirePermission("test:read")(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("No Permission", func(t *testing.T) {
		userID := uuid.New()
		orgID := uuid.New()
		authCtx := &AuthContext{
			UserID:         userID,
			OrganizationID: orgID,
		}

		mockService.On("HasPermission", mock.Anything, userID, orgID, "test:write", mock.Anything).Return(false, nil)

		req, _ := http.NewRequest("GET", "/", nil)
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		middleware.RequirePermission("test:write")(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

func TestAuthMiddleware_AuditMiddleware(t *testing.T) {
	mockService := new(MockRBACService)
	authService := NewAuthService(mockService, []byte("secret"))
	middleware := NewAuthMiddleware(authService, mockService)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Logs Audit Event", func(t *testing.T) {
		mockService.On("LogAuditEvent", mock.Anything, mock.Anything).Return(nil)

		req, _ := http.NewRequest("GET", "/test", nil)

		// Inject AuthContext
		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		middleware.AuditMiddleware(nextHandler).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		// Wait for goroutine to finish (simple sleep for test)
		time.Sleep(50 * time.Millisecond)
		mockService.AssertCalled(t, "LogAuditEvent", mock.Anything, mock.Anything)
	})
}
