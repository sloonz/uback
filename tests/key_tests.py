from .common import *

class KeyTests(unittest.TestCase):
    def test_keygen(self):
        with tempfile.TemporaryDirectory() as d:
            check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            self.assertEqual(
                check_output([uback, "key", "pub"], input=read_file(f"{d}/backup.key")).strip(),
                read_file(f"{d}/backup.pub").strip())

    def test_pubkey_from_privkey(self):
        self.assertEqual(
                b"age1fu6nhq9cvjezr6lffnnfj3txqvxdsv0est5vqzamujcfnj80jfpqdcj87k",
                check_output([uback, "key", "pub"], input=b"AGE-SECRET-KEY-1FZM50PS7W57CZV4EZVFVZZHVPK02Q6WNC0FU3DZ9RHLLYQY42PZQNDKJZW").strip())
