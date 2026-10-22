import QtQuick
import Quickshell
import Quickshell.Io

// Loaded exclusively by scripts/editor.py in a bounded, separate user unit.
ShellRoot {
    Editor {
        id: editor
        Component.onCompleted: open("{}")
    }
    IpcHandler {
        target: "quatrro-editor"
        function show(): string {
            editor.open("{}");
            return "ready";
        }
    }
}
