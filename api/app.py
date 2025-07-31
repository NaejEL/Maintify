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
from plugins.alerts.routes import alerts_bp
from plugins.dashboard.routes import dashboard_bp
from plugins.reports.routes import reports_bp
from core.auth.routes import auth_bp
from api.plugin_loader import register_plugins

# Add a simple root route
@app.route('/')
def index():
    return {
        "message": "Maintify API is running",
        "version": "1.0.0",
        "database": "PostgreSQL" if "postgresql" in app.config['SQLALCHEMY_DATABASE_URI'] else "SQLite",
        "endpoints": {
            "auth": "/api/auth",
            "alerts": "/api/alerts", 
            "dashboard": "/api/dashboard",
            "reports": "/api/reports"
        }
    }

app.register_blueprint(auth_bp, url_prefix='/api/auth')
register_plugins(app)

if __name__ == '__main__':
    with app.app_context():
        db.create_all()
    app.run(debug=True, host='0.0.0.0')
