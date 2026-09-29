pragma Singleton
import QtQuick
import "Translations.js" as Translations

QtObject {
    property string language: "en"
    function tr(spanish) {
        return language === "es" ? spanish : (Translations.english[spanish] || spanish);
    }
    function failure(details) {
        if (details.indexOf("incompatible Quatrro components") !== -1 || details.indexOf("incompatible Omarchy Automations components") !== -1 || details.indexOf("unknown operation: ui") !== -1)
            return language === "es" ? "Componentes de Omarchy Automations incompatibles. Actualiza el motor, la CLI y el plugin juntos, reinicia quatrrod y recarga el plugin." : "Incompatible Omarchy Automations components. Update the engine, CLI and plugin together, restart quatrrod and reload the plugin.";
        if (details.indexOf("backend unavailable:") === 0)
            return language === "es" ? "Motor no disponible. Comprueba que el servicio está iniciado." : "Engine unavailable. Check that the service is running.";
        return (language === "es" ? "No se pudo completar la operación. Detalles técnicos: " : "The operation could not be completed. Technical details: ") + details;
    }
}
