import QtQuick
import qs.Ui as Native
import qs.Commons
import "qml"

Native.BarIconButton {
    id: root
    property var shell: null
    property var manifest: null
    property var settings: ({})
    property string moduleName: "quatrro.automations"
    property alias backendClient: backend
    readonly property var panelHost: shell || (bar ? bar.shell : null)
    readonly property bool opened: panelHost && typeof panelHost.isPluginOpen === "function" ? panelHost.isPluginOpen(moduleName) : false
    readonly property int pending: backend.status.counts ? backend.status.counts.pending || 0 : 0
    readonly property int failed: backend.status.counts ? (backend.status.counts.failed || 0) + (backend.status.counts.uncertain || 0) + (backend.status.counts.denied || 0) : 0
    readonly property string statusText: !backend.connected ? I18n.tr("Sin conexión") : backend.status.paused ? I18n.tr("En pausa") : "Omarchy Automations"
    readonly property string summary: statusText + (backend.connected ? " · " + pending + I18n.tr(" pendientes") + " · " + failed + I18n.tr(" fallos") : "") + "\n" + I18n.tr("Abrir automatizaciones. Deshabilitar el widget no detiene el motor.")
    // Nerd Font glyphs use the same optical sizing and font as Omarchy's icons.
    text: !backend.connected ? "\uf127" : failed > 0 ? "\uf071" : backend.status.paused ? "\uf04c" : "\uf0e8"
    active: !backend.connected || failed > 0
    tooltipText: summary
    activeFocusOnTab: true
    Accessible.role: Accessible.Button
    Accessible.name: summary
    Accessible.onPressAction: activate()
    Keys.onReturnPressed: activate()
    Keys.onEnterPressed: activate()
    Keys.onSpacePressed: activate()
    onPressed: function (button) {
        if (button === Qt.LeftButton) {
            root.forceActiveFocus();
            activate();
        }
    }
    Native.BorderSurface {
        anchors.fill: parent
        z: -1
        visible: root.activeFocus || root.opened
        color: Style.focusFillFor(root.foreground, Color.accent)
        borderSpec: Border.controlSpec("focus", root.foreground, Color.accent)
        radius: Style.cornerRadius
    }
    // The shell owns the separate panel; keep its open/close facade.
    function open() {
        if (panelHost)
            panelHost.summon(moduleName, "{}");
    }
    function close() {
        if (panelHost)
            panelHost.hide(moduleName);
    }
    function activate() {
        if (panelHost)
            panelHost.toggle(moduleName, "{}");
    }
    Backend {
        id: backend
    }
}
