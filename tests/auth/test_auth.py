import unittest
from core.auth.services import create_user, authenticate_user, ROLES

class TestAuthSystem(unittest.TestCase):
    def setUp(self):
        # on suppose la BD en mémoire pour test
        pass

    def test_user_creation_and_auth(self):
        u = create_user('toto', 'pass123', role='technician')
        self.assertEqual(u.username, 'toto')
        self.assertIn(u.role, ROLES)
        auth = authenticate_user('toto', 'pass123')
        self.assertIsNotNone(auth)
        self.assertEqual(auth.username, 'toto')

if __name__ == '__main__':
    unittest.main()
