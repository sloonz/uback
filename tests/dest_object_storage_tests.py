from .common import *

class DestObjectStorageTests(unittest.TestCase):
    def setUp(self):
        subprocess.check_call(["docker", "network", "create", "--driver=bridge", "uback-minio-test-bridge"])
        self.container = subprocess.check_output(["docker", "container", "create", "--network", "uback-minio-test-bridge", "-h", "uback-minio-test", "-p", "9000:9000", "minio/minio", "server", "/data"]).strip().decode()
        subprocess.check_call(["docker", "container", "start", self.container])
        for i in range(300):
            if subprocess.run(["docker", "run", "--network", "uback-minio-test-bridge", "--entrypoint=/bin/sh", "-i", "minio/mc", "-c", "mc alias set minio http://uback-minio-test:9000 minioadmin minioadmin && mc mb minio/testbucket"]).returncode == 0:
                break
            time.sleep(0.1)
        else:
            raise Exception("could not initialize minio")

    def tearDown(self):
        subprocess.check_call(["docker", "container", "rm", "-f", self.container])
        subprocess.check_call(["docker", "network", "rm", "uback-minio-test-bridge"])

    def test_os_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.mkdir(f"{d}/restore")
            os.mkdir(f"{d}/source")
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=object-storage,@retention-policy=daily=3,key-file={d}/backup.key,url=http://minioadmin:minioadmin@localhost:9000/testbucket,prefix=/test"

            # Full 1
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
