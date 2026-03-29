package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func setupHandlerTest(t *testing.T) (*Handler, *MockRBACService) {
	mockService := new(MockRBACService)
	authService := NewAuthService(mockService, []byte("secret"))
	handler := NewHandler(mockService, authService)
	return handler, mockService
}

func addAuthContext(req *http.Request) *http.Request {
	authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
	ctx := context.WithValue(req.Context(), authContextKey, authCtx)
	return req.WithContext(ctx)
}

func TestHandler_CreateUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		user := &User{
			Email:        "test@example.com",
			Username:     "testuser",
			PasswordHash: "password123",
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("CreateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.Email == user.Email && u.Username == user.Username
		})).Return(nil)

		body, _ := json.Marshal(user)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBufferString("invalid json"))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)

		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		user := &User{
			Email:        "error@example.com",
			Username:     "erroruser",
			PasswordHash: "password123",
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(user)
		req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_Login(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &User{
		ID:            uuid.New(),
		Email:         "test@example.com",
		PasswordHash:  string(hashedPassword),
		IsActive:      true,
		IsSystemAdmin: true,
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockService.On("GetOrganizationBySlug", mock.Anything, "system").Return(&Organization{ID: uuid.New(), Slug: "system"}, nil)
		mockService.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

		loginReq := LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("Login failed with status %d. Body: %s", rr.Code, rr.Body.String())
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp LoginResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp.Token)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("GetUserByEmail", mock.Anything, "wrong@example.com").Return(nil, errors.New("not found"))

		loginReq := LoginRequest{
			Email:    "wrong@example.com",
			Password: "password123",
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("GetUserByEmail", mock.Anything, "error@example.com").Return(nil, errors.New("db error"))

		loginReq := LoginRequest{
			Email:    "error@example.com",
			Password: "password123",
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestHandler_GetUser(t *testing.T) {
	userID := uuid.New()
	user := &User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "testuser",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserView, mock.Anything).Return(true, nil)
		mockService.On("GetUser", mock.Anything, userID).Return(user, nil)

		req, _ := http.NewRequest("GET", "/users/"+userID.String(), nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		// Use router to handle variables
		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.GetUser)
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Logf("GetUser failed with status %d. Body: %s", rr.Code, rr.Body.String())
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp User
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, user.Email, resp.Email)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserView, mock.Anything).Return(true, nil)
		mockService.On("GetUser", mock.Anything, userID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/users/"+userID.String(), nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		// Use router to handle variables
		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.GetUser)
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Logf("GetUser Not Found failed with status %d. Body: %s", rr.Code, rr.Body.String())
		}

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserView, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("GET", "/users/invalid", nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.GetUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserView, mock.Anything).Return(true, nil)
		mockService.On("GetUser", mock.Anything, userID).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/users/"+userID.String(), nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.GetUser)
		router.ServeHTTP(rr, req)

		// Current implementation returns 404 for any error
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_UpdateUser(t *testing.T) {
	userID := uuid.New()
	user := &User{
		ID:       userID,
		Email:    "updated@example.com",
		Username: "updateduser",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
			return u.ID == userID && u.Email == user.Email
		})).Return(nil)

		body, _ := json.Marshal(user)
		req, _ := http.NewRequest("PUT", "/users/"+userID.String(), bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.UpdateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("PUT", "/users/invalid", nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.UpdateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("PUT", "/users/"+userID.String(), bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.UpdateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("UpdateUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(user)
		req, _ := http.NewRequest("PUT", "/users/"+userID.String(), bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.UpdateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_DeactivateUser(t *testing.T) {
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("DeactivateUser", mock.Anything, userID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/users/"+userID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.DeactivateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("DELETE", "/users/invalid", nil)
		req = addAuthContext(req)
		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.DeactivateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserManage, mock.Anything).Return(true, nil)
		mockService.On("DeactivateUser", mock.Anything, userID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/users/"+userID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}", handler.DeactivateUser)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ListUsers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		users := []*User{
			{ID: uuid.New(), Email: "user1@example.com"},
			{ID: uuid.New(), Email: "user2@example.com"},
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionUserView, mock.Anything).Return(true, nil)
		mockService.On("ListUsers", mock.Anything, (*uuid.UUID)(nil), 10, 0).Return(users, nil)

		req, _ := http.NewRequest("GET", "/users?limit=10&offset=0", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.ListUsers(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp []*User
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Len(t, resp, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateOrganization(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		org := &Organization{
			Name: "New Org",
			Slug: "new-org",
		}

		mockService.On("CreateOrganization", mock.Anything, mock.MatchedBy(func(o *Organization) bool {
			return o.Name == org.Name && o.Slug == org.Slug
		})).Return(nil)

		body, _ := json.Marshal(org)
		req, _ := http.NewRequest("POST", "/organizations", bytes.NewBuffer(body))

		// System admin required
		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New(), IsSystemAdmin: true}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetOrganization(t *testing.T) {
	orgID := uuid.New()
	org := &Organization{
		ID:   orgID,
		Name: "Test Org",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New(), IsSystemAdmin: true}
		mockService.On("HasPermission", mock.Anything, authCtx.UserID, authCtx.OrganizationID, PermissionOrgView, (*string)(nil)).Return(true, nil)
		mockService.On("GetOrganization", mock.Anything, orgID).Return(org, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String(), nil)

		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.GetOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateOrganization(t *testing.T) {
	orgID := uuid.New()
	org := &Organization{
		ID:   orgID,
		Name: "Updated Org",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New(), IsSystemAdmin: true}
		mockService.On("HasPermission", mock.Anything, authCtx.UserID, authCtx.OrganizationID, PermissionOrgManage, (*string)(nil)).Return(true, nil)
		mockService.On("UpdateOrganization", mock.Anything, mock.MatchedBy(func(o *Organization) bool {
			return o.ID == orgID && o.Name == org.Name
		})).Return(nil)

		body, _ := json.Marshal(org)
		req, _ := http.NewRequest("PUT", "/organizations/"+orgID.String(), bytes.NewBuffer(body))

		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.UpdateOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_DeleteOrganization(t *testing.T) {
	orgID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New(), IsSystemAdmin: true}
		mockService.On("HasPermission", mock.Anything, authCtx.UserID, authCtx.OrganizationID, PermissionOrgManage, (*string)(nil)).Return(true, nil)
		mockService.On("DeleteOrganization", mock.Anything, orgID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/organizations/"+orgID.String(), nil)

		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.DeleteOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListOrganizations(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgs := []*Organization{
			{ID: uuid.New(), Name: "Org 1"},
			{ID: uuid.New(), Name: "Org 2"},
		}

		mockService.On("ListOrganizations", mock.Anything, 10, 0).Return(orgs, nil)

		req, _ := http.NewRequest("GET", "/organizations?limit=10&offset=0", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New(), IsSystemAdmin: true}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.ListOrganizations(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp []*Organization
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Len(t, resp, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateRole(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		role := &Role{
			Name:        "New Role",
			Description: "New Role Description",
		}

		mockService.On("CreateRole", mock.Anything, mock.MatchedBy(func(r *Role) bool {
			return r.Name == role.Name
		})).Return(nil)

		body, _ := json.Marshal(role)
		req, _ := http.NewRequest("POST", "/roles", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.CreateRole(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetRole(t *testing.T) {
	roleID := uuid.New()
	role := &Role{
		ID:   roleID,
		Name: "Test Role",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("GetRole", mock.Anything, roleID).Return(role, nil)

		req, _ := http.NewRequest("GET", "/roles/"+roleID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.GetRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateRole(t *testing.T) {
	roleID := uuid.New()
	role := &Role{
		ID:   roleID,
		Name: "Updated Role",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("UpdateRole", mock.Anything, mock.MatchedBy(func(r *Role) bool {
			return r.ID == roleID && r.Name == role.Name
		})).Return(nil)

		body, _ := json.Marshal(role)
		req, _ := http.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.UpdateRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_DeleteRole(t *testing.T) {
	roleID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("DeleteRole", mock.Anything, roleID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/roles/"+roleID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.DeleteRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListRoles(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		roles := []*Role{
			{ID: uuid.New(), Name: "Role 1"},
			{ID: uuid.New(), Name: "Role 2"},
		}

		mockService.On("ListRoles", mock.Anything, mock.Anything, 10, 0).Return(roles, nil)

		req, _ := http.NewRequest("GET", "/roles?limit=10&offset=0", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.ListRoles(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp []*Role
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Len(t, resp, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreatePermission(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		permission := &Permission{
			Name:        "user:create",
			Description: "Create users",
			Action:      "create",
		}

		mockService.On("CreatePermission", mock.Anything, mock.MatchedBy(func(p *Permission) bool {
			return p.Name == permission.Name
		})).Return(nil)

		body, _ := json.Marshal(permission)
		req, _ := http.NewRequest("POST", "/permissions", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.CreatePermission(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetPermission(t *testing.T) {
	permID := uuid.New()
	permission := &Permission{
		ID:   permID,
		Name: "user:create",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("GetPermission", mock.Anything, permID).Return(permission, nil)

		req, _ := http.NewRequest("GET", "/permissions/"+permID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"permissionId": permID.String()})

		rr := httptest.NewRecorder()

		handler.GetPermission(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListPermissions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		perms := []*Permission{
			{ID: uuid.New(), Name: "perm1"},
			{ID: uuid.New(), Name: "perm2"},
		}

		mockService.On("ListPermissions", mock.Anything, mock.Anything).Return(perms, nil)

		req, _ := http.NewRequest("GET", "/permissions", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.ListPermissions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp []*Permission
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Len(t, resp, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_AssignPermissionToRole(t *testing.T) {
	roleID := uuid.New()
	permID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("AssignPermissionToRole", mock.Anything, roleID, permID).Return(nil)

		body := map[string]string{"permission_id": permID.String()}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(jsonBody))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String()})

		rr := httptest.NewRecorder()

		handler.AssignPermissionToRole(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_RemovePermissionFromRole(t *testing.T) {
	roleID := uuid.New()
	permID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("RemovePermissionFromRole", mock.Anything, roleID, permID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String(), "permissionId": permID.String()})

		rr := httptest.NewRecorder()

		handler.RemovePermissionFromRole(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_AssignRoleToUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		assignment := &UserRoleAssignment{
			UserID: uuid.New(),
			RoleID: uuid.New(),
		}

		mockService.On("AssignRoleToUser", mock.Anything, mock.MatchedBy(func(a *UserRoleAssignment) bool {
			return a.UserID == assignment.UserID && a.RoleID == assignment.RoleID
		})).Return(nil)

		body, _ := json.Marshal(assignment)
		req, _ := http.NewRequest("POST", "/users/"+assignment.UserID.String()+"/roles", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"userId": assignment.UserID.String(), "orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.AssignRoleToUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/users/roles", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()

		handler.AssignRoleToUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/users/roles", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()

		handler.AssignRoleToUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		assignment := &UserRoleAssignment{
			UserID: uuid.New(),
			RoleID: uuid.New(),
		}

		mockService.On("AssignRoleToUser", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(assignment)
		req, _ := http.NewRequest("POST", "/users/"+assignment.UserID.String()+"/roles", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"userId": assignment.UserID.String(), "orgId": authCtx.OrganizationID.String()})

		rr := httptest.NewRecorder()

		handler.AssignRoleToUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_RemoveRoleFromUser(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	orgID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("RemoveRoleFromUser", mock.Anything, userID, roleID, orgID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String(), "roleId": roleID.String(), "orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.RemoveRoleFromUser(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateResourceType(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		resourceType := &ResourceType{
			Name:             "Test Resource Type",
			Description:      "Test Description",
			HierarchyEnabled: true,
		}

		mockService.On("CreateResourceType", mock.Anything, mock.MatchedBy(func(rt *ResourceType) bool {
			return rt.Name == resourceType.Name && rt.OrganizationID == orgID
		})).Return(nil)

		body, _ := json.Marshal(resourceType)
		req, _ := http.NewRequest("POST", "/organizations/"+orgID.String()+"/resource-types", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.CreateResourceType(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetResourceType(t *testing.T) {
	resourceTypeID := uuid.New()
	resourceType := &ResourceType{
		ID:   resourceTypeID,
		Name: "Test Resource Type",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("GetResourceType", mock.Anything, resourceTypeID).Return(resourceType, nil)

		req, _ := http.NewRequest("GET", "/resource-types/"+resourceTypeID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"resourceTypeId": resourceTypeID.String()})

		rr := httptest.NewRecorder()

		handler.GetResourceType(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response ResourceType
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Equal(t, resourceType.ID, response.ID)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListResourceTypes(t *testing.T) {
	orgID := uuid.New()
	resourceTypes := []*ResourceType{
		{ID: uuid.New(), Name: "RT1"},
		{ID: uuid.New(), Name: "RT2"},
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("ListResourceTypes", mock.Anything, orgID).Return(resourceTypes, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/resource-types", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.ListResourceTypes(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*ResourceType
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateResource(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		resource := &Resource{
			Name:           "Test Resource",
			ResourceTypeID: uuid.New(),
		}

		mockService.On("CreateResource", mock.Anything, mock.MatchedBy(func(r *Resource) bool {
			return r.Name == resource.Name && r.OrganizationID == orgID
		})).Return(nil)

		body, _ := json.Marshal(resource)
		req, _ := http.NewRequest("POST", "/organizations/"+orgID.String()+"/resources", bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.CreateResource(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetResource(t *testing.T) {
	resourceID := uuid.New()
	resource := &Resource{
		ID:   resourceID,
		Name: "Test Resource",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("GetResource", mock.Anything, resourceID).Return(resource, nil)

		req, _ := http.NewRequest("GET", "/resources/"+resourceID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"resourceId": resourceID.String()})

		rr := httptest.NewRecorder()

		handler.GetResource(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response Resource
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Equal(t, resource.ID, response.ID)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateResource(t *testing.T) {
	resourceID := uuid.New()
	resource := &Resource{
		ID:   resourceID,
		Name: "Updated Resource",
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("UpdateResource", mock.Anything, mock.MatchedBy(func(r *Resource) bool {
			return r.ID == resourceID && r.Name == resource.Name
		})).Return(nil)

		body, _ := json.Marshal(resource)
		req, _ := http.NewRequest("PUT", "/resources/"+resourceID.String(), bytes.NewBuffer(body))

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"resourceId": resourceID.String()})

		rr := httptest.NewRecorder()

		handler.UpdateResource(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_DeleteResource(t *testing.T) {
	resourceID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("DeleteResource", mock.Anything, resourceID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/resources/"+resourceID.String(), nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"resourceId": resourceID.String()})

		rr := httptest.NewRecorder()

		handler.DeleteResource(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListResources(t *testing.T) {
	orgID := uuid.New()
	resources := []*Resource{
		{ID: uuid.New(), Name: "R1"},
		{ID: uuid.New(), Name: "R2"},
	}

	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("ListResources", mock.Anything, orgID, (*uuid.UUID)(nil), (*string)(nil)).Return(resources, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/resources", nil)

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.ListResources(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*Resource
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 2)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetAuditLog(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		events := []*AuditEvent{
			{ID: uuid.New(), OrganizationID: orgID, Action: "test_action"},
		}

		mockService.On("GetAuditLog", mock.Anything, orgID, mock.Anything).Return(events, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/audit-log", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.GetAuditLog(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*AuditEvent
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 1)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GrantEmergencyAccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		userID := uuid.New()

		access := &EmergencyAccess{
			UserID:             userID,
			GrantedPermissions: []string{"admin:access"},
			Reason:             "Emergency fix",
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("GrantEmergencyAccess", mock.Anything, mock.MatchedBy(func(a *EmergencyAccess) bool {
			return a.UserID == userID && a.OrganizationID == orgID
		})).Return(nil)

		body, _ := json.Marshal(access)
		req, _ := http.NewRequest("POST", "/organizations/"+orgID.String()+"/emergency-access", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GrantEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_RevokeEmergencyAccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		accessID := uuid.New()

		reqBody := map[string]string{
			"reason": "Emergency over",
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("RevokeEmergencyAccess", mock.Anything, accessID, mock.Anything, "Emergency over").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/emergency-access/"+accessID.String()+"/revoke", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"accessId": accessID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.RevokeEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetUserEmergencyAccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		userID := uuid.New()

		accesses := []*EmergencyAccess{
			{ID: uuid.New(), UserID: userID, OrganizationID: orgID},
		}

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("GetActiveEmergencyAccess", mock.Anything, userID, orgID).Return(accesses, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/users/"+userID.String()+"/emergency-access", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String(), "userId": userID.String()})

		rr := httptest.NewRecorder()

		handler.GetUserEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*EmergencyAccess
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 1)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateEmergencyAccessRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		request := &EmergencyAccessRequest{
			RequestedPermissions: []string{"admin:access"},
			Reason:               "Critical issue",
			UrgencyLevel:         EmergencyUrgencyHigh,
		}

		mockService.On("CreateEmergencyAccessRequest", mock.Anything, mock.MatchedBy(func(r *EmergencyAccessRequest) bool {
			return r.OrganizationID == orgID && r.Reason == "Critical issue"
		})).Return(nil)

		body, _ := json.Marshal(request)
		req, _ := http.NewRequest("POST", "/organizations/"+orgID.String()+"/emergency-access/requests", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.CreateEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListEmergencyAccessRequests(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		requests := []*EmergencyAccessRequest{
			{ID: uuid.New(), OrganizationID: orgID},
		}

		mockService.On("ListEmergencyAccessRequests", mock.Anything, orgID, mock.Anything, 50, 0).Return(requests, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/emergency-access/requests", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.ListEmergencyAccessRequests(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NotNil(t, response["requests"])
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetEmergencyAccessRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		request := &EmergencyAccessRequest{ID: requestID}

		mockService.On("GetEmergencyAccessRequest", mock.Anything, requestID).Return(request, nil)

		req, _ := http.NewRequest("GET", "/emergency-access/requests/"+requestID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"requestId": requestID.String()})

		rr := httptest.NewRecorder()

		handler.GetEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ApproveEmergencyAccessRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		reqBody := map[string]interface{}{
			"action": EmergencyAccessApprovalActionApprove,
			"reason": "Approved",
		}

		mockService.On("ApproveEmergencyAccessRequest", mock.Anything, requestID, mock.Anything, EmergencyAccessApprovalActionApprove, "Approved").Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/emergency-access/requests/"+requestID.String()+"/approve", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"requestId": requestID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: uuid.New()}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.ApproveEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetEmergencyAccessApprovals(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		approvals := []*EmergencyAccessApproval{
			{ID: uuid.New(), RequestID: requestID},
		}

		mockService.On("GetEmergencyAccessApprovals", mock.Anything, requestID).Return(approvals, nil)

		req, _ := http.NewRequest("GET", "/emergency-access/requests/"+requestID.String()+"/approvals", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"requestId": requestID.String()})

		rr := httptest.NewRecorder()

		handler.GetEmergencyAccessApprovals(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ProcessBreakGlassAccess(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		mockService.On("ProcessBreakGlassAccess", mock.Anything, requestID).Return(nil)

		req, _ := http.NewRequest("POST", "/emergency-access/requests/"+requestID.String()+"/break-glass", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"requestId": requestID.String()})

		rr := httptest.NewRecorder()

		handler.ProcessBreakGlassAccess(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetBreakGlassConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		config := &BreakGlassConfig{Enabled: true}

		mockService.On("GetBreakGlassConfig", mock.Anything, orgID).Return(config, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/break-glass/config", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.GetBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateBreakGlassConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		config := &BreakGlassConfig{
			MaxDuration: 4 * 3600 * 1000000000, // 4 hours
		}

		mockService.On("UpdateBreakGlassConfig", mock.Anything, orgID, mock.MatchedBy(func(c *BreakGlassConfig) bool {
			return c.MaxDuration == config.MaxDuration
		})).Return(nil)

		body, _ := json.Marshal(config)
		req, _ := http.NewRequest("PUT", "/organizations/"+orgID.String()+"/break-glass/config", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.UpdateBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CreateScheduledRoleAssignment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		userID := uuid.New()
		roleID := uuid.New()

		reqBody := map[string]interface{}{
			"user_id":              userID,
			"role_id":              roleID,
			"assignment_reason":    "Temporary access",
			"scheduled_activation": time.Now().Add(1 * time.Hour),
		}

		mockService.On("CreateScheduledRoleAssignment", mock.Anything, mock.MatchedBy(func(a *ScheduledRoleAssignment) bool {
			return a.UserID == userID && a.RoleID == roleID && a.OrganizationID == orgID
		})).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/organizations/"+orgID.String()+"/scheduled-assignments", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.CreateScheduledRoleAssignment(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_UpdateScheduledRoleAssignment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		assignmentID := uuid.New()
		orgID := uuid.New()

		reason := "Updated reason"
		reqBody := map[string]interface{}{
			"assignment_reason": reason,
		}

		mockService.On("UpdateScheduledRoleAssignment", mock.Anything, mock.MatchedBy(func(a *ScheduledRoleAssignment) bool {
			return a.ID == assignmentID && a.OrganizationID == orgID && a.AssignmentReason == reason
		})).Return(nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PUT", "/organizations/"+orgID.String()+"/scheduled-assignments/"+assignmentID.String(), bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String(), "assignmentId": assignmentID.String()})

		rr := httptest.NewRecorder()

		handler.UpdateScheduledRoleAssignment(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_DeleteScheduledRoleAssignment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		assignmentID := uuid.New()

		mockService.On("DeleteScheduledRoleAssignment", mock.Anything, assignmentID).Return(nil)

		req, _ := http.NewRequest("DELETE", "/scheduled-assignments/"+assignmentID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"assignmentId": assignmentID.String()})

		rr := httptest.NewRecorder()

		handler.DeleteScheduledRoleAssignment(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetUserScheduledAssignments(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		userID := uuid.New()

		assignments := []*ScheduledRoleAssignment{
			{ID: uuid.New(), UserID: userID, OrganizationID: orgID},
		}

		mockService.On("GetScheduledRoleAssignments", mock.Anything, userID, orgID).Return(assignments, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/users/"+userID.String()+"/scheduled-assignments", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String(), "userId": userID.String()})

		rr := httptest.NewRecorder()

		handler.GetUserScheduledAssignments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*ScheduledRoleAssignment
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 1)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListPendingActivations(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		assignments := []*ScheduledRoleAssignment{
			{ID: uuid.New(), OrganizationID: orgID},
		}

		mockService.On("ListPendingActivations", mock.Anything, orgID).Return(assignments, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/scheduled-assignments/pending", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.ListPendingActivations(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*ScheduledRoleAssignment
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 1)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ListExpiredAssignments(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		assignments := []*UserRoleAssignment{
			{ID: uuid.New(), OrganizationID: orgID},
		}

		mockService.On("ListExpiredAssignments", mock.Anything, orgID).Return(assignments, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/assignments/expired", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.ListExpiredAssignments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response []*UserRoleAssignment
		json.Unmarshal(rr.Body.Bytes(), &response)
		assert.Len(t, response, 1)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetTimeBasedAccessStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		status := &TimeBasedAccessStatus{
			ActiveAssignments:  5,
			PendingActivations: 2,
		}

		mockService.On("GetTimeBasedAccessStatus", mock.Anything, orgID).Return(status, nil)

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String()+"/time-based-access/status", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()

		handler.GetTimeBasedAccessStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_ProcessScheduledActivations(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("ProcessScheduledActivations", mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/scheduled-assignments/process", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()

		handler.ProcessScheduledActivations(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_CleanupExpiredAssignments(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("CleanupExpiredAssignments", mock.Anything).Return(nil)

		req, _ := http.NewRequest("POST", "/assignments/cleanup", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()

		handler.CleanupExpiredAssignments(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetUserRoles(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		userID := uuid.New()
		orgID := uuid.New()

		mockService.On("GetUserRoles", mock.Anything, userID, orgID).Return([]*Role{}, nil)

		req, _ := http.NewRequest("GET", "/users/"+userID.String()+"/roles", nil)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetUserRoles(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/users/invalid/roles", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": "invalid"})

		rr := httptest.NewRecorder()

		handler.GetUserRoles(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		userID := uuid.New()
		orgID := uuid.New()

		mockService.On("GetUserRoles", mock.Anything, userID, orgID).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/users/"+userID.String()+"/roles", nil)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetUserRoles(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockService.AssertExpectations(t)
	})
}

func TestHandler_GetUserPermissions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		userID := uuid.New()
		orgID := uuid.New()

		mockService.On("GetUserPermissions", mock.Anything, userID, orgID, (*string)(nil)).Return([]string{"perm1"}, nil)

		req, _ := http.NewRequest("GET", "/users/"+userID.String()+"/permissions", nil)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetUserPermissions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/users/invalid/permissions", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": "invalid"})

		rr := httptest.NewRecorder()

		handler.GetUserPermissions(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		userID := uuid.New()
		orgID := uuid.New()

		mockService.On("GetUserPermissions", mock.Anything, userID, orgID, (*string)(nil)).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/users/"+userID.String()+"/permissions", nil)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String()})

		authCtx := &AuthContext{UserID: uuid.New(), OrganizationID: orgID}
		ctx := context.WithValue(req.Context(), authContextKey, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler.GetUserPermissions(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockService.AssertExpectations(t)
	})
}
func TestHandler_CreateOrganization_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/organizations", bytes.NewBufferString("invalid json"))

		rr := httptest.NewRecorder()
		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		org := &Organization{Name: "Error Org", Slug: "error-org"}

		mockService.On("CreateOrganization", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(org)
		req, _ := http.NewRequest("POST", "/organizations", bytes.NewBuffer(body))

		rr := httptest.NewRecorder()
		handler.CreateOrganization(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetOrganization_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/organizations/invalid", nil)
		req = addAuthContext(req)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgView, mock.Anything).Return(true, nil)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.GetOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgView, mock.Anything).Return(true, nil)
		mockService.On("GetOrganization", mock.Anything, orgID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/organizations/"+orgID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.GetOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_UpdateOrganization_Errors(t *testing.T) {
	orgID := uuid.New()
	org := &Organization{ID: orgID, Name: "Updated Org"}

	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/organizations/invalid", nil)
		req = addAuthContext(req)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgManage, mock.Anything).Return(true, nil)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.UpdateOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/organizations/"+orgID.String(), bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgManage, mock.Anything).Return(true, nil)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.UpdateOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgManage, mock.Anything).Return(true, nil)
		mockService.On("UpdateOrganization", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(org)
		req, _ := http.NewRequest("PUT", "/organizations/"+orgID.String(), bytes.NewBuffer(body))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.UpdateOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_DeleteOrganization_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/organizations/invalid", nil)
		req = addAuthContext(req)

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgManage, mock.Anything).Return(true, nil)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.DeleteOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionOrgManage, mock.Anything).Return(true, nil)
		mockService.On("DeleteOrganization", mock.Anything, orgID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/organizations/"+orgID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/organizations/{orgId}", handler.DeleteOrganization)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ListOrganizations_Errors(t *testing.T) {
	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("ListOrganizations", mock.Anything, 50, 0).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/organizations", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		handler.ListOrganizations(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_CreateRole_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/roles", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.CreateRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/roles", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.CreateRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		role := &Role{Name: "Error Role"}
		orgID := uuid.New()

		mockService.On("CreateRole", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(role)
		req, _ := http.NewRequest("POST", "/roles", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.CreateRole(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetRole_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/roles/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.GetRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		roleID := uuid.New()

		mockService.On("GetRole", mock.Anything, roleID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/roles/"+roleID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.GetRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_UpdateRole_Errors(t *testing.T) {
	roleID := uuid.New()
	role := &Role{ID: roleID, Name: "Updated Role"}

	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/roles/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.UpdateRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.UpdateRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("UpdateRole", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(role)
		req, _ := http.NewRequest("PUT", "/roles/"+roleID.String(), bytes.NewBuffer(body))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.UpdateRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_DeleteRole_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/roles/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.DeleteRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		roleID := uuid.New()

		mockService.On("DeleteRole", mock.Anything, roleID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/roles/"+roleID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/roles/{roleId}", handler.DeleteRole)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ListRoles_Errors(t *testing.T) {
	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/roles", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.ListRoles(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("ListRoles", mock.Anything, orgID, 50, 0).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/roles", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.ListRoles(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_CreatePermission_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/permissions", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.CreatePermission(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/permissions", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.CreatePermission(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		permission := &Permission{Name: "Error Permission"}
		orgID := uuid.New()

		mockService.On("CreatePermission", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(permission)
		req, _ := http.NewRequest("POST", "/permissions", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.CreatePermission(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetPermission_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/permissions/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/permissions/{permissionId}", handler.GetPermission)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		permID := uuid.New()

		mockService.On("GetPermission", mock.Anything, permID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/permissions/"+permID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/permissions/{permissionId}", handler.GetPermission)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_ListPermissions_Errors(t *testing.T) {
	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("ListPermissions", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/permissions", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.ListPermissions(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_AssignPermissionToRole_Errors(t *testing.T) {
	roleID := uuid.New()
	permID := uuid.New()

	t.Run("Invalid Role ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/roles/invalid/permissions", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": "invalid"})

		rr := httptest.NewRecorder()
		handler.AssignPermissionToRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String()})
		rr := httptest.NewRecorder()
		handler.AssignPermissionToRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Permission ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		body := map[string]string{"permission_id": "invalid"}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(jsonBody))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String()})
		rr := httptest.NewRecorder()
		handler.AssignPermissionToRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("AssignPermissionToRole", mock.Anything, roleID, permID).Return(errors.New("db error"))

		body := map[string]string{"permission_id": permID.String()}
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/roles/"+roleID.String()+"/permissions", bytes.NewBuffer(jsonBody))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String()})

		rr := httptest.NewRecorder()
		handler.AssignPermissionToRole(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_RemovePermissionFromRole_Errors(t *testing.T) {
	roleID := uuid.New()
	permID := uuid.New()

	t.Run("Invalid Role ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/roles/invalid/permissions/"+permID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": "invalid", "permissionId": permID.String()})
		rr := httptest.NewRecorder()
		handler.RemovePermissionFromRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Permission ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions/invalid", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String(), "permissionId": "invalid"})
		rr := httptest.NewRecorder()
		handler.RemovePermissionFromRole(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("RemovePermissionFromRole", mock.Anything, roleID, permID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/roles/"+roleID.String()+"/permissions/"+permID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"roleId": roleID.String(), "permissionId": permID.String()})
		rr := httptest.NewRecorder()
		handler.RemovePermissionFromRole(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_RemoveRoleFromUser_Errors(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	orgID := uuid.New()

	t.Run("Invalid User ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/users/invalid/roles/"+roleID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": "invalid", "roleId": roleID.String(), "orgId": orgID.String()})
		rr := httptest.NewRecorder()
		handler.RemoveRoleFromUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Role ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/users/"+userID.String()+"/roles/invalid", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String(), "roleId": "invalid", "orgId": orgID.String()})
		rr := httptest.NewRecorder()
		handler.RemoveRoleFromUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String(), "roleId": roleID.String(), "orgId": "invalid"})
		rr := httptest.NewRecorder()
		handler.RemoveRoleFromUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("RemoveRoleFromUser", mock.Anything, userID, roleID, orgID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/users/"+userID.String()+"/roles/"+roleID.String(), nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String(), "roleId": roleID.String(), "orgId": orgID.String()})
		rr := httptest.NewRecorder()
		handler.RemoveRoleFromUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_CreateResourceType_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/resource-types", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})
		rr := httptest.NewRecorder()
		handler.CreateResourceType(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/resource-types", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})
		rr := httptest.NewRecorder()
		handler.CreateResourceType(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		resourceType := &ResourceType{Name: "Error Resource Type"}
		orgID := uuid.New()

		mockService.On("CreateResourceType", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(resourceType)
		req, _ := http.NewRequest("POST", "/resource-types", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.CreateResourceType(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetResourceType_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/resource-types/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resource-types/{resourceTypeId}", handler.GetResourceType)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		rtID := uuid.New()

		mockService.On("GetResourceType", mock.Anything, rtID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/resource-types/"+rtID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resource-types/{resourceTypeId}", handler.GetResourceType)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_ListResourceTypes_Errors(t *testing.T) {
	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/resource-types", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.ListResourceTypes(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("ListResourceTypes", mock.Anything, orgID).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/resource-types", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.ListResourceTypes(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_CreateResource_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/resources", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.CreateResource(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/resources", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.CreateResource(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		resource := &Resource{Name: "Error Resource"}
		orgID := uuid.New()

		mockService.On("CreateResource", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(resource)
		req, _ := http.NewRequest("POST", "/resources", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.CreateResource(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetResource_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/resources/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.GetResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		resourceID := uuid.New()

		mockService.On("GetResource", mock.Anything, resourceID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/resources/"+resourceID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.GetResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestHandler_UpdateResource_Errors(t *testing.T) {
	resourceID := uuid.New()
	resource := &Resource{ID: resourceID, Name: "Updated Resource"}

	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/resources/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.UpdateResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/resources/"+resourceID.String(), bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.UpdateResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)

		mockService.On("UpdateResource", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body, _ := json.Marshal(resource)
		req, _ := http.NewRequest("PUT", "/resources/"+resourceID.String(), bytes.NewBuffer(body))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.UpdateResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_DeleteResource_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("DELETE", "/resources/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.DeleteResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		resourceID := uuid.New()

		mockService.On("DeleteResource", mock.Anything, resourceID).Return(errors.New("db error"))

		req, _ := http.NewRequest("DELETE", "/resources/"+resourceID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/resources/{resourceId}", handler.DeleteResource)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ListResources_Errors(t *testing.T) {
	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/resources", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.ListResources(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("ListResources", mock.Anything, orgID, (*uuid.UUID)(nil), (*string)(nil)).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/resources", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.ListResources(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GrantEmergencyAccess_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("POST", "/emergency-access/grant", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.GrantEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("POST", "/emergency-access/grant", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.GrantEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("GrantEmergencyAccess", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body := []byte(`{"user_id": "` + uuid.New().String() + `", "role_id": "` + uuid.New().String() + `", "reason": "test", "duration": "1h"}`)
		req, _ := http.NewRequest("POST", "/emergency-access/grant", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		// Add auth context
		ctx := context.WithValue(req.Context(), authContextKey, &AuthContext{UserID: uuid.New()})
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GrantEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_RevokeEmergencyAccess_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("POST", "/emergency-access/revoke", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.RevokeEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("POST", "/emergency-access/revoke", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.RevokeEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()
		accessID := uuid.New()

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("RevokeEmergencyAccess", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

		body := []byte(`{"user_id": "` + uuid.New().String() + `", "role_id": "` + uuid.New().String() + `"}`)
		req, _ := http.NewRequest("POST", "/emergency-access/revoke", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String(), "accessId": accessID.String()})

		// Add auth context
		ctx := context.WithValue(req.Context(), authContextKey, &AuthContext{UserID: uuid.New()})
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.RevokeEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetUserEmergencyAccess_Errors(t *testing.T) {
	t.Run("Invalid User ID", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		req, _ := http.NewRequest("GET", "/users/invalid/emergency-access", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/users/{userId}/emergency-access", handler.GetUserEmergencyAccess)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		userID := uuid.New()
		orgID := uuid.New()

		mockService.On("HasPermission", mock.Anything, mock.Anything, mock.Anything, PermissionRBACManage, mock.Anything).Return(true, nil)
		mockService.On("GetActiveEmergencyAccess", mock.Anything, userID, orgID).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/users/"+userID.String()+"/emergency-access", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"userId": userID.String(), "orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.GetUserEmergencyAccess(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_CreateEmergencyAccessRequest_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/emergency-access/requests", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.CreateEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/emergency-access/requests", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.CreateEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("CreateEmergencyAccessRequest", mock.Anything, mock.Anything).Return(errors.New("db error"))

		body := []byte(`{"user_id": "` + uuid.New().String() + `", "role_id": "` + uuid.New().String() + `", "reason": "test", "duration": "1h", "requested_permissions": ["perm1"]}`)
		req, _ := http.NewRequest("POST", "/emergency-access/requests", bytes.NewBuffer(body))
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		// Add auth context
		ctx := context.WithValue(req.Context(), authContextKey, &AuthContext{UserID: uuid.New()})
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.CreateEmergencyAccessRequest(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ListEmergencyAccessRequests_Errors(t *testing.T) {
	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/emergency-access/requests", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.ListEmergencyAccessRequests(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("ListEmergencyAccessRequests", mock.Anything, orgID, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/emergency-access/requests", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.ListEmergencyAccessRequests(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetEmergencyAccessRequest_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/emergency-access/requests/invalid", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}", handler.GetEmergencyAccessRequest)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		mockService.On("GetEmergencyAccessRequest", mock.Anything, requestID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/emergency-access/requests/"+requestID.String(), nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}", handler.GetEmergencyAccessRequest)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ApproveEmergencyAccessRequest_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/emergency-access/requests/invalid/approve", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}/approve", handler.ApproveEmergencyAccessRequest)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		requestID := uuid.New()
		req, _ := http.NewRequest("POST", "/emergency-access/requests/"+requestID.String()+"/approve", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}/approve", handler.ApproveEmergencyAccessRequest)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		mockService.On("ApproveEmergencyAccessRequest", mock.Anything, requestID, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

		body := []byte(`{"approver_id": "` + uuid.New().String() + `", "action": "approve", "reason": "test"}`)
		req, _ := http.NewRequest("POST", "/emergency-access/requests/"+requestID.String()+"/approve", bytes.NewBuffer(body))
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}/approve", handler.ApproveEmergencyAccessRequest)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetEmergencyAccessApprovals_Errors(t *testing.T) {
	t.Run("Invalid ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/emergency-access/requests/invalid/approvals", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}/approvals", handler.GetEmergencyAccessApprovals)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		mockService.On("GetEmergencyAccessApprovals", mock.Anything, requestID).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/emergency-access/requests/"+requestID.String()+"/approvals", nil)
		req = addAuthContext(req)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/emergency-access/requests/{requestId}/approvals", handler.GetEmergencyAccessApprovals)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_ProcessBreakGlassAccess_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/emergency-access/break-glass", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.ProcessBreakGlassAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("POST", "/emergency-access/break-glass", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.ProcessBreakGlassAccess(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		requestID := uuid.New()

		mockService.On("ProcessBreakGlassAccess", mock.Anything, requestID).Return(errors.New("db error"))

		req, _ := http.NewRequest("POST", "/emergency-access/requests/"+requestID.String()+"/break-glass", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"requestId": requestID.String()})

		rr := httptest.NewRecorder()
		handler.ProcessBreakGlassAccess(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_GetBreakGlassConfig_Errors(t *testing.T) {
	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("GET", "/emergency-access/break-glass/config", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.GetBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("GetBreakGlassConfig", mock.Anything, orgID).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest("GET", "/emergency-access/break-glass/config", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.GetBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestHandler_UpdateBreakGlassConfig_Errors(t *testing.T) {
	t.Run("Invalid JSON", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/emergency-access/break-glass/config", bytes.NewBufferString("invalid json"))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": uuid.New().String()})

		rr := httptest.NewRecorder()
		handler.UpdateBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Invalid Org ID", func(t *testing.T) {
		handler, _ := setupHandlerTest(t)
		req, _ := http.NewRequest("PUT", "/emergency-access/break-glass/config", nil)
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": "invalid"})

		rr := httptest.NewRecorder()
		handler.UpdateBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		handler, mockService := setupHandlerTest(t)
		orgID := uuid.New()

		mockService.On("UpdateBreakGlassConfig", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

		body := []byte(`{"require_approval": true}`)
		req, _ := http.NewRequest("PUT", "/emergency-access/break-glass/config", bytes.NewBuffer(body))
		req = addAuthContext(req)
		req = mux.SetURLVars(req, map[string]string{"orgId": orgID.String()})

		rr := httptest.NewRecorder()
		handler.UpdateBreakGlassConfig(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
