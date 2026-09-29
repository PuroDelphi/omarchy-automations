pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    property var value: []
    property var options: []
    signal edited(var next)
    function change(index, delta) {
        var a = value.slice();
        var target = index + delta;
        var old = a[index];
        a[index] = a[target];
        a[target] = old;
        edited(a);
    }
    Repeater {
        model: root.value
        RowLayout {
            required property int index
            required property string modelData
            Layout.fillWidth: true
            ThemedLabel {
                text: (index + 1) + ". " + modelData
                Layout.fillWidth: true
                color: Color.foreground
            }
            ActionButton {
                text: "↑"
                accessibleLabel: I18n.tr("Subir paso") + " " + (index + 1)
                enabled: index > 0
                onClicked: root.change(index, -1)
            }
            ActionButton {
                text: "↓"
                accessibleLabel: I18n.tr("Bajar paso") + " " + (index + 1)
                enabled: index < root.value.length - 1
                onClicked: root.change(index, 1)
            }
            ActionButton {
                text: I18n.tr("Quitar")
                onClicked: {
                    var a = root.value.slice();
                    a.splice(index, 1);
                    root.edited(a);
                }
            }
        }
    }
    RowLayout {
        ChoiceField {
            id: choice
            Layout.fillWidth: true
            model: root.options
        }
        ActionButton {
            text: I18n.tr("Añadir paso")
            enabled: choice.currentText !== ""
            onClicked: root.edited(root.value.concat([choice.currentText]))
        }
    }
}
