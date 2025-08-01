import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from flask import Flask
from flask_cors import CORS
from config.database import db, jwt, migrate, ma, init_db

# Initialisation de l'app
app = Flask(__name__)

# Initialize database and JWT
init_db(app)
CORS(app)

# Import des modèles pour les migrations
from core.models import *

# Enregistrement des Blueprints
from core.auth.routes import auth_bp
from api.plugin_loader import register_plugins, get_plugin_manager

# Add a simple root route
@app.route('/')
def index():
    plugin_mgr = get_plugin_manager()
    loaded_plugins = plugin_mgr.get_loaded_plugins()
    
    return {
        "message": "Maintify API is running",
        "version": "1.0.0",
        "database": "PostgreSQL" if "postgresql" in app.config['SQLALCHEMY_DATABASE_URI'] else "SQLite",
        "plugins": {
            "count": len(loaded_plugins),
            "loaded": list(loaded_plugins.keys())
        },
        "endpoints": {
            "auth": "/api/auth",
            "plugins": "/api/plugins"
        }
    }

@app.route('/api/plugins', methods=['GET'])
def get_plugins_info():
    """Retourne les informations sur tous les plugins chargés"""
    plugin_mgr = get_plugin_manager()
    return {
        "plugins": plugin_mgr.get_loaded_plugins(),
        "frontend_routes": plugin_mgr.get_frontend_routes(),
        "menu_items": plugin_mgr.get_frontend_menu_items(),
        "locales": plugin_mgr.get_plugins_locales()
    }

@app.route('/api/plugins/locales', methods=['GET'])
def get_plugins_locales():
    """Retourne uniquement les locales des plugins"""
    plugin_mgr = get_plugin_manager()
    return plugin_mgr.get_plugins_locales()

app.register_blueprint(auth_bp, url_prefix='/api/auth')
register_plugins(app)

if __name__ == '__main__':
    with app.app_context():
        db.create_all()
    app.run(debug=True, host='0.0.0.0')
