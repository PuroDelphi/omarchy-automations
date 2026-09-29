# Nombre del producto y compatibilidad

El nombre público es **Omarchy Automations**. Se usa en el manifiesto, título
principal, notificaciones, autorización OAuth, nuevas etiquetas del almacén de
credenciales, documentación y descripciones de servicios. La barra usa el nombre
corto **Automations** para conservar espacio. Inglés es el idioma inicial y el
usuario puede seleccionar español.

Los siguientes identificadores se conservan para actualizar las instalaciones
existentes sin duplicar plugins, perder credenciales o romper integraciones:

| Contrato | Identificador conservado |
|---|---|
| Plugin y preferencias del shell | `quatrro.automations` |
| Binarios y unidad de usuario | `quatrrod`, `quatrroctl`, `quatrrod.service` |
| Broker, socket y acciones Polkit | nombres `quatrro-broker` y `org.quatrro.automations.*` |
| Directorios XDG y atributo de Secret Service | `quatrro` |
| Perfil de desarrollo | `QUATRRO_PROFILE` |
| Firma genérica de webhooks | cabeceras `X-Quatrro-*` |
| Módulo Go | `quatrro.local/automations` |

El cambio de nombre no cambia esquema SQLite, protocolos, permisos aprobados,
revisiones activas ni selección del plugin en la barra. No es necesario renombrar
archivos de datos ni volver a introducir secretos. Las etiquetas de credenciales
ya existentes y los títulos personalizados de flujos del usuario se conservan;
los valores predeterminados nuevos usan el nombre actualizado.

La traducción de errores de incompatibilidad reconoce tanto el texto antiguo
como el nuevo para permitir una actualización con componentes de distinta edad.
La instalación sigue exigiendo versiones compatibles del motor, CLI y panel.
El directorio local de trabajo puede seguir llamándose `quatrro-automations`;
el repositorio remoto elegido se llama `omarchy-automations`.
