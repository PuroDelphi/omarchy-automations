#!/usr/bin/env python3
"""Host opt-in: real launcher/cgroup recovery, with offscreen test-only memory load."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('clipboard_test', ROOT / 'scripts/test-shell-clipboard.py')
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


def wait(check, seconds=15):
    until = time.monotonic() + seconds
    while time.monotonic() < until:
        result = check()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError('Editor isolation test timed out')


def main():
    assert os.environ.get('QUATRRO_HOST_TEST') == '1', 'Explicit host-test opt-in required'
    with tempfile.TemporaryDirectory(prefix='automations-editor-isolation-') as temporary:
        root = Path(temporary)
        plugin = root / 'plugin'
        shutil.copytree(ROOT / 'qml', plugin / 'qml')
        (plugin / 'scripts').mkdir()
        for name in ('editor.py', 'native_ui.py'):
            shutil.copy2(ROOT / 'scripts' / name, plugin / 'scripts' / name)
        for name in ('Panel.qml', 'Editor.qml', 'EditorWindow.qml'):
            shutil.copy2(ROOT / name, plugin / name)
        panel_source = (plugin / 'Panel.qml').read_text()
        panel_source = panel_source.replace('id: root', '''id: root
    function capture(path) {
        launcherCanvas.grabToImage(function(image) { image.saveToFile(path); });
    }''', 1).replace('ColumnLayout {', 'ColumnLayout {\n            id: launcherCanvas', 1)
        (plugin / 'Panel.qml').write_text(panel_source)
        probe.build_probe(plugin)
        # Add a test-only IPC inside the limited editor, never the production tree.
        window = plugin / 'EditorWindow.qml'
        source = window.read_text().replace('import QtQuick', 'import QtQuick\nimport "ClipboardTest"', 1)
        source = source.replace('ShellRoot {', 'ShellRoot {\n    ClipboardProbe { id: probe }', 1)
        source = source.replace('target: "quatrro-editor"', '''target: "quatrro-editor"
        function loaded(): bool { return editor.loaded; }
        function clipboard(): int { probe.offer(); return probe.readCount(); }
        function exhaust(): void { probe.exhaustMemory(); }''')
        window.write_text(source)
        home = root / 'home'
        (home / '.local/bin').mkdir(parents=True)
        shutil.copy2(ROOT / 'build/quatrroctl', home / '.local/bin/quatrroctl')
        env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(root / 'profile'), QT_QPA_PLATFORM='offscreen',
                   QT_QUICK_BACKEND='software')
        attach_native_ui(env, root)
        harness = plugin / 'Host.qml'
        harness.write_text('''import QtQuick
import Quickshell
import Quickshell.Io
import "qml"
ShellRoot {
    Panel { id: panel }
    Timer { id: heartbeat; interval: 20; running: true; repeat: true
        property int ticks: 0
        onTriggered: ticks++
    }
    IpcHandler {
        target: "host-regression"
        function open(): void { panel.open("{}"); }
        function restart(): void { panel.opened = true; panel.launch(true); }
        function state(): string {
            return JSON.stringify({opened: panel.opened, failed: panel.failed, ticks: heartbeat.ticks});
        }
        function failure(language: string): void { I18n.language = language; panel.failed = true; panel.opened = true; }
        function capture(path: string): void {
            panel.capture(path);
        }
    }
}
''')
        import hashlib
        unit = 'quatrro-automations-editor-' + hashlib.sha256(env['QUATRRO_PROFILE'].encode()).hexdigest()[:16] + '.service'
        def properties():
            output = subprocess.check_output(['/usr/bin/systemctl', '--user', 'show', unit,
                '-p', 'MainPID', '-p', 'ControlGroup', '-p', 'MemoryMax', '-p', 'MemorySwapMax',
                '-p', 'CPUQuotaPerSecUSec', '-p', 'TasksMax'], text=True)
            return dict(line.split('=', 1) for line in output.splitlines())
        def ipc(method):
            return subprocess.check_output(['/usr/bin/quickshell', 'ipc', '-p', str(window),
                'call', 'quatrro-editor', method], env=env, text=True, timeout=3, stderr=subprocess.DEVNULL).strip()
        def open_editor(*args):
            subprocess.run(['/usr/bin/python3', str(plugin / 'scripts/editor.py'), *args],
                           env=env, timeout=20, check=True)
        def host_ipc(method, *args):
            return subprocess.check_output(['/usr/bin/quickshell', 'ipc', '-p', str(harness),
                'call', 'host-regression', method, *args], env=env, text=True, timeout=3, stderr=subprocess.DEVNULL).strip()
        def host_ready():
            try:
                return json.loads(host_ipc('state'))
            except (ValueError, subprocess.SubprocessError):
                return None
        with (root / 'engine.log').open('w') as log:
            engine = subprocess.Popen([ROOT / 'build/quatrrod', '--listen='], env=env, stdout=log, stderr=log)
            host = subprocess.Popen(['/usr/bin/quickshell', '-p', str(harness)], env=env, stdout=log, stderr=log)
            try:
                wait(lambda: (root / 'profile/runtime/control.sock').exists())
                wait(host_ready)
                host_ipc('open')
                wait(lambda: not host_ready()['opened'])
                assert not host_ready()['failed']
                wait(lambda: ipc('loaded') == 'true')
                first = properties()
                assert first['MemoryMax'] == '536870912' and first['MemorySwapMax'] == '0', first
                assert first['TasksMax'] == '128' and first['CPUQuotaPerSecUSec'] == '1s', first
                group = Path('/sys/fs/cgroup') / first['ControlGroup'].lstrip('/')
                assert (group / 'memory.max').read_text().strip() == '536870912'
                assert int(ipc('clipboard')) > 0, 'Qt regression not reproduced inside editor'
                open_editor('--open')
                assert properties()['MainPID'] == first['MainPID'], 'duplicate editor process'
                # Cgroup OOM kills only this test unit. The separate engine remains responsive.
                before = host_ready()['ticks']
                ipc('exhaust')
                wait(lambda: not Path('/proc/' + first['MainPID']).exists())
                assert host_ready()['ticks'] > before, 'host event loop stalled'
                status = subprocess.check_output([ROOT / 'build/quatrroctl', 'status'], env=env, text=True, timeout=3)
                assert isinstance(json.loads(status), dict)
                open_editor('--open')
                recovered = properties()
                assert recovered['MainPID'] != first['MainPID'] and recovered['MainPID'] != '0'
                open_editor('--restart')
                assert properties()['MainPID'] != recovered['MainPID']
                artifacts = ROOT / '.dev'
                artifacts.mkdir(exist_ok=True)
                for language in ('en', 'es'):
                    host_ipc('failure', language)
                    time.sleep(.2)
                    image = artifacts / ('editor-launcher-' + language + '.png')
                    image.unlink(missing_ok=True)
                    host_ipc('capture', str(image))
                    wait(image.exists)
                host_ipc('open')
                wait(lambda: not host_ready()['opened'])
                assert not host_ready()['failed']
                print(json.dumps({'effective_limits': first, 'same_instance_reopened': True,
                                  'editor_memory_exhaustion_contained': True,
                                  'host_heartbeat_continued': True,
                                  'engine_responded_after_editor_exit': True, 'restart_recovered': True}))
            finally:
                subprocess.run(['/usr/bin/systemctl', '--user', 'stop', unit], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                engine.terminate()
                engine.wait(timeout=5)
                shutil.copy2(root / 'engine.log', ROOT / '.dev/editor-isolation.log')
                host.terminate()
                host.wait(timeout=5)


if __name__ == '__main__':
    main()
