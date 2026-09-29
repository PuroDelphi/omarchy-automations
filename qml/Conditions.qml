pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    property var value: []
    signal edited(var next)
    function update(i, k, v) {
        var a = JSON.parse(JSON.stringify(value));
        a[i][k] = v;
        edited(a);
    }
    Repeater {
        model: root.value.length
        RowLayout {
            required property int index
            Layout.fillWidth: true
            InputField {
                accessibleLabel: I18n.tr("Campo del evento")
                placeholderText: "data.state"
                text: root.value[index].field
                Layout.fillWidth: true
                onEditingFinished: root.update(index, "field", text)
            }
            ChoiceField {
                accessibleLabel: I18n.tr("Operador de condición")
                model: ["eq", "ne", "gt", "lt", "contains"]
                currentIndex: model.indexOf(root.value[index].op)
                onActivated: root.update(index, "op", currentText)
            }
            InputField {
                placeholderText: I18n.tr("Valor")
                text: String(root.value[index].value)
                Layout.fillWidth: true
                onEditingFinished: root.update(index, "value", type.currentText === I18n.tr("Número") ? Number(text) : type.currentText === I18n.tr("Booleano") ? text === "true" : text)
            }
            ChoiceField {
                id: type
                accessibleLabel: I18n.tr("Tipo de valor")
                model: [I18n.tr("Texto"), I18n.tr("Número"), I18n.tr("Booleano")]
                currentIndex: typeof root.value[index].value === "number" ? 1 : typeof root.value[index].value === "boolean" ? 2 : 0
                onActivated: root.update(index, "value", currentIndex === 1 ? Number(root.value[index].value) : currentIndex === 2 ? String(root.value[index].value) === "true" : String(root.value[index].value))
            }
            ActionButton {
                text: "×"
                accessibleLabel: I18n.tr("Quitar condición") + " " + (index + 1)
                onClicked: {
                    var a = root.value.slice();
                    a.splice(index, 1);
                    root.edited(a);
                }
            }
        }
    }
    ActionButton {
        text: I18n.tr("Añadir condición")
        onClicked: root.edited(root.value.concat([
            {
                field: "data.state",
                op: "eq",
                value: "failed"
            }
        ]))
    }
    ThemedLabel {
        text: I18n.tr("Todas deben cumplirse. eq = igual; ne = diferente; gt/lt = mayor/menor; contains = contiene.")
        color: Qt.alpha(Color.foreground, 0.75)
        wrapMode: Text.Wrap
        Layout.fillWidth: true
    }
}
