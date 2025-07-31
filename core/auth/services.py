from config.database import db
from core.models import User, UserRole
from core.schemas import user_schema, users_schema
from flask_jwt_extended import create_access_token
from datetime import datetime, timedelta

def create_user(username, email, password, first_name, last_name, role='user', phone=None):
    """Créer un nouvel utilisateur"""
    # Vérifier si l'utilisateur existe déjà
    if User.query.filter_by(username=username).first():
        raise ValueError("Nom d'utilisateur déjà existant")
    
    if User.query.filter_by(email=email).first():
        raise ValueError("Email déjà existant")
    
    # Créer l'utilisateur
    user = User(
        username=username,
        email=email,
        first_name=first_name,
        last_name=last_name,
        role=UserRole(role),
        phone=phone
    )
    user.set_password(password)
    
    db.session.add(user)
    db.session.commit()
    return user

def authenticate_user(username, password):
    """Authentifier un utilisateur"""
    user = User.query.filter(
        (User.username == username) | (User.email == username)
    ).first()
    
    if user and user.is_active and user.check_password(password):
        # Mettre à jour la dernière connexion
        user.last_login = datetime.utcnow()
        db.session.commit()
        return user
    return None

def login_user(username, password):
    """Connecter un utilisateur et retourner le token"""
    user = authenticate_user(username, password)
    if not user:
        return None
    
    # Créer le token JWT
    additional_claims = {
        "user_id": user.id,
        "role": user.role.value,
        "full_name": user.full_name
    }
    
    access_token = create_access_token(
        identity=str(user.id),
        additional_claims=additional_claims,
        expires_delta=timedelta(days=30)  # Token valide 30 jours
    )
    
    return {
        "access_token": access_token,
        "user": user_schema.dump(user)
    }

def get_user_by_id(user_id):
    """Récupérer un utilisateur par son ID"""
    return User.query.get(user_id)

def get_all_users():
    """Récupérer tous les utilisateurs"""
    return User.query.all()

def update_user_role(user_id, new_role):
    """Mettre à jour le rôle d'un utilisateur"""
    user = User.query.get(user_id)
    if user:
        user.role = UserRole(new_role)
        db.session.commit()
        return user
    return None

def deactivate_user(user_id):
    """Désactiver un utilisateur"""
    user = User.query.get(user_id)
    if user:
        user.is_active = False
        db.session.commit()
        return user
    return None

def change_password(user_id, old_password, new_password):
    """Changer le mot de passe d'un utilisateur"""
    user = User.query.get(user_id)
    if user and user.check_password(old_password):
        user.set_password(new_password)
        db.session.commit()
        return True
    return False

def reset_password(user_id, new_password):
    """Réinitialiser le mot de passe (admin uniquement)"""
    user = User.query.get(user_id)
    if user:
        user.set_password(new_password)
        db.session.commit()
        return True
    return False

# Constantes pour les rôles
ROLES = [role.value for role in UserRole]
