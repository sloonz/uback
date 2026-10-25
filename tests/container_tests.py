from .common import *

class ContainerTests(unittest.TestCase):
    def test_container_pq_key(self):
        key = tests_path / "fixtures" / "age-pq.key"
        recipient = tests_path / "fixtures" / "age-pq.pub"
        plaintext = b"hello from a post-quantum encrypted container"
        with tempfile.TemporaryDirectory() as d:
            container = f"{d}/test.ubkp"
            with open(container, "wb") as fd:
                run([uback, "container", "create", "-k", recipient, "test"],
                    stdout=fd, input=plaintext, check=True)
            self.assertEqual(b"test", check_output([uback, "container", "type", container]).strip())
            self.assertEqual(plaintext, check_output([
                uback, "container", "extract", "-k", key, container]))

    def test_container(self):
        with tempfile.TemporaryDirectory() as d:
            check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            with open(f"{d}/test.ubkp", "wb+") as fd:
                run([uback, "container", "create", "-k", f"{d}/backup.pub", "test"], stdout=fd, input=b"hello", check=True)
            self.assertEqual(b"test", check_output([uback, "container", "type", f"{d}/test.ubkp"]).strip())
            with open(f"{d}/test.ubkp", "rb") as fd:
                self.assertEqual(b"hello", check_output([uback, "container", "extract", "-k", f"{d}/backup.key"], stdin=fd))
