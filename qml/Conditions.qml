pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import qs.Commons as Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts
import "Events.js" as Events

ColumnLayout {
    id: root
    property var value: []
    property string eventSource: ""
    property bool advancedFields: false
    readonly property var suggestedFields: Events.fields(eventSource)
    signal edited(var next)
    function update(i, k, v) {
        var a = JSON.parse(JSON.stringify(value));
        a[i][k] = v;
        edited(a);
    }
    ActionButton {
        visible: root.suggestedFields.length > 0
        text: root.advancedFields ? I18n.tr("Usar campos sugeridos") : I18n.tr("Editar campos avanzados")
        onClicked: root.advancedFields = !root.advancedFields
    }
    Repeater {
        model: root.value.length
        ColumnLayout {
            required property int index
            Layout.fillWidth: true
            ChoiceField {
                accessibleLabel: I18n.tr("Campo del evento")
                model: root.suggestedFields.map(function (h) {
                    return h.field;
                }).concat(root.suggestedFields.some(function (h) {
                    return h.field === root.value[index].field;
                }) ? [] : [root.value[index].field])
                labels: model.map(function (v) {
                    var h = root.suggestedFields.find(function (h) {
                        return h.field === v;
                    });
                    return h ? I18n.tr(h.fieldLabel) : v;
                })
                editable: root.advancedFields || root.suggestedFields.length === 0
                currentIndex: model.indexOf(root.value[index].field)
                editText: root.value[index].field
                onActivated: root.update(index, "field", currentText)
                onTextEdited: function (next) {
                    root.update(index, "field", next);
                }
            }
            ChoiceField {
                accessibleLabel: I18n.tr("Operador de condición")
                model: ["eq", "ne", "gt", "lt", "contains"]
                labels: [I18n.tr("Es igual a"), I18n.tr("Es diferente de"), I18n.tr("Es mayor que"), I18n.tr("Es menor que"), I18n.tr("Contiene")]
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
                field: root.suggestedFields.length ? root.suggestedFields[0].field : "data.state",
                op: "eq",
                value: root.suggestedFields.length && root.suggestedFields[0].type === "number" ? 20 : ""
            }
        ]))
    }
    ThemedLabel {
        text: I18n.tr("Todas las condiciones deben cumplirse. Puedes elegir un campo sugerido o escribir una ruta avanzada.")
        color: Qt.alpha(Commons.Color.foreground, 0.75)
        wrapMode: Text.Wrap
        Layout.fillWidth: true
    }
}
