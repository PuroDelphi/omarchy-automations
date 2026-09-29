# Idioma de la interfaz

La interfaz usa **inglés por defecto**. El selector **English / Español** está
encima de las secciones del panel. Cambia etiquetas, ayudas, estados, revisión
de capacidades, formularios y botones de diálogos sin modificar los recursos.

La selección se guarda en el motor por perfil. El widget la recibe mediante
la consulta periódica de estado; puede tardar hasta cinco segundos en actualizarse.
Al reiniciar el motor o abrir de nuevo el panel se recupera la preferencia.
Se necesita conexión al motor para guardar un cambio de idioma.

Los nombres, textos de notificaciones y payloads escritos por el usuario no se
traducen. Tampoco cambian identificadores de protocolo, nombres de campos JSON,
rutas, métodos HTTP ni valores técnicos de los recursos. Los errores técnicos
del motor se conservan con una explicación localizada; el JSON de diagnóstico
mantiene su contrato estable.

CLI: `quatrroctl preferences.get` consulta la selección y
`quatrroctl preferences.set '{"language":"es"}'` la cambia. Solo se admiten
`en` y `es`. La exportación de configuración no cambia esta preferencia; el
respaldo completo de SQLite sí la incluye.

El catálogo vive en `qml/Translations.js`; `I18n.qml` comparte el idioma dentro
de cada proceso QML. Los diálogos localizan sus botones explícitamente para no
depender del idioma del escritorio.
