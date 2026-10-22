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
    property var spec: ({})
    property var value
    property string eventSource: ""
    property bool advancedChoice: false
    property var actions: []
    signal actionRequested(string identifier)
    property bool immutable: false
    signal edited(var next)
    function syncEditableChoice() {
        var choice = editorLoader.item;
        if (choice && spec.type === "editable-choice" && !choice.activeFocus)
            choice.editText = value === undefined ? choice.currentText : String(value);
    }
    function ensureChoice() {
        if (visible && spec.type === "choice" && spec.options && spec.options.length > 0 && spec.options.indexOf(value) < 0)
            edited(spec.options[0]);
    }
    onVisibleChanged: Qt.callLater(ensureChoice)
    onValueChanged: {
        Qt.callLater(ensureChoice);
        Qt.callLater(syncEditableChoice);
    }
    Component.onCompleted: Qt.callLater(ensureChoice)
    spacing: 6
    ThemedLabel {
        text: I18n.tr(root.spec.label || "")
        font.bold: true
        color: Commons.Color.foreground
    }
    ThemedLabel {
        text: I18n.tr(root.spec.hint || "")
        visible: text !== ""
        color: Qt.alpha(Commons.Color.foreground, 0.75)
        font.pixelSize: Style.font.body
        wrapMode: Text.Wrap
        Layout.fillWidth: true
    }
    ActionButton {
        visible: root.spec.type === "editable-choice"
        text: root.advancedChoice ? I18n.tr("Elegir de la lista") : I18n.tr("Escribir un origen avanzado")
        onClicked: root.advancedChoice = !root.advancedChoice
    }
    Loader {
        id: editorLoader
        Layout.fillWidth: true
        onLoaded: Qt.callLater(root.syncEditableChoice)
        sourceComponent: root.spec.type === "script-parameters" ? scriptParametersField : root.spec.type === "bool" ? booleanField : root.spec.type === "choice" || root.spec.type === "editable-choice" ? choiceField : root.spec.type === "conditions" ? conditionsField : root.spec.type === "steps" ? stepsField : root.spec.type === "pairs" ? pairsField : root.spec.type === "multiline" || root.spec.type === "lines" ? multiField : textField
    }
    Component {
        id: scriptParametersField
        ScriptParameters {
            value: root.value || []
            onEdited: function (next) {
                root.edited(next);
            }
        }
    }
    Component {
        id: textField
        InputField {
            objectName: "field-" + root.spec.key
            accessibleLabel: I18n.tr(root.spec.label || "")
            Accessible.description: I18n.tr(root.spec.hint || "")
            width: parent.width
            text: root.value === undefined ? "" : String(root.value)
            readOnly: root.immutable
            selectByMouse: true
            onTextEdited: root.edited(root.spec.type === "number" ? Number(text) : text)
        }
    }
    Component {
        id: multiField
        ScrollView {
            width: parent.width
            implicitHeight: 100
            MultiLineField {
                objectName: "field-" + root.spec.key
                accessibleLabel: I18n.tr(root.spec.label || "")
                Accessible.description: I18n.tr(root.spec.hint || "")
                text: root.spec.type === "lines" ? (root.value || []).join("\n") : (root.value || "")
                selectByMouse: true
                wrapMode: TextEdit.Wrap
                onTextChanged: {
                    if (!activeFocus || !root.visible)
                        return;
                    var next = root.spec.type === "lines" ? text.split("\n").filter(function (x) {
                        return x.length > 0;
                    }) : text;
                    if (JSON.stringify(next) !== JSON.stringify(root.value))
                        root.edited(next);
                }
            }
        }
    }
    Component {
        id: choiceField
        ChoiceField {
            objectName: "field-" + root.spec.key
            accessibleLabel: I18n.tr(root.spec.label || "")
            Accessible.description: I18n.tr(root.spec.hint || "")
            width: parent.width
            model: {
                var options = root.spec.options || [];
                if (root.spec.type === "editable-choice" && root.value && options.indexOf(root.value) < 0)
                    return options.concat([root.value]);
                return options;
            }
            labels: model.map(function (v) {
                return I18n.tr(Events.label(v));
            })
            editable: root.spec.type === "editable-choice" && root.advancedChoice
            currentIndex: Math.max(0, model.indexOf(root.value))
            onActivated: root.edited(currentText)
            onTextEdited: function (next) {
                if (editable)
                    root.edited(next);
            }
            onAccepted: if (editable)
                root.edited(editText)
        }
    }
    Component {
        id: booleanField
        ToggleField {
            objectName: "field-" + root.spec.key
            accessibleLabel: I18n.tr(root.spec.label || "")
            Accessible.description: I18n.tr(root.spec.hint || "")
            checked: root.value === true
            text: checked ? I18n.tr("Sí") : "No"
            onClicked: root.edited(checked)
        }
    }
    Component {
        id: conditionsField
        Conditions {
            eventSource: root.eventSource
            width: parent.width
            value: root.value || []
            onEdited: function (next) {
                root.edited(next);
            }
        }
    }
    Component {
        id: stepsField
        Steps {
            actions: root.actions
            onActionRequested: function (identifier) {
                root.actionRequested(identifier);
            }
            width: parent.width
            value: root.value || []
            options: root.spec.options || []
            onEdited: function (next) {
                root.edited(next);
            }
        }
    }
    Component {
        id: pairsField
        Pairs {
            width: parent.width
            value: root.value || {}
            onEdited: function (next) {
                root.edited(next);
            }
        }
    }
}
