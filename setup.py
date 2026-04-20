import os
import subprocess
import sys

from setuptools import setup
from setuptools.command.build_py import build_py


class BuildGoSharedLib(build_py):
    def run(self):
        repo_root = os.path.dirname(os.path.abspath(__file__))
        suffix = {"darwin": "dylib", "win32": "dll"}.get(sys.platform, "so")
        target = os.path.join(repo_root, "cmd", "pylib", "mage", f"libmage.{suffix}")
        os.makedirs(os.path.dirname(target), exist_ok=True)
        subprocess.check_call(
            ["go", "build", "-buildmode=c-shared", "-o", target, "./cmd/pylib"],
            cwd=repo_root,
        )
        super().run()


setup(cmdclass={"build_py": BuildGoSharedLib})
