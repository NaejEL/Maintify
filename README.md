# 🔧 Maintify

> **Un système de gestion de maintenance (CMMS) moderne et open source**

[![Docker](https://img.shields.io/badge/Docker-Enabled-blue?logo=docker)](https://docker.com)
[![Vue.js](https://img.shields.io/badge/Vue.js-3.0-green?logo=vue.js)](https://vuejs.org)
[![Flask](https://img.shields.io/badge/Flask-Python-red?logo=flask)](https://flask.palletsprojects.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Multilingual](https://img.shields.io/badge/i18n-FR%20%7C%20EN%20%7C%20ES-brightgreen)](docs/i18n.md)

## 📋 Table des matières

- [🎯 À propos](#-à-propos)
- [✨ Fonctionnalités](#-fonctionnalités)
- [🛠️ Technologies](#️-technologies)
- [🚀 Installation](#-installation)
- [📖 Utilisation](#-utilisation)
- [🌍 Internationalisation](#-internationalisation)
- [🏗️ Architecture](#️-architecture)
- [🧪 Tests](#-tests)
- [🤝 Contribution](#-contribution)
- [📄 Licence](#-licence)

## 🎯 À propos

**Maintify** est un système de gestion de maintenance assistée par ordinateur (CMMS) moderne, conçu pour simplifier la gestion des équipements, de la maintenance préventive et des interventions dans votre organisation.

### 🎨 Aperçu

Maintify propose une interface utilisateur intuitive et moderne, avec un système d'authentification robuste et une architecture modulaire basée sur des plugins.

## ✨ Fonctionnalités

### � Architecture modulaire avec plugins dynamiques

- **Système de plugins** chargés dynamiquement
- **Routage dynamique** des composants Vue.js
- **API découverte automatique** des modules
- **Chargement à chaud** des fonctionnalités

### �🔐 Authentification & Autorisation

- **Connexion sécurisée** avec JWT
- **Gestion des rôles** (Admin, Technicien, Utilisateur)
- **Sessions persistantes**
- **Guards d'authentification** par route

### 📊 Modules disponibles

- **🏠 Dashboard** : Vue d'ensemble et statistiques
- **🚨 Alertes** : Gestion des alertes système
- **📋 Reports** : Génération de rapports
- **👥 Gestion utilisateurs** : Administration des comptes

### 🌍 Multilingue

- **3 langues supportées** : Français 🇫🇷, Anglais 🇺🇸, Espagnol 🇪🇸
- **Commutation en temps réel**
- **Support i18n des plugins**
- **Persistance des préférences**

## 🛠️ Technologies

### Backend

- **[Flask](https://flask.palletsprojects.com/)** - Framework web Python
- **[SQLAlchemy](https://sqlalchemy.org/)** - ORM et gestion de base de données
- **[PostgreSQL](https://postgresql.org/)** - Base de données relationnelle
- **[Marshmallow](https://marshmallow.readthedocs.io/)** - Sérialisation/validation
- **[Flask-JWT-Extended](https://flask-jwt-extended.readthedocs.io/)** - Authentification JWT

### Frontend

- **[Vue.js 3](https://vuejs.org/)** - Framework JavaScript moderne
- **[Vue Router](https://router.vuejs.org/)** - Routage dynamique et SPA
- **[Vuex](https://vuex.vuejs.org/)** - Gestion d'état centralisée
- **[Vue I18n](https://vue-i18n.intlify.dev/)** - Internationalisation
- **[Axios](https://axios-http.com/)** - Client HTTP
- **[SCSS](https://sass-lang.com/)** - Styles modulaires

### Infrastructure

- **[Docker](https://docker.com/)** - Conteneurisation complète
- **[Docker Compose](https://docs.docker.com/compose/)** - Orchestration des services
- **[Webpack](https://webpack.js.org/)** - Build et modules dynamiques

## 🚀 Installation

### Prérequis

- **Docker** & **Docker Compose**
- **Git**

### Installation rapide

```bash
# Cloner le repository
git clone https://github.com/NaejEL/Maintify.git
cd Maintify

# Lancer l'application
docker-compose up -d

# Initialiser la base de données
docker-compose exec backend python init_db.py
```

### Accès à l'application

- **Frontend** : <http://localhost:8080>
- **Backend API** : <http://localhost:5000>
- **Base de données** : localhost:5432

### Comptes de démonstration

| Utilisateur | Mot de passe | Rôle |
|-------------|--------------|------|
| `admin` | `admin123` | Administrateur |
| `tech` | `tech123` | Technicien |
| `user` | `user123` | Utilisateur |

## 📖 Utilisation

### 1. Connexion

- Accédez à <http://localhost:8080>
- Utilisez un compte de démonstration ou créez un nouveau compte
- Naviguez dans l'interface multilingue

### 2. Navigation

- **🏠 Accueil** : Tableau de bord principal
- **👤 Profil** : Gestion du profil utilisateur
- **👥 Gestion** : Administration des utilisateurs (admin uniquement)
- **🚨 Alertes** : Gestion des alertes système
- **📋 Reports** : Génération de rapports
- **🔌 Plugins** : Navigation dynamique basée sur les plugins chargés

### 3. Système de plugins

- Les **plugins sont chargés automatiquement** au démarrage
- **Navigation dynamique** : les routes apparaissent selon les plugins disponibles
- **Chargement à chaud** : ajout/suppression de plugins sans redémarrage
- **Isolation** : chaque plugin est indépendant

## 🌍 Internationalisation

Maintify supporte nativement trois langues avec une architecture extensible :

```javascript
// Ajouter une nouvelle langue
// 1. Créer le fichier de traduction
frontend/src/locales/de.json

// 2. Mettre à jour la configuration
frontend/src/i18n.js

// 3. Ajouter au sélecteur de langue
{ code: 'de', name: 'Deutsch', flag: '🇩🇪' }
```

## 🏗️ Architecture

### Structure modulaire avec plugins dynamiques

```
Maintify/
├── 🐳 docker-compose.yml     # Orchestration des services
├── 📁 frontend/              # Application Vue.js
│   ├── 📁 src/
│   │   ├── 📁 components/    # Composants Vue
│   │   ├── 📁 router/        # Router dynamique
│   │   │   └── dynamic.js    # Chargement dynamique des routes
│   │   ├── 📁 plugins/       # Gestionnaires de plugins
│   │   │   ├── plugin-manager.js
│   │   │   └── i18n-manager.js
│   │   ├── 📁 locales/       # Traductions i18n
│   │   └── 📁 store/         # État Vuex
├── 📁 api/                   # API Flask
├── 📁 core/                  # Logique métier
│   ├── 📁 auth/              # Authentification
│   └── 📁 models/            # Modèles de données
├── 📁 plugins/               # 🔌 Modules plugins
│   ├── 📁 alerts/            # Plugin alertes
│   │   ├── 📁 backend/       # API et logique
│   │   ├── 📁 frontend/      # Composants Vue
│   │   │   └── 📁 components/
│   │   └── plugin.json       # Configuration
│   ├── 📁 dashboard/         # Plugin tableau de bord
│   └── 📁 reports/           # Plugin rapports
└── 📁 tests/                 # Tests unitaires
```

### Système de plugins dynamiques

**Maintify** utilise une architecture révolutionnaire basée sur des plugins chargés dynamiquement :

#### 🔄 Chargement automatique
- **Découverte automatique** des plugins via l'API backend
- **Routage dynamique** des composants Vue.js
- **Chargement à chaud** sans redémarrage
- **Isolation complète** entre plugins

#### 📁 Structure d'un plugin

```yaml
plugins/mon_plugin/
├── plugin.json              # Configuration du plugin
├── backend/                 # Logique serveur
│   ├── models.py           # Modèles de données
│   ├── routes.py           # Endpoints API
│   └── services.py         # Logique métier
├── frontend/               # Interface utilisateur
│   ├── components/         # Composants Vue
│   └── styles/            # Styles SCSS
└── locales/               # Traductions i18n
    └── messages.json
```

#### ⚙️ Configuration plugin.json

```json
{
  "name": "mon_plugin",
  "version": "1.0.0",
  "description": "Description du plugin",
  "author": "Développeur",
  "backend": {
    "module": "backend.routes",
    "prefix": "/api/mon_plugin"
  },
  "frontend": {
    "routes": [
      {
        "path": "/mon_plugin",
        "component": "MonPluginView",
        "name": "MonPlugin"
      }
    ]
  }
}
```

## 🔌 Développement de plugins

### Créer un nouveau plugin

```bash
# 1. Créer la structure du plugin
mkdir -p plugins/mon_plugin/{backend,frontend/components,locales}

# 2. Créer la configuration
cat > plugins/mon_plugin/plugin.json << EOF
{
  "name": "mon_plugin",
  "version": "1.0.0",
  "description": "Mon nouveau plugin",
  "author": "Développeur",
  "backend": {
    "module": "backend.routes",
    "prefix": "/api/mon_plugin"
  },
  "frontend": {
    "routes": [
      {
        "path": "/mon_plugin",
        "component": "MonPluginView",
        "name": "MonPlugin",
        "meta": {"requiresAuth": true}
      }
    ]
  }
}
EOF

# 3. Créer le composant Vue
cat > plugins/mon_plugin/frontend/components/MonPluginView.vue << EOF
<template>
  <div class="mon-plugin">
    <h1>Mon Plugin</h1>
    <p>Contenu de mon plugin personnalisé</p>
  </div>
</template>

<script>
export default {
  name: 'MonPluginView'
}
</script>
EOF

# 4. Créer les routes backend
cat > plugins/mon_plugin/backend/routes.py << EOF
from flask import Blueprint

bp = Blueprint('mon_plugin', __name__, url_prefix='/api/mon_plugin')

@bp.route('/')
def index():
    return {'message': 'Mon plugin fonctionne !'}
EOF
```

### Chargement automatique

1. **Redémarrer le backend** : `docker-compose restart backend`
2. **Accéder à** `http://localhost:8080/mon_plugin`
3. **Le plugin apparaît automatiquement** dans la navigation !

## 🧪 Tests

```bash
# Tests backend
docker-compose exec backend python -m pytest tests/

# Tests frontend (à venir)
docker-compose exec frontend npm run test
```

### Structure des tests

```
tests/
├── auth/                # Tests d'authentification
├── plugins/            # Tests des plugins
│   └── alerts/         # Tests du module alertes
└── conftest.py         # Configuration pytest
```

## 🤝 Contribution

Les contributions sont les bienvenues ! Voici comment participer :

### 1. Fork & Clone
```bash
git clone https://github.com/VOTRE_USERNAME/Maintify.git
cd Maintify
```

### 2. Développement
```bash
# Créer une branche feature
git checkout -b feature/ma-nouvelle-fonctionnalite

# Lancer l'environnement de développement
docker-compose up -d

# Faire vos modifications...
```

### 3. Tests & Pull Request
```bash
# Tester vos modifications
docker-compose exec backend python -m pytest

# Committer et push
git add .
git commit -m "feat: ajouter nouvelle fonctionnalité"
git push origin feature/ma-nouvelle-fonctionnalite

# Créer une Pull Request sur GitHub
```

### Standards de code

- **Python** : PEP 8, type hints
- **JavaScript** : ESLint, Vue.js style guide
- **Git** : Conventional Commits
- **Documentation** : README à jour, commentaires explicites

## 📈 Roadmap

### Version 1.1 (Q3 2025)

- [ ] API de gestion des plugins à chaud
- [ ] Interface d'administration des plugins
- [ ] Module de gestion des équipements
- [ ] Maintenance préventive programmée
- [ ] Rapports avancés avec graphiques

### Version 1.2 (Q4 2025)

- [ ] Application mobile (React Native)
- [ ] Notifications en temps réel
- [ ] Marketplace de plugins
- [ ] Intégration IoT
- [ ] Export/Import de données

### Version 2.0 (2026)

- [ ] Microservices architecture
- [ ] Machine Learning pour maintenance prédictive
- [ ] Tableau de bord analytics avancé
- [ ] SDK de développement de plugins
- [ ] Intégrations tierces (SAP, etc.)

## 📄 Licence

Ce projet est sous licence **MIT**. Voir le fichier [LICENSE](LICENSE) pour plus de détails.

## 👨‍💻 Auteur

**NaejEL** - [GitHub](https://github.com/NaejEL)

## 🙏 Remerciements

- [Vue.js](https://vuejs.org/) pour le framework frontend
- [Flask](https://flask.palletsprojects.com/) pour le framework backend
- [Docker](https://docker.com/) pour la conteneurisation
- Communauté open source pour l'inspiration

---

<div align="center">

**⭐ N'hésitez pas à donner une étoile si ce projet vous plaît ! ⭐**

[🐛 Signaler un bug](https://github.com/NaejEL/Maintify/issues) | [💡 Proposer une fonctionnalité](https://github.com/NaejEL/Maintify/issues) | [📖 Documentation](https://github.com/NaejEL/Maintify/wiki)

</div>
