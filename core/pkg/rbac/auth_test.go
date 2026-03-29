package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login(t *testing.T) {
	jwtSecret := []byte("secret")
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	t.Run("Success", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		orgID := uuid.New()
		user := &User{
			ID:            userID,
			Email:         "test@example.com",
			PasswordHash:  string(hashedPassword),
			IsActive:      true,
			IsSystemAdmin: false,
		}
		org := &Organization{
			ID:   orgID,
			Slug: "test-org",
		}

		mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockService.On("GetOrganizationBySlug", mock.Anything, "test-org").Return(org, nil)
		mockService.On("GetUserRoles", mock.Anything, userID, orgID).Return([]*Role{{ID: uuid.New()}}, nil)
		mockService.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

		req := &LoginRequest{
			Email:    "test@example.com",
			Password: password,
			OrgSlug:  "test-org",
		}

		resp, err := authService.Login(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, user.ID, resp.User.ID)
		assert.Equal(t, org.ID, resp.Organization.ID)
		assert.NotEmpty(t, resp.Token)
	})

	t.Run("Invalid Credentials - User Not Found", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		mockService.On("GetUserByEmail", mock.Anything, "wrong@example.com").Return(nil, errors.New("not found"))

		req := &LoginRequest{
			Email:    "wrong@example.com",
			Password: password,
		}

		resp, err := authService.Login(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("Invalid Credentials - Wrong Password", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		user := &User{
			ID:           userID,
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     true,
		}

		mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)

		req := &LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		resp, err := authService.Login(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "invalid credentials", err.Error())
	})

	t.Run("Deactivated User", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		user := &User{
			ID:           userID,
			Email:        "inactive@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     false,
		}

		mockService.On("GetUserByEmail", mock.Anything, "inactive@example.com").Return(user, nil)

		req := &LoginRequest{
			Email:    "inactive@example.com",
			Password: password,
		}

		resp, err := authService.Login(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "user account is deactivated", err.Error())
	})

	t.Run("Organization Not Found", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		user := &User{
			ID:           userID,
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			IsActive:     true,
		}

		mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockService.On("GetOrganizationBySlug", mock.Anything, "non-existent").Return(nil, errors.New("not found"))

		req := &LoginRequest{
			Email:    "test@example.com",
			Password: password,
			OrgSlug:  "non-existent",
		}

		resp, err := authService.Login(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "organization not found", err.Error())
	})

	t.Run("No Access to Organization", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		orgID := uuid.New()
		user := &User{
			ID:            userID,
			Email:         "test@example.com",
			PasswordHash:  string(hashedPassword),
			IsActive:      true,
			IsSystemAdmin: false,
		}
		org := &Organization{
			ID:   orgID,
			Slug: "test-org",
		}

		mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
		mockService.On("GetOrganizationBySlug", mock.Anything, "test-org").Return(org, nil)
		mockService.On("GetUserRoles", mock.Anything, userID, orgID).Return([]*Role{}, nil) // No roles

		req := &LoginRequest{
			Email:    "test@example.com",
			Password: password,
			OrgSlug:  "test-org",
		}

		resp, err := authService.Login(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "no access to organization", err.Error())
	})

	t.Run("System Admin Login (No Org)", func(t *testing.T) {
		mockService := new(MockRBACService)
		authService := NewAuthService(mockService, jwtSecret)

		userID := uuid.New()
		orgID := uuid.New()
		user := &User{
			ID:            userID,
			Email:         "admin@example.com",
			PasswordHash:  string(hashedPassword),
			IsActive:      true,
			IsSystemAdmin: true,
		}
		org := &Organization{
			ID:   orgID,
			Slug: "system",
		}

		mockService.On("GetUserByEmail", mock.Anything, "admin@example.com").Return(user, nil)
		mockService.On("GetOrganizationBySlug", mock.Anything, "system").Return(org, nil)
		mockService.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

		req := &LoginRequest{
			Email:    "admin@example.com",
			Password: password,
		}

		resp, err := authService.Login(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, user.ID, resp.User.ID)
		assert.Equal(t, org.ID, resp.Organization.ID)
	})
}

func TestAuthService_GenerateJWT(t *testing.T) {
	// Since generateJWT is private, we test it implicitly via Login,
	// but we can also verify the token structure from the Login response.

	mockService := new(MockRBACService)
	jwtSecret := []byte("secret")
	authService := NewAuthService(mockService, jwtSecret)

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	userID := uuid.New()
	orgID := uuid.New()
	user := &User{
		ID:            userID,
		Email:         "test@example.com",
		PasswordHash:  string(hashedPassword),
		IsActive:      true,
		IsSystemAdmin: false,
	}
	org := &Organization{
		ID:   orgID,
		Slug: "test-org",
	}

	mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
	mockService.On("GetOrganizationBySlug", mock.Anything, "test-org").Return(org, nil)
	mockService.On("GetUserRoles", mock.Anything, userID, orgID).Return([]*Role{{ID: uuid.New()}}, nil)
	mockService.On("UpdateUser", mock.Anything, mock.Anything).Return(nil)

	req := &LoginRequest{
		Email:    "test@example.com",
		Password: password,
		OrgSlug:  "test-org",
	}

	resp, err := authService.Login(context.Background(), req)
	assert.NoError(t, err)

	// Parse and verify token
	token, err := jwt.ParseWithClaims(resp.Token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(*JWTClaims)
	assert.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, orgID, claims.OrganizationID)
}
