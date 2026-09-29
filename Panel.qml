pragma ComponentBehavior: Bound
import QtQuick
import qs.Commons
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import "qml"

Item {
    id: root
    property var shell: null
    property var manifest: null
    property bool opened: false
    property var config: ({
            version: 1,
            entries: [],
            destinations: [],
            actions: [],
            flows: [],
            monitors: [],
            timers: []
        })
    property bool dirty: false
    property bool loaded: false
    property var activeConfig: null
    readonly property int activeFlowCount: activeConfig ? (activeConfig.flows || []).filter(function(flow) { return flow.enabled; }).length : 0
    readonly property int realTestFlowCount: activeConfig ? (activeConfig.flows || []).filter(function(flow) { return flow.enabled && flow.source === sampleSource.text; }).length : 0
    readonly property string realTestGuidance: !activeConfig ? I18n.tr("Consultando la revisión activa…") : realTestFlowCount === 0 ? I18n.tr("No hay flujos activos para este origen. Guardar o simular no activa el borrador. Pulsa Revisar y activar antes de la prueba real.") : I18n.tr("Flujos activos para este origen: ") + realTestFlowCount + I18n.tr(". La prueba real aplica también sus condiciones y permisos.")
    property int section: 0
    property int connectionTab: 0
    property var history: []
    property bool queueOnly: false
    property int queueOffset: 0
    property int queueTotal: 0
    property bool queueHasMore: false
    property string detailTitle: I18n.tr("Detalle de ejecución")
    property var monitorStates: []
    property var timerStates: []
    property var secrets: []
    property var consentSession: ({})
    property string consentURL: ""
    readonly property bool consentPending: consentSession.state === "waiting" || consentSession.state === "exchanging"
    property var oauthStates: ({})
    property var grants: []
    property var storage: ({})
    property var review: ({})
    property string notice: ""
    property string details: ""
    property var simulationReport: ({})
    readonly property string simulation: simulationText()
    property string afterSave: ""
    property string selectedRun: ""
    property string selectedMonitor: ""
    property string localError: ""
    readonly property string resourceKind: section === 0 ? (connectionTab === 0 ? "entries" : "destinations") : section === 1 ? "actions" : section === 2 ? "flows" : section === 6 ? "timers" : section === 7 ? "scripts" : section === 8 ? "adapters" : "monitors"
    readonly property var sectionNames: [I18n.tr("Conexiones"), I18n.tr("Acciones"), I18n.tr("Flujos"), I18n.tr("Monitores"), I18n.tr("Historial"), I18n.tr("Seguridad"), I18n.tr("Programación"), I18n.tr("Scripts"), I18n.tr("Adaptadores")]
    property alias resourceEditor: editor
    property alias backendClient: backend
    property alias approvalDialog: approval
    property alias credentialDialog: secretDialog
    property alias credentialDeleteDialog: deleteCredential
    property alias stopDialog: cancelDialog
    readonly property bool credentialInputEmpty: secretValue.text === ""
    property alias deliveryRetryDialog: retryDialog
    property alias executionDetailDialog: detailDialog
    property alias realTestDialog: realTest
    property alias simulationDialog: simulationResult
    function setLanguage(language) {
        backend.call("preferences.set", {
            language: language
        });
    }
    function exportDiagnostic(path) {
        fileDialog.diagnostic = true;
        fileDialog.exporting = true;
        filePath.text = path;
        fileDialog.open();
        fileDialog.accept();
    }
    function oauthStateText(value) {
        var labels = {
            "unavailable": "No disponible",
            "renewal_required": "Requiere renovación",
            "available": "Disponible",
            "expired": "Expirado",
            "reconnect_required": "Requiere reconexión",
            "client_rejected": "Cliente rechazado",
            "temporary_error": "Error temporal",
            "invalid_response": "Respuesta inválida",
            "token_rejected": "Token rechazado",
            "scopes_missing": "Faltan permisos de Google"
        };
        return I18n.tr(labels[value.state] || "No disponible") + (value.expires_at ? " · " + value.expires_at : "");
    }
    function checkGoogleStatus(id) {
        backend.call("oauth.google.status", {
            id: id
        });
    }
    function googleConsentFields(id, client, secret, scopes) {
        googleCredentialFields(id, client, secret, "");
        credentialType.currentIndex = 2;
        googleScopes.text = scopes;
    }
    function cancelGoogleConsent() {
        backend.call("oauth.google.cancel", {
            session: consentSession.session
        });
    }
    function consentLabel(state) {
        var labels = {
            interrupted: "Sesión interrumpida; actualiza Seguridad para recuperar su estado",
            waiting: "Esperando consentimiento",
            exchanging: "Conectando con Google",
            completed: "Conexión guardada",
            cancelled: "Autorización cancelada",
            denied: "Consentimiento denegado",
            failed: "No se pudo autorizar",
            credential_changed: "La credencial cambió durante la autorización",
            storage_failed: "No se pudo guardar la conexión",
            scopes_missing: "Faltan permisos de Google"
        };
        return I18n.tr(labels[state] || "No disponible");
    }
    function googleCredentialFields(id, client, secret, refresh) {
        credentialType.currentIndex = 1;
        secretID.text = id;
        googleClient.text = client;
        googleSecret.text = secret;
        googleRefresh.text = refresh;
        secretBackend.currentIndex = 1;
    }
    function togglePause() {
        if (backend.connected && !backend.busy)
            backend.call("control", {
                paused: !backend.status.paused,
                admission: "retain"
            });
    }
    function revokePermission(scope) {
        backend.call("permissions.revoke", {
            scope: scope
        });
    }
    function removeCredential(id) {
        deleteCredential.secretID = id;
        deleteCredential.open();
    }
    function credentialFields(id, value, storage) {
        credentialType.currentIndex = 0;
        secretID.text = id;
        secretValue.text = value;
        secretBackend.currentIndex = storage === "file" ? 1 : 0;
    }
    function open(payload) {
        opened = true;
        refresh();
        if (!loaded)
            backend.call("config.get", {});
    }
    function close() {
        opened = false;
    }
    function capture(path) {
        var item = simulationResult.opened ? simulationResult.contentItem : retryDialog.opened ? retryDialog.contentItem : detailDialog.opened ? detailDialog.contentItem : secretDialog.opened ? secretDialog.contentItem : editor.opened ? editor.contentItem : approval.opened ? approval.contentItem : canvas;
        item.grabToImage(function (result) {
            result.saveToFile(path);
        });
    }
    function inspectExecution(id) {
        backend.call("history.detail", {
            id: id
        });
    }
    function retryDelivery(id) {
        selectedRun = id;
        retryDialog.open();
    }
    function stateLabel(state) {
        return ({
                pending: I18n.tr("Pendiente"),
                running: I18n.tr("En ejecución"),
                completed: I18n.tr("Completado"),
                failed: I18n.tr("Fallido"),
                cancelled: I18n.tr("Cancelado"),
                denied: I18n.tr("Permiso revocado"),
                uncertain: I18n.tr("Resultado incierto"),
                scheduled: I18n.tr("Programado"),
                watching: I18n.tr("Observando"),
                normal: "Normal",
                alert: I18n.tr("Alerta"),
                disabled: I18n.tr("Deshabilitado"),
                initializing: I18n.tr("Preparando"),
                "permission revoked": I18n.tr("Permiso revocado")
            })[state] || state;
    }
    function refresh() {
        backend.call("status", {});
        backend.call("config.active", {});
        if (section === 3)
            backend.call("monitors.status", {});
        if (section === 6)
            backend.call("timers.status", {});
        if (section === 4) {
            if (queueOnly)
                backend.call("queue.inspect", {
                    offset: queueOffset,
                    limit: 100
                });
            else
                backend.call("history", {});
        }
        if (section === 5) {
            backend.call("oauth.google.current", {});
            backend.call("secrets.list", {});
            backend.call("permissions.list", {});
            backend.call("storage.status", {});
        }
    }
    function saveResource(resource) {
        var next = JSON.parse(JSON.stringify(config));
        var list = next[editor.kind] || [];
        next[editor.kind] = list;
        var index = list.findIndex(function (x) {
            return x.id === resource.id;
        });
        if (index < 0)
            list.push(resource);
        else
            list[index] = resource;
        config = next;
        dirty = true;
        notice = I18n.tr("Cambio preparado. Guarda el borrador para validarlo.");
    }
    function removeResource(id) {
        var next = JSON.parse(JSON.stringify(config));
        next[resourceKind] = next[resourceKind].filter(function (x) {
            return x.id !== id;
        });
        config = next;
        dirty = true;
    }
    function save(nextOperation) {
        afterSave = nextOperation || "";
        localError = "";
        backend.call("config.save", config);
    }
    function permissionText() {
        var c = review.config || {};
        var lines = [];
        (c.flows || []).filter(function (f) {
            return f.enabled;
        }).forEach(function (f) {
            lines.push((f.name || f.id) + " · " + f.source);
            f.steps.forEach(function (id) {
                var a = (c.actions || []).find(function (x) {
                    return x.id === id;
                });
                if (!a)
                    return;
                var detail = a.kind === "adapter" ? a.adapter + " · " + a.adapter_revision : a.kind === "script" ? a.script + " · " + a.script_revision + " · " + a.timeout_seconds + " s" : a.kind === "omarchy" ? a.operation : (a.kind === "service" || a.kind === "system-service") ? a.operation + " " + a.unit : a.kind === "command" ? (a.command_profile && a.command_profile !== "fixed" ? a.command_profile + " · " + a.command_path : a.executable + " " + (a.args || []).join(" ")) : a.kind === "http" ? ((c.destinations || []).find(function (d) {
                        return d.id === a.destination;
                    }) || {}).url : a.title;
                lines.push("  • " + a.kind + ": " + detail);
                if (a.kind === "system-service")
                    lines.push("    " + I18n.tr("Servicio de sistema: exige permiso del flujo, política root y Polkit. Sin shell root ni reintentos automáticos."));
                if (a.kind === "http") {
                    var destination = (c.destinations || []).find(function (d) {
                        return d.id === a.destination;
                    });
                    if (destination && (destination.private_hosts || []).length)
                        lines.push("    " + I18n.tr("Excepciones de red privada") + ": " + destination.private_hosts.join(", "));
                    if (destination && destination.auth)
                        lines.push("    " + I18n.tr("Autenticación") + ": " + destination.auth + " · " + I18n.tr("Referencia de credencial") + ": " + destination.secret);
                }
                if (a.kind === "adapter") {
                    var adapter = (c.adapters || []).find(function (x) {
                        return x.id === a.adapter;
                    });
                    if (adapter)
                        lines.push("    event.read · data.write · " + adapter.input_limit + " / " + adapter.output_limit + " bytes · " + adapter.timeout_seconds + " s");
                    lines.push("    " + I18n.tr("El adaptador recibe el evento completo y devuelve data.adapter para los pasos siguientes."));
                }
                (a.directories || []).forEach(function (d) {
                    lines.push("    " + d.source + " → " + d.target + " · " + (d.access === "ro" ? I18n.tr("Solo lectura") : I18n.tr("Lectura y escritura")) + " · " + d.device + ":" + d.inode);
                });
                if ((a.directories || []).length) {
                    lines.push("    " + I18n.tr("Directorio de trabajo") + ": " + (a.working_directory || "/tmp"));
                    lines.push("    " + I18n.tr("El permiso abarca todo el árbol. Escritura permite modificar y borrar archivos. Los sockets dentro del árbol pueden comunicar con servicios del host."));
                }
            });
        });
        (c.monitors || []).filter(function (m) {
            return m.enabled;
        }).forEach(function (m) {
            lines.push("Monitor " + m.id + I18n.tr(": leer ") + m.metric + " " + (m.path || m.unit || (m.destination ? ((c.destinations || []).find(function (d) {
                            return d.id === m.destination;
                        }) || {}).url : "")) + (m.metric === "journal" ? I18n.tr(" · prioridad 0..") + (m.priority === undefined ? 0 : m.priority) : ""));
        });
        (c.timers || []).filter(function (t) {
            return t.enabled;
        }).forEach(function (t) {
            lines.push(I18n.tr("Programación ") + t.id + ": " + (t.kind === "interval" ? I18n.tr("cada ") + t.interval_seconds + " s" : t.at + " · " + t.timezone + " · " + (t.weekdays || []).join(",")) + I18n.tr(" · atrasos: ") + t.missed);
        });
        return lines.length ? lines.join("\n") : I18n.tr("No hay flujos habilitados. Se desactivarán los permisos anteriores.");
    }
    onSectionChanged: refresh()
    property var administrationChecks: []
    function administrationChecked(value) {
        if (!administrationChecks.length)
            return;
        var token = administrationChecks[0];
        administrationChecks = administrationChecks.slice(1);
        editor.administrationChecked(value, token);
    }
    Backend {
        id: backend
        polling: root.opened
        onFailed: function (op, message) {
            if (op === "administration.check")
                root.administrationChecked({
                    state: "unavailable"
                });
            if ((op === "scripts.prepare" || op === "adapters.prepare")) {
                editor.preparing = false;
                editor.error = message;
            }
            if (op === "oauth.google.session" || op === "oauth.google.cancel") {
                root.consentSession = {
                    session: root.consentSession.session,
                    id: root.consentSession.id,
                    state: "interrupted"
                };
                root.consentURL = "";
            }
            if (op === "config.active" || op === "status")
                root.activeConfig = null;
            root.afterSave = "";
            root.localError = message;
        }
        onResult: function (op, value) {
            if (op === "administration.check")
                root.administrationChecked(value);
            if ((op === "scripts.prepare" || op === "adapters.prepare")) {
                editor.prepared(value);
            } else if (op === "config.get") {
                root.config = value;
                root.loaded = true;
                root.dirty = false;
            } else if (op === "config.active") {
                root.activeConfig = value;
            } else if (op === "config.save") {
                root.dirty = false;
                root.notice = I18n.tr("Borrador guardado y validado");
                if (root.afterSave !== "") {
                    var next = root.afterSave;
                    root.afterSave = "";
                    if (next === "simulate")
                        root.runSimulation(false);
                    else
                        backend.call(next, {});
                }
            } else if (op === "config.preview") {
                root.review = value;
                approval.open();
            } else if (op === "config.activate") {
                root.notice = I18n.tr("Revisión activada. Ya puedes ejecutar la prueba real desde Simular / probar.");
                root.refresh();
            } else if (op === "queue.inspect") {
                root.history = value.items;
                root.queueTotal = value.total;
                root.queueHasMore = value.has_more;
            } else if (op === "diagnostics") {
                root.detailTitle = I18n.tr("Diagnóstico sin datos de configuración");
                root.details = JSON.stringify(value, null, 2);
                detailDialog.open();
            } else if (op === "history")
                root.history = value;
            else if (op === "history.detail") {
                root.detailTitle = I18n.tr("Detalle de ejecución");
                root.details = JSON.stringify(value, null, 2);
                detailDialog.open();
            } else if (op === "timers.status")
                root.timerStates = value;
            else if (op === "monitors.status")
                root.monitorStates = value;
            else if (op === "permissions.list")
                root.grants = value;
            else if (op === "oauth.google.current") {
                root.consentSession = {
                    session: value.session || "",
                    id: value.id || "",
                    state: value.state
                };
                root.consentURL = value.authorization_url || "";
            } else if (op === "oauth.google.begin") {
                root.consentSession = {
                    session: value.session,
                    id: value.id,
                    state: value.state
                };
                root.consentURL = value.authorization_url;
                root.section = 5;
            } else if (op === "oauth.google.session" || op === "oauth.google.cancel") {
                root.consentSession = value;
                if (!root.consentPending)
                    root.consentURL = "";
                if (value.state === "completed") {
                    root.oauthStates = ({});
                    root.refresh();
                }
            } else if (op === "oauth.google.status") {
                var states = Object.assign({}, root.oauthStates);
                states[value.id] = value;
                root.oauthStates = states;
            } else if (op === "secrets.list")
                root.secrets = value;
            else if (op === "storage.status")
                root.storage = value;
            else if (op === "simulate") {
                root.simulationReport = value;
                simulationResult.open();
            } else if (op === "config.import-file") {
                backend.call("config.get", {});
                root.notice = I18n.tr("Importado como borrador desactivado; revisa y habilita los recursos antes de activar.");
            } else if (op === "config.export-file" || op === "diagnostics.export-file")
                root.notice = I18n.tr("Exportado a ") + value.path;
            else if (op === "control" || op === "cancel" || op === "permissions.revoke" || op === "delivery.retry" || op === "oauth.google.put" || op === "secrets.put" || op === "secrets.delete" || op === "storage.policy" || op === "monitors.reset") {
                if (op === "oauth.google.put" || op === "secrets.put" || op === "secrets.delete")
                    root.oauthStates = ({});
                root.notice = I18n.tr("Operación completada");
                root.refresh();
            } else if (op === "emit") {
                root.localError = "";
                root.notice = value.duplicate ? I18n.tr("Evento duplicado: no se crearon ejecuciones nuevas.") : value.executions > 0 ? I18n.tr("Ejecuciones creadas: ") + value.executions + I18n.tr(". Historial muestra su progreso; una ejecución completada confirma el resultado de la acción.") : I18n.tr("Ningún flujo activo coincidió. No se crearon ejecuciones ni notificaciones. Revisa origen, condiciones y activación del borrador.");
                sampleDialog.close();
                root.section = 4;
                root.queueOnly = false;
                root.queueOffset = 0;
                root.refresh();
            }
        }
    }

    Timer {
        interval: 1000
        repeat: true
        running: root.consentPending
        onTriggered: {
            if (!backend.busy && backend.queue.length === 0)
                backend.call("oauth.google.session", {
                    session: root.consentSession.session
                });
        }
    }
    Timer {
        interval: 5000
        running: root.opened
        repeat: true
        onTriggered: root.refresh()
    }
    function simulationText() {
        var lines = [I18n.tr("Borrador · no se ejecutaron efectos ni se consultaron credenciales.")];
        (simulationReport.evaluations || []).forEach(function (flow) {
            var reason = !flow.enabled ? I18n.tr("Deshabilitado") : !flow.source_matched ? I18n.tr("El origen no coincide") : flow.matched ? I18n.tr("Coincide") : I18n.tr("Las condiciones no coinciden");
            lines.push("\n" + flow.flow + " · " + reason);
            (flow.conditions || []).forEach(function (condition) {
                lines.push("  " + (condition.matched ? "✓ " : "✗ ") + condition.field + " " + condition.operator + " " + JSON.stringify(condition.expected));
            });
            var match = (simulationReport.matches || []).find(function (item) {
                return item.flow === flow.flow;
            });
            if (match) {
                lines.push(I18n.tr("Parámetros y efectos previstos:"));
                (match.steps || []).forEach(function (step, index) {
                    lines.push((index + 1) + ". " + step.action + " · " + step.kind);
                    if (step.requires_adapter_result)
                        lines.push(I18n.tr("Depende del resultado del adaptador; no se ejecuta durante la simulación."));
                    else
                        lines.push(JSON.stringify(step, null, 2));
                });
            }
        });
        if (!(simulationReport.matches || []).length)
            lines.push("\n" + I18n.tr("Ningún flujo coincide con este evento."));
        return lines.join("\n");
    }
    function ingressText() {
        if (!backend.connected)
            return I18n.tr("Receptor desconocido: motor desconectado.");
        var ingress = backend.status.ingress || {};
        if (ingress.state === "listening")
            return I18n.tr("Receptor local activo: ") + "http://" + ingress.address + "/hooks/" + I18n.tr("identificador") + "\n" + I18n.tr("Exposición pública sin verificar. Requiere configurar un proxy TLS externo.");
        if (ingress.state === "disabled")
            return I18n.tr("Receptor HTTP deshabilitado. Los eventos locales siguen disponibles.");
        if (ingress.state === "starting")
            return I18n.tr("Iniciando receptor HTTP…");
        if (ingress.state === "failed" || ingress.state === "stopped")
            return I18n.tr("Receptor HTTP detenido o no disponible. Comprueba el servicio del motor.");
        return I18n.tr("Estado del receptor no disponible.");
    }
    function eventPayload() {
        return {
            source: sampleSource.text,
            type: sampleType.text,
            data: JSON.parse(sampleData.text)
        };
    }
    function sampleEvent(source, data) {
        sampleSource.text = source;
        sampleData.text = data;
    }
    function runSimulation(saveFirst) {
        try {
            eventPayload();
            if (saveFirst && dirty) {
                save("simulate");
                return;
            }
            backend.call("simulate", eventPayload());
        } catch (e) {
            localError = I18n.tr("Los datos de prueba deben ser un objeto JSON válido");
        }
    }
    FloatingWindow {
        id: window
        visible: root.opened
        title: "Omarchy Automations"
        implicitWidth: 1100
        implicitHeight: 800
        minimumSize: Qt.size(760, 560)
        color: Color.background
        onVisibleChanged: if (!visible)
            root.opened = false
        Rectangle {
            id: canvas
            anchors.fill: parent
            color: Color.background
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 24
                spacing: 16
                GridLayout {
                    id: headerActions
                    columns: width < 640 ? 1 : 2
                    Layout.fillWidth: true
                    Layout.minimumWidth: 0
                    Layout.maximumWidth: parent.width
                    ColumnLayout {
                        Layout.fillWidth: true
                        Layout.minimumWidth: 0
                        ThemedLabel {
                            text: "Omarchy Automations"
                            color: Color.foreground
                            font.pixelSize: Style.font.heading
                            font.bold: true
                        }
                        ThemedLabel {
                            text: !backend.connected ? I18n.tr("Motor desconectado") : (backend.status.paused ? I18n.tr("En pausa") : I18n.tr("Motor activo")) + " · " + (backend.status.version || "") + " · " + (backend.status.counts ? (backend.status.counts.pending || 0) : 0) + I18n.tr(" pendientes")
                            color: backend.connected ? Color.foreground : Color.urgent
                        }
                    }
                    RowLayout {
                        Layout.alignment: Qt.AlignRight
                        ActionButton {
                            text: backend.status.paused ? I18n.tr("Reanudar") : I18n.tr("Pausar")
                            enabled: backend.connected
                            onClicked: root.togglePause()
                        }
                        ActionButton {
                            destructive: true
                            text: I18n.tr("Detener ejecuciones")
                            enabled: backend.connected
                            onClicked: cancelDialog.open()
                        }
                    }
                }
                Flow {
                    id: toolbarActions
                    spacing: Style.spacing.controlGap
                    Layout.fillWidth: true
                    Layout.minimumWidth: 0
                    Layout.maximumWidth: parent.width
                    ActionButton {
                        text: root.dirty ? I18n.tr("Guardar borrador •") : I18n.tr("Guardar borrador")
                        enabled: root.loaded
                        onClicked: root.save("")
                    }
                    ActionButton {
                        text: I18n.tr("Revisar y activar")
                        enabled: root.loaded
                        highlighted: true
                        onClicked: root.save("config.preview")
                    }
                    ActionButton {
                        text: I18n.tr("Simular / probar")
                        enabled: root.loaded
                        onClicked: sampleDialog.open()
                    }
                    ActionButton {
                        text: I18n.tr("Importar")
                        onClicked: {
                            fileDialog.diagnostic = false;
                            fileDialog.exporting = false;
                            fileDialog.open();
                        }
                    }
                    ActionButton {
                        text: I18n.tr("Exportar")
                        enabled: root.loaded && !root.dirty
                        onClicked: {
                            fileDialog.diagnostic = false;
                            fileDialog.exporting = true;
                            fileDialog.open();
                        }
                    }
                }
                ThemedLabel {
                    visible: backend.connected && root.activeConfig !== null
                    text: root.activeFlowCount === 0 ? I18n.tr("Borrador sin activar: no hay flujos activos. Revisa y activa para ejecutar automatizaciones.") : I18n.tr("Flujos activos: ") + root.activeFlowCount + I18n.tr(". Los cambios del borrador requieren Revisar y activar.")
                    color: root.activeFlowCount === 0 ? Color.urgent : Qt.alpha(Color.foreground, 0.75)
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                }
                Rectangle {
                    visible: root.localError !== "" || root.notice !== ""
                    Layout.fillWidth: true
                    implicitHeight: banner.implicitHeight + 20
                    radius: Style.cornerRadius
                    color: root.localError !== "" ? Style.selectedFillFor(Color.urgent, Color.urgent) : Style.selectedFillFor(Color.foreground, Color.accent)
                    ThemedLabel {
                        id: banner
                        anchors.fill: parent
                        anchors.margins: 10
                        text: root.localError || root.notice
                        color: Color.foreground
                        wrapMode: Text.Wrap
                    }
                }
                RowLayout {
                    Layout.fillWidth: true
                    Layout.minimumWidth: 0
                    Layout.maximumWidth: parent.width
                    Layout.fillHeight: true
                    Layout.minimumHeight: 0
                    Layout.maximumHeight: Math.max(0, parent.height - y)
                    spacing: 20
                    ScrollView {
                        id: sidebarScroll
                        function reveal(item) {
                            var flick = contentItem;
                            var top = item.mapToItem(flick.contentItem, 0, 0).y;
                            if (top < flick.contentY)
                                flick.contentY = top;
                            else if (top + item.height > flick.contentY + availableHeight)
                                flick.contentY = top + item.height - availableHeight;
                        }
                        clip: true
                        contentWidth: availableWidth
                        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
                        Layout.minimumHeight: 0
                        Layout.minimumWidth: 155
                        Layout.preferredWidth: 155
                        Layout.maximumWidth: 155
                        Layout.fillHeight: true
                        ColumnLayout {
                            width: sidebarScroll.availableWidth
                            height: Math.max(implicitHeight, sidebarScroll.availableHeight)
                            ChoiceField {
                                id: languageChoice
                                objectName: "language-selector"
                                onActiveFocusChanged: if (activeFocus)
                                    sidebarScroll.reveal(this)
                                model: ["English", "Español"]
                                currentIndex: I18n.language === "es" ? 1 : 0
                                enabled: backend.connected
                                onActivated: root.setLanguage(currentIndex === 1 ? "es" : "en")
                                accessibleLabel: I18n.language === "es" ? "Idioma" : "Language"
                            }
                            Repeater {
                                model: root.sectionNames
                                ActionButton {
                                    required property int index
                                    required property string modelData
                                    text: modelData
                                    Layout.fillWidth: true
                                    checkable: true
                                    checked: root.section === index
                                    onActiveFocusChanged: if (activeFocus)
                                        sidebarScroll.reveal(this)
                                    onClicked: root.section = index
                                }
                            }
                            Item {
                                Layout.fillHeight: true
                            }
                            ThemedLabel {
                                text: I18n.tr("Cerrar el panel no detiene el motor.")
                                color: Qt.alpha(Color.foreground, 0.75)
                                wrapMode: Text.Wrap
                                Layout.fillWidth: true
                                font.pixelSize: Style.font.body
                            }
                            ActionButton {
                                id: sidebarRefresh
                                onActiveFocusChanged: if (activeFocus)
                                    sidebarScroll.reveal(this)
                                text: I18n.tr("Actualizar")
                                Layout.fillWidth: true
                                onClicked: root.refresh()
                            }
                        }
                    }
                    ColumnLayout {
                        Layout.fillWidth: true
                        Layout.minimumWidth: 0
                        Layout.fillHeight: true
                        RowLayout {
                            Layout.fillWidth: true
                            ThemedLabel {
                                text: root.sectionNames[root.section]
                                color: Color.foreground
                                font.pixelSize: Style.font.heading
                                font.bold: true
                                Layout.fillWidth: true
                            }
                            ActionButton {
                                text: I18n.tr("Nuevo")
                                visible: root.section < 4 || root.section === 6 || root.section === 7 || root.section === 8
                                enabled: root.loaded
                                onClicked: {
                                    editor.kind = root.resourceKind;
                                    editor.edit(null);
                                }
                            }
                        }
                        RowLayout {
                            visible: root.section === 0
                            Layout.fillWidth: true
                            ActionButton {
                                text: I18n.tr("Entrada")
                                checked: root.connectionTab === 0
                                onClicked: root.connectionTab = 0
                            }
                            ActionButton {
                                text: I18n.tr("Salida")
                                checked: root.connectionTab === 1
                                onClicked: root.connectionTab = 1
                            }
                            Item {
                                Layout.fillWidth: true
                            }
                        }
                        ThemedLabel {
                            visible: root.section === 0
                            text: root.connectionTab === 0 ? root.ingressText() : I18n.tr("Solo se envía a destinos registrados. Las excepciones LAN se autorizan por destino.")
                            wrapMode: Text.Wrap
                            Layout.fillWidth: true
                            color: Qt.alpha(Color.foreground, 0.75)
                        }
                        ListView {
                            visible: root.section < 4 || root.section === 6 || root.section === 7 || root.section === 8
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            clip: true
                            spacing: 10
                            model: root.config[root.resourceKind] || []
                            ScrollBar.vertical: ScrollBar {}
                            delegate: Rectangle {
                                required property var modelData
                                width: ListView.view.width
                                implicitHeight: resourceRow.implicitHeight + 24
                                radius: 10
                                color: Style.normalFillFor(Color.foreground, Color.accent)
                                RowLayout {
                                    id: resourceRow
                                    anchors.fill: parent
                                    anchors.margins: 12
                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        ThemedLabel {
                                            text: modelData.name || modelData.id
                                            color: Color.foreground
                                            font.bold: true
                                        }
                                        ThemedLabel {
                                            text: modelData.id + " · " + (modelData.kind || modelData.metric || modelData.source || modelData.url || modelData.auth || "")
                                            color: Qt.alpha(Color.foreground, 0.75)
                                            wrapMode: Text.Wrap
                                            Layout.fillWidth: true
                                        }
                                        ThemedLabel {
                                            visible: root.section === 3
                                            text: {
                                                var m = root.monitorStates.find(function (x) {
                                                    return x.id === modelData.id;
                                                });
                                                return m ? (m.error || root.stateLabel(m.phase) + " · " + Number(m.value).toFixed(1) + " " + (m.unit || "%")) : I18n.tr("Aún sin muestras");
                                            }
                                            color: Color.foreground
                                        }
                                        ThemedLabel {
                                            visible: root.section === 6
                                            text: {
                                                var t = root.timerStates.find(function (x) {
                                                    return x.id === modelData.id;
                                                });
                                                return t ? root.stateLabel(t.phase) + I18n.tr(" · próxima: ") + (t.next ? Qt.formatDateTime(new Date(t.next * 1000), "yyyy-MM-dd HH:mm:ss") : "—") : I18n.tr("Aún sin activar");
                                            }
                                            color: Color.foreground
                                            wrapMode: Text.Wrap
                                            Layout.fillWidth: true
                                        }
                                        ThemedLabel {
                                            visible: modelData.enabled !== undefined
                                            text: modelData.enabled ? I18n.tr("Habilitado en borrador") : I18n.tr("Deshabilitado")
                                            color: Qt.alpha(Color.foreground, 0.75)
                                            font.pixelSize: Style.font.bodySmall
                                        }
                                    }
                                    ActionButton {
                                        visible: root.section === 3 && modelData.metric === "journal"
                                        text: I18n.tr("Reiniciar cursor")
                                        onClicked: {
                                            root.selectedMonitor = modelData.id;
                                            resetJournalDialog.open();
                                        }
                                    }
                                    ActionButton {
                                        text: I18n.tr("Editar")
                                        onClicked: {
                                            editor.kind = root.resourceKind;
                                            editor.edit(modelData);
                                        }
                                    }
                                    ActionButton {
                                        text: I18n.tr("Quitar")
                                        onClicked: root.removeResource(modelData.id)
                                    }
                                }
                            }
                        }
                        ColumnLayout {
                            visible: root.section === 4
                            Layout.fillWidth: true
                            Layout.minimumWidth: 0
                            ToggleField {
                                id: historyFilter
                                Layout.fillWidth: true
                                text: I18n.tr("Filtrar historial")
                                description: I18n.tr("Solo cola pendiente / incierta")
                                checked: root.queueOnly
                                onToggled: {
                                    root.queueOnly = checked;
                                    root.queueOffset = 0;
                                    root.refresh();
                                }
                            }
                            ThemedLabel {
                                visible: root.queueOnly
                                text: root.queueTotal + I18n.tr(" trabajos · desde ") + (root.queueOffset + 1)
                                color: Qt.alpha(Color.foreground, 0.75)
                                Layout.fillWidth: true
                            }
                            RowLayout {
                                visible: root.queueOnly
                                Layout.fillWidth: true
                                ActionButton {
                                    visible: root.queueOnly
                                    text: I18n.tr("Anterior")
                                    enabled: root.queueOffset > 0
                                    onClicked: {
                                        root.queueOffset = Math.max(0, root.queueOffset - 100);
                                        root.refresh();
                                    }
                                }
                                ActionButton {
                                    visible: root.queueOnly
                                    text: I18n.tr("Siguiente")
                                    enabled: root.queueHasMore
                                    onClicked: {
                                        root.queueOffset += 100;
                                        root.refresh();
                                    }
                                }
                            }
                        }
                        ListView {
                            visible: root.section === 4
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            clip: true
                            spacing: 10
                            model: root.history
                            ScrollBar.vertical: ScrollBar {}
                            delegate: Rectangle {
                                required property var modelData
                                width: ListView.view.width
                                implicitHeight: historyRow.implicitHeight + 24
                                radius: 10
                                color: Style.normalFillFor(Color.foreground, Color.accent)
                                RowLayout {
                                    id: historyRow
                                    anchors.fill: parent
                                    anchors.margins: 12
                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        ThemedLabel {
                                            text: modelData.flow + " · " + root.stateLabel(modelData.state)
                                            color: Color.foreground
                                            font.bold: true
                                        }
                                        ThemedLabel {
                                            text: new Date(modelData.created_at).toLocaleString() + " · " + (modelData.state === "completed" ? modelData.step + I18n.tr(" pasos realizados") : I18n.tr("paso ") + (modelData.step + 1))
                                            color: Qt.alpha(Color.foreground, 0.75)
                                            font.pixelSize: Style.font.body
                                            wrapMode: Text.Wrap
                                            Layout.fillWidth: true
                                        }
                                    }
                                    ActionButton {
                                        text: I18n.tr("Detalles")
                                        onClicked: root.inspectExecution(modelData.id)
                                    }
                                    ActionButton {
                                        text: I18n.tr("Reenviar HTTP")
                                        visible: modelData.state === "failed" || modelData.state === "cancelled"
                                        onClicked: root.retryDelivery(modelData.id)
                                    }
                                }
                            }
                        }
                        ScrollView {
                            id: securityScroll
                            visible: root.section === 5
                            Layout.fillWidth: true
                            Layout.fillHeight: true
                            contentWidth: availableWidth
                            clip: true
                            ColumnLayout {
                                width: securityScroll.availableWidth
                                spacing: 16
                                RowLayout {
                                    ActionButton {
                                        text: I18n.tr("Ver diagnóstico")
                                        onClicked: backend.call("diagnostics", {})
                                    }
                                    ActionButton {
                                        text: I18n.tr("Exportar diagnóstico")
                                        onClicked: {
                                            fileDialog.diagnostic = true;
                                            fileDialog.exporting = true;
                                            fileDialog.open();
                                        }
                                    }
                                }
                                ThemedLabel {
                                    text: I18n.tr("Credenciales")
                                    color: Color.foreground
                                    font.pixelSize: Style.font.title
                                }
                                ActionButton {
                                    text: I18n.tr("Crear o rotar credencial")
                                    onClicked: secretDialog.open()
                                }
                                ColumnLayout {
                                    visible: !!root.consentSession.session
                                    Layout.fillWidth: true
                                    ThemedLabel {
                                        text: (root.consentSession.id || "") + " · " + root.consentLabel(root.consentSession.state)
                                        color: Qt.alpha(Color.foreground, 0.75)
                                        Layout.fillWidth: true
                                        wrapMode: Text.Wrap
                                    }
                                    RowLayout {
                                        ActionButton {
                                            text: I18n.tr("Abrir consentimiento en el navegador")
                                            enabled: root.consentSession.state === "waiting" && root.consentURL.indexOf("https://accounts.google.com/o/oauth2/v2/auth?") === 0
                                            onClicked: {
                                                if (!Qt.openUrlExternally(root.consentURL))
                                                    root.localError = I18n.tr("No se pudo abrir el navegador");
                                            }
                                        }
                                        ActionButton {
                                            text: I18n.tr("Cancelar autorización")
                                            enabled: root.consentPending
                                            onClicked: root.cancelGoogleConsent()
                                        }
                                    }
                                }
                                Repeater {
                                    model: root.secrets
                                    RowLayout {
                                        required property var modelData
                                        Layout.fillWidth: true
                                        ThemedLabel {
                                            text: modelData.id + " · " + (modelData.backend === "keyring" ? I18n.tr("Almacén del escritorio") : I18n.tr("Archivo privado sin cifrar"))
                                            color: Qt.alpha(Color.foreground, 0.75)
                                            Layout.fillWidth: true
                                        }
                                        ThemedLabel {
                                            text: root.oauthStates[modelData.id] ? root.oauthStateText(root.oauthStates[modelData.id]) : ""
                                            color: Qt.alpha(Color.foreground, 0.75)
                                        }
                                        ActionButton {
                                            text: I18n.tr("Estado OAuth2")
                                            onClicked: backend.call("oauth.google.status", {
                                                id: modelData.id
                                            })
                                        }
                                        ActionButton {
                                            destructive: true
                                            text: I18n.tr("Eliminar")
                                            onClicked: root.removeCredential(modelData.id)
                                        }
                                    }
                                }
                                ThemedLabel {
                                    text: I18n.tr("Permisos activos")
                                    color: Color.foreground
                                    font.pixelSize: Style.font.title
                                }
                                Repeater {
                                    model: root.grants
                                    RowLayout {
                                        required property var modelData
                                        Layout.fillWidth: true
                                        ThemedLabel {
                                            text: modelData.scope
                                            Layout.fillWidth: true
                                            color: Qt.alpha(Color.foreground, 0.75)
                                        }
                                        ActionButton {
                                            destructive: true
                                            text: I18n.tr("Revocar")
                                            onClicked: root.revokePermission(modelData.scope)
                                        }
                                    }
                                }
                                ThemedLabel {
                                    text: I18n.tr("Almacenamiento y retención")
                                    color: Color.foreground
                                    font.pixelSize: Style.font.title
                                }
                                ThemedLabel {
                                    text: (root.storage.events || 0) + I18n.tr(" eventos · ") + ((root.storage.payload_bytes || 0) / 1048576).toFixed(1) + I18n.tr(" MiB de datos")
                                    color: Qt.alpha(Color.foreground, 0.75)
                                }
                                RowLayout {
                                    ThemedLabel {
                                        text: I18n.tr("Máximo de eventos")
                                        color: Qt.alpha(Color.foreground, 0.75)
                                        Layout.fillWidth: true
                                    }
                                    NumericField {
                                        id: eventLimit
                                        accessibleLabel: I18n.tr("Máximo de eventos")
                                        from: 100
                                        to: 100000
                                        value: root.storage.policy ? root.storage.policy.max_events : 10000
                                    }
                                }
                                RowLayout {
                                    ThemedLabel {
                                        text: I18n.tr("Datos de eventos (MiB)")
                                        color: Qt.alpha(Color.foreground, 0.75)
                                        Layout.fillWidth: true
                                    }
                                    NumericField {
                                        id: byteLimit
                                        accessibleLabel: I18n.tr("Datos de eventos (MiB)")
                                        from: 1
                                        to: 128
                                        value: root.storage.policy ? root.storage.policy.max_payload_bytes / 1048576 : 64
                                    }
                                }
                                RowLayout {
                                    ThemedLabel {
                                        text: I18n.tr("Conservar historial (días)")
                                        color: Qt.alpha(Color.foreground, 0.75)
                                        Layout.fillWidth: true
                                    }
                                    NumericField {
                                        id: retention
                                        accessibleLabel: I18n.tr("Conservar historial (días)")
                                        from: 1
                                        to: 90
                                        value: root.storage.policy ? root.storage.policy.retention_days : 7
                                    }
                                }
                                RowLayout {
                                    ThemedLabel {
                                        text: I18n.tr("Evitar duplicados durante (días)")
                                        color: Qt.alpha(Color.foreground, 0.75)
                                        Layout.fillWidth: true
                                    }
                                    NumericField {
                                        id: dedup
                                        accessibleLabel: I18n.tr("Evitar duplicados durante (días)")
                                        from: retention.value
                                        to: 365
                                        value: root.storage.policy ? root.storage.policy.dedup_days : 30
                                    }
                                }
                                ActionButton {
                                    text: I18n.tr("Guardar límites")
                                    onClicked: backend.call("storage.policy", {
                                        max_events: eventLimit.value,
                                        max_payload_bytes: byteLimit.value * 1048576,
                                        retention_days: retention.value,
                                        dedup_days: dedup.value
                                    })
                                }
                                ThemedLabel {
                                    text: I18n.tr("Los trabajos pendientes y los resultados inciertos se conservan. Alcanzar un límite rechaza eventos nuevos; no borra trabajos para hacer espacio.")
                                    color: Qt.alpha(Color.foreground, 0.75)
                                    wrapMode: Text.Wrap
                                    Layout.fillWidth: true
                                }
                            }
                        }
                    }
                }
            }
        }
        ResourceEditor {
            id: editor
            onAdministrationCheckRequested: function (request, token) {
                root.administrationChecks = root.administrationChecks.concat([token]);
                if (!backend.call("administration.check", request)) {
                    root.administrationChecks = root.administrationChecks.slice(0, -1);
                    editor.administrationChecked({
                        state: "unavailable"
                    }, token);
                }
            }
            onPrepareRequested: function (request) {
                if (!backend.call(editor.kind === "adapters" ? "adapters.prepare" : "scripts.prepare", request))
                    preparing = false;
            }
            config: root.config
            onSaved: function (resource) {
                root.saveResource(resource);
            }
        }
        LocalizedDialog {
            id: approval
            acceptText: I18n.tr("Autorizar y activar")
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 700)
            height: Math.min(parent.height - 48, 570)
            modal: true
            title: I18n.tr("Revisar capacidades")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call("config.activate", {
                hash: root.review.hash,
                grants: root.review.capabilities
            })
            contentItem: ColumnLayout {
                ThemedLabel {
                    text: I18n.tr("Activar autoriza estas acciones para los eventos indicados. Los cambios de recursos necesitarán otra revisión.")
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                    color: Color.foreground
                }
                ScrollView {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    MultiLineField {
                        text: root.permissionText()
                        readOnly: true
                        wrapMode: TextEdit.Wrap
                        selectByMouse: true
                    }
                }
            }
        }
        LocalizedDialog {
            id: cancelDialog
            anchors.centerIn: parent
            width: 500
            modal: true
            title: I18n.tr("Detener trabajos")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call("cancel", {})
            ThemedLabel {
                width: parent.width
                text: I18n.tr("Se cancelarán los trabajos pendientes y se intentarán detener los procesos activos. Los efectos ya enviados o realizados no se pueden deshacer.")
                wrapMode: Text.Wrap
                color: Color.foreground
            }
        }
        LocalizedDialog {
            id: retryDialog
            anchors.centerIn: parent
            width: 500
            modal: true
            title: I18n.tr("Reenviar entrega HTTP")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call("delivery.retry", {
                id: root.selectedRun
            })
            contentItem: ThemedLabel {
                width: parent.width
                text: I18n.tr("Se reutilizarán el cuerpo y la clave de idempotencia. El receptor podría repetir efectos si no implementa idempotencia.")
                wrapMode: Text.Wrap
                color: Color.foreground
            }
        }
        LocalizedDialog {
            id: deleteCredential
            property string secretID: ""
            anchors.centerIn: parent
            width: 500
            modal: true
            title: I18n.tr("Eliminar credencial")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call("secrets.delete", {
                id: deleteCredential.secretID
            })
            ThemedLabel {
                width: parent.width
                text: I18n.tr("Las conexiones que usan esta referencia dejarán de autenticarse hasta que registres otra credencial con el mismo identificador.")
                wrapMode: Text.Wrap
                color: Color.foreground
            }
        }
        LocalizedDialog {
            id: secretDialog
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 550)
            modal: true
            title: I18n.tr("Credencial")
            standardButtons: Dialog.Save | Dialog.Cancel
            onAccepted: {
                if (credentialType.currentIndex === 2)
                    backend.call("oauth.google.begin", {
                        id: secretID.text,
                        client_id: googleClient.text,
                        client_secret: googleSecret.text,
                        backend: secretBackend.currentIndex === 0 ? "keyring" : "file",
                        scopes: googleScopes.text.trim().split(/\s+/).filter(function (x) {
                            return x.length > 0;
                        })
                    });
                else if (credentialType.currentIndex === 1)
                    backend.call("oauth.google.put", {
                        id: secretID.text,
                        client_id: googleClient.text,
                        client_secret: googleSecret.text,
                        refresh_token: googleRefresh.text,
                        backend: secretBackend.currentIndex === 0 ? "keyring" : "file"
                    });
                else
                    backend.call("secrets.put", {
                        id: secretID.text,
                        value: secretValue.text,
                        backend: secretBackend.currentIndex === 0 ? "keyring" : "file"
                    });
                secretValue.clear();
                googleClient.clear();
                googleSecret.clear();
                googleRefresh.clear();
            }
            onRejected: {
                secretValue.clear();
                googleClient.clear();
                googleSecret.clear();
                googleRefresh.clear();
            }
            contentItem: ColumnLayout {
                ChoiceField {
                    id: credentialType
                    accessibleLabel: I18n.tr("Tipo de credencial")
                    Layout.fillWidth: true
                    model: [I18n.tr("Credencial genérica"), I18n.tr("Google OAuth2: importar token"), I18n.tr("Google OAuth2: autorizar en navegador")]
                }
                ThemedLabel {
                    visible: credentialType.currentIndex === 1
                    text: I18n.tr("Importa una autorización Google existente. Guardar no concede permisos ni conecta con Google.")
                    Layout.fillWidth: true
                    wrapMode: Text.Wrap
                    color: Qt.alpha(Color.foreground, 0.75)
                }
                ThemedLabel {
                    visible: credentialType.currentIndex === 2
                    text: I18n.tr("Usa un cliente Google de tipo Desktop. Indica los scopes necesarios; revisa sus permisos en el navegador. Guardar inicia una sesión de diez minutos, sin conceder permisos a flujos.")
                    Layout.fillWidth: true
                    wrapMode: Text.Wrap
                    color: Qt.alpha(Color.foreground, 0.75)
                }
                InputField {
                    id: googleScopes
                    accessibleLabel: I18n.tr("Permisos OAuth")
                    visible: credentialType.currentIndex === 2
                    Layout.fillWidth: true
                    placeholderText: "https://www.googleapis.com/auth/calendar.events"
                }
                InputField {
                    id: googleClient
                    accessibleLabel: I18n.tr("Client ID")
                    visible: credentialType.currentIndex > 0
                    Layout.fillWidth: true
                    placeholderText: "Client ID"
                }
                InputField {
                    id: googleSecret
                    accessibleLabel: I18n.tr("Client secret")
                    visible: credentialType.currentIndex > 0
                    Layout.fillWidth: true
                    placeholderText: "Client secret"
                    echoMode: TextInput.Password
                }
                InputField {
                    id: googleRefresh
                    accessibleLabel: I18n.tr("Refresh token")
                    visible: credentialType.currentIndex === 1
                    Layout.fillWidth: true
                    placeholderText: "Refresh token"
                    echoMode: TextInput.Password
                }

                ThemedLabel {
                    text: I18n.tr("Identificador")
                    color: Color.foreground
                }
                InputField {
                    id: secretID
                    accessibleLabel: I18n.tr("Identificador")
                    Layout.fillWidth: true
                    placeholderText: "deploy-secret"
                }
                ThemedLabel {
                    visible: credentialType.currentIndex === 0
                    text: I18n.tr("Valor nuevo (16–8192 bytes)")
                    color: Color.foreground
                }
                InputField {
                    id: secretValue
                    accessibleLabel: I18n.tr("Valor nuevo (16–8192 bytes)")
                    visible: credentialType.currentIndex === 0
                    Layout.fillWidth: true
                    echoMode: TextInput.Password
                }
                ChoiceField {
                    id: secretBackend
                    accessibleLabel: I18n.tr("Almacén de credenciales")
                    Layout.fillWidth: true
                    model: [I18n.tr("Almacén seguro del escritorio"), I18n.tr("Archivo privado (sin cifrar)")]
                }
                ThemedLabel {
                    text: I18n.tr("Usar un identificador existente rota la credencial. Los flujos guardan solo su referencia.")
                    color: Qt.alpha(Color.foreground, 0.75)
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                }
            }
        }
        LocalizedDialog {
            id: sampleDialog
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 650)
            height: Math.min(parent.height - 48, 580)
            modal: true
            title: I18n.tr("Simular un evento")
            standardButtons: Dialog.Close
            contentItem: ColumnLayout {
                ThemedLabel {
                    text: I18n.tr("Origen")
                    color: Color.foreground
                }
                InputField {
                    id: sampleSource
                    accessibleLabel: I18n.tr("Origen")
                    text: "local:demo"
                    Layout.fillWidth: true
                }
                ThemedLabel {
                    text: I18n.tr("Tipo")
                    color: Color.foreground
                }
                InputField {
                    id: sampleType
                    accessibleLabel: I18n.tr("Tipo")
                    text: "test"
                    Layout.fillWidth: true
                }
                ThemedLabel {
                    text: I18n.tr("Datos de ejemplo (JSON)")
                    color: Color.foreground
                }
                ScrollView {
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    MultiLineField {
                        id: sampleData
                        accessibleLabel: I18n.tr("Datos de ejemplo (JSON)")
                        text: JSON.stringify({
                            message: I18n.tr("Hola desde Omarchy Automations"),
                            state: "failed"
                        })
                        selectByMouse: true
                        wrapMode: TextEdit.Wrap
                    }
                }
                RowLayout {
                    ActionButton {
                        text: I18n.tr("Simular sin efectos")
                        highlighted: true
                        onClicked: root.runSimulation(true)
                    }
                    ActionButton {
                        text: I18n.tr("Ejecutar prueba real")
                        enabled: backend.connected && !backend.busy && root.realTestFlowCount > 0 && (sampleSource.text.indexOf("local:") === 0 || sampleSource.text.indexOf("hook:") === 0)
                        onClicked: realTest.open()
                    }
                }
                ThemedLabel {
                    text: root.realTestGuidance
                    color: root.realTestFlowCount > 0 ? Color.foreground : Color.urgent
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                }
                ActionButton {
                    text: I18n.tr("Revisar y activar")
                    enabled: root.loaded && !backend.busy
                    onClicked: { sampleDialog.close(); root.save("config.preview"); }
                }
                ThemedLabel {
                    text: I18n.tr("La simulación usa el borrador. La prueba real usa la revisión activa y sus permisos.")
                    color: Qt.alpha(Color.foreground, 0.75)
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                }
            }
        }
        LocalizedDialog {
            id: realTest
            anchors.centerIn: parent
            width: 500
            modal: true
            title: I18n.tr("Ejecutar evento real")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: {
                try {
                    if (!root.activeConfig || root.realTestFlowCount === 0) {
                        root.localError = root.realTestGuidance;
                        return;
                    }
                    backend.call("emit", root.eventPayload());
                } catch (e) {
                    root.localError = I18n.tr("Datos JSON inválidos");
                }
            }
            ThemedLabel {
                width: parent.width
                text: I18n.tr("Esto puede mostrar notificaciones, ejecutar comandos o enviar datos a destinos autorizados.")
                wrapMode: Text.Wrap
                color: Color.foreground
            }
        }
        LocalizedDialog {
            id: simulationResult
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 720)
            height: Math.min(parent.height - 48, 600)
            modal: true
            title: I18n.tr("Resultado de simulación · sin efectos")
            standardButtons: Dialog.Close
            contentItem: ScrollView {
                MultiLineField {
                    text: root.simulation
                    readOnly: true
                    wrapMode: TextEdit.Wrap
                    selectByMouse: true
                }
            }
        }
        LocalizedDialog {
            id: resetJournalDialog
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 550)
            modal: true
            title: I18n.tr("Reiniciar lectura de journal")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call("monitors.reset", {
                id: root.selectedMonitor
            })
            contentItem: ThemedLabel {
                text: I18n.tr("Se descartará el cursor guardado de ") + root.selectedMonitor + I18n.tr(". La siguiente lectura empezará desde ese momento; los registros anteriores no se recuperarán. Los permisos no cambian.")
                wrapMode: Text.Wrap
                color: Color.foreground
            }
        }
        LocalizedDialog {
            id: detailDialog
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 720)
            height: Math.min(parent.height - 48, 600)
            modal: true
            title: root.detailTitle
            standardButtons: Dialog.Close
            contentItem: ScrollView {
                MultiLineField {
                    text: root.details
                    readOnly: true
                    wrapMode: TextEdit.Wrap
                    selectByMouse: true
                }
            }
        }
        LocalizedDialog {
            id: fileDialog
            property bool exporting: false
            property bool diagnostic: false
            anchors.centerIn: parent
            width: Math.min(parent.width - 48, 600)
            modal: true
            title: diagnostic ? I18n.tr("Exportar diagnóstico") : exporting ? I18n.tr("Exportar configuración") : I18n.tr("Importar configuración")
            standardButtons: Dialog.Ok | Dialog.Cancel
            onAccepted: backend.call(diagnostic ? "diagnostics.export-file" : exporting ? "config.export-file" : "config.import-file", {
                path: filePath.text
            })
            contentItem: ColumnLayout {
                ThemedLabel {
                    text: fileDialog.diagnostic ? I18n.tr("Ruta absoluta de un nuevo archivo .json. Solo versiones, contadores y estado; no incluye configuración, rutas, URLs, cuerpos ni valores de credenciales.") : fileDialog.exporting ? I18n.tr("Ruta absoluta de un nuevo archivo .json (no se sobrescribe). Se exportan referencias, no valores del almacén de secretos.") : I18n.tr("Ruta absoluta de un archivo .json. Sustituye el borrador y deja los recursos importados deshabilitados.")
                    color: Qt.alpha(Color.foreground, 0.75)
                    wrapMode: Text.Wrap
                    Layout.fillWidth: true
                }
                InputField {
                    id: filePath
                    accessibleLabel: I18n.tr("Ruta del archivo JSON")
                    Layout.fillWidth: true
                    placeholderText: "/home/usuario/automatizaciones.json"
                }
            }
        }
    }
}
