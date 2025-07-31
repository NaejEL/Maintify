import os
from flask_sqlalchemy import SQLAlchemy
from flask_jwt_extended import JWTManager
from flask_migrate import Migrate
from flask_marshmallow import Marshmallow

# Database instance
db = SQLAlchemy()
jwt = JWTManager()
migrate = Migrate()
ma = Marshmallow()

def init_db(app):
    """Initialize database with Flask app"""
    
    # Configuration de la base de données
    database_url = os.getenv('DATABASE_URL', 'sqlite:///data.db')
    app.config['SQLALCHEMY_DATABASE_URI'] = database_url
    app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
    
    # Configuration JWT
    app.config['JWT_SECRET_KEY'] = os.getenv('JWT_SECRET_KEY', 'change_me_in_production')
    app.config['JWT_ACCESS_TOKEN_EXPIRES'] = False  # Pour le développement
    
    # Initialisation des extensions
    db.init_app(app)
    jwt.init_app(app)
    migrate.init_app(app, db)
    ma.init_app(app)
