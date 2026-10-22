#!/usr/bin/env python3
"""Regression: clipboard offers read by Qt fields never reach the host launcher."""
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]


def build_probe(directory):
    module = directory / 'ClipboardTest'
    module.mkdir()
    shutil.copy2(ROOT / 'tests/clipboard_probe.cpp', module / 'clipboard_probe.cpp')
    subprocess.run(['/usr/lib/qt6/moc', str(module / 'clipboard_probe.cpp'), '-o', str(module / 'clipboard_probe.moc')], check=True)
    flags = shlex.split(subprocess.check_output(['pkg-config', '--cflags', '--libs', 'Qt6Qml', 'Qt6Gui'], text=True))
    subprocess.run(['c++', '-shared', '-fPIC', '-std=c++17', '-o', str(module / 'libclipboardprobe.so'),
                    str(module / 'clipboard_probe.cpp'), *flags], check=True)
    (module / 'qmldir').write_text('module ClipboardTest\nplugin clipboardprobe ' + str(module) + '\n')
    return module


def main():
    with tempfile.TemporaryDirectory(prefix='automations-clipboard-') as temporary:
        directory = Path(temporary)
        build_probe(directory)
        shutil.copytree(ROOT / 'qml', directory / 'qml')
        for name in ('Panel.qml', 'Widget.qml'):
            shutil.copy2(ROOT / name, directory / name)
        env = dict(os.environ, QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
        attach_native_ui(env, directory)
        for controls in (True, False):
            # Reproduce Qt's implicit read first, then exercise the real manifest
            # entries, including their startup/failure/retry surface, without inputs.
            component = 'InputField { text: "regression control" }' if controls else '''
                Panel { id: panel; failed: true; opened: true }
                Widget { id: widget }
            '''
            qml = '''import QtQuick
import Quickshell
import "qml"
import "ClipboardTest"
ShellRoot {
    ClipboardProbe { id: probe }
    %s
    Timer {
        interval: 30; running: true; repeat: true
        property int ticks: 0
        onTriggered: {
            probe.offer();
            ticks++;
            if (ticks === 20) {
                console.log("CLIPBOARD_READS:" + probe.readCount());
                Qt.quit();
            }
        }
    }
}
''' % component
            harness = directory / 'shell.qml'
            harness.write_text(qml)
            try:
                result = subprocess.run(['/usr/bin/quickshell', '-p', str(harness)], env=env,
                                        capture_output=True, text=True, timeout=15)
            except subprocess.TimeoutExpired as error:
                raise AssertionError((error.stdout or b'').decode() + (error.stderr or b'').decode()) from error
            output = result.stdout + result.stderr
            assert result.returncode == 0, output
            assert not any(x in output for x in ('ERROR:', 'ReferenceError:', 'TypeError:')), output
            import re
            count = int(re.search(r'CLIPBOARD_READS:(\d+)', output)[1])
            assert count > 0 if controls else count == 0, output
            print(json.dumps({'surface': 'Qt writable control' if controls else 'manifest launcher and widget',
                              'clipboard_offers': 20, 'implicit_content_reads': count}))


if __name__ == '__main__':
    main()
