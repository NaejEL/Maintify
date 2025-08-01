import os
import json
import importlib
import importlib.util
from typing import Dict, List, Optional, Any
from pathlib import Path
import logging

# Configuration du logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class PluginManager:
    """Gestionnaire de plugins dynamique pour Maintify"""
    
    def __init__(self, plugins_dir: str = "plugins"):
        self.plugins_dir = Path(plugins_dir)
        self.loaded_plugins: Dict[str, Dict] = {}
        self.plugin_blueprints: Dict[str, Any] = {}
        
    def discover_plugins(self) -> List[Dict]:
        """Découvre tous les plugins disponibles en scannant le dossier plugins/"""
        discovered_plugins = []
        
        if not self.plugins_dir.exists():
            logger.warning(f"Le dossier plugins '{self.plugins_dir}' n'existe pas")
            return discovered_plugins
            
        for plugin_dir in self.plugins_dir.iterdir():
            if plugin_dir.is_dir() and not plugin_dir.name.startswith('_'):
                plugin_manifest_path = plugin_dir / "plugin.json"
                
                if plugin_manifest_path.exists():
                    try:
                        plugin_config = self._load_plugin_manifest(plugin_manifest_path)
                        plugin_config['path'] = str(plugin_dir)
                        discovered_plugins.append(plugin_config)
                        logger.info(f"Plugin découvert: {plugin_config['name']} v{plugin_config['version']}")
                    except Exception as e:
                        logger.error(f"Erreur lors du chargement du plugin {plugin_dir.name}: {e}")
                else:
                    logger.warning(f"Plugin {plugin_dir.name} ignoré: pas de plugin.json")
                    
        return discovered_plugins
    
    def _load_plugin_manifest(self, manifest_path: Path) -> Dict:
        """Charge et valide le fichier plugin.json"""
        with open(manifest_path, 'r', encoding='utf-8') as f:
            config = json.load(f)
            
        # Validation basique
        required_fields = ['name', 'version', 'backend']
        for field in required_fields:
            if field not in config:
                raise ValueError(f"Champ requis manquant: {field}")
        
        # Chargement des locales si disponibles
        plugin_dir = manifest_path.parent
        locales_dir = plugin_dir / "locales"
        if locales_dir.exists():
            config['locales'] = self._load_plugin_locales(locales_dir)
                
        return config
    
    def _load_plugin_locales(self, locales_dir: Path) -> Optional[Dict]:
        """Charge les fichiers de traduction d'un plugin"""
        try:
            locales = {}
            
            # Chercher le fichier messages.json
            messages_file = locales_dir / "messages.json"
            if messages_file.exists():
                with open(messages_file, 'r', encoding='utf-8') as f:
                    locales = json.load(f)
                    logger.info(f"Locales chargées depuis: {messages_file}")
            
            # Chercher des fichiers individuels par langue (ex: fr.json, en.json)
            for locale_file in locales_dir.glob("*.json"):
                if locale_file.name != "messages.json":
                    locale_code = locale_file.stem
                    with open(locale_file, 'r', encoding='utf-8') as f:
                        locale_data = json.load(f)
                        locales[locale_code] = locale_data
                        logger.info(f"Locale '{locale_code}' chargée depuis: {locale_file}")
            
            return locales if locales else None
            
        except Exception as e:
            logger.error(f"Erreur lors du chargement des locales depuis {locales_dir}: {e}")
            return None
    
    def load_plugin_backend(self, plugin_config: Dict) -> Optional[Any]:
        """Charge le backend d'un plugin (blueprint Flask)"""
        try:
            plugin_name = plugin_config['name']
            plugin_path = plugin_config['path']
            backend_config = plugin_config['backend']
            
            # Construction du chemin du module
            module_name = f"plugins.{plugin_name}.{backend_config['module']}"
            
            # Import dynamique du module
            module = importlib.import_module(module_name)
            
            # Récupération du blueprint
            blueprint_name = backend_config['blueprint']
            if hasattr(module, blueprint_name):
                blueprint = getattr(module, blueprint_name)
                self.plugin_blueprints[plugin_name] = blueprint
                logger.info(f"Blueprint chargé pour le plugin: {plugin_name}")
                return blueprint
            else:
                logger.error(f"Blueprint '{blueprint_name}' non trouvé dans {module_name}")
                return None
                
        except Exception as e:
            logger.error(f"Erreur lors du chargement du backend pour {plugin_name}: {e}")
            return None
    
    def register_plugins(self, flask_app) -> None:
        """Enregistre tous les plugins découverts avec l'application Flask"""
        logger.info("Démarrage de l'enregistrement automatique des plugins...")
        
        # Découverte des plugins
        plugins = self.discover_plugins()
        
        for plugin_config in plugins:
            if plugin_config.get('enabled', True):
                # Chargement du backend
                blueprint = self.load_plugin_backend(plugin_config)
                
                if blueprint:
                    # Enregistrement du blueprint
                    url_prefix = plugin_config['backend'].get('url_prefix', '/api')
                    flask_app.register_blueprint(blueprint, url_prefix=url_prefix)
                    
                    # Stockage de la configuration
                    self.loaded_plugins[plugin_config['name']] = plugin_config
                    
                    logger.info(f"Plugin enregistré: {plugin_config['name']} -> {url_prefix}")
                else:
                    logger.error(f"Échec de l'enregistrement du plugin: {plugin_config['name']}")
            else:
                logger.info(f"Plugin désactivé: {plugin_config['name']}")
                
        logger.info(f"Enregistrement terminé. {len(self.loaded_plugins)} plugins chargés.")
    
    def get_loaded_plugins(self) -> Dict[str, Dict]:
        """Retourne la liste des plugins chargés"""
        return self.loaded_plugins
    
    def get_plugin_info(self, plugin_name: str) -> Optional[Dict]:
        """Retourne les informations d'un plugin spécifique"""
        return self.loaded_plugins.get(plugin_name)
    
    def get_frontend_routes(self) -> List[Dict]:
        """Extrait toutes les routes frontend des plugins chargés"""
        routes = []
        for plugin_name, plugin_config in self.loaded_plugins.items():
            frontend_config = plugin_config.get('frontend', {})
            plugin_routes = frontend_config.get('routes', [])
            for route in plugin_routes:
                route['plugin'] = plugin_name
                routes.append(route)
        return routes
    
    def get_frontend_menu_items(self) -> List[Dict]:
        """Extrait tous les éléments de menu des plugins chargés"""
        menu_items = []
        for plugin_name, plugin_config in self.loaded_plugins.items():
            frontend_config = plugin_config.get('frontend', {})
            menu_config = frontend_config.get('menu')
            if menu_config:
                menu_item = menu_config.copy()
                menu_item['plugin'] = plugin_name
                menu_items.append(menu_item)
        
        # Tri par ordre
        menu_items.sort(key=lambda x: x.get('order', 999))
        return menu_items
    
    def get_plugins_locales(self) -> Dict[str, Dict]:
        """Retourne toutes les locales des plugins chargés"""
        all_locales = {}
        for plugin_name, plugin_config in self.loaded_plugins.items():
            locales = plugin_config.get('locales')
            if locales:
                all_locales[plugin_name] = locales
        return all_locales

# Instance globale du gestionnaire de plugins
plugin_manager = PluginManager()
