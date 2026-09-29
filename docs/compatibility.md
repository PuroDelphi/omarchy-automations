# Compatibilidad comprobada

Inspección inicial 2026-09-28:

| Componente | Resultado |
|---|---|
| Omarchy | 4.0.4-1; manifiesto `quatrro.automations` validado |
| systemd | 261.2-1-arch; gestor del usuario en estado running |
| Quickshell | Panel instalado abre en shell real; widget visible en barra, interacción y QA visual cerradas en R1.UI; ver ui-audit.md |
| Notificaciones | org.freedesktop.Notifications anunciado por Quickshell |
| Secret Service | org.freedesktop.secrets anunciado por gnome-keyring-daemon |
| Bubblewrap | 0.12.0; namespace de prueba ejecuta `/usr/bin/true` en host |
| Go | 1.26.8 descargado y checksum oficial verificado; local al workspace |
| SQLite Go | modernc.org/sqlite v1.56.0 fijado en go.mod |

Los comandos del bus y Bubblewrap requieren acceso al host fuera del sandbox del agente. Que los servicios estén anunciados no demuestra todavía el recorrido funcional del producto.

## Formatos de entrada comprobados

| Formato | Estado |
|---|---|
| JSON, formulario URL-encoded, raw | Receptor autenticado implementado y probado |
| XML | Receptor autenticado y formulario QML probados; sin DTD, entidades externas ni XPath |
| multipart/form-data | Campos UTF-8 y archivos pequeños en base64; límites y rechazo de rutas probados |

Los contratos y límites de los nuevos formatos figuran en [formatos](formats.md).
Las salidas admiten JSON, formulario URL-encoded y raw. XML/multipart como salida
no se anuncian como implementados; los otros proveedores continúan según el roadmap.

## Proveedores incorporados

| Proveedor | Evidencia | Límite |
|---|---|---|
| GitHub | Vector oficial, firma alterada y replay con cabecera delivery cambiada | Sin frescura firmada; sin cuenta real conectada |
| Slack | Vector oficial, desafío de URL, replay, validación temporal y fallo de commit | JSON/formulario; no interpreta automáticamente `payload` anidado de interacciones; sin cuenta real conectada |

Consultar [contrato y configuración de proveedores](providers.md). Los fixtures
comprueban protocolo y lógica local; no equivalen a certificación del proveedor.

## Exposición opcional

Caddy 2.11.4 comprobado con proxy TLS real y certificados temporales: rutas
restringidas, firma preservada y error cuando falta backend. Solo loopback
durante pruebas. [Guía de exposición](exposure.md); no se ha configurado DNS,
firewall público ni servidor de túnel externo.

## Monitores ampliados

Archivos regulares (metadatos, sin enlaces simbólicos), procesos propios,
comprobación HTTPS HEAD y temperatura implementados. Sensor real del host leído
correctamente; su disponibilidad no se presupone en otros equipos. Journald
probado con una unidad temporal propia y reinicio del motor: cursor conservado,
sin duplicación y sin contenido MESSAGE. Formularios QML verificados junto con
lecturas reales de archivo y HTTPS. Contratos: [monitores](monitoring.md).

## Contratos entre componentes (F3.7)

| Enlace | Contrato actual | Evidencia de rechazo |
|---|---|---|
| CLI → motor | Protocolo local 2, operaciones 1 | Saludo previo sin payload; versión/contrato ajenos no llegan al handler |
| UI → CLI | UI 1 por comando ui y stdin | Contrato ausente/ajeno se rechaza sin conexión al motor |
| Artefactos de usuario | Versiones de release iguales y contratos requeridos por Compatibility.js | Verifica snapshots exactos; actualización incompatible conserva archivos/recibo |
| Motor → adaptador | Protocolo 1, revisión fijada | Validador y decoder rechazan manifiesto/respuesta incompatibles |
| Motor → broker opcional | Protocolo 1 con check_only | Preflight exige autorización; broker anterior que rechaza check no permite activar |

La UI nueva no recurre a comandos sin contrato cuando la CLI no soporta ui.
Comandos manuales directos siguen disponibles y negocian el transporte. Estos
contratos evitan errores de versiones, no autentican código malicioso del mismo
usuario. El estado duradero y los secretos no se eliminan por incompatibilidad.
`scripts/test-ui-compatibility.py` comprueba con QML, CLI y motor reales el
rechazo visible EN/ES de un contrato UI incompatible, conservación del borrador
y configuración persistida intacta. Al restaurar el contrato compatible, el
borrador se guarda y el idioma se conserva. La variación del contrato se realiza
solo en una copia temporal del harness; capturas offscreen inspeccionadas, sin
atribuirles validación dentro del shell instalado. Los tests de transporte y
artefactos cubren por separado versiones mezcladas de CLI/motor/paquete.

El test host `TestHostAdapterRefusalDoesNotAdvance/invalid-response` ejecuta un
adaptador con ID correcto y versión incompatible: el flujo falla y no ejecuta
el paso siguiente. Esto complementa el rechazo de manifiestos antes de preparar.
