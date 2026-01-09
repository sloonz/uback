from .common import *

class SrcTarTests(unittest.TestCase, SrcBaseTests):
    def test_tar_source(self):
        with tempfile.TemporaryDirectory() as d:
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly,@command=tar,@command=--exclude=./c,@command=--exclude=./d"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3,key-file={d}/backup.key"
            b1, b2, b3, _ = self._test_src(d, source, dest, test_ignore=True, test_delete=False)

            # Check that incremental backups are actually incremental
            run(["tar", "-C", f"{d}/restore", "-x"], input=check_output([uback, "container", "extract", "-k", f"{d}/backup.key"], input=read_file(f"{d}/backups/{b2}.ubkp")), check=True)
            self.assertEqual(set(os.listdir(f"{d}/restore/")), {"b"})
            self.assertEqual(b"bv1", read_file(f"{d}/restore/b"))
            shutil.rmtree(f"{d}/restore")
            os.mkdir(f"{d}/restore")

            run(["tar", "-C", f"{d}/restore", "-x"], input=check_output([uback, "container", "extract", "-k", f"{d}/backup.key"], input=read_file(f"{d}/backups/{b3}.ubkp")), check=True)
            self.assertEqual(set(os.listdir(f"{d}/restore/")), {"a"})
            self.assertEqual(b"av2", read_file(f"{d}/restore/a"))
