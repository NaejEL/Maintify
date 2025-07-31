from flask import Blueprint, request, jsonify
from flask_jwt_extended import (
    create_access_token, jwt_required, get_jwt_identity, get_jwt
)
from core.auth.services import (
    create_user, authenticate_user, login_user,
    get_all_users, get_user_by_id, update_user_role, 
    deactivate_user, change_password, reset_password, ROLES
)
from core.schemas import user_schema, users_schema, login_schema, token_schema
from marshmallow import ValidationError

auth_bp = Blueprint('auth', __name__)

@auth_bp.route('/login', methods=['POST'])
def login():
    """Connexion utilisateur"""
    try:
        data = login_schema.load(request.get_json())
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400
    
    result = login_user(data['username'], data['password'])
    if result:
        return jsonify(result), 200
    else:
        return jsonify({"message": "Identifiants invalides"}), 401

@auth_bp.route('/register', methods=['POST'])
@jwt_required()
def register():
    """Créer un nouvel utilisateur (admin uniquement)"""
    current_user = get_jwt()
    if current_user['role'] != 'admin':
        return jsonify({"message": "Permission refusée"}), 403
    
    try:
        data = request.get_json()
        user = create_user(
            username=data['username'],
            email=data['email'],
            password=data['password'],
            first_name=data['first_name'],
            last_name=data['last_name'],
            role=data.get('role', 'user'),
            phone=data.get('phone')
        )
        return jsonify({
            "message": "Utilisateur créé avec succès",
            "user": user_schema.dump(user)
        }), 201
    except ValueError as e:
        return jsonify({"message": str(e)}), 400
    except Exception as e:
        return jsonify({"message": "Erreur lors de la création"}), 500

@auth_bp.route('/profile', methods=['GET'])
@jwt_required()
def get_profile():
    """Récupérer le profil de l'utilisateur connecté"""
    user_id = int(get_jwt_identity())
    user = get_user_by_id(user_id)
    if user:
        return jsonify(user_schema.dump(user)), 200
    return jsonify({"message": "Utilisateur non trouvé"}), 404

@auth_bp.route('/profile', methods=['PUT'])
@jwt_required()
def update_profile():
    """Mettre à jour le profil de l'utilisateur connecté"""
    user_id = int(get_jwt_identity())
    user = get_user_by_id(user_id)
    
    if not user:
        return jsonify({"message": "Utilisateur non trouvé"}), 404
    
    data = request.get_json()
    
    # Champs modifiables par l'utilisateur
    if 'first_name' in data:
        user.first_name = data['first_name']
    if 'last_name' in data:
        user.last_name = data['last_name']
    if 'email' in data:
        user.email = data['email']
    if 'phone' in data:
        user.phone = data['phone']
    
    from config.database import db
    db.session.commit()
    
    return jsonify({
        "message": "Profil mis à jour",
        "user": user_schema.dump(user)
    }), 200

@auth_bp.route('/change-password', methods=['POST'])
@jwt_required()
def change_user_password():
    """Changer le mot de passe de l'utilisateur connecté"""
    user_id = int(get_jwt_identity())
    data = request.get_json()
    
    if not all(k in data for k in ['old_password', 'new_password']):
        return jsonify({"message": "Ancien et nouveau mot de passe requis"}), 400
    
    if change_password(user_id, data['old_password'], data['new_password']):
        return jsonify({"message": "Mot de passe changé avec succès"}), 200
    else:
        return jsonify({"message": "Ancien mot de passe incorrect"}), 400

@auth_bp.route('/users', methods=['GET'])
@jwt_required()
def list_users():
    """Lister tous les utilisateurs (admin/technician)"""
    current_user = get_jwt()
    if current_user['role'] not in ['admin', 'technician']:
        return jsonify({"message": "Permission refusée"}), 403
    
    users = get_all_users()
    return jsonify(users_schema.dump(users)), 200

@auth_bp.route('/users/<int:user_id>/role', methods=['PUT'])
@jwt_required()
def update_role(user_id):
    """Mettre à jour le rôle d'un utilisateur (admin uniquement)"""
    current_user = get_jwt()
    if current_user['role'] != 'admin':
        return jsonify({"message": "Permission refusée"}), 403
    
    data = request.get_json()
    new_role = data.get('role')
    
    if new_role not in ROLES:
        return jsonify({"message": "Rôle invalide"}), 400
    
    user = update_user_role(user_id, new_role)
    if user:
        return jsonify({
            "message": "Rôle mis à jour",
            "user": user_schema.dump(user)
        }), 200
    else:
        return jsonify({"message": "Utilisateur non trouvé"}), 404

@auth_bp.route('/users/<int:user_id>/deactivate', methods=['POST'])
@jwt_required()
def deactivate_user_route(user_id):
    """Désactiver un utilisateur (admin uniquement)"""
    current_user = get_jwt()
    if current_user['role'] != 'admin':
        return jsonify({"message": "Permission refusée"}), 403
    
    user = deactivate_user(user_id)
    if user:
        return jsonify({
            "message": "Utilisateur désactivé",
            "user": user_schema.dump(user)
        }), 200
    else:
        return jsonify({"message": "Utilisateur non trouvé"}), 404

@auth_bp.route('/users/<int:user_id>/reset-password', methods=['POST'])
@jwt_required()
def reset_user_password(user_id):
    """Réinitialiser le mot de passe d'un utilisateur (admin uniquement)"""
    current_user = get_jwt()
    if current_user['role'] != 'admin':
        return jsonify({"message": "Permission refusée"}), 403
    
    data = request.get_json()
    new_password = data.get('new_password')
    
    if not new_password:
        return jsonify({"message": "Nouveau mot de passe requis"}), 400
    
    if reset_password(user_id, new_password):
        return jsonify({"message": "Mot de passe réinitialisé"}), 200
    else:
        return jsonify({"message": "Utilisateur non trouvé"}), 404

@auth_bp.route('/roles', methods=['GET'])
def get_roles():
    """Récupérer la liste des rôles disponibles"""
    return jsonify({"roles": ROLES}), 200
