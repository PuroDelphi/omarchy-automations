import QtQuick
import qs.Ui as Native

Native.NumberField {
    id: root
    property string accessibleLabel: ""
    opacity: enabled ? 1 : 0.45

    function clampValue() {
        var bounded = Math.max(from, Math.min(to, value));
        if (value !== bounded)
            value = bounded;
    }
    onModified: function (next) {
        value = next;
    }
    onFromChanged: clampValue()
    onToChanged: clampValue()
    onValueChanged: clampValue()
    Component.onCompleted: {
        clampValue();
        field.palette.text = Qt.binding(function () {
            return root.foreground;
        });
        field.palette.buttonText = Qt.binding(function () {
            return root.foreground;
        });
        field.palette.windowText = Qt.binding(function () {
            return root.foreground;
        });
        field.Accessible.name = Qt.binding(function () {
            return root.accessibleLabel;
        });
    }
}
