# Omarchy Automations — propuesta de plugin para Omarchy

Estado: diseño para revisión; no se han instalado servicios ni modificado el escritorio.
Fecha: 2026-09-28. Nombre definitivo: Omarchy Automations. Identificador de instalación conservado por compatibilidad: `quatrro.automations`.

## 1. Objetivo y base comprobada

Permitir que el usuario conecte webhooks entrantes y salientes con notificaciones, servicios, comandos y monitoreo del sistema mediante flujos configurables, auditables y revocables.

La instalación consultada informa Omarchy `4.0.4-1`. La integración utiliza las APIs del shell instalado; el nombre definitivo del plugin es Omarchy Automations.

Se revisaron la skill Omarchy, sus guías `plugins.md` y `hooks.md`, y la documentación instalada en `/usr/share/omarchy/shell/README.md` y `/usr/share/omarchy/shell/plugins/README.md`.

Hechos relevantes:

- El escritorio usa Quickshell/QML; admite plugins con `manifest.json`, paneles, widgets y servicios.
- Los plugins de usuario viven en `~/.config/omarchy/plugins/<id>/` y su código puede recargarse al guardar.
- Los plugins se ejecutan sin sandbox dentro del proceso del shell. Sus interfaces restringidas no equivalen a aislamiento del sistema operativo.
- `omarchy plugin add` clona y valida el plugin, pero no ejecuta instaladores ni instala su backend.
- Existen hooks `battery-low`, `font-set`, `post-boot`, `post-update`, `pre-refresh-pacman` y `theme-set`; se instalan con `omarchy hook install`.
- `/usr/share/omarchy/` se consulta, pero no se modifica.

Pendiente del prototipo: comprobar el gestor systemd del usuario, D-Bus de sesión, disponibilidad del almacén de secretos y restricciones efectivas de sandbox. La invocación `systemd --version` no estuvo disponible en el PATH del entorno de inspección; esto no demuestra ausencia de systemd en el host.

## 2. Arquitectura propuesta

```text
Proveedores externos ── HTTPS / autenticación ── receptor de webhooks
Hooks Omarchy / monitores / temporizadores ──── eventos locales
                                                    │
                                  validación → cola persistente
                                                    │
                                      reglas y autorizaciones
                                                    │
                              acciones registradas / envíos HTTP
                                                    │
                                      historial y resultados

Plugin QML ↔ CLI / socket Unix de administración ↔ motor local
```

Componentes:

1. **Plugin QML:** widget de estado y panel con conexiones, flujos, monitores, historial y permisos. No recibe tráfico externo ni guarda secretos.
2. **Motor `quatrrod`:** servicio de usuario con receptor HTTP, validación, reglas, cola SQLite, programación y entrega de webhooks. Debe sobrevivir a recargas del plugin.
3. **CLI `quatrroctl`:** administra, valida, simula y consulta el motor mediante socket Unix bajo `$XDG_RUNTIME_DIR`. También permite introducir eventos desde hooks.
4. **Ejecutor:** acciones tipadas y procesos por trabajo con límites de tiempo, recursos, archivos y red. No expone una API de shell remoto.
5. **Adaptadores:** entradas, salidas y acciones con contratos versionados. Las extensiones ejecutables son código confiable instalado localmente, nunca código recibido en un webhook.
6. **Broker privilegiado opcional y posterior:** operaciones administrativas concretas mediante políticas Polkit y un servicio separado. Sin shell root genérico.

Tecnología propuesta: Go para motor y CLI, QML para interfaz y SQLite para estado transaccional. Es una elección de diseño orientada a distribución sencilla y concurrencia; las dependencias y versiones se fijarán al implementar.

La separación de procesos mejora disponibilidad y reduce acoplamiento. Ejecutar todo con el mismo UID no constituye una frontera fuerte frente a malware local o un plugin malicioso; una futura instalación de mayor aislamiento necesitará identidades y políticas separadas.

## 3. Modelo de configuración

Entidades principales:

| Entidad | Contenido |
|---|---|
| Entrada | Identidad, ruta, método, autenticación, verificador, esquema y límites |
| Destino | URL aprobada, método, cabeceras, credencial referenciada y política de red |
| Evento | ID, origen, tipo, fecha, datos, correlación y profundidad de ejecución |
| Acción | Tipo, parámetros tipados, recursos autorizados, timeout y efectos esperados |
| Flujo | Disparador, condiciones, pasos, política de errores y revisión aprobada |
| Monitor | Métrica/evento, frecuencia, umbral, duración, histéresis y recuperación |
| Ejecución | Revisión del flujo, pasos, intentos, resultados y decisiones de autorización |

