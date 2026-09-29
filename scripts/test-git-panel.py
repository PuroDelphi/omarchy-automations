#!/usr/bin/env python3
"""Run the prebuilt runtime with a Git-managed native panel in an isolated HOME."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='automations-git-panel-') as directory:
    home = Path(directory)
    plugin = home / '.config/omarchy/plugins/quatrro.automations'
    plugin.mkdir(parents=True)
    subprocess.run(['git', 'init', '-q', str(plugin)], check=True)
    for name in ('Panel.qml', 'Widget.qml', 'Development.qml', 'manifest.json'):
        shutil.copy2(ROOT / name, plugin / name)
    shutil.copytree(ROOT / 'qml', plugin / 'qml')
    shutil.copytree(ROOT / 'scripts', plugin / 'scripts', ignore=shutil.ignore_patterns('__pycache__'))
    archive = Path(subprocess.check_output(['python3', str(ROOT / 'scripts/package-release.py'), '--output', str(home / 'artifacts')], text=True).strip())
    subprocess.run(['python3', str(plugin / 'scripts/setup.py'), '--archive', str(archive), '--staging-root', str(home)], check=True)
    env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(home / 'profile'),
               PATH=str(home / '.local/bin') + os.pathsep + os.environ['PATH'],
               QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
    attach_native_ui(env, home)
    def wait(check):
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            value = check()
            if value: return value
            time.sleep(.1)
        raise AssertionError('Native panel readiness timeout')
    def ipc(method, *args):
        return subprocess.check_output(['quickshell', 'ipc', '-p', str(plugin / 'Development.qml'),
            'call', 'quatrro-development', method, *args], env=env, text=True, stderr=subprocess.DEVNULL, timeout=5)
    def ready():
        try:
            state = json.loads(ipc('state'))
            return state if state['loaded'] and state['connected'] and not state['busy'] else None
        except (subprocess.SubprocessError, ValueError): return None
    children=[]
    try:
        with (home / 'runtime.log').open('w') as log:
            children.append(subprocess.Popen([home / '.local/bin/quatrrod', '--listen='], env=env, stdout=log, stderr=log))
            wait(lambda: (home / 'profile/runtime/control.sock').exists())
            children.append(subprocess.Popen(['quickshell', '-p', str(plugin / 'Development.qml')], env=env, stdout=log, stderr=log))
            state = wait(ready)
            assert state['language'] == 'en', state
            ipc('language', 'es')
            wait(lambda: ready() and json.loads(ipc('state'))['language'] == 'es')
            ipc('language', 'en')
            wait(lambda: ready() and json.loads(ipc('state'))['language'] == 'en')
    finally:
        for child in reversed(children):
            child.terminate()
            try: child.wait(timeout=5)
            except subprocess.TimeoutExpired: child.kill(); child.wait()
    log = (home / 'runtime.log').read_text()
    assert not any(message in log for message in ('ERROR:', 'ReferenceError:', 'TypeError:')), log
    subprocess.run(['python3', str(plugin / 'scripts/setup.py'), '--staging-root', str(home), '--uninstall'], check=True)
    assert (plugin / 'manifest.json').is_file()
print(json.dumps({'git_panel_connected_to_installed_runtime': True, 'languages': ['en','es'], 'host_modified': False}))
