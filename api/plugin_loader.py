def register_plugins(app):
    """Register all plugin blueprints with the Flask app"""
    from plugins.alerts.routes import alerts_bp
    from plugins.dashboard.routes import dashboard_bp
    from plugins.reports.routes import reports_bp
    
    app.register_blueprint(alerts_bp, url_prefix='/api')
    app.register_blueprint(dashboard_bp, url_prefix='/api')
    app.register_blueprint(reports_bp, url_prefix='/api')