Un flujo tiene borrador y revisión activa inmutable. Guardar valida; simular muestra condiciones, parámetros y permisos; activar publica una revisión. Las ejecuciones existentes conservan su revisión. Revocar permisos se aplica también a trabajos pendientes y se comprueba antes de cada efecto.

Condiciones declarativas y acotadas: igualdad, comparaciones, pertenencia y combinaciones lógicas. Mapeo de campos con valores tipados y codificación según destino. Sin `eval`, JavaScript arbitrario ni sustitución de shell. Se limita tamaño, profundidad y coste de evaluación.

Los campos sensibles de configuración —ejecutable, unidad, destino HTTP, credencial y política— no pueden sobrescribirse desde un payload. Los parámetros variables deben cumplir el esquema de la acción.

## 4. Webhooks de entrada

- Rutas por integración; no usar el secreto como parte de la URL.
- HTTP con métodos y tipos de contenido definidos por entrada. MVP: POST con JSON, formularios o cuerpo crudo acotado. XML y multipart se incorporan mediante adaptadores con parsers restringidos y límites propios.
- HMAC-SHA256 sobre bytes originales, comparación de tiempo constante y verificadores específicos por proveedor; Bearer sobre TLS cuando el emisor no admite firma.
- Timestamp firmado y ventana de validez cuando el protocolo los ofrece. Si no los ofrece, deduplicación por identificador y retención configurada, documentando que no equivale a una garantía universal contra replay.
- Autenticación antes de normalizar o poner el evento en cola; tamaño máximo aplicado desde la lectura del cuerpo.
- Límites por entrada y globales: solicitudes, concurrencia, tamaño, timeout y espacio de cola.
- Respuesta 202 únicamente después de persistencia satisfactoria; ante disco lleno o fallo de persistencia, devolver error recuperable. Adaptadores pueden ajustar el código y cuerpo al contrato del proveedor.
- Respuestas de desafío y validaciones iniciales mediante lógica específica, sin ejecutar acciones del usuario dentro de la petición.
- No devolver stdout de comandos ni secretos en la respuesta del webhook.

Escuchar en loopback por defecto. Para LAN o Internet, configurar exposición explícita con TLS mediante proxy o túnel y un perfil de confianza. Un túnel no sustituye la autenticación. La API administrativa nunca se publica junto al receptor.

«Cualquier tipo» se aborda como extensibilidad: un núcleo HTTP genérico y adaptadores para firmas, desafíos y formatos particulares. No se promete compatibilidad automática con todos los proveedores. MQTT, WebSocket y otros protocolos serían conectores de eventos adicionales.

## 5. Webhooks de salida

- Destinos registrados con método, cabeceras, cuerpo mapeado y autenticación por referencia a secreto. HTTPS por defecto.
- JSON, formularios y cuerpo crudo en MVP; HMAC o Bearer y adaptadores para otras autenticaciones. OAuth2 queda para una fase posterior.
- Un evento remoto no puede convertir un campo `url` en un destino libre. El flujo selecciona un destino autorizado.
- Política contra SSRF: validar esquema, hostname, puerto y direcciones IPv4/IPv6; comprobar resolución y conectar a una IP validada manteniendo verificación TLS del hostname. Revalidar al reconectar.
- Redirecciones desactivadas por defecto; si se habilitan, revalidar cada salto y no reenviar credenciales a otro origen.
- Bloquear por defecto loopback, link-local, metadata y redes privadas. Permitir servicios LAN locales mediante excepciones explícitas por destino, sin desactivar la protección global.
- Cola duradera, backoff exponencial con jitter, `Retry-After` acotado, máximo de intentos y vencimiento total. Reintentar errores de transporte y estados recuperables; los restantes pasan a revisión.
- Clave de idempotencia estable cuando el receptor la soporte. Entrega al menos una vez; no prometer exactamente una vez entre sistemas independientes.
- Cola de fallos con inspección y reenvío manual. Limitar tamaño de respuestas y no conservar cuerpos sensibles por defecto.

## 6. Acciones y permisos

