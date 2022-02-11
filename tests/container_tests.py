from .common import *

class ContainerTests(unittest.TestCase):
    def test_container(self):
        with tempfile.TemporaryDirectory() as d:
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            with open(f"{d}/test.ubkp", "wb+") as fd:
                subprocess.run([uback, "container", "create", "-k", f"{d}/backup.pub", "test"], stdout=fd, input=b"hello", check=True)
            self.assertEqual(b"test", subprocess.check_output([uback, "container", "type", f"{d}/test.ubkp"]).strip())
            with open(f"{d}/test.ubkp", "rb") as fd:
                self.assertEqual(b"hello", subprocess.check_output([uback, "container", "extract", "-k", f"{d}/backup.key"], stdin=fd))
