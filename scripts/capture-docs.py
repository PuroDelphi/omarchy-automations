#!/usr/bin/env python3
"""Capture the real QML panel with private data and installed native themes."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / '.dev/docs-captures'
OUTPUT.mkdir(parents=True, exist_ok=True)


def wait(check, label):
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        result = check()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError(label)


with tempfile.TemporaryDirectory(prefix='omarchy-doc-captures-') as temporary:
    base = Path(temporary)
    ui = base / 'ui'
    ui.mkdir()
    for name in ('Panel.qml', 'Development.qml'):
        shutil.copy2(ROOT / name, ui / name)
    shutil.copytree(ROOT / 'qml', ui / 'qml')
    panel = (ui / 'Panel.qml').read_text().replace('    id: root\n', '''    id: root
    function documentationBounds() {
        return [eventLimit, byteLimit, retention, dedup].map(function(item) {
            var p = item.mapToItem(window.contentItem, 0, 0);
            return {x:p.x, y:p.y, width:item.width, height:item.height,
                    windowWidth:window.width, windowHeight:window.height};
        });
    }
''', 1)
    (ui / 'Panel.qml').write_text(panel)
    harness = (ui / 'Development.qml').read_text().replace('target: "quatrro-development"',
        'target: "quatrro-development"\n        function documentationBounds(): string { return JSON.stringify(panel.documentationBounds()); }', 1)
    (ui / 'Development.qml').write_text(harness)

    for theme in ('tokyo-night', 'catppuccin-latte'):
        home = base / theme
        current = home / '.local/state/omarchy/current'
        current.mkdir(parents=True)
        (current / 'theme').symlink_to(Path('/usr/share/omarchy/themes') / theme, target_is_directory=True)
        env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(home / 'profile'),
                   PATH=str(ROOT / 'build') + os.pathsep + os.environ['PATH'],
                   QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
        attach_native_ui(env, home)
        children = []
        log_path = OUTPUT / f'{theme}.log'
        def ipc(method, *args):
            return subprocess.check_output(['quickshell', 'ipc', '-p', str(ui / 'Development.qml'),
                                            'call', 'quatrro-development', method, *args],
                                           env=env, text=True, stderr=subprocess.DEVNULL, timeout=10).strip()
        def state():
            try:
                return json.loads(ipc('state'))
            except (subprocess.CalledProcessError, json.JSONDecodeError):
                return {}
        try:
            with log_path.open('w') as log:
                children.append(subprocess.Popen([ROOT / 'build/quatrrod', '--listen='], env=env, stdout=log, stderr=log))
                wait(lambda: (home / 'profile/runtime/control.sock').exists(), 'engine')
                subprocess.run([ROOT / 'build/quatrroctl', 'config.import', '--stdin'],
                               input=(ROOT / 'examples/use-cases/webhook-notification.json').read_bytes(),
                               env=env, check=True, stdout=subprocess.DEVNULL)
                children.append(subprocess.Popen(['quickshell', '-p', str(ui / 'Development.qml')], env=env, stdout=log, stderr=log))
                wait(lambda: state().get('loaded'), 'panel')
                for language in ('en', 'es'):
                    ipc('language', language)
                    wait(lambda: state().get('language') == language and not state().get('busy'), 'language')
                    for section, name in [(0, 'connections'), (5, 'security')]:
                        ipc('section', str(section))
                        time.sleep(.3)
                        wait(lambda: not state().get('busy'), 'section')
                        if section == 5:
                            for rect in json.loads(ipc('documentationBounds')):
                                assert rect['width'] > 0 and rect['height'] > 0, rect
                                assert 0 <= rect['x'] and rect['x'] + rect['width'] <= rect['windowWidth'], rect
                                assert 0 <= rect['y'] and rect['y'] + rect['height'] <= rect['windowHeight'], rect
                        image = OUTPUT / f'{theme}-{language}-{name}.png'
                        image.unlink(missing_ok=True)
                        ipc('capture', str(image))
                        wait(image.exists, 'capture')
        finally:
            for child in reversed(children):
                child.terminate()
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    child.kill()
                    child.wait()
        output = log_path.read_text()
        assert not any(message in output for message in ('ERROR:', 'TypeError:', 'ReferenceError:')), output
    print(json.dumps({'captures': 8, 'output': str(OUTPUT), 'profile': 'temporary',
                      'effects_activated': False, 'ingress': 'disabled', 'host_theme_changed': False}))
