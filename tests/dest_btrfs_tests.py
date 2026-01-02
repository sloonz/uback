from .common import *

class DestBtrfsTests(unittest.TestCase, SrcBaseTests):
    def setUp(self):
        test_root = os.environ.get("BTRFS_ROOT")
        if test_root is None:
            raise unittest.SkipTest("BTRFS_ROOT not set")

        self.tmpdir = tempfile.mkdtemp(dir=test_root)

        subprocess.check_call(["btrfs", "subvolume", "create", f"{self.tmpdir}/source"])
        ensure_dir(f"{self.tmpdir}/snapshots")

    def tearDown(self):
        for s in os.listdir(f"{self.tmpdir}/backups"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/backups/{s}"])
        for s in os.listdir(f"{self.tmpdir}/snapshots"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/snapshots/{s}"])
        for s in os.listdir(f"{self.tmpdir}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/restore/{s}"])
        subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{self.tmpdir}/source"])
        shutil.rmtree(self.tmpdir)

    def _cleanup_restore(self, d):
        for s in os.listdir(f"{d}/restore"):
            subprocess.check_call(["sudo", "btrfs", "subvolume", "delete", f"{d}/restore/{s}"])

    def test_btrfs_dest(self):
        source = f"type=btrfs,path={self.tmpdir}/source,no-encryption=1,state-file={self.tmpdir}/state.json,snapshots-path={self.tmpdir}/snapshots,full-interval=weekly," +\
            "send-command=sudo btrfs send,delete-command=sudo btrfs subvolume delete"
        dest = f"id=test,type=btrfs,path={self.tmpdir}/backups,no-encryption=1," +\
            "send-command=sudo btrfs send,receive-command=sudo btrfs receive,delete-command=sudo btrfs subvolume delete"
        b1, b2, b3, b4 = self._test_src(self.tmpdir, source, dest, "receive-command=sudo btrfs receive", test_ignore=False, test_delete=True)

        # Check that btrfs subvolumes contains the correct data
        s1 = b1.split("-")[0]
        s2 = b2.split("-")[0]
        s3 = b3.split("-")[0]
        s4 = b4.split("-")[0]
        self.assertEqual(b"av1", read_file(f"{self.tmpdir}/backups/{s1}/a"))
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/{s1}")), {"a"})
        self.assertEqual(b"av1", read_file(f"{self.tmpdir}/backups/{s2}/a"))
        self.assertEqual(b"bv1", read_file(f"{self.tmpdir}/backups/{s2}/b"))
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/{s2}")), {"a", "b"})
        self.assertEqual(b"av2", read_file(f"{self.tmpdir}/backups/{s3}/a"))
        self.assertEqual(b"bv1", read_file(f"{self.tmpdir}/backups/{s3}/b"))
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/{s3}")), {"a", "b"})
        self.assertEqual(b"av2", read_file(f"{self.tmpdir}/backups/{s4}/a"))
        self.assertEqual(set(os.listdir(f"{self.tmpdir}/backups/{s4}")), {"a"})
