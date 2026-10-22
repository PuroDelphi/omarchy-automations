import QtQuick
import QtQuick.Layouts
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Commons as Commons
import "qml"

// This entry point runs in Omarchy's shared shell. Never load Editor.qml or
// writable Qt text controls here: Qt probes clipboard contents automatically.
Item {
    id: root
    property var shell: null
    property var manifest: null
    property bool opened: false
    property bool failed: false
    readonly property string launcherPath: decodeURIComponent(Qt.resolvedUrl("scripts/editor.py").toString().replace(/^file:\/\//, ""))

    function open(payload) {
        opened = true;
        launch(false);
    }
    function close() {
        opened = false;
    }
    function launch(restart) {
        if (launcher.running)
            return;
        failed = false;
        launcher.command = ["/usr/bin/python3", root.launcherPath, restart ? "--restart" : "--open"];
        launcher.running = true;
    }
    Process {
        id: launcher
        onExited: function (code, status) {
            root.failed = code !== 0;
            if (!root.failed)
                root.close();
        }
    }
    FloatingWindow {
        visible: root.opened
        title: "Omarchy Automations"
        implicitWidth: 480
        implicitHeight: 230
        color: Commons.Color.background
        onVisibleChanged: if (!visible)
            root.close()
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: Style.spacing.controlPaddingX * 2
            spacing: Style.spacing.controlGap
            ThemedLabel {
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: root.failed ? I18n.tr("No se pudo abrir el editor. Reintenta o reinícialo. Reiniciar descarta los cambios sin guardar del editor.") : I18n.tr("Abriendo automatizaciones…")
            }
            Item {
                Layout.fillHeight: true
            }
            RowLayout {
                ActionButton {
                    visible: root.failed
                    text: I18n.tr("Reintentar")
                    onClicked: root.launch(false)
                }
                ActionButton {
                    visible: root.failed
                    text: I18n.tr("Reiniciar editor")
                    onClicked: root.launch(true)
                }
                ActionButton {
                    text: I18n.tr("Cerrar")
                    onClicked: root.close()
                }
            }
        }
    }
}
