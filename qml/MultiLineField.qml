import QtQuick
import QtQuick.Controls
import qs.Ui as Native
import qs.Commons
import qs.Commons as Commons

TextArea {
    id: root
    Accessible.name: root.accessibleLabel
    property string accessibleLabel: placeholderText
    readonly property var controlBorder: Border.controlSpec(activeFocus ? "focus" : (hovered ? "hover-cursor" : "normal"), Commons.Color.foreground, Commons.Color.accent)
    font.family: Style.font.family
    font.pixelSize: Style.font.body
    color: Commons.Color.foreground
    selectionColor: Style.selectionFillFor(Commons.Color.foreground, Commons.Color.accent)
    selectedTextColor: Commons.Color.foreground
    placeholderTextColor: Commons.Color.muted
    leftPadding: Style.spacing.controlPaddingX + Border.left(controlBorder)
    rightPadding: Style.spacing.controlPaddingX + Border.right(controlBorder)
    topPadding: Style.spacing.inputPaddingY + Border.top(controlBorder)
    bottomPadding: Style.spacing.inputPaddingY + Border.bottom(controlBorder)
    opacity: enabled ? 1 : 0.45
    background: Native.BorderSurface {
        color: Style.controlFill(root.activeFocus, root.hovered, Commons.Color.foreground, Commons.Color.accent)
        borderSpec: root.controlBorder
        radius: Style.cornerRadius
    }
}
