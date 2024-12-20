from .common import *
from pyftpdlib.authorizers import DummyAuthorizer
from pyftpdlib.handlers import FTPHandler
from pyftpdlib.servers import FTPServer
import threading
import unittest

class DestFTPStorageTests(unittest.TestCase):
    def setUp(self):
        self.host = "127.0.0.1"
        self.port = 2121
        self.user = "testuser"
        self.password = "testpass"
        self.ftp_root = tempfile.mkdtemp()

        authorizer = DummyAuthorizer()
        authorizer.add_user(self.user, self.password, self.ftp_root, perm="elradfmw")
        handler = FTPHandler
        handler.authorizer = authorizer
        self.ftp_server = FTPServer((self.host, self.port), handler)
        threading.Thread(target=self.ftp_server.serve_forever, daemon=True).start()

    def tearDown(self):
        self.ftp_server.close_all()
        shutil.rmtree(self.ftp_root)

    def test_ftp_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.mkdir(f"{d}/restore")
            os.mkdir(f"{d}/source")
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=ftp,@retention-policy=daily=3,key-file={d}/backup.key,url=ftp://{self.user}:{self.password}@{self.host}:{self.port},prefix=/test"

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
            parent_dest = f"id=test,type=ftp,@retention-policy=daily=3,key-file={d}/backup.key,url=ftp://{self.user}:{self.password}@{self.host}:{self.port}"
            self.assertEqual(0, len(subprocess.check_output([uback, "list", "backups", parent_dest]).splitlines()))
