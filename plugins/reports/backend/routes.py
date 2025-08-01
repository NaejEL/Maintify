from flask import Blueprint, jsonify

reports_bp = Blueprint('reports', __name__)

@reports_bp.route('/reports', methods=['GET'])
def get_reports():
    return jsonify({"message": "Reports data"})
