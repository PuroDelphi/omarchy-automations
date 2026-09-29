#!/usr/bin/env python3
"""Open only the development panel; isolate state and clean up child processes."""
import json
from native_ui import attach_native_ui
import os
from pathlib import Path
import subprocess
import tempfile
import time
import sys

root = Path(__file__).resolve().parents[1]
artifacts = root / ".dev"
artifacts.mkdir(exist_ok=True)
(artifacts / "panel-render.png").unlink(missing_ok=True)
offscreen = "--offscreen" in sys.argv
children = []
with tempfile.TemporaryDirectory(prefix="quatrro-native-") as profile:
    env = dict(os.environ, QUATRRO_PROFILE=profile, QUATRRO_CAPTURE=str(artifacts / "panel-render.png"),
               PATH=str(root / "build") + os.pathsep + os.environ["PATH"])
    attach_native_ui(env, profile)
    if offscreen:
        env.update(QT_QPA_PLATFORM="offscreen", QT_QUICK_BACKEND="software")
    try:
        with (artifacts / "engine.log").open("w") as engine_log, (artifacts / "qml.log").open("w") as qml_log:
            children.append(subprocess.Popen([root / "build/quatrrod", "--listen="], env=env, stdout=engine_log, stderr=engine_log))
            for _ in range(100):
                if Path(profile, "runtime/control.sock").exists():
                    break
                time.sleep(.05)
            print(subprocess.check_output([root / "build/quatrroctl", "status"], env=env, text=True))
            children.append(subprocess.Popen(["quickshell", "-p", str(root / "Development.qml")], env=env, stdout=qml_log, stderr=qml_log))
            time.sleep(6)
            own = []
            if not offscreen:
                clients = json.loads(subprocess.check_output(["hyprctl", "clients", "-j"]))
                own = [c for c in clients if c.get("title") == "Omarchy Automations"]
            print((artifacts / "qml.log").read_text())
            if not offscreen and not own:
                raise RuntimeError("Development panel did not open")
            if not (artifacts / "panel-render.png").exists():
                raise RuntimeError("Panel render was not produced")
            print("Panel rendered:", artifacts / "panel-render.png")
    finally:
        for child in reversed(children):
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
