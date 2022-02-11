from .common import *

EXPECTED_CONFIG="""
"""

class PresetTests(unittest.TestCase):
    def test_presets(self):
        with tempfile.TemporaryDirectory() as d:
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "tar-src", "@Command=sudo,@Command=tar"])
            self.assertEqual(b"tar-src [[@Command sudo] [@Command tar]]", subprocess.check_output([uback, "-p", f"{d}/presets", "preset", "list", "-v"]).strip())
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "remove", "tar-src"])

            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "tar-src", "@Command=sudo"])
            self.assertEqual(b"tar-src [[@Command sudo]]", subprocess.check_output([uback, "-p", f"{d}/presets", "preset", "list", "-v"]).strip())
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "tar-src", "@Command=tar"])
            self.assertEqual(b"tar-src [[@Command sudo] [@Command tar]]", subprocess.check_output([uback, "-p", f"{d}/presets", "preset", "list", "-v"]).strip())
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "remove", "tar-src"])

            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "escape-path", 'escaped-path={{.Path|clean|replace "/" "-"|trimSuffix "-"}}'])
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "src", "state-file=/var/lib/uback/state/{{.EscapedPath}}.json", "key-file=/etc/uback/backup.pub"])
            subprocess.check_call([uback, "-p", f"{d}/presets", "preset", "set", "tar-src", "type=tar", "preset=escape-path", "preset=src"])
            self.assertEqual(list(sorted(subprocess.check_output([uback, "-p", f"{d}/presets", "preset", "eval", "path=/etc,preset=tar-src"]).splitlines())), [
                b"EscapedPath: -etc",
                b"KeyFile: /etc/uback/backup.pub",
                b"Path: /etc",
                b"StateFile: /var/lib/uback/state/-etc.json",
                b"Type: tar",
            ])
