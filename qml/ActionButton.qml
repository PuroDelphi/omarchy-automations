import QtQuick
import qs.Ui as Native
import qs.Commons

Native.Button {
    id: root
    // Preserve the panel's selection API while using the actual shell control.
    property bool highlighted: false
    property bool checkable: false
    property bool checked: false
    property bool destructive: false
    focusable: true
    selected: highlighted || checked
    foreground: destructive ? Color.urgent : Color.foreground
    opacity: enabled ? 1 : 0.45
    Accessible.role: Accessible.Button
    property string accessibleLabel: text
    Accessible.name: accessibleLabel
    Accessible.checkable: checkable
    Accessible.checked: checked
    Accessible.onPressAction: if (enabled)
        clicked()
}
