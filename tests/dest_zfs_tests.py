from .common import *

import unittest
import random
import json

class DestZfsTests(unittest.TestCase):
    def setUp(self):
        test_root = os.environ.get("ZFS_ROOT")
        if test_root is None or not os.path.exists(test_root):
            raise unittest.SkipTest("zfs not setup")

        tmp_id = random.randbytes(4).hex()
        self.pool = f"{os.environ["ZFS_POOL"]}/{tmp_id}"
        self.tmpdir = f"{test_root}/{tmp_id}"

        for pool in ("", "source", "source/child1", "source/child2"):
            self._mkpool(pool)

    def tearDown(self):
        check_call(["sudo", "zfs", "destroy", "-r", self.pool])

    def _mkpool(self, pool):
        check_call(["sudo", "zfs", "create", f"{self.pool}/{pool}".rstrip("/")])
        check_call(["sudo", "chown", os.environ["USER"], f"{self.tmpdir}/{pool}"])

    def test_zfs_dest(self):
        source = f"type=zfs,dataset={self.pool}/source,state-file={self.tmpdir}/state.json,full-interval=weekly," +\
            f"destroy-command=sudo zfs destroy,no-encryption=1,replicate=true,use-bookmarks=false"
        dest = f"id=zfs,type=zfs,dataset={self.pool}/backups,@retention-policy=daily=3,receive-command=sudo zfs receive"

        d = f"{self.tmpdir}/source"
        with open(f"{d}/a", "w+") as fd: fd.write("av1")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("ac1v1")
        with open(f"{d}/child2/a", "w+") as fd: fd.write("ac2v1")
        b1 = check_output([uback, "backup", source, dest]).decode().split("-")[0]

        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups")), {"a", "child1", "child2"})
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/child1")), {"a"})
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/child2")), {"a"})
        self.assertEqual(read_file(f"{self.tmpdir}/backups/a"), b"av1")
        self.assertEqual(read_file(f"{self.tmpdir}/backups/child1/a"), b"ac1v1")

        check_call(["sudo", "zfs", "destroy", "-r", f"{self.pool}/source/child2"])
        self._mkpool("source/child3")
        with open(f"{d}/a", "w+") as fd: fd.write("av2")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("ac1v2")
        with open(f"{d}/child3/a", "w+") as fd: fd.write("ac3v1")
        b2 = check_output([uback, "backup", source, dest]).decode().split("-")[0]

        zfs_snaps = lambda ds: json.loads(check_output(["zfs", "list", "-j", "-t", "snapshot", "-d", "1", f"{self.pool}/{ds}"]).decode())["datasets"]
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups")), {"a", "child1", "child3"})
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/child1")), {"a"})
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/child2")), {"a"})
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/child3")), {"a"})
        self.assertEqual(read_file(f"{self.tmpdir}/backups/a"), b"av2")
        self.assertEqual(read_file(f"{self.tmpdir}/backups/child1/a"), b"ac1v2")
        self.assertEqual(read_file(f"{self.tmpdir}/backups/child2/a"), b"ac2v1")
        self.assertEqual(read_file(f"{self.tmpdir}/backups/child3/a"), b"ac3v1")
        self.assertEqual(set(zfs_snaps("backups").keys()), {f"{self.pool}/backups@uback-{b1}", f"{self.pool}/backups@uback-{b2}"})
        self.assertEqual(set(zfs_snaps("backups/child1").keys()), {f"{self.pool}/backups/child1@uback-{b1}", f"{self.pool}/backups/child1@uback-{b2}"})
        self.assertEqual(set(zfs_snaps("backups/child2").keys()), {f"{self.pool}/backups/child2@uback-{b1}"})
        self.assertEqual(set(zfs_snaps("backups/child3").keys()), {f"{self.pool}/backups/child3@uback-{b2}"})

    def test_zfs_dest_remove_backup(self):
        source = f"type=zfs,dataset={self.pool}/source,state-file={self.tmpdir}/state.json,full-interval=weekly," +\
            f"destroy-command=sudo zfs destroy,no-encryption=1,replicate=true,use-bookmarks=false"
        dest = f"id=zfs,type=zfs,dataset={self.pool}/backups,@retention-policy=daily=1,receive-command=sudo zfs receive," +\
            f"destroy-command=sudo zfs destroy"

        d = f"{self.tmpdir}/source"
        with open(f"{d}/a", "w+") as fd: fd.write("v1")
        b1 = check_output([uback, "backup", "-n", "-f", source, dest]).decode().split("-")[0]
        time.sleep(0.01)
        with open(f"{d}/a", "w+") as fd: fd.write("v2")
        b2 = check_output([uback, "backup", "-n", source, dest]).decode().split("-")[0]

        zfs_snaps = lambda ds: json.loads(check_output(["zfs", "list", "-j", "-t", "snapshot", "-d", "1", f"{self.pool}/{ds}"]).decode())["datasets"]
        self.assertEqual(set(zfs_snaps("backups").keys()), {f"{self.pool}/backups@uback-{b1}", f"{self.pool}/backups@uback-{b2}"})

        check_call([uback, "prune", "backups", dest])
        self.assertEqual(set(zfs_snaps("backups").keys()), {f"{self.pool}/backups@uback-{b2}"})
