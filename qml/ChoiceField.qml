import QtQuick
import QtQuick.Layouts
import qs.Ui as Native
import qs.Commons

FocusScope {
    id: root
    Accessible.name: root.accessibleLabel
    property string accessibleLabel: ""
    property var model: []
    property int currentIndex: model.length ? 0 : -1
    readonly property string currentText: currentIndex >= 0 && currentIndex < model.length ? String(model[currentIndex]) : ""
    property bool editable: false
    property string editText: currentText
    signal activated(int index)
    signal accepted
    Layout.fillWidth: true
    Layout.minimumWidth: 0
    implicitWidth: Style.spacing.dropdownWidth
    implicitHeight: row.implicitHeight
    opacity: enabled ? 1 : 0.45
    Accessible.role: Accessible.ComboBox
    Accessible.focusable: true
    Accessible.focused: activeFocus
    Accessible.description: currentText
    Accessible.onPressAction: if (enabled)
        dropdown.open()
    RowLayout {
        id: row
        anchors.fill: parent
        spacing: Style.spacing.controlGap
        InputField {
            accessibleLabel: root.accessibleLabel
            visible: root.editable
            Layout.fillWidth: true
            text: root.editText
            onTextEdited: root.editText = text
            onAccepted: root.accepted()
        }
        Native.Dropdown {
            id: dropdown
            Layout.fillWidth: !root.editable
            Layout.preferredWidth: root.editable ? Style.space(130) : implicitWidth
            showLabel: false
            options: root.model.map(function (label, index) {
                return {
                    value: String(index),
                    label: String(label)
                };
            })
            value: String(root.currentIndex)
            onChanged: function (value) {
                root.currentIndex = Number(value);
                root.editText = root.currentText;
                root.activated(root.currentIndex);
                // Native Dropdown owns value while picking; restore the external
                // model binding for later edits or language changes.
                dropdown.value = Qt.binding(function () {
                    return String(root.currentIndex);
                });
            }
        }
    }
}
