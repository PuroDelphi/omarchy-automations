pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "."

ColumnLayout {
    id: root
    property var value: []
    property string workingDirectory: "/tmp"
    property bool preparing: false
    property int generation: 0
    property int requestGeneration: 0
    property string error: ""
    signal edited(var next)
    signal workingDirectoryEdited(string next)
    function reset() {
        generation++;
        error = "";
        source.text = "";
        target.text = "/work/data";
        access.currentIndex = 0;
    }
    function prepare(sourcePath, mountTarget, mode) {
        if (preparing)
            return;
        error = "";
        requestGeneration = generation;
        preparing = backend.call("directories.prepare", {
            source: sourcePath,
            target: mountTarget,
            access: mode
        });
    }
    Backend {
        id: backend
        polling: false
        onResult: function (op, directory) {
            root.preparing = false;
            if (root.requestGeneration !== root.generation)
                return;
            var list = JSON.parse(JSON.stringify(root.value));
            var index = list.findIndex(function (d) {
                return d.target === directory.target;
            });
            if (index < 0) {
                if (list.length >= 8) {
                    root.error = I18n.tr("Máximo ocho directorios por acción.");
                    return;
                }
                list.push(directory);
            } else
                list[index] = directory;
            root.edited(list);
        }
        onFailed: function (op, message) {
            root.preparing = false;
            if (root.requestGeneration === root.generation)
                root.error = message;
        }
    }
    ThemedLabel {
        text: I18n.tr("Directorios autorizados")
        font.bold: true
    }
    ThemedLabel {
        Layout.fillWidth: true
        wrapMode: Text.Wrap
        text: I18n.tr("El permiso abarca todo el árbol. Escritura permite modificar y borrar archivos. Los sockets dentro del árbol pueden comunicar con servicios del host.")
    }
    ThemedLabel {
        visible: text !== ""
        text: root.error
        wrapMode: Text.Wrap
        Layout.fillWidth: true
    }
    Repeater {
        model: root.value
        RowLayout {
            required property var modelData
            Layout.fillWidth: true
            ThemedLabel {
                Layout.fillWidth: true
                wrapMode: Text.WrapAnywhere
                text: modelData.source + " → " + modelData.target + " · " + (modelData.access === "ro" ? I18n.tr("Solo lectura") : I18n.tr("Lectura y escritura")) + "\n" + modelData.device + ":" + modelData.inode
            }
            ActionButton {
                text: I18n.tr("Quitar")
                accessibleLabel: I18n.tr("Quitar") + " " + parent.modelData.target
                enabled: !root.preparing
                onClicked: {
                    var mountTarget = parent.modelData.target;
                    root.edited(root.value.filter(function (d) {
                        return d.target !== mountTarget;
                    }));
                    if (root.workingDirectory === mountTarget)
                        root.workingDirectoryEdited("/tmp");
                }
            }
        }
    }
    ThemedLabel {
        text: I18n.tr("Directorio del host")
    }
    InputField {
        id: source
        accessibleLabel: I18n.tr("Directorio del host")
        Layout.fillWidth: true
        enabled: !root.preparing
        selectByMouse: true
    }
    ThemedLabel {
        text: I18n.tr("Destino dentro del sandbox")
    }
    InputField {
        id: target
        accessibleLabel: I18n.tr("Destino dentro del sandbox")
        Layout.fillWidth: true
        text: "/work/data"
        enabled: !root.preparing
    }
    ChoiceField {
        id: access
        accessibleLabel: I18n.tr("Acceso al directorio")
        enabled: !root.preparing
        model: [I18n.tr("Solo lectura"), I18n.tr("Lectura y escritura")]
    }
    ActionButton {
        text: root.preparing ? I18n.tr("Preparando…") : I18n.tr("Preparar y añadir directorio")
        enabled: !root.preparing
        onClicked: root.prepare(source.text, target.text, access.currentIndex === 0 ? "ro" : "rw")
    }
    ThemedLabel {
        text: I18n.tr("Preparar verifica la identidad; no concede permisos. Reutilizar un destino reemplaza su entrada en el borrador.")
        Layout.fillWidth: true
        wrapMode: Text.Wrap
    }
    ThemedLabel {
        text: I18n.tr("Directorio de trabajo")
    }
    ChoiceField {
        Layout.fillWidth: true
        accessibleLabel: I18n.tr("Directorio de trabajo")
        model: ["/tmp"].concat(root.value.map(function (d) {
            return d.target;
        }))
        currentIndex: model.indexOf(root.workingDirectory || "/tmp")
        onActivated: root.workingDirectoryEdited(currentText)
    }
}
