from .common import *
import unittest
import urllib.request
import json
import base64

class DestFTPStorageTests(unittest.TestCase):
    def setUp(self):
        self.container = subprocess.check_output(["podman", "run", "--rm", "-d", "--network=host",
            "-e", "SFTPGO_DATA_PROVIDER__CREATE_DEFAULT_ADMIN=true",
            "-e", "SFTPGO_DEFAULT_ADMIN_USERNAME=admin",
            "-e", "SFTPGO_DEFAULT_ADMIN_PASSWORD=admin",
            "-e", "SFTPGO_FTPD__BINDINGS__0__PORT=2121",
            "ghcr.io/drakkan/sftpgo:latest"]).strip()
        for i in range(30):
            try:
                urllib.request.urlopen("http://localhost:8080/healthz").read()
            except:
                time.sleep(1)
                pass
        tok = json.load(urllib.request.urlopen(urllib.request.Request("http://localhost:8080/api/v2/token",
            headers={"Authorization":"Basic "+base64.b64encode(b"admin:admin").decode()})))["access_token"]
        urllib.request.urlopen(urllib.request.Request(
            "http://localhost:8080/api/v2/users",
            data=json.dumps({"status":1,"username":"test","password":"test","permissions":{"/":["*"]}}).encode(),
            headers={"Authorization":f"Bearer {tok}", "Content-Type":"application/json"}
        ))

    def tearDown(self):
        subprocess.check_call(["podman", "stop", self.container])

    def test_ftp_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.mkdir(f"{d}/restore")
            os.mkdir(f"{d}/source")
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=ftp,@retention-policy=daily=3,key-file={d}/backup.key,url=ftp://test:test@localhost:2121,prefix=/test"

            # Full 1
            with open(f"{d}/source/a", "w+") as fd: fd.write("hello")
            self.assertEqual(0, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
            subprocess.check_call([uback, "backup", "-n", "-f", source, dest])
            self.assertEqual(1, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
            time.sleep(0.01)

            # Full 2
            subprocess.check_call([uback, "backup", "-n", "-f", source, dest])
            self.assertEqual(2, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

            # Incremental
            with open(f"{d}/source/b", "w+") as fd: fd.write("world")
            subprocess.check_call([uback, "backup", "-n", source, dest])
            self.assertEqual(3, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

            # Prune (remove full 1)
            subprocess.check_call([uback, "prune", "backups", dest])
            self.assertEqual(2, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

            # Restore full 2 + incremental
            subprocess.check_call([uback, "restore", "-d", f"{d}/restore", dest])
            self.assertEqual(b"hello", read_file(glob.glob(f"{d}/restore/*/a")[0]))
            self.assertEqual(b"world", read_file(glob.glob(f"{d}/restore/*/b")[0]))

            # Searching on "/" should not yield any result in the "/test/" prefix
            parent_dest = f"id=test,type=ftp,@retention-policy=daily=3,key-file={d}/backup.key,url=ftp://test:test@localhost:2121"
            self.assertEqual(0, len(subprocess.check_output([uback, "list", "backups", parent_dest]).splitlines()))
