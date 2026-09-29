import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Commons
import qs.Ui as Native
import "."

Dialog {
    id: root
    font.family: Style.font.family
    font.pixelSize: Style.font.body
    palette.windowText: Color.foreground
    palette.text: Color.foreground
    padding: Style.spacing.lg
    header: ThemedLabel {
        text: root.title
        visible: text !== ""
        font.pixelSize: Style.font.heading
        font.bold: true
        wrapMode: Text.Wrap
        padding: Style.spacing.lg
        bottomPadding: Style.spacing.controlGap
    }
    background: Native.BorderSurface {
        color: Color.popups.background
        radius: Style.cornerRadius
        borderSpec: Border.surfaceSpec("popups", "border", Color.popups.border, Style.normalBorderWidth)
    }
    footer: RowLayout {
        function standardButton(button) {
            if (!(root.standardButtons & button))
                return null;
            if (button === Dialog.Cancel)
                return cancelButton;
            if (button === Dialog.Close)
                return closeButton;
            if (button === Dialog.Save)
                return saveButton;
            if (button === Dialog.Ok)
                return acceptButton;
            return null;
        }
        spacing: Style.spacing.controlGap
        visible: root.standardButtons !== Dialog.NoButton
        Item {
            Layout.fillWidth: true
        }
        ActionButton {
            id: cancelButton
            visible: (root.standardButtons & Dialog.Cancel) !== 0
            text: I18n.language === "es" ? "Cancelar" : "Cancel"
            onClicked: root.reject()
        }
        ActionButton {
            id: closeButton
            visible: (root.standardButtons & Dialog.Close) !== 0
            text: I18n.language === "es" ? "Cerrar" : "Close"
            onClicked: root.reject()
        }
        ActionButton {
            id: saveButton
            visible: (root.standardButtons & Dialog.Save) !== 0
            text: I18n.language === "es" ? "Guardar" : "Save"
            highlighted: true
            onClicked: root.accept()
        }
        ActionButton {
            id: acceptButton
            visible: (root.standardButtons & Dialog.Ok) !== 0
            text: I18n.language === "es" ? "Aceptar" : "OK"
            highlighted: true
            onClicked: root.accept()
        }
    }
}
