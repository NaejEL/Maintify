import axios from 'axios';

const API_BASE_URL = '/api';

const api = axios.create({
    baseURL: API_BASE_URL,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Add token to requests
api.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Auth API
export const authAPI = {
    login: (email, password) =>
        api.post('/rbac/auth/login', { email, password }),
};

// RBAC API
export const rbacAPI = {
    // Users
    listUsers: () => api.get('/rbac/users'),
    createUser: (userData) => api.post('/rbac/users', userData),
    updateUser: (userId, userData) => api.put(`/rbac/users/${userId}`, userData),
    deactivateUser: (userId) => api.post(`/rbac/users/${userId}/deactivate`),
    
    // Organizations
    listOrgs: () => api.get('/rbac/organizations'),
    createOrg: (orgData) => api.post('/rbac/organizations', orgData),
    deleteOrg: (orgId) => api.delete(`/rbac/organizations/${orgId}`),
    
    // Roles
    listRoles: (orgId) => api.get(`/rbac/organizations/${orgId}/roles`),
    getRole: (orgId, roleId) => api.get(`/rbac/organizations/${orgId}/roles/${roleId}`),
    createRole: (orgId, roleData) => api.post(`/rbac/organizations/${orgId}/roles`, roleData),
    deleteRole: (orgId, roleId) => api.delete(`/rbac/organizations/${orgId}/roles/${roleId}`),
    
    // Permissions
    listPermissions: (orgId) => api.get(`/rbac/organizations/${orgId}/permissions`),
    createPermission: (orgId, permData) => api.post(`/rbac/organizations/${orgId}/permissions`, permData),
    assignPermission: (orgId, roleId, permissionId) => api.post(`/rbac/organizations/${orgId}/roles/${roleId}/permissions`, { permission_id: permissionId }),
    removePermission: (orgId, roleId, permissionId) => api.delete(`/rbac/organizations/${orgId}/roles/${roleId}/permissions/${permissionId}`),

    // Assignments
    assignRole: (orgId, assignmentData) => api.post(`/rbac/organizations/${orgId}/assignments`, assignmentData),
    removeRoleFromUser: (orgId, userId, roleId) => api.delete(`/rbac/organizations/${orgId}/assignments/${userId}/${roleId}`),
    getUserRoles: (userId) => api.get(`/rbac/users/${userId}/roles`),

    // Audit
    getAuditLog: (orgId) => api.get(`/rbac/organizations/${orgId}/audit`),
};

// Plugin API
export const pluginAPI = {
    list: () => api.get('/plugins'),
    status: () => api.get('/plugins/status'),
    start: (name) => api.post(`/plugins/${name}/start`),
    stop: (name) => api.post(`/plugins/${name}/stop`),
    restart: (name) => api.post(`/plugins/${name}/restart`),
    diagnostics: () => api.get('/plugins/diagnostics'),
};

// Health API
export const healthAPI = {
    check: () => axios.get('/health'),
};

export default api;
