#!/usr/bin/env python3
"""Run the packaged service as a temporary user unit, without enabling it."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
import uuid

project = Path(__file__).resolve().parents[1]
unit = "quatrro-service-test-" + uuid.uuid4().hex[:12] + ".service"
runtime = Path(os.environ["XDG_RUNTIME_DIR"])
unit_file = runtime / "systemd/user" / unit
unit_file.parent.mkdir(parents=True, exist_ok=True)
(project / ".dev").mkdir(exist_ok=True)


def systemctl(*args, check=True):
    return subprocess.run(["systemctl", "--user", *args], capture_output=True, text=True, timeout=30, check=check)


def quote(value):
    return '"' + str(value).replace("\\", "\\\\").replace('"', '\\"').replace("%", "%%") + '"'


with tempfile.TemporaryDirectory(prefix="service-", dir=project / ".dev") as temporary:
    profile = Path(temporary)
    env = {**os.environ, "QUATRRO_PROFILE": str(profile)}

    def call(op, data=None, check=True):
        result = subprocess.run([str(project / "build/quatrroctl"), op, "--stdin"], input=json.dumps(data or {}), env=env, text=True, capture_output=True, timeout=20, check=check)
        return json.loads(result.stdout) if check else result

    def wait_for(predicate, label, seconds=15):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if predicate():
                return
            time.sleep(0.1)
        raise AssertionError(label)

    def pid():
        return int(systemctl("show", unit, "--property=MainPID", "--value").stdout.strip())

    template = (project / "packaging/systemd/quatrrod.service").read_text()
    template = template.replace("ExecStart=%h/.local/bin/quatrrod", "ExecStart=" + quote(project / "build/quatrrod") + " -listen=\nEnvironment=" + quote("QUATRRO_PROFILE=" + str(profile)))
    unit_file.write_text(template)
    try:
        systemctl("daemon-reload")
        systemctl("start", unit)
        wait_for(lambda: call("status", check=False).returncode == 0, "service socket unavailable")
        call("control", {"paused": True, "admission": "retain"})
        config = json.loads((project / "examples/notification.json").read_text())
        config["actions"].append({"id": "bounded", "kind": "command", "executable": "/usr/bin/true", "args": [], "timeout_seconds": 5})
        config["flows"][0]["steps"].append("bounded")
        call("config.save", config)
        preview = call("config.preview")
        call("config.activate", {"hash": preview["hash"], "grants": preview["capabilities"]})
        call("emit", {"source": "local:demo", "data": {"message": "Quatrro: prueba temporal de servicio"}})
        assert call("status")["counts"].get("pending") == 1
        first_pid = pid()
        systemctl("kill", "--signal=KILL", "--kill-whom=main", unit)
        wait_for(lambda: pid() not in (0, first_pid) and call("status", check=False).returncode == 0, "service did not restart")
        assert call("status")["paused"] and call("status")["counts"].get("pending") == 1
        call("control", {"paused": False, "admission": "retain"})
        wait_for(lambda: call("status")["counts"].get("completed") == 1, "service actions did not complete")
        systemctl("stop", unit)
        assert call("status", check=False).returncode != 0
        systemctl("start", unit)
        wait_for(lambda: call("status", check=False).returncode == 0, "service did not start again")
        assert call("status")["counts"].get("completed") == 1
        properties = systemctl("show", unit, "--property=NoNewPrivileges,PrivateTmp,RestrictSUIDSGID,UMask").stdout
        assert "NoNewPrivileges=yes" in properties and "PrivateTmp=yes" in properties and "UMask=0077" in properties
        print(json.dumps({"packaged_unit": True, "crash_restart": True, "persistent_queue": True, "notification_and_isolated_command": True, "stop_start": True}))
    except Exception:
        diagnostic = subprocess.run(["journalctl", "--user", "-u", unit, "-n", "20", "--no-pager"], text=True, capture_output=True, timeout=10)
        print(diagnostic.stdout)
        raise
    finally:
        systemctl("stop", unit, check=False)
        unit_file.unlink(missing_ok=True)
        systemctl("daemon-reload", check=False)
        systemctl("reset-failed", unit, check=False)
