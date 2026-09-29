import QtQuick
import qs.Ui as Native

FocusScope {
    id: root
    property string accessibleLabel: description ? text + ". " + description : text
    property bool checked: false
    property string text: ""
    property string description: ""
    signal clicked
    signal toggled
    Accessible.role: Accessible.CheckBox
    Accessible.name: root.accessibleLabel
    Accessible.checkable: true
    Accessible.checked: checked
    Accessible.focusable: true
    Accessible.focused: activeFocus
    Accessible.onToggleAction: activate()
    Accessible.onPressAction: activate()
    function activate() {
        if (!enabled)
            return;
        control.forceActiveFocus();
        checked = !checked;
        toggled();
        clicked();
    }
    implicitWidth: control.implicitWidth
    implicitHeight: control.implicitHeight
    opacity: enabled ? 1 : 0.45
    Native.Toggle {
        id: control
        anchors.fill: parent
        label: root.text
        description: root.description
        checked: root.checked
        onClicked: root.activate()
    }
}
