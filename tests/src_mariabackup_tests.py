from .common import *

import re

class SrcMariabackupTests(unittest.TestCase):
    def setUp(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        self.tmpdir = tempfile.mkdtemp()
        os.mkdir(f"{self.tmpdir}/snapshots")
        os.mkdir(f"{self.tmpdir}/data")

    def _wait_for_server(self, container):
        # First start involves a server restart, so the server being up is not a good enough reason to consider it available
        # To make it work 100% of the time, we have to
        #  1. Run it, wait for it to be up. At this point "being up" may mean "being up for good" or "will restart very soon".
        #  2. Restart the container. Since it’s no longer the first start, the server will not restart by itself.
        #  3. Wait for it to be up. Now "being up" really means "being up for good".
        for i in range(30):
            if subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest", "-e", "SELECT VERSION()"], capture_output=True).returncode == 0:
                break
            time.sleep(1)
        subprocess.run(["docker", "container", "restart", container], check=True)
        for i in range(30):
            if subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest", "-e", "SELECT VERSION()"], capture_output=True).returncode == 0:
                break
            time.sleep(1)
        else:
            raise Exception("cannot start mariadb")

    def _run_server(self, version):
        return subprocess.check_output(["docker", "container", "run", "--rm",
            "-v", f"{self.tmpdir}/snapshots:{self.tmpdir}/snapshots",
            "-v", f"{self.tmpdir}/data:/var/lib/mysql",
            "-e", "MARIADB_ROOT_PASSWORD=test", "-e", "MARIADB_AUTO_UPGRADE=1",
            "-u", str(os.getuid()), "-di", f"mariadb:{version}"]).strip().decode()

    def _get_source(self, container):
        return f"type=mariabackup,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly," +\
            f"command=docker exec -i {container} mariadb-backup -uroot -ptest,mariadb-command=docker exec -i {container} mariadb -uroot -ptest"

    def tearDown(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        shutil.rmtree(self.tmpdir)

    def test_mariabackup_source(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        container = None
        try:
            container = self._run_server("10.10")
            self._wait_for_server(container)

            subprocess.check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])
            source = self._get_source(container)
            dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"

            # Create 1 full & 2 incrementals
            subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest"], check=True, input=b"CREATE DATABASE ubkptest")
            subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest", "ubkptest"], check=True, input=b"CREATE TABLE test(a INT); INSERT INTO test VALUES (1);")
            subprocess.check_call([uback, "backup", "-n", source, dest])
            time.sleep(0.01)

            subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest", "ubkptest"], check=True, input=b"INSERT INTO test VALUES (2), (3);")
            subprocess.check_call([uback, "backup", "-n", source, dest])
            time.sleep(0.01)

            subprocess.run(["docker", "exec", "-i", container, "mariadb", "-uroot", "-ptest", "ubkptest"], check=True, input=b"UPDATE test SET a=4 WHERE a=1; DELETE FROM test WHERE a=2;")
            subprocess.check_call([uback, "backup", "-n", source, dest])
            time.sleep(0.01)

            # Check restoration
            subprocess.check_call([uback, "restore", "-d", f"{self.tmpdir}/restore", dest])
            restore_path = os.listdir(f"{self.tmpdir}/restore")[0]
            out = subprocess.check_output([f"{self.tmpdir}/restore/{restore_path}/sqldump-docker.sh", "ubkptest"])
            self.assertTrue(re.search(b"(?ms)INSERT INTO `test` VALUES\\s+\\(4\\),\\s*\\(3\\);", out))
            shutil.rmtree(f"{self.tmpdir}/restore")

            subprocess.check_call([uback, "restore", "-d", f"{self.tmpdir}/restore", dest])
            restore_path = os.listdir(f"{self.tmpdir}/restore")[0]
            out = subprocess.check_output([f"{self.tmpdir}/restore/{restore_path}/sqldump-local.sh", "ubkptest"])
            self.assertTrue(re.search(b"(?ms)INSERT INTO `test` VALUES\\s+\\(4\\),\\s*\\(3\\);", out))
            shutil.rmtree(f"{self.tmpdir}/restore")

            # Check that server upgrade forced a full backup
            subprocess.check_call(["docker", "container", "stop", container])
            container = None
            container = self._run_server("10.11")
            self._wait_for_server(container)
            source = self._get_source(container)
            subprocess.check_call([uback, "backup", "-n", source, dest])
            self.assertTrue(list(sorted(os.listdir(f"{self.tmpdir}/backups")))[-1].endswith("-full.ubkp"))
        finally:
            if container:
                subprocess.check_call(["docker", "container", "stop", container])
