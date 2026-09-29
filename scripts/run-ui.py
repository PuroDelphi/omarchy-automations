#!/usr/bin/env python3
"""Launch the development panel with the installed Omarchy UI modules."""
import os
from pathlib import Path
import subprocess
import tempfile
from native_ui import attach_native_ui

root = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='omarchy-ui-imports-') as directory:
    env = dict(os.environ, PATH=str(root / 'build') + os.pathsep + os.environ['PATH'])
    attach_native_ui(env, directory)
    raise SystemExit(subprocess.call(['quickshell', '-p', str(root / 'Development.qml')], env=env))
