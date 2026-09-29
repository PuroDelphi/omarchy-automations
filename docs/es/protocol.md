# Protocolo local, versión 2 — contrato de operaciones 1

[English](../en/protocol.md) · [Arquitectura](architecture.md)

Socket `$XDG_RUNTIME_DIR/quatrro/control.sock`, permisos 0600. Servidor y cliente
comprueban UID del peer mediante SO_PEERCRED. No se publica por HTTP.

Cada conexión negocia primero compatibilidad y luego admite una sola operación.
Cada mensaje es una línea JSON de hasta 1 MiB; plazo de conexión acotado a 15 s
(o el plazo menor del llamador), y 12 s para el handler. Saludo sin payload:

```json
{"version":2,"contract":1,"op":"hello"}
```

El servidor confirma `{"version":2,"contract":1,"ok":true,"data":{"handshake":"ready"}}`.
Solo entonces la CLI envía, en la misma conexión, por ejemplo:

```json
{"version":2,"contract":1,"op":"status","data":{}}
```

Respuesta: `{"version":2,"contract":1,"ok":true,"data":{...}}` o
`{"version":2,"contract":1,"ok":false,"error":"..."}`. El servidor comprueba de
nuevo versión y contrato en la operación; no admite omitir el saludo. Se conserva
un lector por conexión para procesar correctamente mensajes concatenados.

Un cliente versión 1 se rechaza antes de invocar el motor. Frente a un motor
antiguo, el cliente nuevo envía únicamente el saludo, sin evento ni secreto.
No hay fallback automático al protocolo anterior ni reenvío de operaciones.
Versión de transporte y contrato de operaciones se comprueban por igualdad.
El protocolo del broker administrativo continúa siendo independiente (versión 1),
al igual que los esquemas de configuración y adaptadores.

Para recuperar una instalación mixta: actualizar motor, CLI y plugin juntos,
reiniciar quatrrod y recargar el plugin. El estado persistido no se borra por una
incompatibilidad. Este cambio exige actualizar también clientes externos del
socket; los hooks que usan la CLI actual negocian automáticamente.

`quatrroctl OP --stdin` lee el payload de stdin. Es obligatorio usar stdin para secretos; nunca ponerlos en argumentos ni comandos registrados en historial.

| Operación | Datos | Resultado |
|---|---|---|
| status | ninguno | Versión, idioma, disponibilidad e ingress (estado/dirección loopback real; exposición pública sin verificar) |
| scripts.prepare | id, path, interpreter, parameters | Copia y valida revisión local; no guarda ni ejecuta |
| adapters.prepare | path, manifest | Copia código local y devuelve revisión ligada al manifiesto; no guarda, concede permisos ni ejecuta |
| directories.prepare | source, target, access | Devuelve identidad dispositivo/inode de un directorio; no guarda ni concede permisos |
| preferences.get | ninguno | Idioma de interfaz, en por defecto |
| preferences.set | language: en o es | Guarda preferencia por perfil sin alterar recursos |
| config.get / config.active | ninguno | Borrador / configuración activa |
| config.save | Configuración completa versión 1 | Hash del borrador validado |
| config.preview | ninguno | Hash, configuración y capacidades requeridas |
| config.activate | hash y grants exactos revisados | Nueva revisión activa |
| permissions.revoke | scope: `flow:<flujo>:action:<acción>` o `monitor:<id>` / `timer:<id>` | Revoca próximas ejecuciones de esa capacidad |
| secrets.put | id, value, backend: keyring o file | Referencia y backend, nunca valor |
| secrets.list | ninguno | Referencias y backend |
| secrets.delete | id | Elimina el valor y la referencia; las conexiones afectadas fallan cerradas |
| emit | Evento con source local: o hook: | ID y cantidad de ejecuciones persistidas |
| simulate | Evento de prueba | Flujos coincidentes y pasos, sin efectos |
| control | paused: boolean, admission: retain o reject | Estado de pausa |
| cancel | ninguno | Solicita cancelación de procesos; efectos iniciados pueden quedar inciertos |
| history | ninguno | Últimas 100 ejecuciones sin payloads |
| timers.status | ninguno | Permiso, próximo vencimiento y último disparo de las programaciones |
| monitors.status | ninguno | Estado, valor y próximo muestreo de monitores |
| monitors.reset | id de monitor journald | Descarta cursor; comienza en la próxima muestra autorizada, sin cambiar permisos |
| permissions.list | ninguno | Capacidades activas de acciones y monitores |
| history.detail | id | Pasos e intentos de entrega sin cuerpos |
| delivery.retry | id | Reenvía solo HTTP fallido/cancelado, con la misma clave |
| config.import | Configuración | Borrador con recursos desactivados |
| config.import-file | path absoluto .json | Importa un archivo local acotado |
| config.export-file | path absoluto .json nuevo | Exporta configuración sin sobrescribir |
| diagnostics | ninguno | Reporte de versiones y contadores sin datos de configuración |
| diagnostics.export-file | path absoluto .json nuevo | Exporta reporte con permisos 0600, sin sobrescribir |
| queue.inspect | state opcional, offset y limit (1..100) | Trabajos pendientes/en ejecución/inciertos, paginados y sin cuerpos |
| storage.status | ninguno | Uso y política de retención |
| storage.policy | max_events, max_payload_bytes, retention_days, dedup_days | Límites de almacenamiento |

El consentimiento se expresa al enviar las capacidades exactas de la revisión a `config.activate`. La interfaz presenta esas capacidades antes de hacerlo. Un emisor remoto no tiene acceso a esta operación.

## Entradas HTTP

