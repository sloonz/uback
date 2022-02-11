import glob
import os
import pathlib
import shlex
import shutil
import subprocess
import tempfile
import time
import unittest

uback = pathlib.Path(__file__).parent/".."/"uback"
tests_path = pathlib.Path(__file__).parent/".."/"tests"

def read_file(p, mode="rb"):
    with open(p, mode) as fd:
        return fd.read()

class SrcBaseTests:
    def _test_src(self, d, source, dest):
        os.mkdir(f"{d}/backups")
        os.mkdir(f"{d}/restore")
        os.mkdir(f"{d}/source")
        os.mkdir(f"{d}/source/d")
        subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

        with open(f"{d}/source/a", "w+") as fd: fd.write("av1")
        with open(f"{d}/source/c", "w+") as fd: fd.write("c")
        with open(f"{d}/source/d/e", "w+") as fd: fd.write("e")
        b1 = subprocess.check_output([uback, "backup", source, dest]).strip().decode()
        subprocess.check_call([uback, "restore", "-d", f"{d}/restore", dest])
        self.assertEqual(b"av1", read_file(glob.glob(f"{d}/restore/*/a")[0]))
        self.assertEqual(set(os.listdir(glob.glob(f"{d}/restore/*")[0])), {"a"})
        shutil.rmtree(f"{d}/restore")
        os.mkdir(f"{d}/restore")
        time.sleep(0.01)

        with open(f"{d}/source/b", "w+") as fd: fd.write("bv1")
        b2 = subprocess.check_output([uback, "backup", source, dest]).strip().decode()
        subprocess.check_call([uback, "restore", "-d", f"{d}/restore", dest])
        self.assertEqual(b"av1", read_file(glob.glob(f"{d}/restore/*/a")[0]))
        self.assertEqual(b"bv1", read_file(glob.glob(f"{d}/restore/*/b")[0]))
        self.assertEqual(set(os.listdir(glob.glob(f"{d}/restore/*")[0])), {"a", "b"})
        shutil.rmtree(f"{d}/restore")
        os.mkdir(f"{d}/restore")
        time.sleep(0.01)

        with open(f"{d}/source/a", "w+") as fd: fd.write("av2")
        b3 = subprocess.check_output([uback, "backup", source, dest]).strip().decode()
        subprocess.check_call([uback, "restore", "-d", f"{d}/restore", dest])
        self.assertEqual(b"av2", read_file(glob.glob(f"{d}/restore/*/a")[0]))
        self.assertEqual(b"bv1", read_file(glob.glob(f"{d}/restore/*/b")[0]))
        self.assertEqual(set(os.listdir(glob.glob(f"{d}/restore/*")[0])), {"a", "b"})
        shutil.rmtree(f"{d}/restore")
        os.mkdir(f"{d}/restore")
        time.sleep(0.01)

        return b1, b2, b3

class DestBaseTests:
    def _test_dest(self, d, source, dest):
        os.mkdir(f"{d}/backups")
        os.mkdir(f"{d}/restore")
        os.mkdir(f"{d}/source")
        subprocess.check_call([uback, "key", "gen", f"{d}/backup.key", f"{d}/backup.pub"])

        # Full 1
        with open(f"{d}/source/a", "w+") as fd: fd.write("hello")
        self.assertEqual(0, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
        subprocess.check_call([uback, "backup", "-n", "-f", source, dest])
        self.assertEqual(1, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
        time.sleep(0.01)

        # Full 2
        subprocess.check_call([uback, "backup", "-n", "-f", source, dest])
        self.assertEqual(2, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))
        time.sleep(0.01)

        # Incremental
        with open(f"{d}/source/b", "w+") as fd: fd.write("world")
        subprocess.check_call([uback, "backup", "-n", source, dest])
        self.assertEqual(3, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

        # Prune (remove full 1)
        subprocess.check_call([uback, "prune", "backups", dest])
        self.assertEqual(2, len(subprocess.check_output([uback, "list", "backups", dest]).splitlines()))

        # Restore full 2 + incremental
        subprocess.check_call([uback, "restore", "-d", f"{d}/restore", dest])
        self.assertEqual(b"hello", read_file(glob.glob(f"{d}/restore/*/a")[0]))
        self.assertEqual(b"world", read_file(glob.glob(f"{d}/restore/*/b")[0]))
