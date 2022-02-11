from .common import *

class DestFsTests(unittest.TestCase, DestBaseTests):
    def test_fs_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.environ["PATH"] = ":".join((str(tests_path), os.environ["PATH"]))
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3,key-file={d}/backup.key"
            self._test_dest(d, source, dest)
