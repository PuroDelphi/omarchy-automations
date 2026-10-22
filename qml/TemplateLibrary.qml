pragma ComponentBehavior: Bound
import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import qs.Commons
import "."
import "Recipes.js" as Recipes

LocalizedDialog {
    id: root
    property var config: ({})
    signal selected(var next)
    property var categories: ["All", "Desktop", "Reminders", "Schedules", "Monitoring", "Local events"]
    readonly property var filtered: Recipes.recipes.filter(function (r) {
        return (category.currentText === "All" || r.category === category.currentText) && (Recipes.title(r, I18n.language) + " " + r.source).toLowerCase().indexOf(search.text.toLowerCase()) >= 0;
    })
    width: Math.min(parent.width - 48, 780)
    height: Math.min(parent.height - 48, 720)
    anchors.centerIn: parent
    title: I18n.tr("Biblioteca de plantillas") + " · " + Recipes.recipes.length
    modal: true
    standardButtons: Dialog.Close
    contentItem: ColumnLayout {
        ThemedLabel {
            Layout.fillWidth: true
            wrapMode: Text.Wrap
            text: I18n.tr("Las plantillas se añaden deshabilitadas al borrador. Ajusta sus valores, habilita el flujo y sus monitores o temporizadores, guarda y revisa los permisos antes de activar. Los eventos de escritorio requieren conectar su hook. Los horarios usan UTC inicialmente.")
        }
        InputField {
            id: search
            Layout.fillWidth: true
            accessibleLabel: I18n.tr("Buscar plantillas")
            placeholderText: I18n.tr("Buscar plantillas")
        }
        ActionButton {
            text: I18n.tr("Limpiar búsqueda")
            visible: search.text !== ""
            onClicked: search.text = ""
        }
        ChoiceField {
            id: category
            accessibleLabel: I18n.tr("Categoría de plantilla")
            model: root.categories
            labels: [I18n.tr("Todas"), I18n.tr("Escritorio"), I18n.tr("Recordatorios"), I18n.tr("Horarios"), I18n.tr("Monitoreo"), I18n.tr("Eventos locales")]
        }
        ScrollView {
            id: scroll
            Layout.fillWidth: true
            Layout.fillHeight: true
            contentWidth: availableWidth
            clip: true
            ColumnLayout {
                width: scroll.availableWidth
                spacing: 20
                ThemedLabel {
                    visible: root.filtered.length === 0
                    text: I18n.tr("No hay plantillas que coincidan. Cambia la búsqueda o la categoría.")
                    Layout.fillWidth: true
                    wrapMode: Text.Wrap
                }
                Repeater {
                    model: root.filtered
                    ColumnLayout {
                        required property var modelData
                        Layout.fillWidth: true
                        ThemedLabel {
                            text: Recipes.title(modelData, I18n.language)
                            font.bold: true
                            Layout.fillWidth: true
                            wrapMode: Text.Wrap
                        }
                        ThemedLabel {
                            Layout.fillWidth: true
                            wrapMode: Text.Wrap
                            text: modelData.source + (modelData.timer ? " · " + (modelData.timer.at || modelData.timer.interval_seconds + " s") : modelData.monitor ? " · " + modelData.monitor.metric + " · " + modelData.monitor.threshold : "")
                        }
                        ActionButton {
                            text: I18n.tr("Añadir al borrador")
                            onClicked: {
                                root.selected(Recipes.instantiate(modelData, root.config, I18n.language));
                                root.close();
                            }
                        }
                    }
                }
            }
        }
    }
}
