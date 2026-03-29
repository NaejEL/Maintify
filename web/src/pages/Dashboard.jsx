import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { pluginAPI, healthAPI, rbacAPI } from '../api';
import './Dashboard.css';

function Dashboard() {
    const [activeTab, setActiveTab] = useState('overview');
    const [plugins, setPlugins] = useState([]);
    const [health, setHealth] = useState(null);
    const [users, setUsers] = useState([]);
    const [orgs, setOrgs] = useState([]);
    const [roles, setRoles] = useState([]);
    const [permissions, setPermissions] = useState([]);
    const [auditLogs, setAuditLogs] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [showModal, setShowModal] = useState(false);
    const [modalType, setModalType] = useState(''); // 'createUser', 'createOrg', 'createRole', 'createPermission'
    const [formData, setFormData] = useState({});
    
    const navigate = useNavigate();
    const user = JSON.parse(localStorage.getItem('user') || '{}');

    useEffect(() => {
        loadData();
    }, []);

    useEffect(() => {
        if (modalType === 'manageRolePermissions' && formData.role && orgs.length > 0) {
            rbacAPI.getRole(orgs[0].id, formData.role.id)
                .then(res => {
                    setFormData(prev => ({ ...prev, role: res.data }));
                })
                .catch(console.error);
        }
    }, [modalType]);

    const loadData = async () => {
        try {
            const [pluginsRes, healthRes, usersRes, orgsRes] = await Promise.all([
                pluginAPI.list().catch(() => ({ data: [] })),
                healthAPI.check().catch(() => ({ data: { status: 'unknown' } })),
                rbacAPI.listUsers().catch(() => ({ data: [] })),
                rbacAPI.listOrgs().catch(() => ({ data: [] })),
            ]);

            setPlugins(pluginsRes.data || []);
            setHealth(healthRes.data);
            setUsers(usersRes.data || []);
            setOrgs(orgsRes.data || []);

            // Fetch roles and permissions for the first org if available
            if (orgsRes.data && orgsRes.data.length > 0) {
                const orgId = orgsRes.data[0].id;
                const [rolesRes, permissionsRes] = await Promise.all([
                    rbacAPI.listRoles(orgId).catch(() => ({ data: [] })),
                    rbacAPI.listPermissions(orgId).catch(() => ({ data: [] }))
                ]);
                setRoles(rolesRes.data || []);
                setPermissions(permissionsRes.data || []);
            }

        } catch (err) {
            if (err.response?.status === 401) {
                localStorage.clear();
                navigate('/');
            } else {
                console.error(err);
                setError('Failed to load data');
            }
        } finally {
            setLoading(false);
        }
    };

    const handleLogout = () => {
        localStorage.clear();
        navigate('/');
    };

    const handleAction = async (action, type, id, data) => {
        try {
            if (action === 'delete') {
                if (!window.confirm('Are you sure you want to delete this item?')) return;
                
                if (type === 'user') {
                    // Note: API doesn't have delete user, only deactivate
                    await rbacAPI.deactivateUser(id);
                } else if (type === 'org') {
                    await rbacAPI.deleteOrg(id);
                } else if (type === 'role') {
                    // Assuming first org for now
                    await rbacAPI.deleteRole(orgs[0].id, id);
                }
            } else if (action === 'create') {
                if (type === 'user') {
                    await rbacAPI.createUser(data);
                } else if (type === 'org') {
                    await rbacAPI.createOrg(data);
                } else if (type === 'role') {
                    await rbacAPI.createRole(orgs[0].id, data);
                } else if (type === 'permission') {
                    await rbacAPI.createPermission(orgs[0].id, data);
                } else if (type === 'assignment') {
                    await rbacAPI.assignRole(orgs[0].id, {
                        user_id: data.userId,
                        role_id: data.roleId
                    });
                } else if (type === 'rolePermission') {
                    await rbacAPI.assignPermission(orgs[0].id, data.roleId, data.permissionId);
                }
                setShowModal(false);
                setFormData({});
            } else if (action === 'remove') {
                if (type === 'rolePermission') {
                    if (!window.confirm('Remove this permission from the role?')) return;
                    await rbacAPI.removePermission(orgs[0].id, id.roleId, id.permissionId);
                }
            }
            
            // Reload data
            loadData();
        } catch (err) {
            console.error(err);
            alert('Action failed: ' + (err.response?.data?.message || err.message));
        }
    };

    const openModal = (type, data = {}) => {
        setModalType(type);
        setFormData(data);
        setShowModal(true);
    };

    if (loading) {
        return <div className="loading"><div className="loading-spinner"></div></div>;
    }

    return (
        <div className="dashboard-container">
            {/* Sidebar */}
            <aside className="sidebar">
                <div className="sidebar-header">
                    <h1>Maintify</h1>
                </div>
                
                <nav className="sidebar-nav">
                    <button 
                        className={`nav-item ${activeTab === 'overview' ? 'active' : ''}`}
                        onClick={() => setActiveTab('overview')}
                    >
                        Overview
                    </button>
                    <button 
                        className={`nav-item ${activeTab === 'plugins' ? 'active' : ''}`}
                        onClick={() => setActiveTab('plugins')}
                    >
                        Plugins
                    </button>
                    <button 
                        className={`nav-item ${activeTab === 'rbac' ? 'active' : ''}`}
                        onClick={() => setActiveTab('rbac')}
                    >
                        Access Control
                    </button>
                    <button 
                        className={`nav-item ${activeTab === 'audit' ? 'active' : ''}`}
                        onClick={() => {
                            setActiveTab('audit');
                            if (orgs.length > 0) {
                                rbacAPI.getAuditLog(orgs[0].id)
                                    .then(res => setAuditLogs(res.data || []))
                                    .catch(console.error);
                            }
                        }}
                    >
                        Audit Logs
                    </button>
                </nav>

                <div className="sidebar-footer">
                    <div className="user-profile">
                        <div className="user-avatar">
                            {user.email ? user.email[0].toUpperCase() : 'U'}
                        </div>
                        <div className="user-details">
                            <span className="user-email">{user.email}</span>
                            <span className="user-role">{user.is_system_admin ? 'System Admin' : 'User'}</span>
                        </div>
                    </div>
                    <button onClick={handleLogout} className="logout-btn">Sign Out</button>
                </div>
            </aside>

            {/* Main Content */}
            <main className="main-content">
                {error && <div className="error-banner">{error}</div>}

                {activeTab === 'overview' && (
                    <>
                        <div className="page-header">
                            <div className="page-title">
                                <h2>System Overview</h2>
                                <p>Monitor system health and key metrics</p>
                            </div>
                            <button className="action-btn secondary" onClick={loadData}>Refresh</button>
                        </div>

                        <div className="stats-grid">
                            <div className="stat-card">
                                <div className="stat-label">System Status</div>
                                <div className="stat-value">
                                    <span className={`badge ${health?.status === 'healthy' ? 'success' : 'error'}`}>
                                        {health?.status || 'Unknown'}
                                    </span>
                                </div>
                            </div>
                            <div className="stat-card">
                                <div className="stat-label">Active Plugins</div>
                                <div className="stat-value">{plugins.length}</div>
                            </div>
                            <div className="stat-card">
                                <div className="stat-label">Total Users</div>
                                <div className="stat-value">{users.length}</div>
                            </div>
                            <div className="stat-card">
                                <div className="stat-label">Organizations</div>
                                <div className="stat-value">{orgs.length}</div>
                            </div>
                        </div>

                        <div className="content-card">
                            <div className="modal-header">
                                <h3>Component Health</h3>
                            </div>
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Component</th>
                                            <th>Status</th>
                                            <th>Last Check</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {health?.components && Object.entries(health.components).map(([key, value]) => (
                                            <tr key={key}>
                                                <td>{key}</td>
                                                <td>
                                                    <span className={`badge ${value?.status === 'healthy' ? 'success' : 'error'}`}>
                                                        {value?.status || 'Unknown'}
                                                    </span>
                                                </td>
                                                <td>{new Date().toLocaleTimeString()}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </>
                )}

                {activeTab === 'plugins' && (
                    <>
                        <div className="page-header">
                            <div className="page-title">
                                <h2>Plugins</h2>
                                <p>Manage installed system plugins</p>
                            </div>
                        </div>

                        <div className="content-card">
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Name</th>
                                            <th>Version</th>
                                            <th>Description</th>
                                            <th>Resources</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {plugins.length === 0 ? (
                                            <tr>
                                                <td colSpan="4" style={{textAlign: 'center', padding: '40px'}}>No plugins registered</td>
                                            </tr>
                                        ) : (
                                            plugins.map((plugin) => (
                                                <tr key={plugin.name}>
                                                    <td><strong>{plugin.name}</strong></td>
                                                    <td><span className="badge neutral">v{plugin.version}</span></td>
                                                    <td>{plugin.description}</td>
                                                    <td>
                                                        {plugin.resources && (
                                                            <small>CPU: {plugin.resources.cpu_milli_cores}m | Mem: {plugin.resources.memory_mb}MB</small>
                                                        )}
                                                    </td>
                                                </tr>
                                            ))
                                        )}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </>
                )}

                {activeTab === 'rbac' && (
                    <>
                        <div className="page-header">
                            <div className="page-title">
                                <h2>Access Control</h2>
                                <p>Manage users, organizations, and roles</p>
                            </div>
                        </div>

                        {/* Users Section */}
                        <div className="content-card" style={{marginBottom: '32px'}}>
                            <div className="modal-header">
                                <h3>Users</h3>
                                <button className="action-btn small" onClick={() => openModal('createUser')}>Add User</button>
                            </div>
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>User</th>
                                            <th>Email</th>
                                            <th>Status</th>
                                            <th>Role</th>
                                            <th>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {users.map(u => (
                                            <tr key={u.id}>
                                                <td>{u.first_name} {u.last_name} <br/><small style={{color: '#94a3b8'}}>{u.username}</small></td>
                                                <td>{u.email}</td>
                                                <td>
                                                    <span className={`badge ${u.is_active ? 'success' : 'error'}`}>
                                                        {u.is_active ? 'Active' : 'Inactive'}
                                                    </span>
                                                </td>
                                                <td>{u.is_system_admin ? 'System Admin' : 'User'}</td>
                                                <td className="actions-cell">
                                                    <button 
                                                        className="action-btn small"
                                                        style={{marginRight: '8px'}}
                                                        onClick={() => openModal('assignRole', { userId: u.id })}
                                                    >
                                                        Assign Role
                                                    </button>
                                                    <button 
                                                        className="action-btn small danger"
                                                        onClick={() => handleAction('delete', 'user', u.id)}
                                                        disabled={!u.is_active}
                                                    >
                                                        Deactivate
                                                    </button>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>

                        {/* Organizations Section */}
                        <div className="content-card" style={{marginBottom: '32px'}}>
                            <div className="modal-header">
                                <h3>Organizations</h3>
                                <button className="action-btn small" onClick={() => openModal('createOrg')}>Add Org</button>
                            </div>
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Name</th>
                                            <th>Slug</th>
                                            <th>Description</th>
                                            <th>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {orgs.map(o => (
                                            <tr key={o.id}>
                                                <td><strong>{o.name}</strong></td>
                                                <td>{o.slug}</td>
                                                <td>{o.description}</td>
                                                <td className="actions-cell">
                                                    <button 
                                                        className="action-btn small danger"
                                                        onClick={() => handleAction('delete', 'org', o.id)}
                                                    >
                                                        Delete
                                                    </button>
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>

                        {/* Roles Section */}
                        <div className="content-card" style={{marginBottom: '32px'}}>
                            <div className="modal-header">
                                <h3>Roles</h3>
                                <button className="action-btn small" onClick={() => openModal('createRole')}>Add Role</button>
                            </div>
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Name</th>
                                            <th>Description</th>
                                            <th>System Role</th>
                                            <th>Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {roles.map(r => (
                                            <tr key={r.id}>
                                                <td><strong>{r.name}</strong></td>
                                                <td>{r.description}</td>
                                                <td>{r.is_system_role ? 'Yes' : 'No'}</td>
                                                <td className="actions-cell">
                                                    <button 
                                                        className="action-btn small"
                                                        style={{marginRight: '8px'}}
                                                        onClick={() => openModal('manageRolePermissions', { role: r })}
                                                    >
                                                        Permissions
                                                    </button>
                                                    {!r.is_system_role && (
                                                        <button 
                                                            className="action-btn small danger"
                                                            onClick={() => handleAction('delete', 'role', r.id)}
                                                        >
                                                            Delete
                                                        </button>
                                                    )}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>

                        {/* Permissions Section */}
                        <div className="content-card">
                            <div className="modal-header">
                                <h3>Permissions</h3>
                                <button className="action-btn small" onClick={() => openModal('createPermission')}>Add Permission</button>
                            </div>
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Name</th>
                                            <th>Description</th>
                                            <th>Action</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {permissions.map(p => (
                                            <tr key={p.id}>
                                                <td><strong>{p.name}</strong></td>
                                                <td>{p.description}</td>
                                                <td><span className="badge neutral">{p.action}</span></td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </>
                )}

                {activeTab === 'audit' && (
                    <>
                        <div className="page-header">
                            <div className="page-title">
                                <h2>Audit Logs</h2>
                                <p>View system activity and security events</p>
                            </div>
                            <button className="action-btn secondary" onClick={() => {
                                if (orgs.length > 0) {
                                    rbacAPI.getAuditLog(orgs[0].id)
                                        .then(res => setAuditLogs(res.data || []))
                                        .catch(console.error);
                                }
                            }}>Refresh</button>
                        </div>

                        <div className="content-card">
                            <div className="table-wrapper">
                                <table>
                                    <thead>
                                        <tr>
                                            <th>Time</th>
                                            <th>User</th>
                                            <th>Action</th>
                                            <th>Resource</th>
                                            <th>Status</th>
                                            <th>IP Address</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {auditLogs.length === 0 ? (
                                            <tr>
                                                <td colSpan="6" style={{textAlign: 'center', padding: '40px'}}>No audit logs found</td>
                                            </tr>
                                        ) : (
                                            auditLogs.map((log) => (
                                                <tr key={log.id}>
                                                    <td>{new Date(log.created_at).toLocaleString()}</td>
                                                    <td>{users.find(u => u.id === log.user_id)?.username || 'System'}</td>
                                                    <td><strong>{log.action}</strong></td>
                                                    <td>{log.resource_type}</td>
                                                    <td>
                                                        <span className={`badge ${log.success ? 'success' : 'error'}`}>
                                                            {log.success ? 'Success' : 'Failed'}
                                                        </span>
                                                    </td>
                                                    <td>{log.ip_address}</td>
                                                </tr>
                                            ))
                                        )}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </>
                )}
            </main>

            {/* Modal */}
            {showModal && (
                <div className="modal-overlay" onClick={(e) => {if(e.target === e.currentTarget) setShowModal(false)}}>
                    <div className="modal">
                        <div className="modal-header">
                            <h3>
                                {modalType === 'createUser' && 'Create New User'}
                                {modalType === 'createOrg' && 'Create Organization'}
                                {modalType === 'createRole' && 'Create Role'}
                                {modalType === 'createPermission' && 'Create Permission'}
                                {modalType === 'assignRole' && 'Assign Role to User'}
                                {modalType === 'manageRolePermissions' && `Manage Permissions: ${formData.role?.name}`}
                            </h3>
                            <button className="close-btn" onClick={() => setShowModal(false)}>&times;</button>
                        </div>
                        <div className="modal-body">
                            {modalType === 'createUser' && (
                                <>
                                    <div className="form-field">
                                        <label>Username</label>
                                        <input type="text" onChange={e => setFormData({...formData, username: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Email</label>
                                        <input type="email" onChange={e => setFormData({...formData, email: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>First Name</label>
                                        <input type="text" onChange={e => setFormData({...formData, first_name: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Last Name</label>
                                        <input type="text" onChange={e => setFormData({...formData, last_name: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Password</label>
                                        <input type="password" onChange={e => setFormData({...formData, password: e.target.value})} />
                                    </div>
                                </>
                            )}
                            {modalType === 'createOrg' && (
                                <>
                                    <div className="form-field">
                                        <label>Name</label>
                                        <input type="text" onChange={e => setFormData({...formData, name: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Slug</label>
                                        <input type="text" onChange={e => setFormData({...formData, slug: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Description</label>
                                        <textarea onChange={e => setFormData({...formData, description: e.target.value})}></textarea>
                                    </div>
                                </>
                            )}
                            {modalType === 'createRole' && (
                                <>
                                    <div className="form-field">
                                        <label>Name</label>
                                        <input type="text" onChange={e => setFormData({...formData, name: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Description</label>
                                        <textarea onChange={e => setFormData({...formData, description: e.target.value})}></textarea>
                                    </div>
                                </>
                            )}
                            {modalType === 'createPermission' && (
                                <>
                                    <div className="form-field">
                                        <label>Name (e.g. user.create)</label>
                                        <input type="text" onChange={e => setFormData({...formData, name: e.target.value})} />
                                    </div>
                                    <div className="form-field">
                                        <label>Description</label>
                                        <textarea onChange={e => setFormData({...formData, description: e.target.value})}></textarea>
                                    </div>
                                    <div className="form-field">
                                        <label>Action</label>
                                        <select 
                                            className="form-select"
                                            style={{width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #cbd5e1'}}
                                            onChange={e => setFormData({...formData, action: e.target.value})}
                                        >
                                            <option value="">Select action...</option>
                                            <option value="create">Create</option>
                                            <option value="read">Read</option>
                                            <option value="update">Update</option>
                                            <option value="delete">Delete</option>
                                            <option value="manage">Manage</option>
                                        </select>
                                    </div>
                                </>
                            )}
                            {modalType === 'assignRole' && (
                                <div className="form-field">
                                    <label>Select Role</label>
                                    <select 
                                        className="form-select"
                                        style={{width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid #cbd5e1'}}
                                        onChange={e => setFormData({...formData, roleId: e.target.value})}
                                    >
                                        <option value="">Select a role...</option>
                                        {roles.map(r => (
                                            <option key={r.id} value={r.id}>{r.name}</option>
                                        ))}
                                    </select>
                                </div>
                            )}
                            {modalType === 'manageRolePermissions' && (
                                <>
                                    <div className="form-field">
                                        <label>Add Permission</label>
                                        <div style={{display: 'flex', gap: '8px'}}>
                                            <select 
                                                className="form-select"
                                                style={{flex: 1, padding: '8px', borderRadius: '4px', border: '1px solid #cbd5e1'}}
                                                onChange={e => setFormData({...formData, selectedPermissionId: e.target.value})}
                                            >
                                                <option value="">Select permission to add...</option>
                                                {permissions.map(p => (
                                                    <option key={p.id} value={p.id}>{p.name}</option>
                                                ))}
                                            </select>
                                            <button 
                                                className="action-btn small"
                                                onClick={() => handleAction('create', 'rolePermission', null, { roleId: formData.role.id, permissionId: formData.selectedPermissionId })}
                                            >
                                                Add
                                            </button>
                                        </div>
                                    </div>
                                    <div style={{marginTop: '16px'}}>
                                        <label style={{display: 'block', marginBottom: '8px', fontSize: '14px', fontWeight: '500'}}>Current Permissions</label>
                                        {formData.role?.permissions && formData.role.permissions.length > 0 ? (
                                            <div style={{display: 'flex', flexWrap: 'wrap', gap: '8px'}}>
                                                {formData.role.permissions.map(p => (
                                                    <span key={p.id} className="badge neutral" style={{display: 'flex', alignItems: 'center', gap: '6px'}}>
                                                        {p.name}
                                                        <span 
                                                            style={{cursor: 'pointer', fontWeight: 'bold'}}
                                                            onClick={() => handleAction('remove', 'rolePermission', { roleId: formData.role.id, permissionId: p.id })}
                                                        >
                                                            &times;
                                                        </span>
                                                    </span>
                                                ))}
                                            </div>
                                        ) : (
                                            <p style={{color: '#94a3b8', fontSize: '14px'}}>No permissions assigned</p>
                                        )}
                                    </div>
                                </>
                            )}
                        </div>
                        <div className="modal-footer">
                            <button className="action-btn secondary" onClick={() => setShowModal(false)}>Close</button>
                            {modalType !== 'manageRolePermissions' && (
                                <button 
                                    className="action-btn"
                                    onClick={() => handleAction('create', modalType === 'createUser' ? 'user' : modalType === 'createOrg' ? 'org' : modalType === 'createRole' ? 'role' : modalType === 'createPermission' ? 'permission' : 'assignment', null, formData)}
                                >
                                    {modalType === 'assignRole' ? 'Assign' : 'Create'}
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

export default Dashboard;
