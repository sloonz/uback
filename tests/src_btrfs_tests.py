from .common import *

import json
import unittest
import tempfile

class SrcBtrfsTests(unittest.TestCase, SrcBaseTests):
    def setUp(self):
        test_root = os.environ.get("BTRFS_ROOT")
        if test_root is None or not os.path.exists(test_root):
            raise unittest.SkipTest("btrfs not setup")

        self.tmpdir = tempfile.mkdtemp(dir=test_root)

        check_call(["btrfs", "subvolume", "create", f"{self.tmpdir}/source"])
        ensure_dir(f"{self.tmpdir}/snapshots")

    def tearDown(self):
        if self.tmpdir is None:
            return

        for s in os.listdir(f"{self.tmpdir}/snapshots"):
            check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/snapshots/{s}"])
        for s in os.listdir(f"{self.tmpdir}/restore"):
            check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/restore/{s}"])
        check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/source"])
        shutil.rmtree(self.tmpdir)

    def _cleanup_restore(self, d):
        for s in os.listdir(f"{d}/restore"):
            check_call(["sudo", "btrfs", "subvolume", "delete", f"{d}/restore/{s}"])

    def test_btrfs_source(self):
        source = f"type=btrfs,path={self.tmpdir}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly," +\
            "send-command=sudo btrfs send,delete-command=sudo btrfs subvolume delete"
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"
        self._test_src(self.tmpdir, source, dest, "receive-command=sudo btrfs receive", test_ignore=False, test_delete=True)

    def test_btrfs_reuse_snapshots(self):
        source = f"type=btrfs,path={self.tmpdir}/source,no-encryption=1,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly," +\
            "send-command=sudo btrfs send,delete-command=sudo btrfs subvolume delete,reuse-snapshots=1d"
        dest = f"type=btrfs,@retention-policy=daily=3"

        ensure_dir(f"{self.tmpdir}/backups1")
        ensure_dir(f"{self.tmpdir}/backups2")
        ensure_dir(f"{self.tmpdir}/restore")
        ensure_dir(f"{self.tmpdir}/source")
        check_call([uback, "key", "gen", f"{self.tmpdir}/backup.key", f"{self.tmpdir}/backup.pub"])
        with open(f"{self.tmpdir}/source/a", "w+") as fd: fd.write("av1")

        b1 = check_output([uback, "backup", source, f"id=test1,{dest},path={self.tmpdir}/backups1"]).strip().decode()
        time.sleep(0.01)
        b2 = check_output([uback, "backup", source, f"id=test2,{dest},path={self.tmpdir}/backups2"]).strip().decode()
        s = b1.split("-")[0]
        self.assertEqual(b1, b2)
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/snapshots")), {s})
        with open(f"{self.tmpdir}/state.json") as fd:
            self.assertEqual(json.load(fd), {"test1": s, "test2": s})

        check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/backups1/{s}"])
        check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/backups2/{s}"])