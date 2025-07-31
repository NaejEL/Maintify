import unittest
from plugins.alerts.services import get_all_alerts

class TestAlertsPlugin(unittest.TestCase):
    def test_get_all_alerts(self):
        alerts = get_all_alerts()
        self.assertIsInstance(alerts, list)
        self.assertGreater(len(alerts), 0)
        self.assertIn('message', alerts[0])

if __name__ == '__main__':
    unittest.main()
