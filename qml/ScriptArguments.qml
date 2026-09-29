pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "."

ColumnLayout {
    id: root
    property var script: null
    property var action: ({})
    property int revision: 0
    signal pinRequested
    signal edited(string name, bool binding, var value)
    ThemedLabel {
        Layout.fillWidth: true
        wrapMode: Text.WrapAnywhere
        text: I18n.tr("Revisión seleccionada") + ": " + (root.action.script_revision || "—")
    }
    ThemedLabel {
        Layout.fillWidth: true
        wrapMode: Text.WrapAnywhere
        text: I18n.tr("Revisión disponible") + ": " + (root.script ? root.script.revision : "—")
    }
    ScrollView {
        Layout.fillWidth: true
        implicitHeight: 120
        MultiLineField {
            accessibleLabel: I18n.tr("Código del script")
            text: root.script ? root.script.code : ""
            textFormat: TextEdit.PlainText
            readOnly: true
            selectByMouse: true
            wrapMode: TextEdit.Wrap
        }
    }
    ActionButton {
        text: I18n.tr("Usar esta revisión y reiniciar parámetros")
        enabled: !!root.script
        onClicked: root.pinRequested()
    }
    Repeater {
        model: root.script ? root.script.parameters : []
        ColumnLayout {
            id: row
            required property var modelData
            readonly property bool bound: {
                root.revision;
                return Object.prototype.hasOwnProperty.call(root.action.script_bindings || {}, modelData.name);
            }
            readonly property var current: {
                root.revision;
                return ((bound ? root.action.script_bindings : root.action.script_values) || {})[modelData.name];
            }
            Layout.fillWidth: true
            enabled: !!root.script && root.action.script_revision === root.script.revision
            ThemedLabel {
                text: row.modelData.name + " · " + row.modelData.type
            }
            ChoiceField {
                accessibleLabel: I18n.tr("Origen del parámetro") + " " + row.modelData.name
                model: [I18n.tr("Valor literal"), I18n.tr("Campo del evento")]
                currentIndex: row.bound ? 1 : 0
                onActivated: root.edited(row.modelData.name, currentIndex === 1, currentIndex === 1 ? "data." + row.modelData.name : row.modelData.type === "boolean" ? false : row.modelData.type === "integer" ? row.modelData.minimum : "")
            }
            InputField {
                accessibleLabel: row.modelData.name + " · " + (row.bound ? I18n.tr("Campo del evento") : I18n.tr("Valor literal"))
                visible: row.bound || row.modelData.type !== "boolean"
                Layout.fillWidth: true
                text: row.current === undefined ? "" : String(row.current)
                placeholderText: row.bound ? "data.message" : ""
                selectByMouse: true
                onTextEdited: root.edited(row.modelData.name, row.bound, !row.bound && row.modelData.type === "integer" ? Number(text) : text)
            }
            ToggleField {
                visible: !row.bound && row.modelData.type === "boolean"
                checked: row.current === true
                text: "true"
                accessibleLabel: row.modelData.name
                onClicked: root.edited(row.modelData.name, false, checked)
            }
            ThemedLabel {
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: row.modelData.type === "integer" ? row.modelData.minimum + " … " + row.modelData.maximum : row.modelData.type === "string" ? I18n.tr("Longitud máxima (bytes)") + ": " + row.modelData.max_length + (row.modelData.pattern ? " · " + row.modelData.pattern : "") + (row.modelData.choices && row.modelData.choices.length ? " · " + row.modelData.choices.join(", ") : "") : ""
            }
        }
    }
}
