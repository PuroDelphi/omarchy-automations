pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import "."
import QtQuick.Controls
import QtQuick.Layouts
import "Schema.js" as Schema

LocalizedDialog {
    id: root
    property alias directoriesEditor: directories
    property alias scriptArgumentsEditor: scriptArguments
    readonly property bool localCode: kind === "scripts" || kind === "adapters"
    readonly property var selectedAdapter: {
        revision;
        return (config.adapters || []).find(function (a) {
            return a.id === root.draft.adapter;
        }) || null;
    }
    function pinAdapter() {
        if (selectedAdapter) {
            draft.adapter_revision = selectedAdapter.revision;
            revision++;
        }
    }
    property string kind: "entries"
    property var config: ({})
    property var draft: ({})
    property bool existing: false
    property int revision: 0
    property string error: ""
    property string administrationState: "unchecked"
    property bool checkingAdministration: false
    signal administrationCheckRequested(var request, int token)
    function resetAdministrationCheck() {
        administrationState = "unchecked";
        checkingAdministration = false;
    }
    function checkAdministration() {
        if (checkingAdministration || kind !== "actions" || draft.kind !== "system-service")
            return;
        checkingAdministration = true;
        administrationState = "checking";
        administrationCheckRequested({
            unit: draft.unit || "",
            operation: draft.operation || ""
        }, revision);
    }
    function administrationChecked(value, token) {
        if (token !== revision || !visible)
            return;
        checkingAdministration = false;
        administrationState = value.state || "unavailable";
    }
    function administrationText() {
        switch (administrationState) {
        case "checking":
            return I18n.tr("Comprobando permisos administrativos…");
        case "authorized":
            return I18n.tr("Operación autorizada ahora. No se ejecutó; se comprobará de nuevo al activar y ejecutar.");
        case "denied":
            return I18n.tr("La política administrativa o Polkit deniega esta operación.");
        case "unsupported":
            return I18n.tr("El broker instalado no admite esta comprobación. Actualiza el componente administrativo.");
        case "unavailable":
            return I18n.tr("Broker no disponible o respuesta no verificable. Comprueba su instalación y activación.");
        default:
            return I18n.tr("Permisos administrativos sin comprobar. Esta consulta no ejecuta la operación ni concede permisos al flujo.");
        }
    }
    onVisibleChanged: {
        if (!visible) {
            revision++;
            resetAdministrationCheck();
        }
    }
    property bool preparing: false
    property var preparedRevision: null
    readonly property var selectedScript: {
        revision;
        return (config.scripts || []).find(function (s) {
            return s.id === root.draft.script;
        }) || null;
    }
    function pinScript() {
        if (!selectedScript)
            return;
        draft.script_revision = selectedScript.revision;
        draft.script_values = {};
        draft.script_bindings = {};
        selectedScript.parameters.forEach(function (p) {
            draft.script_values[p.name] = p.type === "boolean" ? false : p.type === "integer" ? p.minimum : p.choices && p.choices.length ? p.choices[0] : "";
        });
        revision++;
    }
    function setScriptArgument(name, binding, value) {
        var literals = JSON.parse(JSON.stringify(draft.script_values || {}));
        var bindings = JSON.parse(JSON.stringify(draft.script_bindings || {}));
        delete literals[name];
        delete bindings[name];
        if (binding)
            bindings[name] = value;
        else
            literals[name] = value;
        draft.script_values = literals;
        draft.script_bindings = bindings;
        revision++;
    }
    signal prepareRequested(var request)
    signal saved(var resource)
    function prepare() {
        if (preparing)
            return;
        preparedRevision = null;
        preparing = true;
        error = "";
        if (kind === "adapters") {
            prepareRequested({
                path: draft.path || "",
                manifest: {
                    id: draft.id,
                    protocol: draft.protocol,
                    runtime: draft.runtime,
                    capabilities: draft.capabilities,
                    input_limit: draft.input_limit,
                    output_limit: draft.output_limit,
                    timeout_seconds: draft.timeout_seconds
                }
            });
            return;
        }
        prepareRequested({
            id: draft.id,
            path: draft.path || "",
            interpreter: draft.interpreter,
            parameters: draft.parameters || []
        });
    }
    function prepared(value) {
        if (!preparing)
            return;
        preparing = false;
        if (!localCode)
            return;
        preparedRevision = value;
    }
    width: Math.min(parent.width - 48, 740)
    height: Math.min(parent.height - 48, 700)
    anchors.centerIn: parent
    modal: true
    title: existing ? I18n.tr("Editar recurso") : I18n.tr("Nuevo recurso")
    standardButtons: Dialog.Save | Dialog.Cancel
    function edit(resource) {
        if (preparing)
            return;
        resetAdministrationCheck();
        directories.reset();
        preparedRevision = resource && localCode ? JSON.parse(JSON.stringify(resource)) : null;
        existing = !!resource;
        draft = resource ? JSON.parse(JSON.stringify(resource)) : Schema.defaults(kind);
        if (kind === "monitors" && draft.metric === "journal" && draft.priority === undefined)
            draft.priority = 0;
        revision++;
        error = "";
        open();
    }
    function setField(key, next) {
        if (JSON.stringify(draft[key]) === JSON.stringify(next))
            return;
        if (preparing)
            return;
        preparedRevision = null;
        resetAdministrationCheck();
        draft[key] = next;
        if (key === "adapter")
            draft.adapter_revision = "";
        if (key === "script") {
            draft.script_revision = "";
            draft.script_values = {};
            draft.script_bindings = {};
        }
        revision++;
    }
    onAccepted: {
        if (directories.preparing) {
            error = I18n.tr("Espera a que termine la preparación del directorio.");
            open();
            return;
        }
        if (localCode) {
            if (preparing || !preparedRevision) {
                error = I18n.tr("Prepara y revisa el código antes de guardarlo.");
                open();
                return;
            }
            saved(JSON.parse(JSON.stringify(preparedRevision)));
            return;
        }
        if (!/^[a-z][a-z0-9._-]{0,63}$/.test(draft.id || "")) {
            error = I18n.tr("Usa un identificador en minúsculas, sin espacios, de hasta 64 caracteres.");
            open();
            return;
        }
        var resource = {};
        Schema.fields(kind, config).forEach(function (field) {
            if (Schema.visible(field, root.draft) && root.draft[field.key] !== undefined)
                resource[field.key] = root.draft[field.key];
        });
        if (kind === "actions" && draft.kind === "adapter") {
            if (!selectedAdapter || draft.adapter_revision !== selectedAdapter.revision) {
                error = I18n.tr("Revisa y selecciona la revisión actual del adaptador.");
                open();
                return;
            }
            resource.adapter_revision = draft.adapter_revision;
        }
        if (kind === "actions" && draft.kind === "script") {
            if (!selectedScript || draft.script_revision !== selectedScript.revision) {
                error = I18n.tr("Revisa y selecciona la revisión actual del script.");
                open();
                return;
            }
            resource.script_revision = draft.script_revision;
            resource.script_values = draft.script_values || {};
            resource.script_bindings = draft.script_bindings || {};
        }
        if (kind === "actions" && (draft.kind === "command" || draft.kind === "script")) {
            if (draft.directories !== undefined)
                resource.directories = draft.directories;
            if (draft.working_directory !== undefined)
                resource.working_directory = draft.working_directory;
        }
        saved(JSON.parse(JSON.stringify(resource)));
    }
    contentItem: ScrollView {
        id: scroller
        contentWidth: availableWidth
        clip: true
        ColumnLayout {
            width: scroller.availableWidth
            spacing: 16
            ThemedLabel {
                visible: {
                    root.revision;
                    return root.kind === "actions" && root.draft.kind === "system-service";
                }
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: I18n.tr("Requiere el broker administrativo opcional y una política del administrador para esta unidad y operación. Guardar no instala el broker ni concede privilegios. Un resultado incierto no se reintenta automáticamente.")
                color: Color.accent
            }
            ColumnLayout {
                visible: {
                    root.revision;
                    return root.kind === "actions" && root.draft.kind === "system-service";
                }
                Layout.fillWidth: true
                ActionButton {
                    text: I18n.tr("Comprobar permisos administrativos")
                    enabled: !root.checkingAdministration
                    onClicked: root.checkAdministration()
                }
                ThemedLabel {
                    text: root.administrationText()
                    Layout.fillWidth: true
                    wrapMode: Text.Wrap
                    color: Color.foreground
                }
            }
            ThemedLabel {
                text: root.error
                visible: text !== ""
                color: Color.urgent
                wrapMode: Text.Wrap
                Layout.fillWidth: true
            }
            ThemedLabel {
                visible: root.localCode
                text: I18n.tr("Preparar copia el archivo para revisión. Guardar no ejecuta ni concede permisos.")
                wrapMode: Text.Wrap
                Layout.fillWidth: true
            }
            ThemedLabel {
                visible: root.localCode && !!root.preparedRevision
                text: I18n.tr("Revisión del código") + ": " + (root.preparedRevision ? root.preparedRevision.revision : "")
                wrapMode: Text.WrapAnywhere
                Layout.fillWidth: true
            }
            ScrollView {
                visible: root.localCode && !!root.preparedRevision
                Layout.fillWidth: true
                implicitHeight: 180
                MultiLineField {
                    readOnly: true
                    selectByMouse: true
                    textFormat: TextEdit.PlainText
                    text: root.preparedRevision ? root.preparedRevision.code : ""
                    wrapMode: TextEdit.Wrap
                }
            }
            Repeater {
                model: Schema.fields(root.kind, root.config)
                Field {
                    required property var modelData
                    Layout.fillWidth: true
                    spec: modelData
                    enabled: !root.preparing
                    immutable: root.existing && modelData.key === "id"
                    visible: {
                        root.revision;
                        return Schema.visible(modelData, root.draft);
                    }
                    value: {
                        root.revision;
                        return root.draft[modelData.key];
                    }
                    onEdited: function (next) {
                        root.setField(modelData.key, next);
                    }
                }
            }
            ThemedLabel {
                visible: root.kind === "adapters"
                Layout.fillWidth: true
                wrapMode: Text.Wrap
                text: I18n.tr("Protocolo 1 · event.read: recibe el evento completo · data.write: devuelve datos. Sin red ni directorios del usuario.")
            }
            ColumnLayout {
                visible: {
                    root.revision;
                    return root.kind === "actions" && root.draft.kind === "adapter";
                }
                Layout.fillWidth: true
                ThemedLabel {
                    Layout.fillWidth: true
                    wrapMode: Text.WrapAnywhere
                    text: {
                        root.revision;
                        return I18n.tr("Revisión seleccionada") + ": " + (root.draft.adapter_revision || "—");
                    }
                }
                ThemedLabel {
                    Layout.fillWidth: true
                    wrapMode: Text.WrapAnywhere
                    text: I18n.tr("Revisión disponible") + ": " + (root.selectedAdapter ? root.selectedAdapter.revision : "—")
                }
                ThemedLabel {
                    Layout.fillWidth: true
                    wrapMode: Text.Wrap
                    text: root.selectedAdapter ? "event.read · data.write · " + root.selectedAdapter.input_limit + " / " + root.selectedAdapter.output_limit + " bytes · " + root.selectedAdapter.timeout_seconds + " s" : ""
                }
                ScrollView {
                    Layout.fillWidth: true
                    implicitHeight: 160
                    MultiLineField {
                        text: root.selectedAdapter ? root.selectedAdapter.code : ""
                        readOnly: true
                        selectByMouse: true
                        textFormat: TextEdit.PlainText
                        wrapMode: TextEdit.Wrap
                    }
                }
                ActionButton {
                    text: I18n.tr("Usar esta revisión")
                    enabled: !!root.selectedAdapter
                    onClicked: root.pinAdapter()
                }
            }
            Directories {
                id: directories
                Layout.fillWidth: true
                visible: {
                    root.revision;
                    return root.kind === "actions" && (root.draft.kind === "command" || root.draft.kind === "script");
                }
                value: {
                    root.revision;
                    return root.draft.directories || [];
                }
                workingDirectory: {
                    root.revision;
                    return root.draft.working_directory || "/tmp";
                }
                onEdited: function (next) {
                    root.setField("directories", next);
                }
                onWorkingDirectoryEdited: function (next) {
                    root.setField("working_directory", next);
                }
            }
            ScriptArguments {
                id: scriptArguments
                Layout.fillWidth: true
                visible: {
                    root.revision;
                    return root.kind === "actions" && root.draft.kind === "script";
                }
                script: root.selectedScript
                action: {
                    root.revision;
                    return JSON.parse(JSON.stringify(root.draft));
                }
                revision: root.revision
                onPinRequested: root.pinScript()
                onEdited: function (name, binding, value) {
                    root.setScriptArgument(name, binding, value);
                }
            }
            ActionButton {
                visible: root.localCode
                enabled: !root.preparing
                text: root.preparing ? I18n.tr("Preparando…") : I18n.tr("Preparar revisión")
                onClicked: root.prepare()
            }
        }
    }
}
