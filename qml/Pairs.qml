pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    property var value: ({})
    signal edited(var next)
    Repeater {
        model: Object.keys(root.value)
        RowLayout {
            required property string modelData
            ThemedLabel {
                text: modelData
                color: Color.foreground
                Layout.fillWidth: true
            }
            InputField {
                accessibleLabel: modelData
                text: root.value[modelData]
                Layout.fillWidth: true
                onEditingFinished: {
                    var next = Object.assign({}, root.value);
                    next[modelData] = text;
                    root.edited(next);
                }
            }
            ActionButton {
                text: "×"
                accessibleLabel: I18n.tr("Quitar") + " " + modelData
                onClicked: {
                    var next = Object.assign({}, root.value);
                    delete next[modelData];
                    root.edited(next);
                }
            }
        }
    }
    RowLayout {
        InputField {
            id: key
            placeholderText: I18n.tr("Nombre")
            Layout.fillWidth: true
        }
        InputField {
            id: val
            placeholderText: I18n.tr("Valor")
            Layout.fillWidth: true
        }
        ActionButton {
            text: I18n.tr("Añadir")
            enabled: key.text !== ""
            onClicked: {
                var next = Object.assign({}, root.value);
                next[key.text] = val.text;
                root.edited(next);
                key.clear();
                val.clear();
            }
        }
    }
}
