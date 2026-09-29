pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import "."

ColumnLayout {
    id: root
    property var value: []
    signal edited(var next)
    function update(index, key, next) {
        var list = JSON.parse(JSON.stringify(value));
        if (key === "type") {
            list[index] = {
                name: list[index].name,
                type: next
            };
            if (next === "string")
                list[index].max_length = 256;
            if (next === "integer") {
                list[index].minimum = 0;
                list[index].maximum = 100;
            }
        } else
            list[index][key] = next;
        edited(list);
    }
    Repeater {
        model: root.value.length
        ColumnLayout {
            id: row
            required property int index
            Layout.fillWidth: true
            RowLayout {
                Layout.fillWidth: true
                InputField {
                    Layout.fillWidth: true
                    placeholderText: I18n.tr("Nombre del parámetro")
                    text: root.value[row.index].name
                    onTextEdited: root.update(row.index, "name", text)
                }
                ChoiceField {
                    accessibleLabel: I18n.tr("Tipo de parámetro")
                    model: ["string", "integer", "boolean"]
                    currentIndex: model.indexOf(root.value[row.index].type)
                    onActivated: root.update(row.index, "type", currentText)
                }
                ActionButton {
                    text: I18n.tr("Quitar")
                    accessibleLabel: I18n.tr("Quitar") + " " + root.value[row.index].name
                    onClicked: {
                        var list = root.value.slice();
                        list.splice(row.index, 1);
                        root.edited(list);
                    }
                }
            }
            Repeater {
                model: root.value[row.index].type === "string" ? [
                    {
                        key: "max_length",
                        label: "Longitud máxima (bytes)",
                        type: "number"
                    },
                    {
                        key: "pattern",
                        label: "Patrón RE2 (opcional)"
                    },
                    {
                        key: "choices",
                        label: "Opciones permitidas (una por línea)",
                        type: "lines"
                    }
                ] : root.value[row.index].type === "integer" ? [
                    {
                        key: "minimum",
                        label: "Mínimo",
                        type: "number"
                    },
                    {
                        key: "maximum",
                        label: "Máximo",
                        type: "number"
                    }
                ] : []
                ColumnLayout {
                    id: bound
                    required property var modelData
                    property var current: (root.value[row.index] || {})[modelData.key]
                    Layout.fillWidth: true
                    ThemedLabel {
                        text: I18n.tr(bound.modelData.label)
                    }
                    InputField {
                        accessibleLabel: I18n.tr(bound.modelData.label)
                        visible: bound.modelData.type !== "lines"
                        Layout.fillWidth: true
                        text: bound.current === undefined ? "" : String(bound.current)
                        onTextEdited: root.update(row.index, bound.modelData.key, bound.modelData.type === "number" ? Number(text) : text)
                    }
                    ScrollView {
                        visible: bound.modelData.type === "lines"
                        Layout.fillWidth: true
                        implicitHeight: 70
                        MultiLineField {
                            accessibleLabel: I18n.tr(bound.modelData.label)
                            text: (bound.current instanceof Array ? bound.current : []).join("\n")
                            wrapMode: TextEdit.Wrap
                            onTextChanged: if (activeFocus)
                                root.update(row.index, bound.modelData.key, text.split("\n").filter(function (x) {
                                    return x.length > 0;
                                }))
                        }
                    }
                }
            }
        }
    }
    ActionButton {
        text: I18n.tr("Añadir parámetro")
        enabled: root.value.length < 16
        onClicked: root.edited(root.value.concat([
            {
                name: "parameter" + (root.value.length + 1),
                type: "string",
                max_length: 256
            }
        ]))
    }
}
