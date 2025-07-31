#!/usr/bin/env python3
"""
Script d'initialisation de la base de données Maintify
Crée les tables et insère des données de test
"""

import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from api.app import app
from config.database import db
from core.models import *
from datetime import datetime, date

def init_database():
    """Initialiser la base de données avec des données de test"""
    
    with app.app_context():
        print("🗃️  Création des tables...")
        db.create_all()
        
        print("👥 Création des utilisateurs de test...")
        
        # Créer un administrateur
        admin = User(
            username='admin',
            email='admin@maintify.com',
            first_name='Admin',
            last_name='Maintify',
            role=UserRole.ADMIN
        )
        admin.set_password('admin123')
        db.session.add(admin)
        
        # Créer un technicien
        tech = User(
            username='technicien',
            email='tech@maintify.com',
            first_name='Jean',
            last_name='Dupont',
            role=UserRole.TECHNICIAN,
            phone='0123456789'
        )
        tech.set_password('tech123')
        db.session.add(tech)
        
        # Créer un utilisateur normal
        user = User(
            username='utilisateur',
            email='user@maintify.com',
            first_name='Marie',
            last_name='Martin',
            role=UserRole.USER
        )
        user.set_password('user123')
        db.session.add(user)
        
        print("📍 Création des lieux...")
        
        # Créer des lieux
        location1 = Location(
            name='Bâtiment Principal',
            description='Bâtiment principal de l\'entreprise',
            address='123 Rue de la République',
            city='Paris',
            postal_code='75001',
            latitude=48.8566,
            longitude=2.3522
        )
        db.session.add(location1)
        
        location2 = Location(
            name='Entrepôt Nord',
            description='Entrepôt de stockage principal',
            address='456 Avenue de la Logistique',
            city='Lyon',
            postal_code='69000',
            latitude=45.7640,
            longitude=4.8357
        )
        db.session.add(location2)
        
        print("🏷️  Création des catégories d'équipement...")
        
        # Créer des catégories d'équipement
        cat_hvac = EquipmentCategory(
            name='CVC (Chauffage, Ventilation, Climatisation)',
            description='Systèmes de chauffage, ventilation et climatisation',
            color='#ff6b6b'
        )
        db.session.add(cat_hvac)
        
        cat_elec = EquipmentCategory(
            name='Électrique',
            description='Équipements électriques et électroniques',
            color='#4ecdc4'
        )
        db.session.add(cat_elec)
        
        cat_mech = EquipmentCategory(
            name='Mécanique',
            description='Équipements mécaniques et machines',
            color='#45b7d1'
        )
        db.session.add(cat_mech)
        
        print("🏭 Création des équipements...")
        
        # Commit pour obtenir les IDs
        db.session.commit()
        
        # Créer des équipements
        equipment1 = Equipment(
            name='Climatiseur Central',
            model='HVAC-2000X',
            serial_number='HV123456789',
            manufacturer='AirTech Solutions',
            description='Climatiseur central pour le bâtiment principal',
            status=EquipmentStatus.OPERATIONAL,
            purchase_date=date(2022, 3, 15),
            installation_date=date(2022, 4, 1),
            warranty_expiry=date(2025, 4, 1),
            purchase_price=15000.00,
            location_id=location1.id,
            category_id=cat_hvac.id
        )
        db.session.add(equipment1)
        
        equipment2 = Equipment(
            name='Groupe Électrogène',
            model='GEN-500KW',
            serial_number='GE987654321',
            manufacturer='PowerGen',
            description='Groupe électrogène de secours',
            status=EquipmentStatus.OPERATIONAL,
            purchase_date=date(2021, 8, 10),
            installation_date=date(2021, 9, 1),
            warranty_expiry=date(2024, 9, 1),
            purchase_price=25000.00,
            location_id=location1.id,
            category_id=cat_elec.id
        )
        db.session.add(equipment2)
        
        equipment3 = Equipment(
            name='Convoyeur Automatique',
            model='CONV-AUTO-3000',
            serial_number='CA111222333',
            manufacturer='LogiMech',
            description='Convoyeur automatique pour l\'entrepôt',
            status=EquipmentStatus.MAINTENANCE,
            purchase_date=date(2020, 12, 5),
            installation_date=date(2021, 1, 15),
            warranty_expiry=date(2023, 1, 15),
            purchase_price=35000.00,
            location_id=location2.id,
            category_id=cat_mech.id
        )
        db.session.add(equipment3)
        
        print("🚨 Création des alertes...")
        
        # Commit pour obtenir les IDs des équipements
        db.session.commit()
        
        # Créer des alertes
        alert1 = Alert(
            title='Température élevée détectée',
            message='Le climatiseur central affiche une température de fonctionnement anormalement élevée',
            level=AlertLevel.CRITICAL,
            status=AlertStatus.OPEN,
            creator_id=tech.id,
            location_id=location1.id,
            equipment_id=equipment1.id
        )
        db.session.add(alert1)
        
        alert2 = Alert(
            title='Maintenance préventive programmée',
            message='Maintenance préventive du groupe électrogène prévue pour la semaine prochaine',
            level=AlertLevel.INFO,
            status=AlertStatus.OPEN,
            creator_id=admin.id,
            location_id=location1.id,
            equipment_id=equipment2.id
        )
        db.session.add(alert2)
        
        alert3 = Alert(
            title='Convoyeur en panne',
            message='Le convoyeur automatique s\'est arrêté de fonctionner ce matin',
            level=AlertLevel.HIGH,
            status=AlertStatus.IN_PROGRESS,
            creator_id=user.id,
            location_id=location2.id,
            equipment_id=equipment3.id
        )
        db.session.add(alert3)
        
        print("🔧 Création des tâches de maintenance...")
        
        # Créer des tâches de maintenance
        task1 = MaintenanceTask(
            title='Nettoyage filtres climatiseur',
            description='Nettoyage et remplacement des filtres du climatiseur central',
            scheduled_date=datetime(2025, 8, 15, 9, 0),
            estimated_duration=120,
            priority=AlertLevel.MEDIUM,
            equipment_id=equipment1.id,
            assigned_to_id=tech.id,
            estimated_cost=150.00
        )
        db.session.add(task1)
        
        task2 = MaintenanceTask(
            title='Test groupe électrogène',
            description='Test mensuel du groupe électrogène et vérification du carburant',
            scheduled_date=datetime(2025, 8, 10, 14, 0),
            estimated_duration=60,
            priority=AlertLevel.HIGH,
            equipment_id=equipment2.id,
            assigned_to_id=tech.id,
            estimated_cost=75.00
        )
        db.session.add(task2)
        
        print("📦 Création des catégories d'inventaire...")
        
        # Créer des catégories d'inventaire
        inv_cat1 = InventoryCategory(
            name='Filtres et Consommables',
            description='Filtres, huiles, consommables divers'
        )
        db.session.add(inv_cat1)
        
        inv_cat2 = InventoryCategory(
            name='Pièces Électriques',
            description='Composants électriques et électroniques'
        )
        db.session.add(inv_cat2)
        
        inv_cat3 = InventoryCategory(
            name='Outils et Équipements',
            description='Outils de maintenance et équipements de sécurité'
        )
        db.session.add(inv_cat3)
        
        print("📋 Création des éléments d'inventaire...")
        
        # Commit pour obtenir les IDs des catégories
        db.session.commit()
        
        # Créer des éléments d'inventaire
        item1 = InventoryItem(
            name='Filtre HEPA pour climatiseur',
            description='Filtre haute efficacité pour climatiseur central',
            part_number='HEPA-CLM-001',
            quantity_in_stock=12,
            minimum_stock=5,
            maximum_stock=25,
            unit='pcs',
            unit_price=45.50,
            supplier='FilterTech Pro',
            category_id=inv_cat1.id,
            equipment_id=equipment1.id,
            location_id=location1.id
        )
        db.session.add(item1)
        
        item2 = InventoryItem(
            name='Fusible 32A',
            description='Fusible de protection 32 ampères',
            part_number='FUS-32A-001',
            quantity_in_stock=25,
            minimum_stock=10,
            maximum_stock=50,
            unit='pcs',
            unit_price=3.20,
            supplier='ElectroStock',
            category_id=inv_cat2.id,
            location_id=location1.id
        )
        db.session.add(item2)
        
        item3 = InventoryItem(
            name='Huile moteur SAE 15W-40',
            description='Huile moteur pour groupe électrogène',
            part_number='OIL-15W40-5L',
            quantity_in_stock=8,
            minimum_stock=3,
            maximum_stock=15,
            unit='bidons 5L',
            unit_price=28.90,
            supplier='LubriTech',
            category_id=inv_cat1.id,
            equipment_id=equipment2.id,
            location_id=location1.id
        )
        db.session.add(item3)
        
        item4 = InventoryItem(
            name='Gants de sécurité',
            description='Gants de protection pour maintenance',
            part_number='GLOVE-SAFE-001',
            quantity_in_stock=2,  # Stock bas intentionnel
            minimum_stock=5,
            maximum_stock=20,
            unit='paires',
            unit_price=12.50,
            supplier='SafetyFirst',
            category_id=inv_cat3.id,
            location_id=location1.id
        )
        db.session.add(item4)
        
        # Commit final
        db.session.commit()
        
        print("✅ Base de données initialisée avec succès!")
        print("\n📊 Résumé des données créées:")
        print(f"   - Utilisateurs: {User.query.count()}")
        print(f"   - Lieux: {Location.query.count()}")
        print(f"   - Catégories d'équipement: {EquipmentCategory.query.count()}")
        print(f"   - Équipements: {Equipment.query.count()}")
        print(f"   - Alertes: {Alert.query.count()}")
        print(f"   - Tâches de maintenance: {MaintenanceTask.query.count()}")
        print(f"   - Catégories d'inventaire: {InventoryCategory.query.count()}")
        print(f"   - Éléments d'inventaire: {InventoryItem.query.count()}")
        
        print("\n🔑 Comptes de test créés:")
        print("   Admin: admin / admin123")
        print("   Technicien: technicien / tech123")
        print("   Utilisateur: utilisateur / user123")

if __name__ == '__main__':
    init_database()
