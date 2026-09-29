# Referencia de opciones

[EN / ES](../en/options.md) · [Guide / Guía](user-guide.md)

Esta entrega cubre campos compartidos, entradas, destinos, flujos, condiciones, programaciones, monitores, acciones, scripts y adaptadores. Incluye credenciales, retención, controles operativos, simulación y preparación. La revisión semántica y el alcance comprobado se registran en docs/documentation-audit.md. Los valores iniciales describen un formulario nuevo, no valores implícitos para todo JSON importado. Los ejemplos son valores independientes, no configuraciones completas. Las referencias deben existir. Los tokens técnicos son idénticos en ambos idiomas.

- [common](#group-common) (3)
- [entries](#group-entries) (3)
- [destinations](#group-destinations) (7)
- [flows](#group-flows) (3)
- [conditions](#group-conditions) (4)
- [timers](#group-timers) (6)
- [monitors](#group-monitors) (12)
- [actions](#group-actions) (16)
- [scripts](#group-scripts) (3)
- [adapters](#group-adapters) (5)
- [parameters](#group-parameters) (7)
- [arguments](#group-arguments) (2)
- [directories](#group-directories) (4)
- [credentials](#group-credentials) (8)
- [storage](#group-storage) (4)
- [panel](#group-panel) (11)
- [files](#group-files) (1)
- [history](#group-history) (4)
- [simulation](#group-simulation) (5)
- [security](#group-security) (9)
- [code](#group-code) (1)
- [administration](#group-administration) (1)

<a id="group-common"></a>
## common

<a id="common.id"></a>
### Identificador — `common.id`

**Valor inicial:** `""`.

Obligatorio y único dentro del tipo de recurso. Empieza con letra minúscula; después minúsculas, dígitos, punto, guion bajo o guion; 1–64 caracteres. Cambiarlo exige actualizar referencias.

- Ejemplo 1: `deploy`.
- Ejemplo 2: `disk-space`.

<a id="common.name"></a>
### Nombre — `common.name`

**Valor inicial:** `""`.

Nombre visible opcional de entradas, destinos, flujos y programaciones. No cambia el origen técnico ni la dirección.

- Ejemplo 1: `Deployment alerts`.
- Ejemplo 2: `Daily reminder`.

<a id="common.enabled"></a>
### Habilitado al activar la revisión — `common.enabled`

**Valor inicial:** `true`.

Se aplica a entradas, flujos, monitores y programaciones. Guardar no lo activa. Importar fuerza false. Un flujo deshabilitado no coincide al simular ni ejecutar.

- Ejemplo 1: `true`.
- Ejemplo 2: `false`.

<a id="group-entries"></a>
## entries

<a id="entries.auth"></a>
### Autenticación — `entries.auth`

**Valor inicial:** `hmac`.

Elige hmac, github, slack o bearer. El emisor debe usar exactamente ese contrato; timestamps HMAC genérico/Slack admiten cinco minutos de diferencia. Slack solo acepta json/form. Firmas incorrectas o credenciales ausentes rechazan antes de crear eventos.

- Ejemplo 1: `hmac`.
- Ejemplo 2: `slack`.

<a id="entries.secret"></a>
### Referencia de credencial — `entries.secret`

**Valor inicial:** `""`.

ID obligatorio de credencial creada en Seguridad, con la misma sintaxis que los recursos. Escribe la referencia, no el valor. Debe existir al autenticar solicitudes reales.

- Ejemplo 1: `deploy-signing`.
- Ejemplo 2: `slack-signing`.

<a id="entries.format"></a>
### Formato del cuerpo — `entries.format`

**Valor inicial:** `json`.

Elige json, form, raw, xml o multipart; cuerpo y tipo de contenido deben coincidir. JSON debe ser un objeto; formulario rechaza nombres duplicados. El formato es fijo por entrada, no autodetectado. Entrada HTTP limitada a 256 KiB.

- Ejemplo 1: `json`.
- Ejemplo 2: `multipart`.

<a id="group-destinations"></a>
## destinations

<a id="destinations.url"></a>
### Dirección HTTPS — `destinations.url`

**Valor inicial:** `""`.

URL HTTPS obligatoria con host, sin credenciales en URL ni fragmento. No sigue redirecciones. Las direcciones privadas exigen la excepción exacta; sigue siendo obligatorio un certificado confiable. Sustituye las direcciones de ejemplo por tu receptor.

- Ejemplo 1: `https://status.example.com/events`.
- Ejemplo 2: `https://receiver.example.com:8443/hook`.

<a id="destinations.method"></a>
### Método — `destinations.method`

**Valor inicial:** `POST`.

Elige POST, PUT o PATCH según el receptor. Las acciones HTTP usan ese método; los monitores de conectividad siempre usan HEAD. Otro método es un error de configuración.

- Ejemplo 1: `PUT`.
- Ejemplo 2: `PATCH`.

<a id="destinations.format"></a>
### Formato de salida — `destinations.format`

**Valor inicial:** `json`.

Elige json, form o raw. Los cuerpos JSON/form son plantillas JSON válidas codificadas tras sustituir valores; raw usa texto. Omitir formato en una configuración importada equivale a JSON.

- Ejemplo 1: `form`.
- Ejemplo 2: `raw`.

<a id="destinations.auth"></a>
### Autenticación — `destinations.auth`

**Valor inicial:** `""`.

Vacío no añade autenticación; otras opciones son hmac, bearer y oauth2-google. Los modos autenticados requieren referencia de credencial. Google limita además hosts y puerto.

- Ejemplo 1: `bearer`.
- Ejemplo 2: `oauth2-google`.

<a id="destinations.secret"></a>
### Referencia de credencial — `destinations.secret`

**Valor inicial:** `""`.

Visible en destinos autenticados. Usa credencial genérica existente para HMAC/bearer o conexión Google para oauth2-google. Un conjunto de tokens Google no es un secreto bearer genérico.

- Ejemplo 1: `status-token`.
- Ejemplo 2: `google-calendar`.

<a id="destinations.headers"></a>
### Cabeceras adicionales — `destinations.headers`

**Valor inicial:** `{}`.

Pares nombre/valor enviados y exportados en configuración. No pongas secretos aquí. Se rechazan Authorization, Host, Cookie, Proxy-* y CR/LF en nombres/valores.

- Ejemplo 1: `{"X-Application":"omarchy"}`.
- Ejemplo 2: `{"X-Environment":"staging"}`.

<a id="destinations.private_hosts"></a>
### Excepciones de red privada — `destinations.private_hosts`

**Valor inicial:** `[]`.

Vacío bloquea destinos privados. Como máximo una excepción, exactamente el host y puerto explícito de esta URL (443 si la URL omite puerto); IPv6 entre corchetes. Autoriza unicast local, no redes arbitrarias, y no desactiva TLS. Cada ejemplo exige su URL correspondiente.

- Ejemplo 1: `["status.internal:443"]`.
- Ejemplo 2: `["[fd00::10]:8443"]`.

<a id="group-flows"></a>
## flows

<a id="flows.source"></a>
### Cuando llegue un evento de — `flows.source`

**Valor inicial:** `local:demo`.

Origen exacto: entry:ID, monitor:ID o timer:ID de un recurso registrado, o local:NOMBRE/hook:NOMBRE. Un emisor HTTP no accede a la API de control local. Seleccionar origen no lo habilita ni autoriza acciones.

- Ejemplo 1: `entry:deploy`.
- Ejemplo 2: `monitor:disk-space`.

<a id="flows.conditions"></a>
### Si se cumplen estas condiciones — `flows.conditions`

**Valor inicial:** `[]`.

De cero a 32 condiciones; deben cumplirse todas. Usa ruta de campo, operador y valor escalar tipado. Campos ausentes nunca coinciden. Rutas de hasta 128 caracteres y ocho puntos. Consulta las opciones de condición siguientes.

- Ejemplo 1: `[{"field":"data.state","op":"eq","value":"failed"}]`.
- Ejemplo 2: `[{"field":"data.percent","op":"gt","value":90}]`.

<a id="flows.steps"></a>
### Ejecutar en este orden — `flows.steps`

**Valor inicial:** `[]`.

Lista obligatoria de 1–32 IDs de acciones registradas, ejecutadas en ese orden. Un flujo vacío no se guarda. Cada acción exige su permiso revisado; los pasos posteriores pueden consumir salida de adaptadores.

- Ejemplo 1: `["notice"]`.
- Ejemplo 2: `["normalize","report","notice"]`.

<a id="group-conditions"></a>
## conditions

<a id="conditions.field"></a>
### Campo — `conditions.field`

**Valor inicial:** `data.state`.

Ruta del evento: source, type o campos bajo data. Índices de arrays en decimal ordinario no negativo. Selecciona datos; no evalúa código.

- Ejemplo 1: `data.repository.full_name`.
- Ejemplo 2: `data.xml.children.0.text`.

<a id="conditions.op"></a>
### Operador — `conditions.op`

**Valor inicial:** `eq`.

eq/ne comparan valores escalares tipados; gt/lt requieren números; contains busca una subcadena en texto. Un campo ausente falla incluso con ne.

- Ejemplo 1: `eq`.
- Ejemplo 2: `contains`.

<a id="conditions.type"></a>
### Tipo de valor — `conditions.type`

**Valor inicial:** `string`.

Elige Texto, Número o Booleano en la fila. Importa el tipo: texto "90" no es número 90. Booleano usa true o false; al seleccionarlo convierte según si el texto actual es exactamente true.

- Ejemplo 1: `number: 90`.
- Ejemplo 2: `boolean: true`.

<a id="conditions.value"></a>
### Valor — `conditions.value`

**Valor inicial:** `failed`.

Valor que se compara con el campo. Una condición nueva empieza con texto failed. Elige el tipo antes de introducir una comparación numérica/booleana.

- Ejemplo 1: `failed`.
- Ejemplo 2: `90 (number)`.

<a id="group-timers"></a>
## timers

<a id="timers.kind"></a>
### Programación — `timers.kind`

**Valor inicial:** `interval`.

interval repite tras un número de segundos; calendar elige una hora local en zona explícita. Cambiar la configuración crea una programación nueva.

- Ejemplo 1: `interval`.
- Ejemplo 2: `calendar`.

<a id="timers.interval_seconds"></a>
### Intervalo en segundos (mínimo 5) — `timers.interval_seconds`

**Valor inicial:** `60`.

Solo interval: entero de 5–31536000 segundos. El primer vencimiento parte de la primera evaluación; reiniciar conserva el próximo guardado. No garantiza tiempo real.

- Ejemplo 1: `300`.
- Ejemplo 2: `3600`.

<a id="timers.at"></a>
### Hora local HH:MM — `timers.at`

**Valor inicial:** `09:00`.

Solo calendar: HH:MM exacto en formato de 24 horas. Una hora inexistente por cambio horario se omite; de una repetida solo puede ejecutarse la primera. Usa programaciones separadas para varias horas diarias.

- Ejemplo 1: `08:30`.
- Ejemplo 2: `18:00`.

<a id="timers.timezone"></a>
### Zona horaria IANA — `timers.timezone`

**Valor inicial:** `America/Bogota`.

Solo calendar: zona IANA conocida explícita o UTC. Se rechaza Local. No se sustituye implícitamente por la zona del escritorio.

- Ejemplo 1: `UTC`.
- Ejemplo 2: `Europe/Madrid`.

<a id="timers.weekdays"></a>
### Días de la semana — `timers.weekdays`

**Valor inicial:** `[]`.

Para calendar: códigos únicos mon tue wed thu fri sat sun, uno por línea en la interfaz. Vacío significa diario. Códigos desconocidos o repetidos fallan validación.

- Ejemplo 1: `["mon","tue","wed","thu","fri"]`.
- Ejemplo 2: `["sat","sun"]`.

<a id="timers.missed"></a>
### Si hay ejecuciones atrasadas — `timers.missed`

**Valor inicial:** `coalesce`.

coalesce emite un evento por atrasos acumulados; skip descarta atrasos mayores de 5 segundos en intervalos o 60 en calendarios. Ninguno despierta el equipo suspendido.

- Ejemplo 1: `coalesce`.
- Ejemplo 2: `skip`.

<a id="group-monitors"></a>
## monitors

<a id="monitors.metric"></a>
### Métrica — `monitors.metric`

**Valor inicial:** `disk`.

Elige cpu, memory, disk, battery, service, process, file_exists, file_age, file_size, connectivity, temperature o journal. CPU/memoria usan porcentaje ocupado; disco/batería disponible. Presencia/conectividad vale 0 o 100. Antigüedad en segundos, tamaño en bytes y temperatura en Celsius. Ajusta ambos umbrales al cambiar métrica.

- Ejemplo 1: `cpu`.
- Ejemplo 2: `file_age`.

<a id="monitors.path"></a>
### Ruta del disco — `monitors.path`

**Valor inicial:** `/`.

La etiqueta cambia según métrica. Disco: ruta absoluta del sistema de archivos. Proceso: ejecutable absoluto canónico, solo del usuario. Archivos: ruta absoluta canónica a archivo regular, sin enlaces simbólicos; solo metadatos. Temperatura: sensor thermal/hwmon explícito. Recursos ausentes/inaccesibles pueden quedar no disponibles; file_exists devuelve 0 para archivo ausente.

- Ejemplo 1: `/home (disk)`.
- Ejemplo 2: `/sys/class/thermal/thermal_zone0/temp (temperature)`.

<a id="monitors.unit"></a>
### Unidad de usuario — `monitors.unit`

**Valor inicial:** `""`.

Obligatoria para service/journal: unidad .service exacta de usuario. No es unidad de sistema ni comodín. Service devuelve 100 activa y 0 inactiva. Journal lee solo esa unidad permitida y no incluye contenido MESSAGE en sus eventos.

- Ejemplo 1: `backup.service`.
- Ejemplo 2: `worker.service`.

<a id="monitors.destination"></a>
### Destino HTTPS para comprobar con HEAD — `monitors.destination`

**Valor inicial:** `config.destinations[0]?.id`.

El selector elige inicialmente el primer destino registrado, si existe. Para connectivity: destino HTTPS registrado. Usa HEAD con sus credenciales/cabeceras, plazo de tres segundos y sin redirecciones. HTTP 2xx vale 100, fallo HTTP/transporte 0; credenciales ausentes o rechazo de política producen no disponible. Cambiar destino exige revisar el permiso del monitor.

- Ejemplo 1: `status`.
- Ejemplo 2: `health-endpoint`.

<a id="monitors.priority"></a>
### Prioridad máxima de journal (0 emergencia … 7 debug) — `monitors.priority`

**Valor inicial:** `3`.

Solo journal: entero 0–7, incluye esa prioridad y registros más urgentes. 0 emergencia, 3 error, 4 advertencia, 6 información, 7 debug. La primera muestra comienza ahora, sin importar historial anterior.

- Ejemplo 1: `3`.
- Ejemplo 2: `4`.

<a id="monitors.threshold"></a>
### Umbral de alerta — `monitors.threshold`

**Valor inicial:** `10`.

No journal. Alerta al bajar en disk/battery/service/process/file_exists/connectivity y al subir en las demás métricas numéricas. Porcentaje 0–100; temperatura −273.15–1000; antigüedad 0–31536000; tamaño 0–10^15. Una recuperación en dirección inválida rechaza la configuración. Las comparaciones son estrictas: la igualdad no inicia una alerta. Consulta monitor-recipes.md para dos configuraciones por métrica.

- Ejemplo 1: `10 (disk; recovery 15)`.
- Ejemplo 2: `90 (cpu; recovery 75)`.

<a id="monitors.recovery"></a>
### Umbral de recuperación — `monitors.recovery`

**Valor inicial:** `15`.

Mismas unidades/rangos que el umbral. Debe ser estrictamente mayor para métricas que alertan al bajar y menor para las que alertan al subir. Volver a cruzarlo emite recovered tras la duración de recuperación. La igualdad no recupera; el valor debe cruzar estrictamente el umbral.

- Ejemplo 1: `15 (disk; threshold 10)`.
- Ejemplo 2: `75 (cpu; threshold 90)`.

<a id="monitors.duration_seconds"></a>
### Duración mínima de la condición (segundos) — `monitors.duration_seconds`

**Valor inicial:** `60`.

No journal: entero 0–86400 segundos de condición observada antes de confirmar alerta. Cero permite confirmar en una muestra elegible. Un hueco prolongado reinicia la confirmación; no es observación continua en tiempo real.

- Ejemplo 1: `0`.
- Ejemplo 2: `120`.

<a id="monitors.recovery_duration_seconds"></a>
### Duración mínima de recuperación (segundos) — `monitors.recovery_duration_seconds`

**Valor inicial:** `0`.

No journal: entero 0–86400 segundos de recuperación observada antes de emitir recovered. Cero elimina la espera adicional de confirmación, no el intervalo de muestreo.

- Ejemplo 1: `0`.
- Ejemplo 2: `30`.

<a id="monitors.interval_seconds"></a>
### Intervalo de muestreo (segundos) — `monitors.interval_seconds`

**Valor inicial:** `10`.

Entero 5–3600 segundos para toda métrica, incluido journal. El muestreo es secuencial y comprobaciones lentas pueden retrasar las siguientes. El motor no despierta el equipo para muestrear.

- Ejemplo 1: `5`.
- Ejemplo 2: `60`.

<a id="monitors.cooldown_seconds"></a>
### Separación mínima entre alertas (segundos) — `monitors.cooldown_seconds`

**Valor inicial:** `300`.

No journal: mínimo cinco segundos entre alertas. Limita su frecuencia; no cambia el muestreo ni garantiza recordatorios repetidos mientras persista la condición.

- Ejemplo 1: `60`.
- Ejemplo 2: `600`.

<a id="monitors.reset"></a>
### Reiniciar cursor — `monitors.reset`

Solo monitores journal. Confirmar descarta cursor guardado; siguiente muestra autorizada comienza desde entonces sin recuperar registros anteriores. No concede permisos y registra auditoría.

- Ejemplo 1: `Reiniciar si rotación eliminó cursor`.
- Ejemplo 2: `Iniciar deliberadamente periodo de observación nuevo`.

<a id="group-actions"></a>
## actions

<a id="actions.kind"></a>
### Acción — `actions.kind`

**Valor inicial:** `notify`.

Elige notify, http, service, command, omarchy, script, adapter o system-service. Solo se guardan campos aplicables. Un flujo debe referenciar la acción y conceder su capacidad exacta antes de ejecutar.

- Ejemplo 1: `notify`.
- Ejemplo 2: `http`.

<a id="actions.title"></a>
### Título — `actions.title`

**Valor inicial:** `Omarchy Automations`.

Título de notificación de hasta 200 bytes. Admite referencias escalares al evento. Es texto plano; los campos ausentes provocan fallo de renderizado.

- Ejemplo 1: `Deployment finished`.
- Ejemplo 2: `Backup: {{data.project}}`.

<a id="actions.body"></a>
### Mensaje — `actions.body`

**Valor inicial:** `{{data.message}}`.

Mensaje de notificación (hasta 4096 bytes) o cuerpo HTTP (hasta 16 KiB al validar configuración). En HTTP la etiqueta es Cuerpo del envío y exige plantilla JSON para json/form; raw usa texto. Campos ausentes o no escalares pueden fallar al resolver; simula para previsualizar. Al resolver se limita además cada plantilla textual a 4096 bytes y el texto expandido a 16384; superar la comprobación de tamaño de configuración no garantiza poder resolverla.

- Ejemplo 1: `Build: {{data.message}}`.
- Ejemplo 2: `{"state":"{{type}}","value":"{{data.value}}"}`.

<a id="actions.destination"></a>
### Destino registrado — `actions.destination`

**Valor inicial:** `config.destinations[0]?.id`.

Para http: elige destino existente, inicialmente el primero disponible. URL, cabeceras y referencia de credencial forman parte de la capacidad revisada; cambiarlas exige nueva revisión.

- Ejemplo 1: `status`.
- Ejemplo 2: `deploy-receiver`.

<a id="actions.unit"></a>
### Unidad de usuario — `actions.unit`

**Valor inicial:** `""`.

Para service: .service exacta de usuario. En system-service la etiqueta es Unidad de sistema y exige política del broker opcional y autorización Polkit. Sin patrones ni argumentos systemctl arbitrarios. Comprobar política no ejecuta el servicio ni autoriza un flujo.

- Ejemplo 1: `backup.service`.
- Ejemplo 2: `worker.service`.

<a id="actions.operation"></a>
### Operación — `actions.operation`

**Valor inicial:** `status`.

Servicios de usuario/sistema ofrecen status, start, stop y restart. Omarchy ofrece theme.current, nightlight.status, nightlight.toggle y system.lock; al cambiar a Omarchy se elige la primera operación válida. Status consulta sin modificar servicio. Bloqueo y toggle afectan realmente el escritorio. En status de servicio de usuario, inactivo es acción fallida: comprueba is-active, no devuelve informe detallado.

- Ejemplo 1: `restart (service)`.
- Ejemplo 2: `nightlight.status (omarchy)`.

<a id="actions.command_profile"></a>
### Perfil de comando — `actions.command_profile`

**Valor inicial:** `fixed`.

Para command: fixed ejecuta binario y argumentos fijos; file-exists comprueba archivo regular hijo directo de un montaje aprobado; make-directory crea hijo directo con permisos 0700 y exige escritura. Los perfiles nombrados no admiten sustituir ejecutable/argumentos.

- Ejemplo 1: `file-exists`.
- Ejemplo 2: `make-directory`.

<a id="actions.command_path"></a>
### Ruta del perfil — `actions.command_path`

**Valor inicial:** `""`.

Para file-exists/make-directory: ruta absoluta canónica a hijo directo de montaje preparado, sin plantillas. El padre debe ser destino de directorio autorizado. make-directory falla si el hijo ya existe.

- Ejemplo 1: `/work/data/report.json`.
- Ejemplo 2: `/work/data/new-backup`.

<a id="actions.executable"></a>
### Ejecutable local — `actions.executable`

**Valor inicial:** `""`.

Para comandos fixed: ejecutable absoluto canónico bajo /usr/bin/. Debe estar disponible dentro del sandbox; sin HOME ni red IP por defecto. Un fallo no se reintenta automáticamente.

- Ejemplo 1: `/usr/bin/true`.
- Ejemplo 2: `/usr/bin/test`.

<a id="actions.args"></a>
### Argumentos fijos — `actions.args`

**Valor inicial:** `[]`.

Para fixed: hasta 32 argumentos, uno por línea; hasta 4096 bytes por argumento, sin NUL ni plantillas {{. Sin expansión de shell. Revisa también el binario: un intérprete puede tratar deliberadamente un argumento como código.

- Ejemplo 1: `["-f","/work/data/report.json"]`.
- Ejemplo 2: `["-d","/work/data"]`.

<a id="actions.adapter"></a>
### Adaptador registrado — `actions.adapter`

**Valor inicial:** `config.adapters[0]?.id`.

Para adapter: elige adaptador registrado, inspecciona código/revisión y selecciona esa revisión expresamente. Cambiar código o límites invalida la referencia anterior. Usa límites del manifiesto, no timeout separado.

- Ejemplo 1: `normalize`.
- Ejemplo 2: `summarize`.

<a id="actions.script"></a>
### Script registrado — `actions.script`

**Valor inicial:** `config.scripts[0]?.id`.

Para script: elige script registrado y selecciona expresamente su revisión exacta. Proporciona cada parámetro una vez como literal o campo del evento. No se vuelve a abrir el archivo original al ejecutar.

- Ejemplo 1: `check-project`.
- Ejemplo 2: `prepare-report`.

<a id="actions.timeout_seconds"></a>
### Tiempo máximo (segundos) — `actions.timeout_seconds`

**Valor inicial:** `30`.

Para command/script: entero 1–300 segundos. Al vencer se termina el grupo aislado; no deshace efectos realizados. El timeout del adaptador se configura en el propio adaptador.

- Ejemplo 1: `5`.
- Ejemplo 2: `120`.

<a id="actions.script_revision"></a>
### Usar esta revisión y reiniciar parámetros — `actions.script_revision`

**Valor inicial:** `""`.

Inspecciona código/hash disponible antes de seleccionar. Fija esa revisión y reinicia valores: false para booleano, mínimo para entero, primera opción o vacío para string. Si cambia el script hay que seleccionar otra vez; nunca se sustituye silenciosamente. Los hashes se obtienen al preparar, no se inventan.

- Ejemplo 1: `check-project: seleccionar revisión preparada`.
- Ejemplo 2: `prepare-report: seleccionar revisión actualizada`.

<a id="actions.adapter_revision"></a>
### Usar esta revisión — `actions.adapter_revision`

**Valor inicial:** `""`.

Inspecciona código y límites del manifiesto y fija la revisión preparada. Preparar o seleccionar no ejecuta ni concede permisos. Cambiar código/manifiesto exige nueva selección exacta y capacidad revisada.

- Ejemplo 1: `normalize: seleccionar revisión preparada`.
- Ejemplo 2: `summarize: seleccionar revisión actualizada`.

<a id="actions.working_directory"></a>
### Directorio de trabajo — `actions.working_directory`

**Valor inicial:** `/tmp`.

Elige /tmp privado o uno de los destinos de montaje aprobados de la acción, no un subdirectorio arbitrario. Quitar el destino elegido devuelve la interfaz a /tmp.

- Ejemplo 1: `/tmp`.
- Ejemplo 2: `/work/reports`.

<a id="group-scripts"></a>
## scripts

<a id="scripts.path"></a>
### Archivo local del script — `scripts.path`

**Valor inicial:** `""`.

Ruta absoluta a archivo local regular UTF-8, 1–32768 bytes, sin NUL ni enlaces simbólicos. Preparar revisión lee y copia el código para inspección; no ejecuta ni autoriza. Edita el origen y prepara de nuevo para reemplazar código; las exportaciones incluyen el código preparado.

- Ejemplo 1: `/home/alex/automations/check.py`.
- Ejemplo 2: `/home/alex/automations/report.py`.

<a id="scripts.interpreter"></a>
### Intérprete — `scripts.interpreter`

**Valor inicial:** `bash`.

bash o python3. Bash usa --noprofile --norc; Python -I -S. Código e intérprete forman parte de la revisión. Los parámetros son valores argv separados, no código shell interpolado.

- Ejemplo 1: `bash`.
- Ejemplo 2: `python3`.

<a id="scripts.parameters"></a>
### Parámetros del script — `scripts.parameters`

**Valor inicial:** `[]`.

Lista ordenada de hasta 16 parámetros tipados obligatorios. Nombres únicos con sintaxis de ID; tipos string, integer y boolean. El orden declarado es el de argv. No se admiten valores extra ni ausentes.

- Ejemplo 1: `[{"name":"message","type":"string","max_length":256}]`.
- Ejemplo 2: `[{"name":"count","type":"integer","minimum":1,"maximum":10}]`.

<a id="group-adapters"></a>
## adapters

<a id="adapters.path"></a>
### Archivo local del adaptador — `adapters.path`

**Valor inicial:** `""`.

Ruta absoluta a archivo local regular UTF-8, 1–32768 bytes, sin NUL ni enlaces simbólicos. Preparar revisión lee y copia el código para inspección; no ejecuta ni autoriza. Edita el origen y prepara de nuevo para reemplazar código; las exportaciones incluyen el código preparado.

- Ejemplo 1: `/home/alex/automations/check.py`.
- Ejemplo 2: `/home/alex/automations/report.py`.

<a id="adapters.runtime"></a>
### Intérprete — `adapters.runtime`

**Valor inicial:** `python3`.

Solo admite python3. Usa Python aislado (-I -S) y protocolo JSON solicitud/respuesta versión 1. No permite intérpretes arbitrarios; ambos ejemplos usan el único admitido para transformaciones distintas.

- Ejemplo 1: `python3: normalizar campos`.
- Ejemplo 2: `python3: calcular un resumen`.

<a id="adapters.input_limit"></a>
### Límite de entrada (bytes) — `adapters.input_limit`

**Valor inicial:** `262144`.

Entero 1–262144 bytes para toda la solicitud codificada, incluido salto final, no solo datos del evento. Si excede el límite falla antes de lanzar. Reserva espacio para el sobre.

- Ejemplo 1: `4096`.
- Ejemplo 2: `65536`.

<a id="adapters.output_limit"></a>
### Límite de salida (bytes) — `adapters.output_limit`

**Valor inicial:** `4096`.

Entero 1–65536 bytes para la respuesta JSON completa. Salida excesiva/adicional, ID de invocación incorrecto o datos inválidos rechazan el resultado. Los pasos siguientes acceden a datos bajo data.adapter.

- Ejemplo 1: `2048`.
- Ejemplo 2: `16384`.

<a id="adapters.timeout_seconds"></a>
### Tiempo máximo (segundos) — `adapters.timeout_seconds`

**Valor inicial:** `5`.

Entero 1–30 segundos. La transformación aislada debe terminar dentro del plazo. No se ofrece red ni directorios autorizados del host a adaptadores. Simular informa de la dependencia pero no ejecuta el adaptador.

- Ejemplo 1: `2`.
- Ejemplo 2: `10`.

<a id="group-parameters"></a>
## parameters

<a id="parameters.name"></a>
### Nombre del parámetro — `parameters.name`

**Valor inicial:** `parameterN`.

Un parámetro nuevo usa parameter más su posición. Sustitúyelo por identificador único (misma sintaxis de recursos). Renombrar cambia el contrato y exige preparar/revisar nueva revisión.

- Ejemplo 1: `message`.
- Ejemplo 2: `count`.

<a id="parameters.type"></a>
### Tipo de valor — `parameters.type`

**Valor inicial:** `string`.

Elige string, integer o boolean. Cambiar tipo reinicia límites: string empieza con max_length 256, integer con mínimo 0/máximo 100, boolean sin límites.

- Ejemplo 1: `integer`.
- Ejemplo 2: `boolean`.

<a id="parameters.max_length"></a>
### Longitud máxima (bytes) — `parameters.max_length`

**Valor inicial:** `256`.

Solo string: entero 1–4096 bytes UTF-8, no caracteres. Valores mayores se rechazan antes de ejecutar.

- Ejemplo 1: `64`.
- Ejemplo 2: `1024`.

<a id="parameters.pattern"></a>
### Patrón RE2 (opcional) — `parameters.pattern`

**Valor inicial:** `""`.

Solo string: patrón RE2 opcional de hasta 256 bytes, aplicado al valor completo. Vacío no restringe por patrón. Un patrón inválido falla al preparar.

- Ejemplo 1: `[a-z][a-z0-9-]*`.
- Ejemplo 2: `[0-9]{4}`.

<a id="parameters.choices"></a>
### Opciones permitidas (una por línea) — `parameters.choices`

**Valor inicial:** `[]`.

Solo string: hasta 32 valores permitidos únicos, uno por línea. Vacío no impone enumeración. Cada opción debe cumplir longitud/patrón.

- Ejemplo 1: `["staging","production"]`.
- Ejemplo 2: `["small","large"]`.

<a id="parameters.minimum"></a>
### Mínimo — `parameters.minimum`

**Valor inicial:** `0`.

Solo integer: límite inferior inclusivo explícito, no mayor que máximo. Límites dentro de ±9007199254740991. Texto y fracciones no se convierten automáticamente a enteros.

- Ejemplo 1: `1`.
- Ejemplo 2: `-10`.

<a id="parameters.maximum"></a>
### Máximo — `parameters.maximum`

**Valor inicial:** `100`.

Solo integer: límite superior inclusivo explícito, no menor que mínimo y dentro del rango entero exacto JSON. Se comprueba otra vez tras resolver un campo del evento.

- Ejemplo 1: `10`.
- Ejemplo 2: `1000`.

<a id="group-arguments"></a>
## arguments

<a id="arguments.mode"></a>
### Valor literal — `arguments.mode`

**Valor inicial:** `script_values`.

Por parámetro: elige Valor literal o Campo del evento. Exige exactamente un origen. Cambiar a campo sugiere data.PARAMETRO; volver reinicia literal según tipo. Los valores deben cumplir el contrato del script elegido.

- Ejemplo 1: `script_values: {"count":3}`.
- Ejemplo 2: `script_bindings: {"count":"data.count"}`.

<a id="arguments.value"></a>
### Valor — `arguments.value`

**Valor inicial:** `"" / minimum / false / data.PARAMETER`.

Los literales conservan tipo string/integer/boolean; booleano usa interruptor. Los vínculos aceptan source, type o data.CAMPO con hasta 128 caracteres/ocho puntos. Campos ausentes, tipos incorrectos o valores fuera de rango fallan antes de lanzar.

- Ejemplo 1: `data.message`.
- Ejemplo 2: `true (literal booleano)`.

<a id="group-directories"></a>
## directories

<a id="directories.source"></a>
### Directorio del host — `directories.source`

**Valor inicial:** `""`.

Solo command/script: directorio existente absoluto canónico, sin enlaces simbólicos. Se rechazan raíz y árboles /proc, /sys, /dev, /run. Preparar captura identidad dispositivo/inode; reemplazar el directorio exige preparar y revisar de nuevo.

- Ejemplo 1: `/home/alex/reports`.
- Ejemplo 2: `/home/alex/project-data`.

<a id="directories.target"></a>
### Destino dentro del sandbox — `directories.target`

**Valor inicial:** `/work/data`.

Destino /work/ID con sintaxis minúscula de ID. Hasta ocho destinos únicos por acción. Preparar un destino existente reemplaza su entrada del borrador; el árbol origen se expone en esa ruta del sandbox.

- Ejemplo 1: `/work/reports`.
- Ejemplo 2: `/work/project`.

<a id="directories.access"></a>
### Solo lectura — `directories.access`

**Valor inicial:** `ro`.

Elige solo lectura (ro) o lectura/escritura (rw). El permiso abarca todo el árbol. Escritura permite modificar/borrar archivos del host; sockets del árbol pueden comunicar con servicios del host. No se concede permiso hasta activar.

- Ejemplo 1: `ro`.
- Ejemplo 2: `rw`.

<a id="directories.prepare"></a>
### Preparar y añadir directorio — `directories.prepare`

Verifica identidad de directorio existente y añade/reemplaza destino en borrador local de acción, hasta ocho. No concede acceso hasta aprobar revisión.

- Ejemplo 1: `Preparar árbol de informes solo lectura`.
- Ejemplo 2: `Preparar árbol de salida con escritura explícita`.

<a id="group-credentials"></a>
## credentials

<a id="credentials.type"></a>
### Tipo de credencial — `credentials.type`

**Valor inicial:** `generic`.

Genérica guarda secreto HMAC/bearer. Importar Google guarda autorización existente sin contactar Google. Autorizar Google en navegador inicia consentimiento de diez minutos; no autoriza flujos. Elige el modo correspondiente a la integración.

- Ejemplo 1: `generic: secreto de firma HMAC`.
- Ejemplo 2: `Google OAuth2: autorización en navegador`.

<a id="credentials.id"></a>
### Identificador — `credentials.id`

**Valor inicial:** `""`.

Referencia local con sintaxis de ID de recurso. Reutilizar ID rota valor en el mismo backend. Otro backend se rechaza hasta eliminar la referencia existente; eliminar afecta todos sus consumidores.

- Ejemplo 1: `deploy-token`.
- Ejemplo 2: `google-calendar`.

<a id="credentials.value"></a>
### Valor nuevo (16–8192 bytes) — `credentials.value`

**Valor inicial:** `""`.

Solo genérica: 16–8192 bytes sin CR, LF ni NUL. El campo oculto se limpia al Guardar/Cancelar. Los ejemplos describen orígenes de valores privados; no uses texto público como credencial real. Guarda secreto de firma para HMAC/Slack o token esperado por receptor bearer.

- Ejemplo 1: `secreto privado de firma del emisor de webhooks`.
- Ejemplo 2: `token privado de API del receptor HTTPS`.

<a id="credentials.backend"></a>
### Almacén de credenciales — `credentials.backend`

**Valor inicial:** `keyring`.

El almacén del escritorio usa Secret Service y puede necesitar sesión desbloqueada. Archivo privado guarda explícitamente valor local sin cifrar con permisos restrictivos. Sin fallback automático. Exportar configuración incluye referencias, no estos valores.

- Ejemplo 1: `keyring`.
- Ejemplo 2: `file`.

<a id="credentials.client_id"></a>
### Client ID — `credentials.client_id`

**Valor inicial:** `""`.

Modos Google: ID de tu cliente OAuth Desktop, no correo ni contraseña Google. ASCII imprimible obligatorio sin espacios/controles, hasta 8192 bytes por valor; todo el conjunto guardado tiene también límite de 8192 bytes. Ejemplos son marcadores no funcionales.

- Ejemplo 1: `YOUR_DESKTOP_CLIENT_A.apps.googleusercontent.com`.
- Ejemplo 2: `YOUR_DESKTOP_CLIENT_B.apps.googleusercontent.com`.

<a id="credentials.client_secret"></a>
### Client secret — `credentials.client_secret`

**Valor inicial:** `""`.

Modos Google: secreto del cliente Desktop elegido. Mismos límites ASCII/valor y conjunto que client ID. Oculto y borrado tras Guardar/Cancelar. Introdúcelo localmente; ejemplos son marcadores no utilizables.

- Ejemplo 1: `CLIENT_SECRET_FROM_DESKTOP_CLIENT_A`.
- Ejemplo 2: `CLIENT_SECRET_FROM_DESKTOP_CLIENT_B`.

<a id="credentials.refresh_token"></a>
### Refresh token — `credentials.refresh_token`

**Valor inicial:** `""`.

Solo importación Google: refresh token existente del cliente correspondiente, no access token de corta duración. Guardar no valida scopes con Google. Mismos límites ASCII/tamaño. Ejemplos son marcadores; el valor real va en el campo local oculto.

- Ejemplo 1: `REFRESH_TOKEN_FOR_ACCOUNT_A`.
- Ejemplo 2: `REFRESH_TOKEN_FOR_ACCOUNT_B`.

<a id="credentials.scopes"></a>
### Permisos OAuth — `credentials.scopes`

**Valor inicial:** `""`.

Solo autorización en navegador: 1–16 URLs HTTPS distintas bajo www.googleapis.com/auth/, hasta 256 bytes cada una, separadas por espacios. Sin query, fragmento, credenciales URL ni puerto. La sugerencia calendar.events no es valor inicial seleccionado. Solicita scopes necesarios para tu API; Google debe concederlos todos. Los nombres de ejemplo figuran en la [referencia oficial de scopes de Google Calendar](https://developers.google.com/workspace/calendar/api/auth). Un scope válido no añade soporte GET: las acciones de salida siguen ofreciendo POST/PUT/PATCH.

- Ejemplo 1: `https://www.googleapis.com/auth/calendar.events`.
- Ejemplo 2: `https://www.googleapis.com/auth/calendar.readonly`.

<a id="group-storage"></a>
## storage

<a id="storage.max_events"></a>
### Máximo de eventos — `storage.max_events`

**Valor inicial:** `10000`.

Entero 100–100000. Guardar límites aplica inmediatamente, separado de activar borrador. Alcanzar cuota rechaza eventos nuevos sin borrar trabajo pendiente/incierto. Aumentarla no sustituye resolver una cola atascada.

- Ejemplo 1: `5000`.
- Ejemplo 2: `20000`.

<a id="storage.max_payload_bytes"></a>
### Datos de eventos (MiB) — `storage.max_payload_bytes`

**Valor inicial:** `64 MiB`.

La UI admite entero 1–128 MiB; API guarda bytes (MiB × 1048576). Cuenta cuerpos de eventos y contextos privados de ejecución, no todo SQLite/WAL. Guardar límites no exporta ni sube datos.

- Ejemplo 1: `16 MiB = 16777216 bytes`.
- Ejemplo 2: `128 MiB = 134217728 bytes`.

<a id="storage.retention_days"></a>
### Conservar historial (días) — `storage.retention_days`

**Valor inicial:** `7`.

Entero 1–90 días de historial terminal. Trabajo pendiente/incierto se conserva. Reducir retención afecta limpieza posterior; dedup_days debe ser al menos igual.

- Ejemplo 1: `3`.
- Ejemplo 2: `30`.

<a id="storage.dedup_days"></a>
### Evitar duplicados durante (días) — `storage.dedup_days`

**Valor inicial:** `30`.

Entero entre retention_days y 365. Conserva identidades de entrega independientemente de cuerpos ordinarios. No hace idempotente al receptor HTTP ni acredita frescura de firmas antiguas fuera de la ventana conservada.

- Ejemplo 1: `30`.
- Ejemplo 2: `90`.

<a id="group-panel"></a>
## panel

<a id="panel.language"></a>
### Idioma — `panel.language`

**Valor inicial:** `en`.

English o Español, guardado inmediatamente por perfil. No cambia IDs ni mensajes propios. Requiere conexión con el motor.

- Ejemplo 1: `English para escritorio en inglés`.
- Ejemplo 2: `Español para escritorio en español`.

<a id="panel.save"></a>
### Guardar borrador — `panel.save`

Valida y guarda la configuración editada. No concede capacidades ni cambia revisión activa. El punto indica cambios locales sin guardar; corrige errores antes de revisar.

- Ejemplo 1: `Guardar un flujo de notificación nuevo`.
- Ejemplo 2: `Guardar umbrales de recuperación modificados`.

<a id="panel.activate"></a>
### Revisar y activar — `panel.activate`

Guarda borrador y muestra capacidades exactas y hash. OK activa esa revisión; Cancelar la deja sin activar. Cambiar recursos requiere otra revisión.

- Ejemplo 1: `Aprobar una capacidad de notificación`.
- Ejemplo 2: `Revisar monitor y acción de salida`.

<a id="panel.pause"></a>
### Pausar — `panel.pause`

Alterna Pausar/Reanudar con admisión retain. Pausa detiene despachos posteriores y conserva eventos entrantes; no deshace efectos iniciados. Reanudar deja continuar la cola.

- Ejemplo 1: `Pausar para inspeccionar pendientes`.
- Ejemplo 2: `Reanudar tras restaurar una credencial`.

<a id="panel.stop"></a>
### Detener ejecuciones — `panel.stop`

Abre confirmación. Aceptar cancela pendientes e intenta detener procesos; no deshace efectos enviados y puede dejar resultados inciertos. No rechaza eventos futuros.

- Ejemplo 1: `Cancelar pruebas en cola tras un error`.
- Ejemplo 2: `Detener un script activo prolongado`.

<a id="panel.new"></a>
### Nuevo — `panel.new`

Abre recurso nuevo en la sección actual. Guardar en el editor prepara recurso localmente; Guardar borrador valida toda la configuración. Cancelar abandona ese envío del editor.

- Ejemplo 1: `Crear destino en Conexiones`.
- Ejemplo 2: `Crear intervalo en Programación`.

<a id="panel.edit"></a>
### Editar — `panel.edit`

Abre recurso seleccionado. El ID existente es inmutable en el formulario; referencias y revisiones exactas requieren validación. Guardar no activa.

- Ejemplo 1: `Cambiar mensaje de notificación`.
- Ejemplo 2: `Cambiar zona de programación`.

<a id="panel.remove"></a>
### Quitar — `panel.remove`

Quita recurso de cambios locales del borrador, sin diálogo de confirmación. Guardar valida referencias restantes; activar es separado. Quitar acción referenciada exige actualizar sus flujos.

- Ejemplo 1: `Quitar destino sin uso`.
- Ejemplo 2: `Quitar programación deshabilitada obsoleta`.

<a id="panel.refresh"></a>
### Actualizar — `panel.refresh`

Actualiza estado de sección, como monitores/programaciones, cola o metadatos de seguridad. No activa borrador ni prueba por sí mismo servicio externo.

- Ejemplo 1: `Actualizar tras completar entrega`.
- Ejemplo 2: `Actualizar credenciales tras consentimiento`.

<a id="panel.import"></a>
### Importar — `panel.import`

Lee .json regular absoluto de hasta 1 MiB y reemplaza borrador guardado con orígenes/flujos deshabilitados. No fusiona ni reemplaza revisión activa. Exporta antes cambios que quieras conservar.

- Ejemplo 1: `Importar ejemplo de notificación`.
- Ejemplo 2: `Importar configuración exportada antes`.

<a id="panel.export"></a>
### Exportar — `panel.export`

Requiere borrador guardado. Escribe configuración en .json absoluto nuevo; rechaza sobrescribir. Incluye código, cabeceras ordinarias y cuerpos, pero solo referencias a credenciales guardadas. No sube automáticamente.

- Ejemplo 1: `Exportar antes de importar otra configuración`.
- Ejemplo 2: `Exportar configuración como respaldo local`.

<a id="group-files"></a>
## files

<a id="files.path"></a>
### Ruta de archivo — `files.path`

**Valor inicial:** `""`.

Usada en importación y exportación de configuración/diagnóstico. Importar exige archivo regular existente; exportar nombre nuevo. Rechaza enlace simbólico final. Los directorios padre de ejemplo deben existir.

- Ejemplo 1: `/home/alex/automations-backup.json`.
- Ejemplo 2: `/home/alex/diagnostics-01.json`.

<a id="group-history"></a>
## history

<a id="history.queue"></a>
### Solo cola pendiente / incierta — `history.queue`

**Valor inicial:** `false`.

Desactivado muestra últimas 100 ejecuciones. Activado muestra pendientes, en ejecución e inciertas, por páginas de 100. La cola cambia mientras trabaja; contadores y páginas pueden variar.

- Ejemplo 1: `Activar para inspeccionar entrega atascada`.
- Ejemplo 2: `Desactivar para localizar entrega fallida terminal`.

<a id="history.pages"></a>
### Anterior — `history.pages`

Anterior/Siguiente mueve páginas de cola de 100 si están disponibles; no cambia trabajos. Actualizar puede cambiar contenido de página.

- Ejemplo 1: `Siguiente para inspeccionar después de los primeros 100`.
- Ejemplo 2: `Anterior para volver a la primera página`.

<a id="history.details"></a>
### Detalles — `history.details`

Abre estados de pasos e intentos HTTP sin cuerpos ni credenciales. Un paso completado puede pertenecer a ejecución pendiente. Inspecciona efectos inciertos antes de decidir reintentos.

- Ejemplo 1: `Consultar estado HTTP de un envío fallido`.
- Ejemplo 2: `Identificar paso que perdió permiso`.

<a id="history.retry"></a>
### Reenviar HTTP — `history.retry`

Visible en ejecuciones fallidas/canceladas; el motor solo admite pasos HTTP elegibles. Confirmar reutiliza cuerpo/clave de idempotencia originales. El receptor debe evitar duplicados. Este botón no reintenta acciones no HTTP.

- Ejemplo 1: `Reenviar tras reparar receptor`.
- Ejemplo 2: `Reenviar tras corregir credenciales rechazadas`.

<a id="group-simulation"></a>
## simulation

<a id="simulation.source"></a>
### Origen — `simulation.source`

**Valor inicial:** `local:demo`.

Origen exacto del evento. Simular acepta entradas/monitores/programaciones registradas y local/hook. Prueba real solo admite local: y hook:; no falsifica webhook firmado.

- Ejemplo 1: `local:demo`.
- Ejemplo 2: `monitor:disk-space`.

<a id="simulation.type"></a>
### Tipo — `simulation.type`

**Valor inicial:** `test`.

Tipo de evento usado por condiciones/plantillas. No selecciona por sí solo autenticación de entrada ni acción de sistema.

- Ejemplo 1: `alert`.
- Ejemplo 2: `recovered`.

<a id="simulation.data"></a>
### Datos de ejemplo (JSON) — `simulation.data`

**Valor inicial:** `{"message":"…","state":"failed"}`.

Objeto JSON válido, no array ni texto sin comillas. Usa campos referenciados por condiciones/acciones; evita credenciales en datos de prueba. El mensaje inicial sigue idioma de interfaz.

- Ejemplo 1: `{"message":"Compilación terminada","state":"ok"}`.
- Ejemplo 2: `{"metric":"disk","value":8,"state":"alert"}`.

<a id="simulation.preview"></a>
### Simular sin efectos — `simulation.preview`

Guarda primero borrador modificado y explica flujos coincidentes/rechazados, resolviendo pasos sin ejecutar, consultar credenciales ni crear trabajos. Adaptadores quedan como dependencias sin resolver.

- Ejemplo 1: `Previsualizar cuerpo de notificación`.
- Ejemplo 2: `Comprobar que no coincide condición numérica`.

<a id="simulation.real"></a>
### Ejecutar prueba real — `simulation.real`

Abre confirmación separada y envía evento local/hook a revisión activa. Puede notificar, ejecutar o enviar HTTP con permisos activos. Cancelar no envía evento. No activa cambios sin guardar.

- Ejemplo 1: `Enviar prueba local real de notificación`.
- Ejemplo 2: `Ejercitar flujo hook autorizado`.

<a id="group-security"></a>
## security

<a id="security.create"></a>
### Crear o rotar credencial — `security.create`

Abre campos de credencial. Guardar aplica operación genérica/importación/navegador inmediatamente, independiente de activar borrador. Cancelar limpia campos sensibles y no envía nada.

- Ejemplo 1: `Crear referencia de firma entrante`.
- Ejemplo 2: `Rotar token bearer con su ID existente`.

<a id="security.delete"></a>
### Eliminar — `security.delete`

Abre confirmación de eliminación. Aceptar retira valor/referencia local y hace fallar autenticación dependiente hasta restaurar. No deshace solicitudes completadas ni revoca remotamente consentimiento Google.

- Ejemplo 1: `Eliminar credencial API sin uso`.
- Ejemplo 2: `Retirar conexión Google localmente`.

<a id="security.revoke"></a>
### Revocar — `security.revoke`

Revoca inmediatamente capacidad activa elegida, sin diálogo de confirmación. Impide futuros despachos bajo ese permiso, no efectos completados. Revocar monitor/programación detiene eventos nuevos, no permisos de acciones ya encoladas.

- Ejemplo 1: `Revocar permiso de acción de salida`.
- Ejemplo 2: `Revocar capacidad de eventos de programación`.

<a id="security.oauth_open"></a>
### Abrir consentimiento en el navegador — `security.oauth_open`

Disponible mientras sesión Google espera consentimiento. Abre URL del proveedor en navegador; revisa allí permisos pedidos. Caduca a los diez minutos. No compartas URL/código como diagnóstico.

- Ejemplo 1: `Abrir consentimiento de conexión nueva`.
- Ejemplo 2: `Reabrir sesión en espera tras recargar panel`.

<a id="security.oauth_cancel"></a>
### Cancelar autorización — `security.oauth_cancel`

Cancela sesión local pendiente de consentimiento/canje y cierra listener. No revoca conexión ya guardada con éxito. Volver a iniciar crea sesión nueva; reiniciar motor también exige empezar otra.

- Ejemplo 1: `Cancelar tras elegir cliente incorrecto`.
- Ejemplo 2: `Cancelar autorización que ya no necesitas`.

<a id="security.oauth_status"></a>
### Estado OAuth2 — `security.oauth_status`

Muestra metadatos locales, incluidos estados de renovación/reconexión. No contacta Google ni prueba validez remota actual. Scopes ausentes/rechazados o invalid_grant pueden exigir reconectar.

- Ejemplo 1: `Consultar tras consentimiento exitoso`.
- Ejemplo 2: `Consultar tras HTTP 401`.

<a id="security.diagnostics"></a>
### Ver diagnóstico — `security.diagnostics`

Muestra versiones, contadores, pausa/almacenamiento y disponibilidad de herramientas. Excluye configuración, URLs, rutas, cuerpos y valores secretos. Tener herramienta no demuestra servicio operativo.

- Ejemplo 1: `Comprobar versiones tras actualizar`.
- Ejemplo 2: `Inspeccionar cola/almacenamiento durante incidente`.

<a id="security.export_diagnostics"></a>
### Exportar diagnóstico — `security.export_diagnostics`

Escribe diagnóstico acotado en .json absoluto nuevo con permisos 0600, sin sobrescribir ni subir. Un fallo puede dejar archivo incompleto; elige otra ruta nueva tras revisar error.

- Ejemplo 1: `Guardar informe antes de actualizar`.
- Ejemplo 2: `Guardar informe tras entrega fallida`.

<a id="security.save_limits"></a>
### Guardar límites — `security.save_limits`

Aplica inmediatamente cuatro límites, separado del borrador/revisión. Rechaza combinaciones inválidas. No borra trabajo pendiente/incierto para cumplir cuota nueva.

- Ejemplo 1: `Ampliar capacidad tras inspeccionar crecimiento de cola`.
- Ejemplo 2: `Reducir historial terminal conservando deduplicación mayor`.

<a id="group-code"></a>
## code

<a id="code.prepare"></a>
### Preparar revisión — `code.prepare`

Lee script/adaptador local y prepara código y revisión exacta. Revisa antes de Guardar. Cambiar campos invalida preparación y exige repetir. Nunca ejecuta ni concede capacidades.

- Ejemplo 1: `Preparar script Python con parámetros tipados`.
- Ejemplo 2: `Preparar manifiesto/código de adaptador modificados`.

<a id="group-administration"></a>
## administration

<a id="administration.check"></a>
### Comprobar permisos administrativos — `administration.check`

Para system-service: comprueba unidad/operación exactas con broker opcional sin ejecutar systemctl ni autorizar flujo. Autorizado no acredita que exista unidad ni garantiza permiso futuro; ejecución vuelve a comprobar política/Polkit.

- Ejemplo 1: `Comprobar autorización status de backup.service`.
- Ejemplo 2: `Comprobar autorización restart de worker.service`.
