from .common import *

class SrcProxyTests(unittest.TestCase, SrcBaseTests):
    def test_proxy_source(self):
        with tempfile.TemporaryDirectory() as d:
            source = f"type=proxy,command={uback} proxy,proxy-type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly,proxy-command=tar --exclude=./c --exclude=./d"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3,key-file={d}/backup.key"
            self._test_src(d, source, dest, test_ignore=True, test_delete=False)
