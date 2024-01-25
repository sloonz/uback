from .common import *

class DestProxyTests(unittest.TestCase, DestBaseTests):
    def test_proxy_destination(self):
        with tempfile.TemporaryDirectory() as d:
            os.environ["PATH"] = ":".join((str(tests_path), os.environ["PATH"]))
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=proxy,proxy-type=fs,command={uback} proxy,path={d}/backups,@retention-policy=daily=3,key-file={d}/backup.key"
            self._test_dest(d, source, dest)
