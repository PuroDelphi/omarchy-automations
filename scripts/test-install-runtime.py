#!/usr/bin/env python3
"""Exercise installed binaries and restored state without touching the desktop."""
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import time

project = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix="quatrro-install-") as temporary:
    home = Path(temporary)
    runtime = home / "run"
    runtime.mkdir(mode=0o700)
    env = {**os.environ, "HOME": str(home), "XDG_CONFIG_HOME": str(home / ".config"), "XDG_STATE_HOME": str(home / ".local/state"), "XDG_RUNTIME_DIR": str(runtime)}
    env.pop("QUATRRO_PROFILE", None)
    installer = ["python", str(project / "scripts/install.py"), "--staging-root", str(home)]
    process = None

    def call(op, data=None, check=True):
        result = subprocess.run([str(home / ".local/bin/quatrroctl"), op, "--stdin"], input=json.dumps(data or {}), text=True, capture_output=True, env=env, timeout=20, check=check)
        return json.loads(result.stdout) if check else result

    def start():
        global process
        process = subprocess.Popen([str(home / ".local/bin/quatrrod"), "-listen="], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise AssertionError(process.stderr.read().decode())
            if call("status", check=False).returncode == 0:
                return
            time.sleep(0.05)
        raise AssertionError("engine did not start")

    def stop():
        global process
        if process:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise
            finally:
                process.stderr.close()
                process = None

    try:
        subprocess.run(installer, check=True, capture_output=True)
        start()
        call("control", {"paused": True, "admission": "retain"})
        config = json.loads((project / "examples/notification.json").read_text())
        call("config.save", config)
        preview = call("config.preview")
        call("config.activate", {"hash": preview["hash"], "grants": preview["capabilities"]})
        call("emit", {"source": "local:demo", "type": "demo", "data": {"message": "preserved fixture"}})
        assert call("status")["counts"].get("pending") == 1
        refused = subprocess.run(installer, capture_output=True, text=True)
        assert refused.returncode != 0 and "Stop the engine" in refused.stderr
        call("preferences.set", {"language": "es"})
        stop()
        # Reconstruct the immediately previous data layout (v4 had no execution
        # context column). This is a schema fixture, not an archived old binary.
        database = home / '.local/state/quatrro/quatrro.db'
        with sqlite3.connect(database) as db:
            db.execute('ALTER TABLE executions DROP COLUMN context')
            db.execute('DELETE FROM schema_version')
            db.execute('INSERT INTO schema_version VALUES(4)')
        update = json.loads(subprocess.check_output(installer, text=True))
        start()
        assert call("status")["language"] == "es"
        assert call("status")["counts"].get("pending") == 1
        with sqlite3.connect(f'file:{database}?mode=ro', uri=True) as db:
            assert db.execute('SELECT version FROM schema_version').fetchone()[0] == 5
            assert db.execute('SELECT state,context FROM executions').fetchone() == ('pending', '')
        config["flows"][0]["name"] = "newer draft"
        call("config.save", config)
        call("preferences.set", {"language": "en"})
        stop()
        subprocess.run(installer + ["--restore-data", update["backup"]], check=True, capture_output=True)
        start()
        status = call("status")
        assert status["paused"] and status["counts"].get("uncertain") == 1
        assert status["language"] == "es"
        assert call("permissions.list") == []
        assert call("config.get")["flows"][0]["name"] == "Notificación local"
        assert call("emit", {"source": "local:demo", "data": {}}, check=False).returncode != 0
        stop()
        subprocess.run(installer + ["--uninstall"], check=True, capture_output=True)
        assert (home / ".local/state/quatrro/quatrro.db").is_file()
        assert not (home / ".local/bin/quatrrod").exists()
        print(json.dumps({"installed_runtime": True, "schema_4_to_5_preserves_pending": True, "update_backup": True, "restore_paused": True, "queue_uncertain": True, "grants_revoked": True, "uninstall_preserved_data": True}))
    finally:
        stop()
