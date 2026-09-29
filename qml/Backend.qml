pragma ComponentBehavior: Bound
import QtQuick
import "."
import Quickshell.Io
import "Compatibility.js" as Compatibility

Item {
    id: root
    property bool connected: false
    property bool polling: true
    property var status: ({})
    property string error: ""
    property string cli: "quatrroctl"
    property bool busy: false
    property var queue: []
    property string currentOperation: ""
    property string output: ""
    property string failure: ""
    property string payloadText: ""
    property bool exited: false
    property bool stdoutDone: false
    property bool stderrDone: false
    property bool finished: false
    property int exitCode: -1
    signal result(string operation, var value)
    signal failed(string operation, string message)
    function finish() {
        if (finished || !exited || !stdoutDone || !stderrDone)
            return;
        finished = true;
        busy = false;
        if (exitCode !== 0) {
            error = failure.trim() ? I18n.failure(failure.trim()) : I18n.tr("No se pudo completar la operación");
            if (currentOperation === "status")
                connected = false;
            failed(currentOperation, error);
        } else {
            try {
                var value = JSON.parse(output);
                error = "";
                connected = true;
                if (currentOperation === "status") {
                    status = value;
                    I18n.language = value.language === "es" ? "es" : "en";
                }
                if (currentOperation === "preferences.get" || currentOperation === "preferences.set")
                    I18n.language = value.language === "es" ? "es" : "en";
                result(currentOperation, value);
            } catch (e) {
                error = I18n.tr("Respuesta inválida del motor");
                failed(currentOperation, error);
            }
        }
        Qt.callLater(dispatch);
    }
    function dispatch() {
        if (busy || proc.running || queue.length === 0)
            return;
        var item = queue[0];
        queue = queue.slice(1);
        currentOperation = item.op;
        payloadText = JSON.stringify({
            contract: Compatibility.requirements.ui_contract,
            operation: item.op,
            data: JSON.parse(item.payload)
        });
        output = "";
        failure = "";
        exited = false;
        stdoutDone = false;
        stderrDone = false;
        finished = false;
        busy = true;
        proc.stdinEnabled = true;
        proc.command = [cli, "ui", "--stdin"];
        proc.running = true;
    }
    function call(operation, payload) {
        if (queue.length >= 32) {
            error = I18n.tr("Hay demasiadas operaciones pendientes");
            return false;
        }
        if (operation === "status" && (busy || queue.length > 0))
            return false;
        queue = queue.concat([
            {
                op: operation,
                payload: JSON.stringify(payload || {})
            }
        ]);
        dispatch();
        return true;
    }
    Process {
        id: proc
        stdinEnabled: true
        onStarted: {
            write(root.payloadText);
            root.payloadText = "";
            stdinEnabled = false;
        }
        stdout: StdioCollector {
            onStreamFinished: {
                root.output = text;
                root.stdoutDone = true;
                root.finish();
            }
        }
        stderr: StdioCollector {
            onStreamFinished: {
                root.failure = text;
                root.stderrDone = true;
                root.finish();
            }
        }
        onExited: function (code) {
            root.exitCode = code;
            root.exited = true;
            root.finish();
        }
    }
    Timer {
        interval: 5000
        running: root.polling
        repeat: true
        triggeredOnStart: true
        onTriggered: root.call("status", {})
    }
}
