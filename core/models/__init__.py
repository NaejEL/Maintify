from config.database import db, ma
from datetime import datetime
from flask_bcrypt import generate_password_hash, check_password_hash
from enum import Enum

# Énumérations pour les types
class UserRole(Enum):
    ADMIN = "admin"
    TECHNICIAN = "technician" 
    USER = "user"

class AlertLevel(Enum):
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"

class AlertStatus(Enum):
    OPEN = "open"
    IN_PROGRESS = "in_progress"
    RESOLVED = "resolved"
    CLOSED = "closed"

class EquipmentStatus(Enum):
    OPERATIONAL = "operational"
    MAINTENANCE = "maintenance"
    OUT_OF_ORDER = "out_of_order"
    RETIRED = "retired"

# Table d'association pour les utilisateurs et les lieux
user_locations = db.Table('user_locations',
    db.Column('user_id', db.Integer, db.ForeignKey('users.id'), primary_key=True),
    db.Column('location_id', db.Integer, db.ForeignKey('locations.id'), primary_key=True)
)

class User(db.Model):
    __tablename__ = 'users'
    
    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(80), unique=True, nullable=False, index=True)
    email = db.Column(db.String(120), unique=True, nullable=False, index=True)
    password_hash = db.Column(db.String(255), nullable=False)
    first_name = db.Column(db.String(50), nullable=False)
    last_name = db.Column(db.String(50), nullable=False)
    role = db.Column(db.Enum(UserRole), nullable=False, default=UserRole.USER)
    is_active = db.Column(db.Boolean, default=True)
    phone = db.Column(db.String(20))
    
    # Dates
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    last_login = db.Column(db.DateTime)
    
    # Relations
    locations = db.relationship('Location', secondary=user_locations, backref='users')
    created_alerts = db.relationship('Alert', backref='creator', lazy='dynamic')
    assigned_tasks = db.relationship('MaintenanceTask', backref='assigned_to', lazy='dynamic')
    
    def set_password(self, password):
        self.password_hash = generate_password_hash(password).decode('utf-8')
    
    def check_password(self, password):
        return check_password_hash(self.password_hash, password)
    
    @property
    def full_name(self):
        return f"{self.first_name} {self.last_name}"
    
    def __repr__(self):
        return f'<User {self.username}: {self.role.value}>'

class Location(db.Model):
    __tablename__ = 'locations'
    
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text)
    address = db.Column(db.String(255))
    city = db.Column(db.String(100))
    postal_code = db.Column(db.String(20))
    country = db.Column(db.String(50), default='France')
    
    # Coordonnées GPS
    latitude = db.Column(db.Float)
    longitude = db.Column(db.Float)
    
    # Métadonnées
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relations
    equipment = db.relationship('Equipment', backref='location', lazy='dynamic')
    alerts = db.relationship('Alert', backref='location', lazy='dynamic')
    
    def __repr__(self):
        return f'<Location {self.name}>'

class Equipment(db.Model):
    __tablename__ = 'equipment'
    
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False)
    model = db.Column(db.String(100))
    serial_number = db.Column(db.String(100), unique=True)
    manufacturer = db.Column(db.String(100))
    description = db.Column(db.Text)
    
    # Statut et état
    status = db.Column(db.Enum(EquipmentStatus), default=EquipmentStatus.OPERATIONAL)
    
    # Dates importantes
    purchase_date = db.Column(db.Date)
    installation_date = db.Column(db.Date)
    warranty_expiry = db.Column(db.Date)
    last_maintenance = db.Column(db.DateTime)
    next_maintenance = db.Column(db.DateTime)
    
    # Prix et coûts
    purchase_price = db.Column(db.Numeric(10, 2))
    maintenance_cost = db.Column(db.Numeric(10, 2), default=0)
    
    # Relations
    location_id = db.Column(db.Integer, db.ForeignKey('locations.id'), nullable=False)
    category_id = db.Column(db.Integer, db.ForeignKey('equipment_categories.id'))
    
    # Métadonnées
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relations
    alerts = db.relationship('Alert', backref='equipment', lazy='dynamic')
    maintenance_tasks = db.relationship('MaintenanceTask', backref='equipment', lazy='dynamic')
    inventory_items = db.relationship('InventoryItem', backref='equipment', lazy='dynamic')
    
    def __repr__(self):
        return f'<Equipment {self.name} - {self.serial_number}>'