| Acción | Comportamiento inicial | Permiso |
|---|---|---|
| Notificación | Título/cuerpo acotados, texto sin markup remoto | Notificar |
| Métrica | Leer CPU, RAM, disco o batería | Métricas seleccionadas |
| Servicio de usuario | Consultar, iniciar, detener o reiniciar unidades concretas | Unidad + operación |
| Comando | Ejecutable local registrado y argumentos validados | Perfil de comando |
| Omarchy | Acciones concretas del catálogo local `omarchy` | Acción seleccionada |
| HTTP saliente | Enviar a un destino registrado | Destino + datos permitidos |
| Servicio de sistema | Operación administrativa concreta | Broker + política específica |

Las autorizaciones se conceden al crear o activar el flujo, no en cada ejecución rutinaria. Los cambios que amplían capacidades requieren nueva autorización local. Entradas HTTP solo disparan flujos ya autorizados; no crean acciones ni modifican políticas.

Para comandos:

- Ejecutable y directorio de trabajo fijos; argumentos en arreglo, sin `sh -c` automático.
- Un arreglo de argumentos evita expansión del shell, pero no vuelve seguro cualquier programa. Restringir opciones e intérpretes y validar semánticamente parámetros de cada acción.
- Entorno mínimo; secretos fuera de argv; stdin o mecanismo de credenciales según acción.
- Timeout, terminación de todo el grupo de procesos/cgroup, límite de salida, memoria, CPU y concurrencia.
- Acceso a archivos y red deshabilitado o acotado por perfil, con pruebas de que el aislamiento se aplica realmente.
- Modo avanzado: scripts locales versionados y revisados, con parámetros estructurados. Cambios de contenido invalidan su autorización; ejecutar la revisión aprobada evitando carreras entre validación y ejecución.
- Si una acción exige un aislamiento que el host no puede aplicar, rechazar su activación; no degradar silenciosamente.

No ejecutar el motor como root. No usar `sudo` sin contraseña de alcance general. Las operaciones administrativas automáticas necesitan una autorización previamente instalada y estrecha; las interactivas pueden usar Polkit. Un webhook no debe provocar diálogos de privilegios repetitivos. Para instalación privilegiada sin terminal se seguirá la guía Omarchy de `pkexec`.

## 7. Secretos y frontera de confianza

- Almacén de secretos del escritorio si está disponible; comprobar su funcionamiento al iniciar sesión y al ejecutar sin sesión gráfica.
- Configuración exportable contiene referencias, nunca valores. No guardar secretos en `shell.json`, repositorio, historial o logs.
- Alternativa explícita: archivo propietario 0600 en directorio 0700, con limitación claramente visible: protege mediante permisos, no cifra frente al mismo UID. No fingir cifrado guardando clave y ciphertext juntos.
- Credenciales por conexión, rotación y revocación independientes. Si el almacén está bloqueado, pausar la conexión afectada.
- Socket administrativo con permisos del usuario y verificación de credenciales del peer; sigue confiando en procesos del mismo usuario.
- Separar contrato de ingestión local del de administración; validar origen y permisos de eventos internos sin confiar en campos que se autodeclaren privilegiados.
- Redactar secretos conocidos y evitar registrar cuerpos y stdout completos. Lo que se envía hacia fuera se define por campo, con vista previa de datos.
- Auditoría local para diagnóstico; no se presenta como inalterable frente al dueño de la cuenta o root.

## 8. Monitoreo y eventos del sistema

MVP: CPU, RAM, espacio libre, batería y estado de unidades systemd de usuario. Recoger métricas con APIs del sistema y archivos adecuados; evitar lanzar herramientas pesadas en bucle.

Monitores configurables: intervalo mínimo, ventana temporal, umbral, recuperación, histéresis, cooldown y máximo de alertas. Ejemplo: CPU superior al 90% durante dos minutos; recuperación debajo del 75% durante un minuto.

Preferir suscripciones de eventos para cambios de servicio, energía y red cuando existan. Los hooks Omarchy envían un evento pequeño mediante `quatrroctl emit`, con timeout corto; un motor caído no bloquea el arranque ni una actualización. La garantía de persistencia empieza cuando el motor confirma recepción; un spool local opcional puede cubrir caídas del motor más adelante.

Fases posteriores: procesos, cambios de archivos con rutas aprobadas, conectividad y temperatura cuando el hardware la exponga. Journald requiere comprobar permisos; no solicitar acceso global por defecto.

Eventos de bloqueo o actividad pueden transportar información privada: deben ser opcionales y tener un contrato verificado con la instalación concreta.

## 9. Experiencia de usuario

