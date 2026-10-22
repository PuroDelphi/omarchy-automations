pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import qs.Commons as Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    property var value: []
    property var options: []
    property var actions: []
    signal actionRequested(string identifier)
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
                text: {
                    var a = root.actions.find(function (a) {
                        return a.id === modelData;
                    });
                    return (index + 1) + ". " + (a ? (a.title || a.name || a.id) + " · " + a.kind : modelData);
                }
                Layout.fillWidth: true
                Layout.minimumWidth: 0
                wrapMode: Text.Wrap
                color: Commons.Color.foreground
            }
            ActionButton {
                text: I18n.tr("Editar acción")
                onClicked: root.actionRequested(modelData)
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
    ActionButton {
        text: I18n.tr("Crear acción y añadirla")
        onClicked: root.actionRequested("")
    }
    RowLayout {
        ChoiceField {
            id: choice
            Layout.fillWidth: true
            model: root.options
            labels: model.map(function (id) {
                var a = root.actions.find(function (a) {
                    return a.id === id;
                });
                return a ? (a.title || a.name || id) + " · " + a.kind : id;
            })
        }
        ActionButton {
            text: I18n.tr("Añadir paso")
            enabled: choice.currentText !== ""
            onClicked: root.edited(root.value.concat([choice.currentText]))
        }
    }
}
