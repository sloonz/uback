from .common import *

class PruneTests(unittest.TestCase):
    def test_manual_pruning(self):
        with tempfile.TemporaryDirectory() as d:
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            os.mkdir(f"{d}/snapshots")
            os.mkdir(f"{d}/backups")
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3"

            pathlib.Path(f"{d}/snapshots/20210101T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210102T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210103T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210104T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210105T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210106T000000.000").touch()
            pathlib.Path(f"{d}/backups/20210101T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210102T000000.000-from-20210101T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210103T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210104T000000.000-from-20210103T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210105T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210106T000000.000-from-20210105T000000.000.ubkp").touch()
            with open(f"{d}/state.json", "w+") as fd: fd.write('{"test":"20210106T000000.000"}')

            self.assertEqual(6, len(subprocess.check_output([uback, "list", "snapshots", source]).splitlines()))
            self.assertEqual(6, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
            
            subprocess.check_call([uback, "prune", "snapshots", source])
            subprocess.check_call([uback, "prune", "backups", dest])

            self.assertEqual(set(os.listdir(f"{d}/snapshots")),
                {"20210106T000000.000"})
            self.assertEqual(set(os.listdir(f"{d}/backups")),
                {"20210103T000000.000-full.ubkp", "20210104T000000.000-from-20210103T000000.000.ubkp", "20210105T000000.000-full.ubkp", "20210106T000000.000-from-20210105T000000.000.ubkp"})

    def test_automatic_pruning(self):
        with tempfile.TemporaryDirectory() as d:
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            os.mkdir(f"{d}/snapshots")
            os.mkdir(f"{d}/backups")
            os.mkdir(f"{d}/source")
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3"

            pathlib.Path(f"{d}/snapshots/20210101T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210102T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210103T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210104T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210105T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210106T000000.000").touch()
            pathlib.Path(f"{d}/backups/20210101T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210102T000000.000-from-20210101T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210103T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210104T000000.000-from-20210103T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210105T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210106T000000.000-from-20210105T000000.000.ubkp").touch()
            with open(f"{d}/state.json", "w+") as fd: fd.write('{"test":"20210106T000000.000"}')

            self.assertEqual(6, len(subprocess.check_output([uback, "list", "snapshots", source]).splitlines()))
            self.assertEqual(6, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

            b = subprocess.check_output([uback, "backup", source, dest]).strip().decode()
            s = b.split("-")[0]

            self.assertEqual(set(os.listdir(f"{d}/snapshots")), {s})
            self.assertEqual(set(os.listdir(f"{d}/backups")),
                {"20210105T000000.000-full.ubkp", "20210106T000000.000-from-20210105T000000.000.ubkp", f"{b}.ubkp"})
            self.assertTrue(b.endswith("-full"))

    def test_disabled_automatic_pruning(self):
        with tempfile.TemporaryDirectory() as d:
            subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])
            os.mkdir(f"{d}/snapshots")
            os.mkdir(f"{d}/backups")
            os.mkdir(f"{d}/source")
            source = f"type=tar,path={d}/source,key-file={d}/backup.pub,state-file={d}/state.json,snapshots-path={d}/snapshots,full-interval=weekly"
            dest = f"id=test,type=fs,path={d}/backups,@retention-policy=daily=3"

            pathlib.Path(f"{d}/snapshots/20210101T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210102T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210103T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210104T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210105T000000.000").touch()
            pathlib.Path(f"{d}/snapshots/20210106T000000.000").touch()
            pathlib.Path(f"{d}/backups/20210101T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210102T000000.000-from-20210101T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210103T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210104T000000.000-from-20210103T000000.000.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210105T000000.000-full.ubkp").touch()
            pathlib.Path(f"{d}/backups/20210106T000000.000-from-20210105T000000.000.ubkp").touch()
            with open(f"{d}/state.json", "w+") as fd: fd.write('{"test":"20210106T000000.000"}')

            self.assertEqual(6, len(subprocess.check_output([uback, "list", "snapshots", source]).splitlines()))
            self.assertEqual(6, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

            b = subprocess.check_output([uback, "backup", "-n", source, dest]).strip().decode()
            s = b.split("-")[0]

            self.assertEqual(set(os.listdir(f"{d}/snapshots")),
                {"20210101T000000.000", "20210102T000000.000", "20210103T000000.000", "20210104T000000.000", "20210105T000000.000", "20210106T000000.000", s})
            self.assertEqual(set(os.listdir(f"{d}/backups")),
                {"20210101T000000.000-full.ubkp", "20210102T000000.000-from-20210101T000000.000.ubkp", "20210103T000000.000-full.ubkp", "20210104T000000.000-from-20210103T000000.000.ubkp",
                    "20210105T000000.000-full.ubkp", "20210106T000000.000-from-20210105T000000.000.ubkp", f"{b}.ubkp"})
            self.assertTrue(b.endswith("-full"))
