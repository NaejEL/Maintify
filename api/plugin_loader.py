from api.dynamic_plugin_loader import plugin_manager

def register_plugins(app):
    """Register all plugin blueprints with the Flask app using dynamic discovery"""
    # Utilise le nouveau système de chargement dynamique
    plugin_manager.register_plugins(app)

def get_plugin_manager():
    """Retourne l'instance du gestionnaire de plugins"""
    return plugin_manager
