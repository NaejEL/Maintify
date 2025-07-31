from marshmallow import fields, validate, post_load
from config.database import ma
from core.models import User, Location, Equipment, Alert, MaintenanceTask, InventoryItem

class UserSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = User
        load_instance = True
        exclude = ('password_hash',)
    
    password = fields.Str(load_only=True, required=True, validate=validate.Length(min=6))
    full_name = fields.Str(dump_only=True)
    role = fields.Str()
    
    @post_load
    def make_user(self, data, **kwargs):
        if 'password' in data:
            password = data.pop('password')
            user = User(**data)
            user.set_password(password)
            return user
        return User(**data)

class LocationSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = Location
        load_instance = True

class EquipmentSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = Equipment
        load_instance = True
    
    location = fields.Nested(LocationSchema, dump_only=True)
    status = fields.Str()

class AlertSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = Alert
        load_instance = True
    
    creator = fields.Nested(UserSchema, dump_only=True, exclude=('created_alerts', 'assigned_tasks'))
    location = fields.Nested(LocationSchema, dump_only=True)
    equipment = fields.Nested(EquipmentSchema, dump_only=True)
    level = fields.Str()
    status = fields.Str()

class MaintenanceTaskSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = MaintenanceTask
        load_instance = True
    
    equipment = fields.Nested(EquipmentSchema, dump_only=True)
    assigned_to = fields.Nested(UserSchema, dump_only=True, exclude=('created_alerts', 'assigned_tasks'))

class InventoryItemSchema(ma.SQLAlchemyAutoSchema):
    class Meta:
        model = InventoryItem
        load_instance = True
    
    location = fields.Nested(LocationSchema, dump_only=True)
    equipment = fields.Nested(EquipmentSchema, dump_only=True)
    is_low_stock = fields.Bool(dump_only=True)
    total_value = fields.Float(dump_only=True)

# Schémas pour les réponses API
class LoginSchema(ma.Schema):
    username = fields.Str(required=True)
    password = fields.Str(required=True)

class TokenSchema(ma.Schema):
    access_token = fields.Str()
    user = fields.Nested(UserSchema, exclude=('created_alerts', 'assigned_tasks'))

# Instances des schémas
user_schema = UserSchema()
users_schema = UserSchema(many=True)
location_schema = LocationSchema()
locations_schema = LocationSchema(many=True)
equipment_schema = EquipmentSchema()
equipments_schema = EquipmentSchema(many=True)
alert_schema = AlertSchema()
alerts_schema = AlertSchema(many=True)
maintenance_task_schema = MaintenanceTaskSchema()
maintenance_tasks_schema = MaintenanceTaskSchema(many=True)
inventory_item_schema = InventoryItemSchema()
inventory_items_schema = InventoryItemSchema(many=True)
login_schema = LoginSchema()
token_schema = TokenSchema()