Panel nativo con cinco secciones:

1. Conexiones: entrada/salida, autenticación, exposición y prueba.
2. Flujos: asistente «cuando → si → ejecutar → enviar», con pasos ordenados.
3. Monitores: valores actuales, condiciones, recuperación y últimas alertas.
4. Historial: recorrido del evento, revisión, resultado, errores e intentos, con datos redactados.
5. Seguridad: capacidades otorgadas, secretos referenciados y revocación.

Cada flujo permite guardar borrador, simular sin efectos, realizar una prueba real explícita y activar. La simulación muestra datos mapeados y capacidades necesarias sin leer secretos ni ejecutar comandos.

El widget muestra salud, cola y fallos. «Pausar todo» detiene nuevos efectos y reintentos; permite elegir si retener o rechazar entradas. «Detener ejecuciones» cancela procesos activos, explicando que los efectos ya realizados o solicitudes remotas enviadas no se pueden deshacer.

Deshabilitar el plugin visual no equivale a detener el backend. El panel y el desinstalador deben mostrar esta diferencia y ofrecer detener/deshabilitar el servicio.

## 10. Ejemplos de flujos

- GitHub informa un despliegue fallido → verificar firma → filtrar repositorio y entorno → notificar → enviar resumen a un destino registrado.
- Espacio libre inferior al 10% durante cinco minutos → notificar → emitir webhook; al recuperarse, enviar un evento distinto.
- Webhook autenticado de mantenimiento → reiniciar únicamente `mi-app.service` del usuario → comprobar estado real → emitir resultado a un destino fijado localmente.
- Hook `theme-set` → evento con nombre de tema → envío al sistema domótico autorizado en LAN.
- Temporizador → script local de respaldo autorizado → notificación y webhook de éxito/fallo con campos de resultado seleccionados.

## 11. Persistencia y fallos

- SQLite transaccional para inbox, ejecuciones, outbox e historial; límites de disco, retención y limpieza definidos.
- Estados explícitos: pendiente, ejecutando, completado, fallido, cancelado e incierto.
- Tras una caída, no repetir ciegamente comandos con efectos: consultar estado o requerir revisión cuando no sea posible saber si terminaron.
- Reservas de trabajos y recuperación al arrancar; limitar concurrencia por flujo y recurso para impedir reinicios simultáneos.
- Fallos parciales conservan resultado por paso. Sin rollback universal; compensaciones solo si se configuran expresamente.
- ID de correlación, profundidad máxima y cooldown para evitar bucles entre flujos. No es posible detectar todos los ciclos en servicios externos que descartan la correlación.
- Backpressure y cuotas; informar sobre cola llena y sobre pérdida de eventos de monitores. Las métricas repetidas pueden consolidarse, los trabajos de usuario no se descartan silenciosamente.
- Suspensión y reanudación: decidir vencimiento de eventos y ejecución de temporizadores atrasados. Por defecto consolidar muestras y evitar tormentas de tareas pendientes.
- Dependencia gráfica explícita: notificaciones requieren sesión. Sin sesión, registrar o aplazar según política. Mantener el servicio tras logout mediante linger es una opción posterior explícita.

## 12. Distribución e integración Omarchy

Repositorio propuesto:

```text
manifest.json
Widget.qml
Panel.qml
qml/
cmd/quatrrod/
cmd/quatrroctl/
internal/{ingress,egress,rules,actions,monitors,policy,store}/
schemas/
packaging/systemd/
hooks/
examples/
tests/
docs/
```

El manifiesto declarará `bar-widget` y `panel`, con entrypoints compatibles con el esquema instalado; validar con `omarchy plugin validate` en el prototipo. No inventar claves de manifest sin verificarlas contra `PluginRegistry.qml`.

Instalación en dos componentes: plugin por el mecanismo de Omarchy y backend mediante paquete o instalador explícito de usuario. La descarga del plugin no ejecutará instalación encubierta. UI y backend negocian versión de protocolo y muestran incompatibilidad sin ejecutar acciones.

Rutas previstas:

- `~/.config/omarchy/plugins/quatrro.automations/`: interfaz y fuentes del plugin.
- `~/.config/omarchy/shell.json`: únicamente posición, activación y ajustes propios del widget/panel siguiendo el contrato Omarchy.
- `$XDG_CONFIG_HOME/quatrro/`: configuración del backend, esquemas y políticas.
- `$XDG_STATE_HOME/quatrro/`: base de datos e historial.
- `$XDG_RUNTIME_DIR/quatrro/`: sockets efímeros.
- `~/.config/systemd/user/quatrrod.service`: unidad del servicio si se instala por usuario.
- `~/.config/omarchy/hooks/<evento>.d/`: adaptadores instalados mediante `omarchy hook install`.

