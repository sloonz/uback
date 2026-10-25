from .common import *

class DestObjectStorageTests(unittest.TestCase):
    def setUp(self):
        self.container = check_output([
            "podman", "run", "--rm", "-dp", "7070:7070",
            "-e", "ROOT_ACCESS_KEY=testuser", "-e", "ROOT_SECRET_KEY=testsecret",
            "docker.io/versity/versitygw:latest", "--port", ":7070", "posix", "/tmp/vgw",
        ]).strip().decode()
        self.addCleanup(check_call, ["podman", "stop", self.container])
        for i in range(300):
            if run([
                "podman", "exec", self.container, "versitygw", "admin",
                "-a", "testuser", "-s", "testsecret", "-er", "http://127.0.0.1:7070",
                "create-bucket", "--bucket", "testbucket", "--owner", "testuser",
            ]).returncode == 0:
                break
            time.sleep(0.1)
        else:
            raise Exception("could not initialize versitygw")

    def test_os_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.mkdir(f"{d}/restore")
            os.mkdir(f"{d}/source")
            check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=object-storage,@retention-policy=daily=3,key-file={d}/backup.key,url=http://testuser:testsecret@localhost:7070/testbucket,prefix=/test"

            # Full 1
            with open(f"{d}/source/a", "w+") as fd: fd.write("hello")
            self.assertEqual(0, len(check_output([uback, "list", "backups", dest]).splitlines()))
            check_call([uback, "backup", "-n", "-f", source, dest])
            self.assertEqual(1, len(check_output([uback, "list", "backups", dest]).splitlines()))
            time.sleep(0.01)

            # Full 2
            check_call([uback, "backup", "-n", "-f", source, dest])
            self.assertEqual(2, len(check_output([uback, "list", "backups", dest]).splitlines()))

            # Incremental
            with open(f"{d}/source/b", "w+") as fd: fd.write("world")
            check_call([uback, "backup", "-n", source, dest])
            self.assertEqual(3, len(check_output([uback, "list", "backups", dest]).splitlines()))

            # Prune (remove full 1)
            check_call([uback, "prune", "backups", dest])
            self.assertEqual(2, len(check_output([uback, "list", "backups", dest]).splitlines()))

            # Restore full 2 + incremental
            check_call([uback, "restore", "-d", f"{d}/restore", dest])
            self.assertEqual(b"hello", read_file(glob.glob(f"{d}/restore/*/a")[0]))
            self.assertEqual(b"world", read_file(glob.glob(f"{d}/restore/*/b")[0]))

            # Searching on "/" should not yield any result in the "/test/" prefix
            parent_dest = f"id=test,type=object-storage,@retention-policy=daily=3,key-file={d}/backup.key,url=http://testuser:testsecret@localhost:7070/testbucket"
            self.assertEqual(0, len(check_output([uback, "list", "backups", parent_dest]).splitlines()))
