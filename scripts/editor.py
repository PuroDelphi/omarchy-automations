#!/usr/bin/env python3
"""Open the native editor outside the shared shell with enforced cgroup limits."""
import argparse
import hashlib
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time

from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
MEMORY = 512 * 1024 * 1024
TASKS = 128
PROPERTIES = (
    'MemoryMax=512M', 'MemorySwapMax=0', 'CPUQuota=100%', 'TasksMax=128',
    'OOMPolicy=kill', 'KillMode=control-group', 'TimeoutStopSec=3s',
    'LimitCORE=0', 'StandardOutput=null', 'StandardError=null',
)
# Session context only. Clipboard/credential content is never an argument,
# environment value, log entry, or response sent to the shared shell.
ENVIRONMENT = (
    'HOME', 'XDG_CONFIG_HOME', 'XDG_STATE_HOME', 'XDG_DATA_HOME',
    'XDG_RUNTIME_DIR', 'DBUS_SESSION_BUS_ADDRESS', 'WAYLAND_DISPLAY', 'DISPLAY',
    'XDG_CURRENT_DESKTOP', 'XDG_SESSION_TYPE', 'LANG', 'LC_ALL',
    'QT_QPA_PLATFORM', 'QT_QUICK_BACKEND', 'QT_IM_MODULE', 'QT_IM_MODULES',
    'QUATRRO_PROFILE',
)


def unit_name():
    # Development profiles never collide with the installed editor.
    profile = os.environ.get('QUATRRO_PROFILE')
    suffix = '-' + hashlib.sha256(profile.encode()).hexdigest()[:16] if profile else ''
    return 'quatrro-automations-editor' + suffix + '.service'


def command():
    args = ['/usr/bin/systemd-run', '--user', '--quiet', '--collect',
            '--service-type=exec', '--expand-environment=no', '--unit=' + unit_name()]
    args += ['--property=' + prop for prop in PROPERTIES]
    args += ['--setenv=' + name + '=' + os.environ[name] for name in ENVIRONMENT if name in os.environ]
    return args + ['--', '/usr/bin/python3', str(Path(__file__).resolve()), '--worker']


def verify_limits():
    """Fail closed before creating Qt controls, including direct --worker calls."""
    groups = Path('/proc/self/cgroup').read_text().splitlines()
    relative = next((line[3:] for line in groups if line.startswith('0::')), None)
    if not relative or not relative.startswith('/') or '..' in Path(relative).parts:
        raise RuntimeError('Unified cgroup limits unavailable')
    group = Path('/sys/fs/cgroup') / relative.lstrip('/')
    if group.name != unit_name():
        raise RuntimeError('Editor must run in its dedicated user unit')
    memory = int((group / 'memory.max').read_text())
    swap = int((group / 'memory.swap.max').read_text())
    tasks = int((group / 'pids.max').read_text())
    quota, period = map(int, (group / 'cpu.max').read_text().split())
    if not (0 < memory <= MEMORY and swap == 0 and 0 < tasks <= TASKS and 0 < quota <= period):
        raise RuntimeError('Editor resource limits were not enforced')


def worker():
    verify_limits()
    with tempfile.TemporaryDirectory(prefix='quatrro-editor-imports-') as directory:
        env = {name: os.environ[name] for name in ENVIRONMENT if name in os.environ}
        env['PATH'] = str(Path.home() / '.local/bin') + ':/usr/local/bin:/usr/bin'
        # Only installed native modules; no inherited development import paths.
        attach_native_ui(env, directory)
        return subprocess.call(['/usr/bin/quickshell', '--no-duplicate', '-p', str(ROOT / 'EditorWindow.qml')], env=env)


def show():
    result = subprocess.run(
        ['/usr/bin/quickshell', 'ipc', '-p', str(ROOT / 'EditorWindow.qml'),
         'call', 'quatrro-editor', 'show'], stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL, timeout=2)
    return result.returncode == 0


def launch(restart=False):
    if restart:
        subprocess.run(['/usr/bin/systemctl', '--user', 'stop', unit_name()],
                       check=True, timeout=6, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    active = subprocess.run(['/usr/bin/systemctl', '--user', 'is-active', '--quiet', unit_name()],
                            timeout=3).returncode == 0
    if active:
        if not show():
            raise RuntimeError('Running editor unavailable; restart it to recover')
        return
    subprocess.run(command(), check=True, timeout=6, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    until = time.monotonic() + 10
    while time.monotonic() < until:
        if show():
            return
        time.sleep(.1)
    raise RuntimeError('Editor did not become ready')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--open', action='store_true')
    mode.add_argument('--restart', action='store_true')
    mode.add_argument('--worker', action='store_true')
    args = parser.parse_args()
    try:
        if args.worker:
            return worker()
        launch(restart=args.restart)
        return 0
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError):
        # Do not forward Qt/IPC output, session paths, or user input to the host.
        print('Omarchy Automations editor unavailable; retry or restart the editor.', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
