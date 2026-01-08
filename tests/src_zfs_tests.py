from .common import *

import json
import unittest
import random

class SrcZfsTests(unittest.TestCase):
    def setUp(self):
        test_root = os.environ.get("ZFS_ROOT")
        if test_root is None or not os.path.exists(test_root):
            raise unittest.SkipTest("zfs not setup")
        
        tmp_id = random.randbytes(4).hex()
        self.pool = f"{os.environ["ZFS_POOL"]}/{tmp_id}"
        self.tmpdir = f"{test_root}/{tmp_id}"

        for pool in ("", "source", "source/child1", "source/ignored"):
            self._mkpool(pool)

    def tearDown(self):
        check_call(["sudo", "zfs", "destroy", "-r", self.pool])

    def _mkpool(self, pool):
        check_call(["sudo", "zfs", "create", f"{self.pool}/{pool}".rstrip("/")])
        check_call(["sudo", "chown", os.environ["USER"], f"{self.tmpdir}/{pool}"])

    def test_zfs_replicate(self):
        source = f"type=zfs,dataset={self.pool}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,full-interval=weekly," +\
            f"destroy-command=sudo zfs destroy,exclude={self.pool}/source/ignored,replicate=true,use-bookmarks=false"
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"

        check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])

        d = f"{self.tmpdir}/source"
        with open(f"{d}/a", "w+") as fd: fd.write("av1")
        with open(f"{d}/b", "w+") as fd: fd.write("bv1")
        with open(f"{d}/c", "w+") as fd: fd.write("cv1")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("acv1")
        with open(f"{d}/ignored/a", "w+") as fd: fd.write("aiv1")
        check_call([uback, "backup", source, dest])

        self._mkpool("source/child2")
        with open(f"{d}/a", "w+") as fd: fd.write("av2")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("acv2")
        with open(f"{d}/child2/a", "w+") as fd: fd.write("a2cv1")
        os.unlink(f"{d}/b")
        b = check_output([uback, "backup", source, dest]).decode().strip()

        check_call([uback, "restore", "-o", f"dataset={self.pool}/restore,receive-command=sudo zfs receive", "-d", self.tmpdir, dest, b])
        d = f"{self.tmpdir}/restore"
        self.assertEqual(set(os.listdir(d)), {"a", "c", "child1", "child2", "ignored"})
        self.assertEqual(set(os.listdir(f"{d}/child1")), {"a"})
        self.assertEqual(set(os.listdir(f"{d}/child2")), {"a"})
        self.assertEqual(set(os.listdir(f"{d}/ignored")), set())
        self.assertEqual(read_file(f"{d}/a"), b"av2")
        self.assertEqual(read_file(f"{d}/c"), b"cv1")
        self.assertEqual(read_file(f"{d}/child1/a"), b"acv2")
        self.assertEqual(read_file(f"{d}/child2/a"), b"a2cv1")

    def test_zfs_bookmarks(self):
        source = f"type=zfs,dataset={self.pool}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,full-interval=weekly," +\
            f"destroy-command=sudo zfs destroy"
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"

        check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])

        d = f"{self.tmpdir}/source"
        with open(f"{d}/a", "w+") as fd: fd.write("av1")
        with open(f"{d}/b", "w+") as fd: fd.write("bv1")
        with open(f"{d}/c", "w+") as fd: fd.write("cv1")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("acv1")
        with open(f"{d}/ignored/a", "w+") as fd: fd.write("aiv1")
        check_call([uback, "backup", source, dest])

        self._mkpool("source/child2")
        with open(f"{d}/a", "w+") as fd: fd.write("av2")
        with open(f"{d}/child1/a", "w+") as fd: fd.write("acv2")
        with open(f"{d}/child2/a", "w+") as fd: fd.write("a2cv1")
        os.unlink(f"{d}/b")
        b = check_output([uback, "backup", source, dest]).decode().strip()

        check_call([uback, "restore", "-o", f"dataset={self.pool}/restore,receive-command=sudo zfs receive", "-d", self.tmpdir, dest, b])
        d = f"{self.tmpdir}/restore"
        self.assertEqual(set(os.listdir(d)), {"a", "c", "child1", "child2", "ignored"})
        self.assertEqual(set(os.listdir(f"{d}/child1")), set())
        self.assertEqual(set(os.listdir(f"{d}/child2")), set())
        self.assertEqual(set(os.listdir(f"{d}/ignored")), set())
        self.assertEqual(read_file(f"{d}/a"), b"av2")
        self.assertEqual(read_file(f"{d}/c"), b"cv1")
        self.assertEqual(json.loads(check_output(["zfs", "list", "-j", "-r", "-t", "snapshot", f"{self.pool}/source"]).decode())["datasets"], {})

    def test_zfs_reuse(self):
        source = f"type=zfs,dataset={self.pool}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,full-interval=weekly," +\
            f"destroy-command=sudo zfs destroy,reuse-snapshots=1d,use-bookmarks=false"
        dest = f"type=fs,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"

        ensure_dir(f"{self.tmpdir}/backups1")
        ensure_dir(f"{self.tmpdir}/backups2")
        check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])
        with open(f"{self.tmpdir}/source/a", "w+") as fd: fd.write("av1")

        b1 = check_output([uback, "backup", source, f"id=test1,{dest},path={self.tmpdir}/backups1"]).strip().decode()
        time.sleep(0.01)
        b2 = check_output([uback, "backup", source, f"id=test2,{dest},path={self.tmpdir}/backups2"]).strip().decode()
        s = b1.split("-")[0]
        zfs_snaps = json.loads(check_output(["zfs", "list", "-j", "-t", "snapshot", "-d", "1", f"{self.pool}/source"]).decode())
        self.assertEqual(b1, b2)
        self.assertEqual(set(zfs_snaps["datasets"].keys()), {f"{self.pool}/source@uback-{s}"})
        with open(f"{self.tmpdir}/state.json") as fd:
            self.assertEqual(json.load(fd), {"test1": s, "test2": s})
