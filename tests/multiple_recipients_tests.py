from .common import *

def fread(f):
    with open(f) as fd:
        return fd.read()

class SrcTarTests(unittest.TestCase, SrcBaseTests):
    def test_tar_source(self):
        with tempfile.TemporaryDirectory() as d:
            source = f"type=tar,path={d}/source,key-file={d}/backup-all.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly,@command=tar"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3"

            ensure_dir(f"{d}/backups")
            ensure_dir(f"{d}/restore")
            ensure_dir(f"{d}/source")
            check_call([uback, "key", "gen", f"{d}/backup1.key", f"{d}/backup1.pub"])
            check_call([uback, "key", "gen", f"{d}/backup2.key", f"{d}/backup2.pub"])
            with open(f"{d}/backup-all.pub", "w+") as fd:
                fd.write(fread(f"{d}/backup1.pub") + fread(f"{d}/backup2.pub"))

            with open(f"{d}/source/a", "w+") as fd: fd.write("av1")
            b = check_output([uback, "backup", source, dest]).strip().decode()
            s = b.split("-")[0]
            check_call([uback, "restore", "-d", f"{d}/restore", f"{dest},key-file={d}/backup1.key"])
            self.assertEqual(b"av1", read_file(f"{d}/restore/{s}/a"))
            self.assertEqual(set(os.listdir(f"{d}/restore/{s}")), {"a"})
            self._cleanup_restore(d)

            check_call([uback, "restore", "-d", f"{d}/restore", f"{dest},key-file={d}/backup2.key"])
            self.assertEqual(b"av1", read_file(f"{d}/restore/{s}/a"))
            self.assertEqual(set(os.listdir(f"{d}/restore/{s}")), {"a"})
            self._cleanup_restore(d)
