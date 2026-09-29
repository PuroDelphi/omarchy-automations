#!/usr/bin/env python3
"""Render populated tutorial forms in isolated profiles; never activate effects."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / '.dev/tutorial-captures'
OUTPUT.mkdir(parents=True, exist_ok=True)
CASES = {
    'webhook-notification': ('entries', 'flows'),
    'monitor-outbound': ('monitors', 'destinations'),
    'scheduled-notification': ('timers', 'actions'),
    'github-release': ('entries', 'flows'),
    'user-service': ('actions', 'flows'),
    'http-recovery': ('destinations', 'actions'),
    'google-oauth': ('destinations', 'actions'),
    'typed-script': ('scripts', 'actions'),
    'adapter-notification': ('adapters', 'actions'),
}

def wait(check, label):
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline:
        result = check()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError(label)

with tempfile.TemporaryDirectory(prefix='omarchy-tutorials-') as temporary:
    base = Path(temporary)
    ui = base / 'ui'
    ui.mkdir()
    shutil.copy2(ROOT / 'Panel.qml', ui / 'Panel.qml')
    shutil.copytree(ROOT / 'qml', ui / 'qml')
    harness = (ROOT / 'Development.qml').read_text().replace(
        'target: "quatrro-development"', '''target: "quatrro-development"
        function tutorialEdit(kind: string): void {
            panel.resourceEditor.kind = kind;
            panel.resourceEditor.edit(panel.config[kind][0]);
        }
        function tutorialCapture(path: string): void {
            panel.resourceEditor.background.parent.grabToImage(function(result) { result.saveToFile(path); });
        }
        function tutorialReload(): void { panel.open("{}"); panel.backendClient.call("config.get", {}); }
''', 1)
    (ui / 'Development.qml').write_text(harness)
    home = base / 'home'
    current = home / '.local/state/omarchy/current'
    current.mkdir(parents=True)
    (current / 'theme').symlink_to('/usr/share/omarchy/themes/tokyo-night', target_is_directory=True)
    env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(home / 'profile'),
               PATH=str(ROOT / 'build') + os.pathsep + os.environ['PATH'],
               QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
    attach_native_ui(env, home)
    children = []
    def ipc(method, *args):
        return subprocess.check_output(['quickshell', 'ipc', '-p', str(ui / 'Development.qml'),
            'call', 'quatrro-development', method, *args], env=env, text=True,
            stderr=subprocess.DEVNULL, timeout=10).strip()
    def state():
        try: return json.loads(ipc('state'))
        except (subprocess.CalledProcessError, json.JSONDecodeError): return {}
    try:
        with (OUTPUT / 'capture.log').open('w') as log:
            children.append(subprocess.Popen([ROOT / 'build/quatrrod', '--listen='], env=env, stdout=log, stderr=log))
            wait(lambda: (home / 'profile/runtime/control.sock').exists(), 'engine')
            children.append(subprocess.Popen(['quickshell', '-p', str(ui / 'Development.qml')], env=env, stdout=log, stderr=log))
            wait(lambda: state().get('loaded'), 'panel')
            for case, kinds in CASES.items():
                fixture = json.loads((ROOT / 'examples/use-cases' / (case + '.json')).read_text())
                assert all(not f['enabled'] for f in fixture['flows'])
                subprocess.run([ROOT / 'build/quatrroctl', 'config.import', '--stdin'], input=json.dumps(fixture),
                               text=True, env=env, check=True, stdout=subprocess.DEVNULL)
                ipc('tutorialReload')
                wait(lambda: state().get('config', {}).get('flows', [{}])[0].get('id') == fixture['flows'][0]['id'] and not state().get('busy'), case)
                for language in ('en', 'es'):
                    ipc('language', language)
                    wait(lambda: state().get('language') == language and not state().get('busy'), 'language')
                    for kind in kinds:
                        ipc('tutorialEdit', kind)
                        time.sleep(.3)
                        assert not state().get('editorError'), state().get('editorError')
                        image = OUTPUT / f'{case}-{kind}-{language}.png'
                        image.unlink(missing_ok=True)
                        ipc('tutorialCapture', str(image))
                        wait(image.exists, str(image))
                        ipc('dismissEditor')
                print(case, flush=True)
            status = json.loads(subprocess.check_output([ROOT / 'build/quatrroctl', 'status'], env=env))
            assert not status['counts'], status
    finally:
        for child in reversed(children):
            child.terminate()
            try: child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill(); child.wait()
    log = (OUTPUT / 'capture.log').read_text()
    assert not any(message in log for message in ('ERROR:', 'TypeError:', 'ReferenceError:')), log
print(json.dumps({'captures': len(CASES)*4, 'output': str(OUTPUT), 'effects_activated': False}))