POST `/hooks/ID`, por defecto en `127.0.0.1:8791`. El motor solo escucha en loopback; para exposición remota se requiere proxy TLS configurado explícitamente. Cuerpos máximos de 256 KiB.

HMAC genérico:

- `X-Quatrro-Timestamp`: segundos Unix, tolerancia de cinco minutos.
- `X-Quatrro-Delivery`: identificador único de entrega.
- `X-Quatrro-Signature`: `sha256=` + HMAC-SHA256 hexadecimal de `timestamp.delivery.` seguido de los bytes originales del cuerpo.
- El tipo se toma del campo `type` del cuerpo autenticado, o se usa `webhook`. No se confía en cabeceras de tipo no cubiertas por la firma.

GitHub utiliza `X-Hub-Signature-256` sobre el cuerpo original. Se exige `X-GitHub-Delivery`, pero la deduplicación se liga al digest del cuerpo autenticado: cambiar una cabecera no permite repetir el mismo payload. Dos entregas con cuerpo exactamente igual se consolidan durante la ventana de deduplicación. El tipo de evento no se toma de `X-GitHub-Event`, que no está firmado; filtra campos del cuerpo y el origen registrado. Su contrato no ofrece timestamp firmado, por lo que no demuestra frescura fuera de la ventana retenida.

Bearer utiliza `Authorization: Bearer ...`; el transporte público siempre debe usar TLS. JSON requiere un objeto; formularios no aceptan nombres de campo duplicados; raw entrega `data.body` y `data.content_type`.

Slack comprueba firma v0 y timestamp, responde al desafío autenticado sin crear ejecuciones y confirma los eventos normales con HTTP 200 después del commit. Detalles en [proveedores](../providers.md).

Una respuesta 202 indica commit de entrada y ejecuciones, no que todas las acciones hayan terminado. La deduplicación usa origen e identidad autenticada y conserva registros separados del cuerpo (30 días por defecto).

## Salida HTTP

Destinos HTTPS registrados. `private_hosts` habilita únicamente excepciones exactas `host:puerto`, sin omitir validación TLS. No usar excepciones globales.

Los destinos admiten `format: json`, `form` o `raw`. JSON y formulario usan una plantilla JSON en `body`: se parsea, se mapean valores y se codifica según destino. Una referencia escalar completa conserva su tipo. La salida raw usa texto. El cuerpo resultante se guarda en outbox para los reintentos; los datos remotos no añaden claves ni campos por romper comillas o introducir `&`.

La entrega incluye `Idempotency-Key` estable por ejecución/paso. Las respuestas 408, 429 y 5xx y los fallos de transporte pueden reintentarse, hasta ocho intentos o 24 horas. El receptor debe implementar idempotencia para evitar efectos duplicados tras caídas.

## Almacenamiento

Migración transaccional a esquema SQLite 5; se rechazan esquemas futuros. Por defecto: 10.000 eventos, 64 MiB de payload (eventos más contextos privados de ejecución), siete días de historial terminal y 30 días de deduplicación. La cuota cuenta bytes UTF-8, no caracteres. El motor no elimina trabajos pendientes ni resultados inciertos para aceptar otros. SQLite limita su archivo principal a 65.536 páginas (256 MiB con páginas de 4 KiB); el WAL se consolida periódicamente. No se promete que la suma instantánea de WAL y archivo principal sea igual a ese límite.

La admisión devuelve un error ante falta de espacio y no confirma eventos sin commit. La limpieza de eventos y el estado de los monitores usan transacciones. Las credenciales nunca forman parte de la exportación desde su almacén; evita poner valores secretos manualmente en campos comunes como cuerpos o cabeceras.

### administration.check

Entrada: `{"unit":"backup.service","operation":"restart"}`. Comprueba la
política administrativa para el usuario del motor sin ejecutar systemctl ni
conceder capacidades a flujos. Operaciones admitidas: status/start/stop/restart;
unidad fija válida, sin campos de identidad, scripts o argv.

Respuesta: `{"state":"authorized","unit":"backup.service","operation":"restart","effects_executed":false}`.
Estados: authorized, denied, unavailable (incluye respuesta inválida/desconexión)
y unsupported (broker anterior rechaza la extensión). Consulta acotada a ocho
segundos; no reintenta. No acredita existencia de la unidad ni garantiza permisos
futuros: la ejecución vuelve a comprobar política y Polkit. No activa un flujo.

### Contrato UI/CLI

La interfaz envía todas sus solicitudes mediante `quatrroctl ui --stdin`:

```json
{"contract":1,"operation":"status","data":{}}
```

La CLI valida contrato, campos y claves duplicadas antes de resolver rutas o
conectar al motor. No admite envelopes concatenados ni operaciones reservadas
hello/ui/hook. El contenido de data se conserva y viaja únicamente por stdin y
el socket negociado; no se incorpora a argumentos del proceso. El resultado
mantiene el formato habitual de la operación para los consumidores QML.

La UI nueva con una CLI vieja no reintenta por la ruta sin contrato: el motor
anterior rechaza la operación desconocida ui, o el transporte nuevo rechaza su
protocolo antiguo. La interfaz presenta instrucciones EN/ES de actualización.
Los comandos manuales directos de la CLI siguen disponibles; este contrato no
pretende autenticar procesos del mismo usuario como una frontera de seguridad.

`quatrroctl --version` y `quatrrod --version` devuelven JSON con component, version,
protocol, contract y ui_contract sin abrir perfiles, bases de datos o sockets.
El instalador comprueba conjuntamente estos metadatos antes de instalar componentes.


Las operaciones Google `oauth.google.begin`, `current`, `session`, `cancel`, `put` y `status` bajo ese prefijo se describen en el [contrato OAuth](../oauth2.md) y [ejemplo guiado](oauth-example.md).