class EquipmentCategory(db.Model):
    __tablename__ = 'equipment_categories'
    
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False, unique=True)
    description = db.Column(db.Text)
    color = db.Column(db.String(7), default='#007bff')  # Couleur hex pour l'interface
    
    # Relations
    equipment = db.relationship('Equipment', backref='category', lazy='dynamic')
    
    def __repr__(self):
        return f'<EquipmentCategory {self.name}>'

class Alert(db.Model):
    __tablename__ = 'alerts'
    
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String(200), nullable=False)
    message = db.Column(db.Text, nullable=False)
    level = db.Column(db.Enum(AlertLevel), nullable=False, default=AlertLevel.INFO)
    status = db.Column(db.Enum(AlertStatus), nullable=False, default=AlertStatus.OPEN)
    
    # Relations
    creator_id = db.Column(db.Integer, db.ForeignKey('users.id'), nullable=False)
    location_id = db.Column(db.Integer, db.ForeignKey('locations.id'))
    equipment_id = db.Column(db.Integer, db.ForeignKey('equipment.id'))
    
    # Dates
    created_at = db.Column(db.DateTime, default=datetime.utcnow, index=True)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    resolved_at = db.Column(db.DateTime)
    
    # Métadonnées
    notes = db.Column(db.Text)
    
    def __repr__(self):
        return f'<Alert {self.title} - {self.level.value}>'

class MaintenanceTask(db.Model):
    __tablename__ = 'maintenance_tasks'
    
    id = db.Column(db.Integer, primary_key=True)
    title = db.Column(db.String(200), nullable=False)
    description = db.Column(db.Text)
    
    # Planification
    scheduled_date = db.Column(db.DateTime, nullable=False)
    estimated_duration = db.Column(db.Integer)  # en minutes
    priority = db.Column(db.Enum(AlertLevel), default=AlertLevel.MEDIUM)
    
    # Statut
    status = db.Column(db.String(20), default='planned')  # planned, in_progress, completed, cancelled
    
    # Relations
    equipment_id = db.Column(db.Integer, db.ForeignKey('equipment.id'), nullable=False)
    assigned_to_id = db.Column(db.Integer, db.ForeignKey('users.id'))
    
    # Dates
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    started_at = db.Column(db.DateTime)
    completed_at = db.Column(db.DateTime)
    
    # Coûts
    estimated_cost = db.Column(db.Numeric(10, 2))
    actual_cost = db.Column(db.Numeric(10, 2))
    
    # Notes et commentaires
    notes = db.Column(db.Text)
    completion_notes = db.Column(db.Text)
    
    def __repr__(self):
        return f'<MaintenanceTask {self.title}>'

class InventoryItem(db.Model):
    __tablename__ = 'inventory_items'
    
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False)
    description = db.Column(db.Text)
    part_number = db.Column(db.String(100), unique=True)
    barcode = db.Column(db.String(100))
    
    # Stock
    quantity_in_stock = db.Column(db.Integer, default=0)
    minimum_stock = db.Column(db.Integer, default=0)
    maximum_stock = db.Column(db.Integer)
    unit = db.Column(db.String(20), default='pcs')  # unité (pcs, kg, m, etc.)
    
    # Prix
    unit_price = db.Column(db.Numeric(10, 2))
    supplier = db.Column(db.String(100))
    
    # Relations
    category_id = db.Column(db.Integer, db.ForeignKey('inventory_categories.id'))
    equipment_id = db.Column(db.Integer, db.ForeignKey('equipment.id'))  # Optionnel: lié à un équipement spécifique
    location_id = db.Column(db.Integer, db.ForeignKey('locations.id'))  # Lieu de stockage
    
    # Métadonnées
    created_at = db.Column(db.DateTime, default=datetime.utcnow)
    updated_at = db.Column(db.DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    @property
    def is_low_stock(self):
        return self.quantity_in_stock <= self.minimum_stock
    
    @property
    def total_value(self):
        if self.unit_price:
            return float(self.quantity_in_stock * self.unit_price)
        return 0
    
    def __repr__(self):
        return f'<InventoryItem {self.name} - Stock: {self.quantity_in_stock}>'

class InventoryCategory(db.Model):
    __tablename__ = 'inventory_categories'
    
    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False, unique=True)
    description = db.Column(db.Text)
    
    # Relations
    items = db.relationship('InventoryItem', backref='category', lazy='dynamic')
    
    def __repr__(self):
        return f'<InventoryCategory {self.name}>'
