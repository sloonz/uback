from .common import *

class SrcCommandTests(unittest.TestCase, SrcBaseTests):
    def test_command_source(self):
        with tempfile.TemporaryDirectory() as d:
            os.environ["PATH"] = ":".join((str(tests_path), os.environ["PATH"]))
            source = f"type=command,command=uback-tar-src,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly,@extra-args=--exclude=./c,@extra-args=--exclude=./d"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3,key-file={d}/backup.key"
            self._test_src(d, source, dest)
