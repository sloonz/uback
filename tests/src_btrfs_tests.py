from .common import *

class SrcBtrfsTests(unittest.TestCase, SrcBaseTests):
    def setUp(self):
        test_root = os.environ.get("UBACK_BTRFS_TEST_ROOT")
        if test_root is None:
            self.tmpdir = None
            return

        ensure_dir(test_root)
        self.tmpdir = f"{test_root}/src-test"
        if os.path.exists(self.tmpdir):
            raise Exception("UBACK_BTRFS_TEST_ROOT already exists")

        subprocess.check_call(["btrfs", "subvolume", "create", self.tmpdir])
        subprocess.check_call(["btrfs", "subvolume", "create", f"{self.tmpdir}/source"])
        ensure_dir(f"{self.tmpdir}/snapshots")

    def tearDown(self):
        if self.tmpdir is None:
            return

        for s in os.listdir(f"{self.tmpdir}/snapshots"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/snapshots/{s}"])
        for s in os.listdir(f"{self.tmpdir}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/restore/{s}"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/source"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", self.tmpdir])
        try:
            os.rmdir(os.environ.get("UBACK_BTRFS_TEST_ROOT"))
        except:
            pass

    def _cleanup_restore(self, d):
        for s in os.listdir(f"{d}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{d}/restore/{s}"])

    def test_btrfs_source(self):
        if self.tmpdir is None:
            return

        source = f"type=btrfs,path={self.tmpdir}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly," +\
            "send-command=sudo btrfs send,delete-command=sudo btrfs subvolume delete"
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"
        self._test_src(self.tmpdir, source, dest, "receive-command=sudo btrfs receive", test_ignore=False, test_delete=True)
