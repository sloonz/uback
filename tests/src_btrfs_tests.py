from .common import *

class SrcBtrfsTests(unittest.TestCase, SrcBaseTests):
    def setUp(self):
        self.tmpdir = os.environ.get("UBACK_BTRFS_TEST_ROOT")
        if self.tmpdir is None:
            return

        if os.path.exists(self.tmpdir):
            raise Exception("UBACK_BTRFS_TEST_ROOT already exists")

        subprocess.check_call(["btrfs", "subvolume", "create", self.tmpdir])
        subprocess.check_call(["btrfs", "subvolume", "create", f"{self.tmpdir}/snapshots"])
        subprocess.check_call(["btrfs", "subvolume", "create", f"{self.tmpdir}/source"])

    def tearDown(self):
        if os.environ.get("UBACK_BTRFS_TEST_ROOT") is None:
            return

        for s in os.listdir(f"{self.tmpdir}/snapshots"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/snapshots/{s}"])
        for s in os.listdir(f"{self.tmpdir}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/restore/{s}"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/snapshots"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/source"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", self.tmpdir])

    def _cleanup_restore(self, d):
        for s in os.listdir(f"{d}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{d}/restore/{s}"])

    def test_btrfs_source(self):
        if os.environ.get("UBACK_BTRFS_TEST_ROOT") is None:
            return

        source = [f"type=btrfs,path={self.tmpdir}/source,key-file={self.tmpdir}/backup.pub,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly",
            "send-command=sudo btrfs send", "delete-command=sudo btrfs subvolume delete"]
        dest = f"id=test,type=fs,path={self.tmpdir}/backups,@retention-policy=daily=3,key-file={self.tmpdir}/backup.key"
        self._test_src(self.tmpdir, ",".join(source), dest, "receive-command=sudo btrfs receive", test_ignore=False, test_delete=True)
