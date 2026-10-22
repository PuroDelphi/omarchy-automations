pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Commons
import qs.Commons as Commons
import "."
import "Events.js" as Events

LocalizedDialog {
    id: root
    property var adapters: []
    property bool busy: false
    property string error: ""
    signal requested(string operation, var data)
    anchors.centerIn: parent
    width: Math.min(parent.width - 48, 700)
    height: Math.min(parent.height - 48, 650)
    title: I18n.tr("Eventos de Omarchy")
    modal: true
    standardButtons: Dialog.Close
    function receive(value) {
        adapters = value;
        busy = false;
        error = "";
    }
    function request(operation, name) {
        busy = true;
        error = "";
        requested(operation, {
            name: name
        });
    }
    onOpened: request("hooks.status", "")
    contentItem: ScrollView {
        contentWidth: availableWidth
        clip: true
        ColumnLayout {
            width: parent.width
            spacing: 16
            ThemedLabel {
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: I18n.tr("Conecta los eventos del escritorio con tus flujos. Instalar una conexión no ejecuta acciones. Solo se retiran archivos propios sin modificaciones.")
            }
            ThemedLabel {
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: root.error
                visible: text !== ""
                color: Commons.Color.urgent
            }
            Repeater {
                model: root.adapters
                ColumnLayout {
                    required property var modelData
                    Layout.fillWidth: true
                    ThemedLabel {
                        text: I18n.tr(Events.label("hook:" + modelData.name))
                        font.bold: true
                    }
                    ThemedLabel {
                        Layout.fillWidth: true
                        wrapMode: Text.Wrap
                        text: modelData.state === "installed" ? I18n.tr("Conectado") : modelData.state === "conflict" ? I18n.tr("Archivo modificado o ruta no segura. Revisión manual necesaria.") : I18n.tr("Sin conectar")
                    }
                    ActionButton {
                        text: modelData.state === "installed" ? I18n.tr("Desconectar evento") : I18n.tr("Conectar evento")
                        enabled: !root.busy && modelData.state !== "conflict"
                        onClicked: root.request(modelData.state === "installed" ? "hooks.remove" : "hooks.install", modelData.name)
                    }
                }
            }
            ActionButton {
                text: I18n.tr("Actualizar")
                enabled: !root.busy
                onClicked: root.request("hooks.status", "")
            }
        }
    }
}
