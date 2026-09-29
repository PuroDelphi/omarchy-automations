import QtQuick
import qs.Ui as Native

Native.TextField {
    id: root
    property string accessibleLabel: placeholderText
    Accessible.name: root.accessibleLabel
    // Qt masks Accessible.name for password controls; describe only the label.
    Accessible.description: echoMode !== TextInput.Normal ? root.accessibleLabel : ""
    Accessible.passwordEdit: echoMode !== TextInput.Normal
    opacity: enabled ? 1 : 0.45
}