Aplicar valores XDG por defecto cuando las variables estén ausentes. Copias de seguridad antes de modificar configuración existente; no reemplazar el `shell.json` del usuario. La configuración separada pertenece al backend independiente, no duplica las preferencias del plugin.

Desinstalación: detener y deshabilitar backend, retirar únicamente hooks propios, desinstalar plugin y ofrecer conservar o eliminar estado y secretos por separado. No borrar otros hooks ni configuración ajena.

## 13. Fases y criterios de aceptación

### Fase 0 — contratos y viabilidad

Prototipo de panel/widget, comunicación QML–CLI–socket, notificación y servicio de usuario. Verificar capacidades de sandbox y almacén de secretos en el host. Inspeccionar esquema real del manifiesto y definir modelos versionados.

Aceptación: el plugin valida, la UI se recarga sin interrumpir el motor y no se modifica `/usr/share/omarchy/`. Las limitaciones detectadas quedan registradas antes de activar comandos.

### Fase 1 — MVP seguro de extremo a extremo

Entrada HTTP autenticada, salida HTTPS, cola persistente, flujos secuenciales, simulación, notificaciones, métricas básicas y catálogo reducido de comandos y unidades de usuario. Sin operaciones root, código remoto ni editor visual de grafos.

Aceptación: configurar desde UI un webhook que notifica y emite una salida; configurar un monitor que alerta y se recupera; ejecutar una acción previamente autorizada; pausar y revocar; conservar cola tras reinicio.

### Fase 2 — integraciones y operación continua

Hooks Omarchy, verificadores de proveedores, temporizadores, conectores adicionales, gestión de fallos, exportación/importación sin secretos, rotación de credenciales y herramientas de diagnóstico. Importaciones llegan desactivadas y sin permisos otorgados.

Aceptación: comportamiento reproducible ante desconexión, suspensión, repetición de eventos, destinos caídos y actualización de configuración durante una ejecución.

### Fase 3 — extensibilidad avanzada

Scripts con revisiones aprobadas, adaptadores en procesos separados, OAuth2, recursos LAN con políticas específicas y broker administrativo opcional. Evaluar una interfaz de grafos solo cuando la complejidad de los flujos la justifique.

Aceptación: cada extensión declara capacidades; el broker rechaza unidades, argumentos y operaciones fuera de política; ninguna ampliación de permisos se aplica por recibir datos remotos.

## 14. Pruebas necesarias antes de distribuir

- Firmas correctas/incorrectas, cuerpo alterado, revocación, timestamp cuando aplique y replay.
- Inyección de shell y de opciones, parámetros fuera de esquema, cambios de scripts tras aprobación y revocación de trabajos en cola.
- SSRF con IPv4/IPv6, DNS cambiante, redirecciones y destinos privados explícitamente permitidos.
- Caídas entre persistencia/ejecución/envío; duplicados; acciones inciertas; backpressure y disco lleno.
- Timeout y cancelación de procesos hijos, límites efectivos de recursos y denegación ante sandbox no disponible.
- Redacción de secretos, permisos de archivos/socket y exportación sin credenciales.
- Recarga QML, reinicio del shell, actualización de backend y migración/rollback de configuración y datos.
- Cierre y bloqueo de sesión, suspensión, reanudación y almacén de secretos bloqueado.
- Desinstalación que deja intactos otros plugins, hooks y preferencias.

## 15. Referencias

- Skill local: `/home/macondo/.codex/skills/omarchy/SKILL.md`, `plugins.md` y `hooks.md`.
- Contrato de plugins instalado: `/usr/share/omarchy/shell/README.md`; esquema: `/usr/share/omarchy/shell/services/PluginRegistry.qml`.
- GitHub, validación de entregas: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
- OWASP, prevención de SSRF: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- systemd, opciones y limitaciones de ejecución: https://github.com/systemd/systemd/blob/main/man/systemd.exec.xml

Las firmas deben respetar los bytes y el contrato del proveedor. Las defensas SSRF deben abarcar resolución y redirecciones. El aislamiento de unidades de usuario depende del host y debe verificarse, no asumirse por declarar opciones en un archivo de unidad.
