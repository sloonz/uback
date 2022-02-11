from .common import *

class SrcMariabackupTests(unittest.TestCase):
    def setUp(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        self.tmpdir = tempfile.mkdtemp()
        self.container = subprocess.check_output(["docker", "container", "create", "-v", f"{self.tmpdir}/snapshots:{self.tmpdir}/snapshots", "-e", "MARIADB_ROOT_PASSWORD=test", "mariadb:latest"]).strip().decode()
        subprocess.check_call(["docker", "container", "start", self.container])
        subprocess.check_call(["docker", "exec", self.container, "chmod", "777", f"{self.tmpdir}/snapshots"])

        for i in range(300):
            if subprocess.run(["docker", "exec", "-i", self.container, "mysql", "-uroot", "-ptest", "-e", "SELECT VERSION()"]).returncode == 0:
                break
            time.sleep(0.1)
        else:
            raise Exception("cannot start mariadb")

    def tearDown(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        subprocess.check_call(["docker", "exec", "-i", self.container, "bash", "-c", f"rm -rf {shlex.quote(self.tmpdir)}/snapshots/*"])
        shutil.rmtree(self.tmpdir)
        subprocess.check_call(["docker", "container", "rm", "-f", self.container])

    def test_mariabackup_source(self):
        if os.getenv("SKIP_MARIADB_TESTS"):
            return

        subprocess.check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])
        source = f"type=mariabackup,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,command=docker exec -i {self.container} mariabackup -uroot -ptest,full-interval=weekly"
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"

        subprocess.run(["docker", "exec", "-i", self.container, "mysql", "-uroot", "-ptest"], check=True, input=b"CREATE DATABASE ubkptest")
        subprocess.run(["docker", "exec", "-i", self.container, "mysql", "-uroot", "-ptest", "ubkptest"], check=True, input=b"CREATE TABLE test(a INT); INSERT INTO test VALUES (1);")
        subprocess.check_call([uback, "backup", "-n", source, dest])
        time.sleep(0.01)

        subprocess.run(["docker", "exec", "-i", self.container, "mysql", "-uroot", "-ptest", "ubkptest"], check=True, input=b"INSERT INTO test VALUES (2), (3);")
        subprocess.check_call([uback, "backup", "-n", source, dest])
        time.sleep(0.01)

        subprocess.run(["docker", "exec", "-i", self.container, "mysql", "-uroot", "-ptest", "ubkptest"], check=True, input=b"UPDATE test SET a=4 WHERE a=1; DELETE FROM test WHERE a=2;")
        subprocess.check_call([uback, "backup", "-n", source, dest])

        subprocess.check_call([uback, "restore", "-d", f"{self.tmpdir}/restore", dest])
        restore_path = os.listdir(f"{self.tmpdir}/restore")[0]
        shutil.copytree(f"{self.tmpdir}/restore/{restore_path}", f"{self.tmpdir}/restore2")
        self.assertIn(b"INSERT INTO `test` VALUES (4),(3);", subprocess.check_output([f"{self.tmpdir}/restore2/sqldump-docker.sh", "ubkptest"]))
        subprocess.check_call(["docker", "container", "run", "-v", f"{self.tmpdir}/restore2:/var/lib/mysql", "mariadb:latest", "bash", "-c", "rm -rf /var/lib/mysql/*"])

        self.assertIn(b"INSERT INTO `test` VALUES (4),(3);", subprocess.check_output([f"{self.tmpdir}/restore/{restore_path}/sqldump-local.sh", "ubkptest"]))
