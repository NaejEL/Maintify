package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RBACClient provides a simple client for third-party plugins to interact with Maintify's RBAC system
type RBACClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewRBACClient creates a new RBAC client for plugins
func NewRBACClient(maintifyURL, token string) *RBACClient {
	return &RBACClient{
		baseURL: maintifyURL + "/api/rbac",
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AuthenticatePlugin authenticates a plugin user and returns a JWT token
func (c *RBACClient) AuthenticatePlugin(email, password string) (string, error) {
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}
	
	resp, err := c.makeRequest("POST", "/auth/login", loginData)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}
	
	var result struct {
		Token string `json:"token"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	c.token = result.Token
	return result.Token, nil
}

// CheckPermission verifies if the current user has permission to perform an action on a resource
func (c *RBACClient) CheckPermission(action, resourceType, resourceID string) (bool, error) {
	url := fmt.Sprintf("/permissions/check?action=%s&resource_type=%s&resource_id=%s", 
		action, resourceType, resourceID)
	
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("permission check failed: %w", err)
	}
	
	var result struct {
		HasPermission bool `json:"has_permission"`
	}
	
	if err := json.Unmarshal(resp, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result.HasPermission, nil
}

// GetUserInfo retrieves information about the current authenticated user
func (c *RBACClient) GetUserInfo() (*UserInfo, error) {
	resp, err := c.makeRequest("GET", "/user/current", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	
	var user UserInfo
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}
	
	return &user, nil
}

// GetUserPermissions retrieves all permissions for the current user
func (c *RBACClient) GetUserPermissions() ([]Permission, error) {
	resp, err := c.makeRequest("GET", "/user/permissions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	
	var permissions []Permission
	if err := json.Unmarshal(resp, &permissions); err != nil {
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}
	
	return permissions, nil
}

// ListResources retrieves all resources the user has access to
func (c *RBACClient) ListResources(resourceType string) ([]Resource, error) {
	url := "/resources"
	if resourceType != "" {
		url += "?type=" + resourceType
	}
	
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	
	var resources []Resource
	if err := json.Unmarshal(resp, &resources); err != nil {
		return nil, fmt.Errorf("failed to parse resources: %w", err)
	}
	
	return resources, nil
}

// CreateAuditLog creates an audit log entry for plugin actions
func (c *RBACClient) CreateAuditLog(action, description string, metadata map[string]interface{}) error {
	auditData := map[string]interface{}{
		"action":      action,
		"description": description,
		"metadata":    metadata,
		"timestamp":   time.Now(),
	}
	
	_, err := c.makeRequest("POST", "/audit", auditData)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	
	return nil
}

// RequestEmergencyAccess requests temporary elevated permissions
func (c *RBACClient) RequestEmergencyAccess(roleID, reason, emergencyType string, duration time.Duration) error {
	requestData := map[string]interface{}{
		"role_id":        roleID,
		"reason":         reason,
		"emergency_type": emergencyType,
		"duration":       duration.String(),
	}
	
	_, err := c.makeRequest("POST", "/emergency-access", requestData)
	if err != nil {
		return fmt.Errorf("failed to request emergency access: %w", err)
	}
	
	return nil
}

// Helper method to make authenticated HTTP requests
func (c *RBACClient) makeRequest(method, endpoint string, data interface{}) ([]byte, error) {
	var body io.Reader
	
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}
	
	req, err := http.NewRequest(method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

// Data structures matching the RBAC API responses
type UserInfo struct {
	ID             string            `json:"id"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	OrganizationID string            `json:"organization_id"`
	Department     string            `json:"department"`
	IsActive       bool              `json:"is_active"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Permission struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resource_type"`
	Conditions   map[string]interface{} `json:"conditions"`
}

type Resource struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ParentID     *string           `json:"parent_id"`
	Description  string            `json:"description"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Example usage in a plugin
func ExamplePluginUsage() {
	// Initialize RBAC client (typically done in plugin initialization)
	rbacClient := NewRBACClient("http://core:8080", "")
	
	// Authenticate the plugin user
	token, err := rbacClient.AuthenticatePlugin("plugin-user@example.com", "secure-password")
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		return
	}
	
	fmt.Printf("Authenticated successfully, token: %s\n", token)
	
	// Check if user can read equipment data
	canRead, err := rbacClient.CheckPermission("read", "equipment", "eq-123")
	if err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		return
	}
	
	if !canRead {
		fmt.Println("User does not have permission to read equipment data")
		return
	}
	
	// Get user information
	userInfo, err := rbacClient.GetUserInfo()
	if err != nil {
		fmt.Printf("Failed to get user info: %v\n", err)
		return
	}
	
	fmt.Printf("User: %s %s (%s)\n", userInfo.FirstName, userInfo.LastName, userInfo.Email)
	
	// List accessible equipment
	equipment, err := rbacClient.ListResources("equipment")
	if err != nil {
		fmt.Printf("Failed to list equipment: %v\n", err)
		return
	}
	
	fmt.Printf("User has access to %d pieces of equipment\n", len(equipment))
	
	// Create audit log for plugin action
	err = rbacClient.CreateAuditLog(
		"equipment_maintenance_scheduled",
		"Maintenance scheduled for equipment eq-123",
		map[string]interface{}{
			"equipment_id": "eq-123",
			"plugin_name": "maintenance-scheduler",
			"action_type": "schedule",
		},
	)
	if err != nil {
		fmt.Printf("Failed to create audit log: %v\n", err)
	}
	
	fmt.Println("Plugin operations completed successfully")
}