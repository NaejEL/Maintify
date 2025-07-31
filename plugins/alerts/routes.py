from flask import Blueprint, jsonify
from plugins.alerts.services import get_all_alerts

alerts_bp = Blueprint('alerts', __name__)

@alerts_bp.route('/alerts', methods=['GET'])
def list_alerts():
    return jsonify(get_all_alerts())
