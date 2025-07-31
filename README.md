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

### 🔐 Authentification & Autorisation

- **Connexion sécurisée** avec JWT
- **Gestion des rôles** (Admin, Technicien, Utilisateur)
- **Sessions persistantes**
- **Comptes de démonstration** pour les tests

### 📊 Tableau de bord

- **Vue d'ensemble** en temps réel
- **Statistiques** des équipements et alertes
- **Activité récente**
- **Actions rapides** contextuelles

### 👥 Gestion des utilisateurs

- **Administration** complète des comptes
- **Attribution de rôles**
- **Suivi des connexions**
- **Interface de gestion** intuitive

### 🚨 Système d'alertes

- **Création d'alertes** personnalisées
- **Niveaux de gravité** (Élevée, Moyenne, Faible)
- **Suivi du statut** (Active, Résolue, En cours)
- **Interface de gestion** dédiée

### 🌍 Multilingue

- **3 langues supportées** : Français 🇫🇷, Anglais 🇺🇸, Espagnol 🇪🇸
- **Commutation en temps réel**
- **Traductions complètes** de l'interface
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
- **[Vue Router](https://router.vuejs.org/)** - Routage SPA
- **[Vuex](https://vuex.vuejs.org/)** - Gestion d'état centralisée
- **[Vue I18n](https://vue-i18n.intlify.dev/)** - Internationalisation
- **[Axios](https://axios-http.com/)** - Client HTTP

### Infrastructure

- **[Docker](https://docker.com/)** - Conteneurisation
- **[Docker Compose](https://docs.docker.com/compose/)** - Orchestration
- **[Nginx](https://nginx.org/)** (optionnel) - Serveur web/proxy

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

### 3. Changement de langue

- Cliquez sur le sélecteur de langue (🇫🇷 FR) en haut à droite
- Choisissez parmi Français, English ou Español
- L'interface change instantanément

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

### Structure modulaire

```
Maintify/
├── 🐳 docker-compose.yml     # Orchestration des services
├── 📁 frontend/              # Application Vue.js
│   ├── 📁 src/components/    # Composants Vue
│   ├── 📁 src/locales/       # Traductions i18n
│   └── 📁 src/store/         # État Vuex
├── 📁 api/                   # API Flask
├── 📁 core/                  # Logique métier
│   ├── 📁 auth/              # Authentification
│   └── 📁 models/            # Modèles de données
├── 📁 plugins/               # Modules fonctionnels
│   ├── 📁 alerts/            # Gestion des alertes
│   ├── 📁 dashboard/         # Tableau de bord
│   └── 📁 reports/           # Rapports
└── 📁 tests/                 # Tests unitaires
```

### Architecture des plugins

Maintify utilise une architecture modulaire basée sur des plugins :

```python
# Exemple de plugin
plugins/
├── mon_plugin/
│   ├── models.py      # Modèles de données
│   ├── routes.py      # Endpoints API
│   └── services.py    # Logique métier
```

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
- [ ] Module de gestion des équipements
- [ ] Maintenance préventive programmée
- [ ] Rapports avancés avec graphiques
- [ ] API REST complète

### Version 1.2 (Q4 2025)
- [ ] Application mobile (React Native)
- [ ] Notifications en temps réel
- [ ] Intégration IoT
- [ ] Export/Import de données

### Version 2.0 (2026)
- [ ] Microservices architecture
- [ ] Machine Learning pour maintenance prédictive
- [ ] Tableau de bord analytics avancé
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
