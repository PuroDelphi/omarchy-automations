import QtQuick
import QtQuick.Controls
import "qml"
import Quickshell
import Quickshell.Io

ShellRoot {
    Panel {
        id: panel
        Component.onCompleted: open("{}")
    }
    Timer {
        interval: 4000
        running: !!Quickshell.env("QUATRRO_CAPTURE")
        onTriggered: panel.capture(Quickshell.env("QUATRRO_CAPTURE"))
    }
    // Local-only UI integration harness, loaded only through Development.qml.
    // Production manifest never loads this file.
    IpcHandler {
        target: "quatrro-development"
        function state(): string {
            return JSON.stringify({
                loaded: panel.loaded,
                oauthStates: panel.oauthStates,
                consentSession: panel.consentSession,
                consentURL: panel.consentURL,
                language: I18n.language,
                sections: panel.sectionNames,
                sample: panel.eventPayload(),
                editorTitle: panel.resourceEditor.title,
                preparedScript: panel.resourceEditor.preparedRevision,
                editorError: panel.resourceEditor.error,
                administrationState: panel.resourceEditor.administrationState,
                administrationText: panel.resourceEditor.administrationText(),
                directoryValues: panel.resourceEditor.directoriesEditor.value,
                directoriesVisible: panel.resourceEditor.directoriesEditor.visible,
                directoryError: panel.resourceEditor.directoriesEditor.error,
                scriptArgumentsVisible: panel.resourceEditor.scriptArgumentsEditor.visible,
                scriptArgumentsAction: panel.resourceEditor.scriptArgumentsEditor.action,
                saveLabel: panel.resourceEditor.footer.standardButton(Dialog.Save).text,
                dirty: panel.dirty,
                connected: panel.backendClient.connected,
                paused: !!panel.backendClient.status.paused,
                stopVisible: panel.stopDialog.opened,
                stopTitle: panel.stopDialog.title,
                ingressText: panel.ingressText(),
                busy: panel.backendClient.busy,
                queued: panel.backendClient.queue.length,
                config: panel.config,
                error: panel.localError,
                notice: panel.notice,
                review: panel.review,
                permissionText: panel.permissionText(),
                history: panel.history,
                monitors: panel.monitorStates,
                timers: panel.timerStates,
                queueTotal: panel.queueTotal,
                details: panel.details,
                retryVisible: panel.deliveryRetryDialog.opened,
                retryTitle: panel.deliveryRetryDialog.title,
                credentials: panel.secrets,
                grants: panel.grants,
                credentialInputEmpty: panel.credentialInputEmpty,
                credentialDeleteVisible: panel.credentialDeleteDialog.opened,
                credentialDeleteTitle: panel.credentialDeleteDialog.title,
                simulation: panel.simulation,
                simulationReport: panel.simulationReport,
                realTestTitle: panel.realTestDialog.title,
                realTestVisible: panel.realTestDialog.opened
            });
        }
        function resource(kind: string, fields: string): string {
            if (["entries", "destinations", "actions", "flows", "monitors", "timers"].indexOf(kind) < 0)
                return "invalid kind";
            panel.resourceEditor.kind = kind;
            panel.resourceEditor.edit(null);
            var values = JSON.parse(fields);
            Object.keys(values).forEach(function (key) {
                panel.resourceEditor.setField(key, values[key]);
            });
            panel.resourceEditor.accept();
            return "ok";
        }
        function scriptAction(): void {
            panel.resourceEditor.kind = "actions";
            panel.resourceEditor.edit(null);
            panel.resourceEditor.setField("id", "run-script");
            panel.resourceEditor.setField("kind", "script");
            panel.resourceEditor.setField("script", "local-script");
            panel.resourceEditor.pinScript();
            panel.resourceEditor.setScriptArgument("message", true, "data.message");
            panel.resourceEditor.setScriptArgument("copies", false, 2);
            panel.resourceEditor.setScriptArgument("enabled", false, true);
        }
        function prepareAdapter(path: string): void {
            panel.resourceEditor.kind = "adapters";
            panel.resourceEditor.edit(null);
            panel.resourceEditor.setField("id", "normalizer");
            panel.resourceEditor.setField("path", path);
            panel.resourceEditor.prepare();
        }
        function adapterAction(pin: bool): void {
            panel.resourceEditor.kind = "actions";
            panel.resourceEditor.edit(null);
            panel.resourceEditor.setField("id", "normalize");
            panel.resourceEditor.setField("kind", "adapter");
            panel.resourceEditor.setField("adapter", "normalizer");
            if (pin)
                panel.resourceEditor.pinAdapter();
        }
        function prepareScript(fields: string): void {
            panel.resourceEditor.kind = "scripts";
            panel.resourceEditor.edit(null);
            var values = JSON.parse(fields);
            Object.keys(values).forEach(function (key) {
                panel.resourceEditor.setField(key, values[key]);
            });
            panel.resourceEditor.prepare();
        }
        function prepareDirectory(source: string): void {
            panel.resourceEditor.directoriesEditor.prepare(source, "/work/data", "rw");
        }
        function editSavedScriptAction(): void {
            panel.resourceEditor.kind = "actions";
            panel.resourceEditor.edit(panel.config.actions.find(function (a) {
                return a.id === "run-script";
            }));
        }
        function directoryCwd(): void {
            panel.resourceEditor.directoriesEditor.workingDirectoryEdited("/work/data");
        }
        function staleScriptRevision(): void {
            panel.resourceEditor.setField("script_revision", "stale");
        }
        function acceptEditor(): void {
            panel.resourceEditor.accept();
        }
        function openEditor(kind: string): void {
            panel.resourceEditor.kind = kind;
            panel.resourceEditor.edit(null);
        }
        function language(value: string): void {
            panel.setLanguage(value);
        }
        function dismissEditor(): void {
            panel.resourceEditor.reject();
        }
        function save(): void {
            panel.save("");
        }
        function review(): void {
            panel.save("config.preview");
        }
        function activate(): void {
            panel.approvalDialog.accept();
        }
        function staleAdministrationResult(): void {
            var token = panel.resourceEditor.revision;
            panel.resourceEditor.setField("unit", "changed.service");
            panel.resourceEditor.administrationChecked({
                state: "authorized"
            }, token);
        }
        function adminOperation(value: string): void {
            panel.resourceEditor.setField("operation", value);
        }
        function adminUnit(value: string): void {
            panel.resourceEditor.setField("unit", value);
        }
        function checkAdministration(): void {
            panel.resourceEditor.checkAdministration();
        }
        function adminAction(): void {
            panel.resourceEditor.kind = "actions";
            panel.resourceEditor.edit(null);
            panel.resourceEditor.setField("id", "admin-status");
            panel.resourceEditor.setField("kind", "system-service");
            panel.resourceEditor.setField("unit", "quatrro-fixture.service");
            panel.resourceEditor.setField("operation", "status");
        }
        function googleConsent(id: string, client: string, secret: string, scopes: string): void {
            panel.googleConsentFields(id, client, secret, scopes);
            panel.credentialDialog.open();
        }
        function recoverGoogleConsent(): void {
            panel.consentSession = ({});
            panel.consentURL = "";
            panel.section = 5;
            panel.refresh();
        }
        function cancelGoogleConsent(): void {
            panel.cancelGoogleConsent();
        }
        function googleCredential(id: string, client: string, secret: string, refresh: string): void {
            panel.googleCredentialFields(id, client, secret, refresh);
            panel.credentialDialog.open();
        }
        function saveCredential(): void {
            panel.credentialDialog.accept();
        }
        function googleStatus(id: string): void {
            panel.checkGoogleStatus(id);
        }
        function revokePermission(scope: string): void {
            panel.revokePermission(scope);
        }
        function removeCredential(id: string): void {
            panel.removeCredential(id);
        }
        function confirmCredentialDelete(accept: bool): void {
            if (accept)
                panel.credentialDeleteDialog.accept();
            else
                panel.credentialDeleteDialog.reject();
        }
        function credential(id: string, value: string): void {
            panel.credentialFields(id, value, "file");
            panel.credentialDialog.accept();
        }
        function togglePause(): void {
            panel.togglePause();
        }
        function stopExecutions(): void {
            panel.stopDialog.open();
        }
        function confirmStop(accept: bool): void {
            if (accept)
                panel.stopDialog.accept();
            else
                panel.stopDialog.reject();
        }
        function closePanel(): void {
            panel.close();
        }
        function openPanel(): void {
            panel.open("{}");
        }
        function refreshStatus(): void {
            panel.backendClient.call("status", {});
        }
        function inspectQueue(enabled: bool): void {
            panel.queueOnly = enabled;
            panel.queueOffset = 0;
            panel.section = 4;
            panel.refresh();
        }
        function inspectExecution(id: string): void {
            panel.inspectExecution(id);
        }
        function simulate(source: string, data: string): void {
            panel.sampleEvent(source, data);
            panel.runSimulation(true);
        }
        function dismissSimulation(): void {
            panel.simulationDialog.close();
        }
        function realTest(): void {
            panel.realTestDialog.open();
        }
        function confirmRealTest(accept: bool): void {
            if (accept)
                panel.realTestDialog.accept();
            else
                panel.realTestDialog.reject();
        }
        function dismissDetails(): void {
            panel.executionDetailDialog.close();
        }
        function retryDelivery(id: string): void {
            panel.retryDelivery(id);
        }
        function confirmRetry(accept: bool): void {
            if (accept)
                panel.deliveryRetryDialog.accept();
            else
                panel.deliveryRetryDialog.reject();
        }
        function exportDiagnostic(path: string): void {
            panel.exportDiagnostic(path);
        }
        function section(index: int): void {
            panel.section = index;
        }
        function capture(path: string): void {
            panel.capture(path);
        }
    }
}
