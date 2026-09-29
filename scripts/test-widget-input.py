#!/usr/bin/env python3
"""Actual Qt mouse/key events against Widget.qml; backend stub isolates input QA."""
import os
from native_ui import attach_native_ui
from pathlib import Path
import shutil
import subprocess
import tempfile

root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='quatrro-widget-input-') as temp:
    directory=Path(temp)
    shutil.copy2(root/'Widget.qml',directory/'Widget.qml')
    (directory/'qml').mkdir()
    for name in ('I18n.qml','Translations.js'):
        shutil.copy2(root/'qml'/name,directory/'qml'/name)
    (directory/'qml/qmldir').write_text('singleton I18n 1.0 I18n.qml\nBackend 1.0 Backend.qml\n')
    (directory/'qml/Backend.qml').write_text('import QtQuick\nItem { property bool connected: true; property var status: ({paused:false,counts:{}}) }\n')
    (directory/'tst_widget.qml').write_text('''import QtQuick
import QtTest
import Quickshell
import "."
FloatingWindow {
    visible: true
    implicitWidth: 400; implicitHeight: 100
TestCase {
    id: testCase
    name: "WidgetInput"
    when: true
    visible: true
    width: 400; height: 100
    QtObject {
        id: host
        property int calls: 0
        function toggle(name, payload) {
            testCase.compare(name, "quatrro.automations"); testCase.compare(payload, "{}"); calls++;
        }
    }
    Widget { id: widget; shell: host }
    function initTestCase() { wait(250); }
    function init() { host.calls = 0; }
    function test_pointer() {
        mouseClick(widget, widget.width / 2, widget.height / 2, Qt.LeftButton);
        compare(host.calls, 1);
        mouseClick(widget, widget.width / 2, widget.height / 2, Qt.RightButton);
        compare(host.calls, 1); console.log("WIDGET_PASS:pointer");
    }
    function test_keyboard() {
        widget.forceActiveFocus(); verify(widget.activeFocus);
        keyClick(Qt.Key_Space); compare(host.calls, 1);
        keyClick(Qt.Key_Return); compare(host.calls, 2);
        keyClick(Qt.Key_Enter); compare(host.calls, 3);
        verify(widget.activeFocusOnTab); console.log("WIDGET_PASS:keyboard");
    }
}
}
''')
    env=dict(os.environ,QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software',QT_QPA_PLATFORMTHEME='',QT_STYLE_OVERRIDE='Fusion')
    attach_native_ui(env, directory)
    result = subprocess.run(['quickshell','-p',str(directory/'tst_widget.qml')],env=env,check=True,timeout=30,capture_output=True,text=True)
    output = result.stdout + result.stderr
    for name in ('pointer', 'keyboard'):
        assert 'WIDGET_PASS:' + name in output, output
    print('Native widget: mouse and Space/Return/Enter passed')
