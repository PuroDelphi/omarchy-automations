#!/usr/bin/env python3
"""Real Qt input events against the installed native controls and local adapters."""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='omarchy-controls-') as directory:
    base = Path(directory)
    shutil.copytree(ROOT / 'qml', base / 'qml')
    (base / 'tst_controls.qml').write_text('''
import QtQuick
import QtQuick.Controls
import QtTest
import Quickshell
import qs.Commons
import "qml"
FloatingWindow {
    visible: true
    implicitWidth: 640; implicitHeight: 480
TestCase {
    name: "NativeControlContracts"
    width: 640; height: 480; visible: true
    when: true
    InputField { id: input; x: 10; y: 10; width: 300 }
    ChoiceField { id: choice; x: 10; y: 70; width: 300; model: ["First", "Second", "Third"] }
    ChoiceField { id: editable; x: 10; y: 130; width: 400; model: ["Suggested"]; editable: true }
    ToggleField { id: toggle; x: 10; y: 200; width: 300; text: "Enabled" }
    InputField { id: secret; x:10; y:300; width:300; text:"fixture-secret"; echoMode:TextInput.Password; accessibleLabel:"API token" }
    Field { id: dynamicField; x:340; y:200; width:250; spec:({key:"identity",label:"Identificador",type:"text"}); value:"sample" }
    NumericField { id: numeric; x:340; y:320; from:1; to:10; value:5; accessibleLabel:"Event limit" }
    Conditions { id: conditions; x:1000; width:600; value: [{field:"data.state",op:"eq",value:"failed"}] }
    ScriptParameters { id: parameters; x:1000; y:200; width:600; value: [{name:"project",type:"string",max_length:64,choices:["alpha","beta"]}] }
    Pairs { id: pairs; x:1000; y:400; width:600; value: ({"X-Event":"demo"}) }
    ActionButton { id: action; x:10; y:370; text:"Run" }
    SignalSpy { id: actionClicked; target: action; signalName: "clicked" }
    Steps { id: steps; x:1000; y:600; value:["notice","report"]; options:["notice","report"] }
    SignalSpy { id: picked; target: choice; signalName: "activated" }
    SignalSpy { id: toggled; target: toggle; signalName: "toggled" }
    SignalSpy { id: accepted; target: editable; signalName: "accepted" }
    function initTestCase() { wait(250); }
    function test_accessibility() {
        compare(choice.Accessible.role, Accessible.ComboBox);
        compare(toggle.Accessible.role, Accessible.CheckBox);
        compare(secret.Accessible.name, "");
        compare(secret.Accessible.description, "API token");
        verify(secret.Accessible.description.indexOf(secret.text) === -1);
        verify(secret.Accessible.passwordEdit);
        var field = findChild(dynamicField, "field-identity");
        verify(field !== null);
        I18n.language = "en";
        compare(field.Accessible.name, "Identifier");
        I18n.language = "es";
        compare(field.Accessible.name, "Identificador");
        I18n.language = "en";
        toggle.enabled = false;
        toggle.Accessible.toggleAction();
        compare(toggle.checked, false);
        toggle.enabled = true;
        toggle.Accessible.toggleAction();
        compare(toggle.Accessible.checked, true);
        toggle.Accessible.toggleAction();
        compare(toggle.Accessible.checked, false);
        toggled.clear();
        console.log("CONTROL_PASS:accessibility");
    }
    function accessibleNames(item) {
        var names = [item.Accessible.name];
        for (var i=0; i<item.children.length; i++)
            names = names.concat(accessibleNames(item.children[i]));
        return names;
    }
    function test_nested_labels() {
        var cases = [
            {language:"en", conditions:["Event field","Condition operator","Value type","Remove condition 1"], parameters:["Parameter type","Maximum length (bytes)","Allowed choices (one per line)","Remove project"], pairs:["X-Event","Remove X-Event"], steps:["Move step up 1","Move step down 2"]},
            {language:"es", conditions:["Campo del evento","Operador de condición","Tipo de valor","Quitar condición 1"], parameters:["Tipo de parámetro","Longitud máxima (bytes)","Opciones permitidas (una por línea)","Quitar project"], pairs:["X-Event","Quitar X-Event"], steps:["Subir paso 1","Bajar paso 2"]}
        ];
        for (var c=0; c<cases.length; c++) {
            I18n.language = cases[c].language;
            wait(20);
            var groups = [conditions, parameters, pairs, steps];
            var expected = [cases[c].conditions, cases[c].parameters, cases[c].pairs, cases[c].steps];
            for (var g=0; g<groups.length; g++) {
                var names = accessibleNames(groups[g]);
                for (var n=0; n<expected[g].length; n++)
                    verify(names.indexOf(expected[g][n]) >= 0, expected[g][n] + " absent: " + JSON.stringify(names));
            }
        }
        I18n.language = "en";
        console.log("CONTROL_PASS:nested_labels");
    }
    function test_numeric() {
        compare(numeric.field.Accessible.name, "Event limit");
        numeric.field.forceActiveFocus();
        keyClick(Qt.Key_Up);
        compare(numeric.value, 6);
        keyClick(Qt.Key_Down);
        compare(numeric.value, 5);
        numeric.from = 7;
        compare(numeric.value, 7);
        compare(numeric.field.value, 7);
        numeric.value = 99;
        compare(numeric.value, 10);
        compare(numeric.field.value, 10);
        numeric.field.contentItem.forceActiveFocus();
        keyClick(Qt.Key_A, Qt.ControlModifier);
        keyClick(Qt.Key_8); keyClick(Qt.Key_Return);
        compare(numeric.value, 8);
        numeric.enabled = false;
        keyClick(Qt.Key_Up);
        compare(numeric.value, 8);
        console.log("CONTROL_PASS:numeric");
    }
    function test_action_button() {
        action.enabled = true;
        actionClicked.clear();
        action.forceActiveFocus();
        verify(action.activeFocus);
        verify(action._showFocusRing);
        keyClick(Qt.Key_Return); keyClick(Qt.Key_Enter); keyClick(Qt.Key_Space);
        compare(actionClicked.count, 3);
        var originalWidth = action.width;
        var originalHeight = action.height;
        mouseMove(action, action.width / 2, action.height / 2);
        verify(action.hot);
        mousePress(action, action.width / 2, action.height / 2);
        wait(150);
        compare(action.color, Style.pressedFillFor(action.foreground, action.accent));
        mouseRelease(action, action.width / 2, action.height / 2);
        compare(actionClicked.count, 4);
        compare(action.width, originalWidth);
        compare(action.height, originalHeight);
        action.destructive = true;
        compare(action.foreground, Color.urgent);
        action.enabled = false;
        compare(action.opacity, 0.45);
        keyClick(Qt.Key_Return); keyClick(Qt.Key_Space);
        mouseClick(action, action.width / 2, action.height / 2);
        compare(actionClicked.count, 4);
        action.enabled = true;
        action.destructive = false;
        action.highlighted = true;
        verify(action.selected);
        action.highlighted = false;
        action.checkable = true; action.checked = true;
        verify(action.selected);
        verify(action.Accessible.checked);
        action.checked = false;
        console.log("CONTROL_PASS:action_button");
    }
    function test_text_input() {
        mouseClick(input, 25, input.height / 2);
        keyClick(Qt.Key_A); keyClick(Qt.Key_B);
        compare(input.text, "ab"); console.log("CONTROL_PASS:text");
    }
    function test_dropdown_selection() {
        mouseClick(choice, choice.width - 15, choice.height / 2);
        wait(150);
        keyClick(Qt.Key_Down); keyClick(Qt.Key_Return);
        tryCompare(choice, "currentIndex", 1);
        compare(choice.currentText, "Second");
        compare(picked.count, 1);
        choice.currentIndex = 2;
        compare(choice.currentText, "Third"); console.log("CONTROL_PASS:choice");
    }
    function test_editable_choice() {
        mouseClick(editable, 25, editable.height / 2);
        keyClick(Qt.Key_A, Qt.ControlModifier);
        keyClick(Qt.Key_X); keyClick(Qt.Key_Return);
        compare(editable.editText, "x");
        compare(accepted.count, 1); console.log("CONTROL_PASS:editable");
    }
    function test_toggle() {
        mouseClick(toggle, 25, toggle.height / 2);
        compare(toggle.checked, true);
        compare(toggled.count, 1);
        keyClick(Qt.Key_Space);
        compare(toggle.checked, false);
        compare(toggled.count, 2); console.log("CONTROL_PASS:toggle");
    }
}
}
''')
    env = dict(os.environ, QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software',
               QT_QPA_PLATFORMTHEME='', QT_STYLE_OVERRIDE='Fusion')
    attach_native_ui(env, base)
    result = subprocess.run(['quickshell', '-p', str(base / 'tst_controls.qml')],
                            env=env, check=True, timeout=45, capture_output=True, text=True)
    output = result.stdout + result.stderr
    for name in ('text', 'choice', 'editable', 'toggle', 'accessibility', 'numeric', 'nested_labels', 'action_button'):
        assert 'CONTROL_PASS:' + name in output, output
    print('Native controls: real mouse/keyboard input passed for text, choice, editable choice, toggle, numeric input, button states and accessibility contracts')
