# Registro de validación

## 2026-09-28 — Base

- `omarchy plugin validate .`: código 0.
- `systemctl --version`: systemd 261.2-1-arch.
- `systemctl --user is-system-running` en host: running.
- `busctl --user list`: notificaciones y Secret Service presentes (sin consultar valores).
- `bwrap --unshare-all --ro-bind /usr /usr --symlink usr/lib /lib --symlink usr/lib /lib64 --proc /proc --dev /dev /usr/bin/true`: código 0 en host.
- Archivo Go 1.26.8 linux-amd64: SHA256 `d0f743b33e8d8945e6b1f432edd15785c70507121d6e2a723b21285eddf8b57b`, coincide con metadatos HTTPS de go.dev.
- `qmllint`: código 0; advertencia de metadatos de Quickshell sobre QProcess::ExitStatus. Falta prueba de runtime para cerrar integración gráfica.

No se han validado todavía todas las fases; el roadmap conserva sus casillas pendientes hasta demostrar cada requisito.

## 2026-09-28 — Núcleo y host

- `make build` usando Go 1.26.8: ambos binarios compilados.
- `go test -race ./...` fuera del sandbox: pasa. Pruebas de socket, singleton, revisiones, cambio de permisos, persistencia tras reapertura, inbox y firmas.
- Vector de firma GitHub contrastado con documentación oficial; cuerpo alterado, timestamp vencido y cuerpo mayor de 256 KiB rechazados.
- Servidor TLS local de prueba: conexión privada bloqueada por defecto, excepción exacta funciona con CA de prueba y redirección 302 no se sigue; resolución mixta pública/privada bloqueada.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa incluyendo notificación entregada al servicio, reinicio de una unidad temporal propia y ejecución Bubblewrap/systemd; escritura en `/usr` rechazada. No se modifican servicios existentes.
- `scripts/smoke-native.py`: panel carga sin error QML y render interno muestra motor conectado. Inspección inicial mediante captura de escritorio no fue válida porque la sesión estaba bloqueada; se sustituyó por render del contenido propio. No se desbloqueó la sesión.
- Render de desarrollo en `.dev/panel-render.png`, excluido de Git. La advertencia del portal sobre registro de app no impide cargar el panel.
- Pendientes: interacción completa de UI, barra Omarchy real, servicio backend persistente, límites CPU/memoria/red comprobados individualmente, pruebas de crash/retry completas y resto de fases.

## 2026-09-28 — Formularios y controles del motor

- `scripts/test-ui-flow.py`: pasa con formularios de entrada, destino, acciones, flujo, monitor y credencial; tres capacidades revisadas y activadas. Recibe una entrega HTTPS y verifica muestra real de disco e historial completado. Es automatización de señales de formularios QML, no prueba de puntero/teclado manual ni prueba de la barra instalada.
- Renders fuera de pantalla revisados; archivos anteriores se eliminan antes de producir nuevos. Resuelto un fallo de reutilización de campos entre tipos y un problema de ancho de navegación.
- `go test -race` con modo host: pruebas de Secret Service, CPU/RAM/disco, notificación, servicio temporal, comando aislado y lectura de tema Omarchy. Se elimina la credencial temporal al terminar.
- Nuevas pruebas de migración, rechazo de versión futura, retención sin perder trabajos pendientes, cuota de payload, SQLite lleno sin commit parcial, transición atómica de monitor, duración continua de recuperación, referencia tipada JSON y codificación formulario/raw.
- Nuevas pruebas de cancelación HTTP, recuperación de envío ya confirmado, persistencia de clave/cuerpo/retry-after entre reaperturas y rechazo de repetición de entrega completada.
- Entrada GitHub deduplica por cuerpo autenticado; cambiar delivery header no cambia identidad. El tipo normalizado procede de cuerpo autenticado o `webhook`, nunca de cabeceras no firmadas.
- Permisos independientes para monitores incluidos en la revisión de activación. Esquemas estructurales de configuración/evento en `schemas/`; validación semántica adicional en Go.
- `go vet ./...` y validación de manifiesto Omarchy: pasan. `qmllint` conserva advertencias de acceso a propiedades en delegates y metadatos Quickshell, sin errores sintácticos; la carga QML y el recorrido funcional se comprueban en runtime.

## 2026-09-28 — Hooks y permisos

- `go test -race ./internal/hooks ./internal/core`: pasa; incluye backend ausente, backend conectado sin respuesta y separación entre permisos de monitor y acción con nombres coincidentes.
- `make build`: motor y CLI compilados con Go 1.26.8.
- `python3 scripts/test-hooks.py`: pasa. Usa `/usr/share/omarchy/bin/omarchy-hook` sin modificarlo, HOME temporal y perfil aislado; verifica los seis eventos en SQLite, tipado de porcentaje y argumentos literales, conservación y ejecución de hook ajeno. Tiempo total de seis invocaciones con motor detenido: 0,070 s.
- Los hooks reales del usuario no se modificaron. La instalación permanente queda para las tareas de integración y empaquetado.

## 2026-09-28 — Formatos XML y multipart

- `go test -race ./...`: pasa tras introducir los dos parsers y acceso por índice a arrays.
- `TestXMLNormalizationAndLimits`: namespaces, entidades predefinidas, hijos repetidos y rechazo de DTD, entidades externas/personalizadas, múltiples raíces, atributos duplicados, procesamiento, profundidad y exceso de tokens/elementos/tamaño.
- `TestMultipartNormalizationAndLimits`: texto y binario base64; rechaza campos/archivos grandes, nombres duplicados, traversal Unix/Windows, UTF-8 inválido, más de 16 partes, truncamiento y falta de boundary.
- `TestStructuredFormatsPersistOnlyAuthenticatedInput`: cada formato integrado con el handler, una entrada no autenticada rechazada y exactamente un evento autenticado persistido.
- `python3 scripts/test-ui-flow.py`: pasa; crea entradas XML/multipart mediante el editor QML y comprueba los formatos guardados. Conserva una entrega HTTPS real, notificación e historial del flujo de prueba.

## 2026-09-28 — Proveedores incorporados

- `go test -race ./...`: pasa tras separar el contrato de proveedores. Vectores oficiales GitHub y Slack verificados; timestamps de fixtures controlados en las pruebas.
- Slack: firma sobre bytes originales, desafío válido devuelto sin ejecuciones, desafío no autenticado rechazado sin reflejar contenido, desafío malformado rechazado, evento persistido y retry con timestamp/cabeceras nuevos deduplicado.
- Trigger SQLite de fallo impide commit y obtiene HTTP 503 en lugar de confirmar el evento. Verificación de versiones de firma, proveedor desconocido y cabeceras repetidas; formato Slack XML rechazado y formulario permitido.
- `make build`, `go vet ./...`, `omarchy plugin validate .`: pasan.
- `scripts/test-ui-flow.py`: pasa incluyendo el selector Slack. Las peticiones Slack/GitHub son fixtures locales; no se conectaron cuentas externas.

## 2026-09-28 — Programación persistente

- `go test -race ./...`: pasa con intervalos, calendario, zona horaria y permiso independiente. Migración desde esquema 2 a 3 comprobada.
- Trigger de fallo al actualizar timer_state prueba rollback de evento y estado. Reapertura real del Engine conserva próxima ejecución; retroceso del reloj no duplica; revocación detiene eventos y reactivación aplica la política pendiente.
- Reloj controlado: `coalesce` acumula atrasos en un solo evento, `skip` descarta vencimientos fuera de tolerancia y la fase del intervalo se conserva. Pausa con rechazo no consume la ocurrencia; retención guarda ejecución pendiente.
- Calendarios: Bogotá/UTC, filtro semanal, hora inexistente en primavera y repetida en otoño de Nueva York. Scheduler integrado genera una sola ejecución en la hora repetida; cambiar calendario aplica una próxima ocurrencia nueva.
- UI QML: crea intervalo y calendario, revisa cuatro capacidades, observa un disparo real y muestra próximo vencimiento. `.dev/ui-timers.png` revisada; fechas numéricas y estados en español. La prueba conserva notificación, HTTPS e historial.
- `make build`, `go vet ./...` y validación de manifiesto: pasan. Esta evidencia no cierra las tareas pendientes de barra real, empaquetado ni funciones avanzadas.

## 2026-09-28 — Aislamiento real del ejecutor

- Consulta previa: `DefaultTimeoutStopUSec=1min 30s`. Corregido en unidades propias mediante parada de 1 s, KillMode de grupo, SIGKILL y limpieza tras error/cancelación.
- `TestHostIsolationEffectiveCgroupAndCPU`: lee memory.max=268435456, memory.swap.max=0, pids.max=32 y cuota CPU/periodo=0,5. Bucle ocupado incrementa nr_throttled y mantiene uso bajo el umbral de tolerancia medido; el cgroup desaparece tras cancelar.
- `TestHostIsolationMemoryAndTaskLimits`: asignación acotada de 300 MiB termina con evidencia `oom-kill` en journal propio; fork acotado a 40 intentos alcanza el máximo de tareas y limpia sus hijos.
- `TestHostIsolationFilesystemNetworkAndEnvironment`: bandera ST_RDONLY en /usr, archivo temporal del host inaccesible, /etc y /run/user ausentes, NoNewPrivs=1, entorno limpio y listener loopback del host inaccesible.
- `TestHostIsolationTimeoutAndDescendantCancellation`: timeout y cancelación terminan padre/hijo, incluso con SIGTERM ignorado y setsid. Ambas pruebas pasan dentro de sus plazos; no permanecen PID.
- Preflight real antes de activar comando: ejecuta perfil de producción y comprueba los controles del kernel. Pruebas negativas de límites ausentes/ilimitados, cancelación de preflight, NUL y ruta no canónica.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa; core en 10,256 s. Listado final de unidades quatrro-isolation-test/quatrro-probe vacío.

## 2026-09-28 — Operación, diagnóstico y portabilidad

- `TestDiagnosticsAllowlistAndExclusiveExport`: coloca marcadores sensibles en credencial, payload, URL, cabecera, título/nombre y error arbitrario. Ninguno aparece en el reporte; exportación 0600 sin sobrescritura ni seguimiento de symlink final.
- `TestQueueInspectionPaginationAndRedaction`: 105 trabajos en dos páginas, filtro incierto y rechazo de parámetros inválidos; cuerpos ausentes del resultado.
- `TestImportRejectsFIFOAndSymlinkWithoutBlocking`: FIFO, enlace y archivo mayor de 1 MiB rechazados. El lector usa un descriptor único y límite aplicado a la lectura.
- `TestCredentialRotationImmediatelyChangesIngress`: autenticación con valor original, rechazo tras rotación, aceptación del nuevo y fallo después de eliminación.
- `go test -race ./...`, `go vet ./...`, compilación y validación Omarchy: pasan.
- Recorrido QML: cola pendiente visible durante pausa, diagnóstico exportado desde diálogo y comprobado sin token/cuerpo/destino de prueba; archivo 0600; cancelación retira el trabajo y mantiene una sola entrega HTTPS. Captura nueva `.dev/ui-queue.png` revisada.

## 2026-09-28 — Exposición TLS

- Caddy v2.11.4 linux-amd64, artefacto oficial: SHA512 `8220d1f013b6f27510247b2360c9e0ca9f018feebd82515f07635318b34ff9777ccc8fd0b6e6f2486ce3a33fe389fbb7db12d05baa474f4587509fb4f5ebf1c9`, comprobado antes de ejecutar. Herramienta en `/home/macondo/Work/.quatrro-tools/caddy/caddy`.
- `CADDY=... python3 scripts/test-tls-proxy.py`: pasa con Caddyfile del repositorio. Adaptación JSON confirma administración deshabilitada, sin persistencia automática y listener único loopback.
- TLS 1.2 con CA de prueba confiada explícitamente; certificado rechazado sin esa confianza. Petición Unicode/espacios conserva firma; replay produce un solo evento. Firma alterada y spoof X-Forwarded-For no evitan autenticación.
- Ocho rutas administrativas/ajenas y GET a webhook devuelven 404; cuerpo >256 KiB devuelve 413; motor detenido devuelve 502. No se encontró la credencial de prueba en el log del proxy.
- `ssh -G` valida el reenvío inverso a loopback y opciones de fallo/keepalive sin conectar. Despliegue remoto no realizado.
- `systemd-analyze --user verify`: unidad opcional sintácticamente válida con binario local en una copia temporal. Original requiere Caddy instalado en `/usr/bin`; instalación permanente pendiente de R1.

## 2026-09-28 — Monitores ampliados (F2.5)

- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa; core 13,746 s. Archivos sin symlinks en componentes, metadatos sin contenidos, procesos propios, temperatura finita, HEAD autenticado, restricciones privadas y cambio de capacidad al editar destino.
- Sensor real del host: lectura de 42,0 °C durante la verificación. No garantiza sensores disponibles o numeración estable en otras máquinas.
- `TestHostJournalSelectedUnit`: unidad temporal propia emite un registro; evento contiene únicamente unidad/prioridad/fecha. Se cierra/reabre Engine, se muestrea nuevamente y se comprueba cursor válido y ausencia de duplicado. Un cursor inválido es rechazado.
- Rollback conjunto de lote/cursor, permiso revocado entre lectura y commit, reset durante lectura, límite de salida sin bypass de io.Copy. Migración 3→4 conserva un evento y el próximo vencimiento de una programación.
- `python scripts/test-ui-flow.py`: pasa; seis capacidades, monitores de archivo y HTTPS con valor 100, formularios de procesos/temperatura/journal; conserva flujo de notificación/HTTPS, timers, cola y diagnóstico. Captura `.dev/ui-monitors.png` revisada; artefactos `.dev` no se publican.
- Compilación, `go vet ./...`, `omarchy plugin validate .` y `git diff --check`: pasan.

## 2026-09-28 — Recuperación adicional (F2.7 en curso)

- Detectado: después de persistir outbox fallida/incierta, un fallo al cerrar el paso dejaba ejecución running y el reinicio podía reenviar. `deliver` ahora reconoce esos estados terminales y solo finaliza el paso. Entregas confirmadas conservan la regla de no reenviar.
- `TestTerminalOutboxSurvivesFailedStepCommitAndRestart`: tres estados terminales, fallo SQLite inyectado sobre steps, reapertura y worker real; ningún segundo envío. Incluida en la suite host anterior.
- `go test -race ./internal/core -run 'TestAbruptProcessCrashRecovery|TestNotificationWithoutSession' -count=1 -v`: pasa, 3,024 s. Cinco subprocesos terminados con SIGKILL, antes del despacho, tras reserva, tras efecto, tras commit de paso y después del envío HTTP. Reapertura sin Close previo; integrity_check=ok y estados esperados. Efecto externo representado por archivo sincronizado; no se afirma prueba de pérdida de alimentación.
- Bus inexistente explícito: notify-send falla, historial redactado y no se repite la notificación. No se cerró la sesión real ni se suspendió el equipo.
- F2.7 permanece pendiente de completar matriz de almacenamiento y ciclo instalado del servicio al cerrar sesión. Ver `docs/recovery.md`.

## 2026-09-28 — Reservas y cancelación duraderas

- `TestHTTPStorageBoundariesPreventUntrackedDispatch`: cuatro fallos SQL inyectados. No hay envío antes de persistir outbox/reserva/intento; después del efecto con resultado no persistido no se repite en el mismo proceso.
- `TestFailedCancellationDoesNotAcknowledgeOrSignal`: falla el commit de cancelación; conserva pendiente, devuelve error y no llama a los canceladores.
- `TestAbruptProcessCrashRecovery/http-cancel-ack`: SIGKILL tras confirmar cancelación, reapertura WAL, estado incierto y ninguna repetición. Ambos escenarios HTTP comprueban que el intento ya estaba persistido antes del efecto.
- Pruebas dirigidas de almacenamiento, caídas, cancelación, resultados terminales y reintentos: pasan con race, 5,354 s.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 16,952 s. `go vet ./...` y `git diff --check`: pasan.
- No se alteró la sesión gráfica ni se instalaron servicios permanentes en esta verificación.

## 2026-09-28 — Instalación temporal

- `python -m unittest discover -s tests -p test_install.py -v`: seis pruebas pasan (0,407 s). Copia inicial de 13 archivos, permisos de binarios/recibo, referencia absoluta CLI, backup de cada archivo al actualizar, preservación de shell.json, rechazo de archivo ajeno/modificado/symlink y retirada sin borrar datos/secretos/notas.
- Recibo adulterado con ruta fuera del paquete: retirada rechazada antes de borrar archivos.
- `git diff --check`: pasa. Ningún servicio ni plugin permanente instalado por estas pruebas; activación systemd/Omarchy todavía pendiente.

## 2026-09-28 — Snapshot SQLite y bloqueo del instalador

- `python -m unittest discover -s tests -p test_install.py -v`: ocho pruebas pasan, 0,616 s.
- Base de prueba en modo WAL con autocheckpoint desactivado: snapshot conserva fila de cola pendiente y versión 4, checksum coincide, permisos 0600 y no necesita archivo WAL. Conexiones de copia cerradas antes de calcular checksum.
- Bloqueo flock mantenido por otro descriptor: actualización rechazada sin sustituir archivos. El bloqueo se libera antes del arranque opcional del servicio.
- Pruebas previas de instalación/actualización/retirada conservadas. No se afirma restauración completa ni instalación permanente.

## 2026-09-28 — Restauración y binarios instalados

- `python -m unittest discover -s tests -p test_install.py -v`: once pruebas pasan, 1,005 s. Restauración conserva configuración y completados, revoca grants, pausa/rechaza admisión, marca pendientes inciertos y respalda el estado sustituido. Backup alterado y motor bloqueado rechazados.
- Binarios recompilados con `make build`. `python scripts/test-install-runtime.py`: pasa con permiso de socket local. HOME y XDG temporales; arranque del binario instalado, activación de flujo, cola pausada, rechazo de actualización activa, respaldo, cambio de borrador, restauración y reapertura real del motor.
- Tras restaurar: borrador original, cero permisos, un incierto, pausa activa y emisión rechazada. Retirada elimina binarios pero conserva SQLite. Ningún listener HTTP, servicio permanente ni preferencia del escritorio modificados.

## 2026-09-28 — systemd e integración permanente

- `python scripts/test-systemd-service.py`: pasa. Unidad temporal real, sin enable, retirada al terminar. Restart tras SIGKILL, cola persistente, notificación y comando /usr/bin/true aislado, stop/start, NoNewPrivileges/PrivateTmp/UMask efectivos.
- `python scripts/install.py --activate`: éxito. Unidad quatrrod.service habilitada, ActiveState=active/SubState=running; CLI status ready/protocol1/version0.1.0-dev, sin trabajos. Omarchy confirma plugin habilitado; debugBarGeometry confirma visible en derecha.
- Capturas nativas `.dev/installed-panel.png`, `installed-disconnected.png`, `installed-reconnected.png` revisadas: conexión, desconexión y reconexión. Shell rescan conserva PID 118764 durante prueba. No se desbloqueó sesión.
- Actualización instalada posterior respalda archivos/SQLite en ~/.local/state/quatrro-install/backup-h0e8aq6m; servicio restaurado. Texto nuevo de versión está en archivo instalado pero captura aún refleja componente anterior: QA de recarga continúa pendiente.
- `TestLanguagePreferenceDefaultsPersistsAndValidates`: pasa con race, 1,089 s. API de idioma preparada; integración/traducción UI pendiente.

## 2026-09-28 — Interfaz bilingüe instalada

- UI por defecto en inglés; `scripts/test-ui-flow.py` comprueba cambio a español y vuelta a inglés, secciones reactivas, títulos y botones estándar, y datos de ejemplo JSON traducidos sin corromper el JSON. Recorrido funcional anterior pasa completo. Capturas `.dev/ui-language-en.png`, `ui-language-es.png` y panel en inglés revisadas; formulario ahora usa todo el ancho disponible.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 16,542 s. `go vet ./...`, validación del manifiesto de origen/instalado y diff-check pasan.
- Once pruebas del instalador pasan con UI por hash, 1,065 s. `scripts/test-install-runtime.py` verifica además preferencia es después de reiniciar y restaurarla desde backup después de cambiar a en.
- Caché de QML reproducida: método setLanguage desconocido tras simple rescan. Con entrypoints ui-revisions/SHA256 devuelve ok; permissionText en el shell real devuelve texto español/inglés según selector y preferences.get confirma persistencia. Inglés queda seleccionado al terminar.
- Actualización permanente conserva archivos anteriores registrados; backup ~/.local/state/quatrro-install/backup-ei5l8qrc. Motor y widget siguen habilitados, sin flujos en perfil permanente. No se reinició el shell bloqueado. La captura solicitada de la ventana nueva no fue producida; no se presenta la captura antigua como evidencia del cambio.

## 2026-09-28 — Preparación de scripts

- Pruebas de archivo original modificado conservan contenido preparado y generan revisión nueva; contenido cambiado con hash anterior es rechazado. Fuentes relativas, directorios, symlinks y archivos mayores de 32 KiB rechazados.
- Parámetros prueban cadenas literales con metacaracteres, enteros sin coerción, rechazo de fracciones/fuera de rango, parámetros extra, NUL, patrones completos y opciones autorizadas.
- `go test -race ./internal/core`: pasa, 9,774 s (sin QUATRRO_HOST_TEST). Pruebas dirigidas posteriores a normalizar parámetros vacíos pasan. La ejecución real de scripts no está implementada aún y no se atribuye esta evidencia al sandbox.

## 2026-09-28 — Scripts: ejecución exacta, permisos y simulación

- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 27,092 s. Incluye ejecución real Bash/Python por systemd/bubblewrap y validación de entrada de código en preflight. Fuente editada después de activar no cambia contenido ejecutado; argumentos con `$()` llegan literales; código montado rechaza escritura y directorio personal no está visible.
- Detectada expansión de argumentos por systemd durante integración; `--expand-environment=no` corrige pérdida de valores. No se interpolan argumentos en código shell.
- Pruebas con cola persistida: revocar capacidad o activar una revisión nueva deja el trabajo anterior `denied`; campo ausente, booleano donde se requiere cadena y exceso de longitud terminan `failed` por validación del parámetro.
- Tras compartir resolución con simulación: `QUATRRO_HOST_TEST=1 go test -race ./internal/core -run 'Test.*Script' -count=1` pasa, 11,325 s. Simulación muestra argumentos/revisión exactos, rechaza tipos incorrectos y no crea ejecuciones; suite dirigida vuelve a ejecutar ambos intérpretes y escenarios de permisos.
- `go vet ./...` y `git diff --check` pasan. Contrato actualizado en `docs/scripts.md`; la instalación permanente conserva su versión anterior. F3.1 no se cierra: faltan directorios autorizados, UI y validaciones restantes.

## 2026-09-28 — UI de preparación de scripts

- Compilación de ambos binarios pasa. `python scripts/test-ui-flow.py` pasa tras corregir dependencia circular QML y acceso a filas eliminadas. El formulario llama a `scripts.prepare` sobre archivo temporal real, impide guardar antes de preparar, verifica contenido/hash y guarda sin `path` en configuración.
- Se conserva el recorrido inglés/español y de seis capacidades con una entrega HTTPS, monitores, programación, historial, cola y diagnóstico. La revisión preparada está en el borrador activado sin conceder ejecución de scripts; todavía no se prueba un flujo de script configurado desde UI.
- Captura `.dev/ui-script-review.png` revisada: hash y código aparecen al comienzo del formulario. Es un render offscreen; temas, aspecto nativo y teclado siguen pendientes en sus tareas de QA.
- `python -m unittest discover -s tests -p test_install.py -q`: once pruebas pasan (1,064 s), con 18 archivos de instalación incluyendo ScriptParameters.qml. Manifiesto y diff-check válidos. No se modificó la instalación permanente.

## 2026-09-28 — Flujo gráfico con script real

- `python scripts/test-ui-flow.py`: pasa con siete capacidades. UI prepara script, selecciona revisión explícita y configura `message` desde evento, `copies=2` entero y `enabled=true` booleano. El script comprueba los tres argumentos; el paso HTTPS posterior entrega una vez y el flujo queda completado.
- Prueba de revisión obsoleta rechazada por editor, sin acción guardada. Revisión de capacidades muestra ID/hash; el valor guardado coincide con `scripts.prepare`. Se conservan casos de idiomas, programación, monitores, pausa, cola, cancelación y diagnóstico.
- Captura inicial encontró controles ocultos por falta de dependencia reactiva; corregida y prueba de visibilidad añadida. Captura final `.dev/ui-script-action.png` revisada con revisión seleccionada/disponible, código y entrada de parámetro visibles. El render offscreen no sustituye QA nativa/teclado/temas.
- Una ejecución intermedia expiró al esperar programación mientras se editaban archivos QML; se repitió con archivos estabilizados y pasó. El recorrido final no registra ERROR/ReferenceError/TypeError.
- Once pruebas del instalador pasan (1,238 s) incluyendo ScriptArguments.qml, total 19 archivos; manifiesto y diff-check pasan. No se reinstaló el perfil permanente ni se cerraron F3.1/F3.6.

## 2026-09-28 — Directorios fijados por descriptor

- `QUATRRO_HOST_TEST=1 go test -race ./internal/sandbox -count=1 -v`: pasa, 1,065 s. Rutas relativas/no canónicas, raíz, symlinks, interfaces del kernel/sesión y destinos fuera de `/work/id` rechazados. Transporte rechaza versión/campos/tamaño y cwd no autorizado.
- Pruebas ro/rw abren y verifican directorio, lo renombran y crean reemplazo antes de arrancar Bubblewrap. Proceso lee el original; ro impide escritura, rw escribe resultado en original, reemplazo intacto. Próximo intento con identidad vieja falla. Código de script permanece solo lectura y argumentos literales.
- Binarios recompilados. `python scripts/test-sandbox-runner.py`: pasa mediante systemd-run con memoria 256 MiB, CPU 50 %, tareas 32 y timeout 10 s. Python lee/escribe directorio temporal y recibe argumentos literales mediante protocolo privado. Versión desconocida rechazada por ejecutable real.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 27,107 s y sandbox 1,067 s; vet y diff-check pasan. Motor aún no referencia el auxiliar para sus acciones: integrar permisos/preflight/UI es la siguiente tarea, sin atribuir a estas pruebas la autorización completa.

## 2026-09-28 — API, capacidades y acciones con directorios

- `TestHostDirectoryActionsAndRevocation`: pasa, 17,531 s. Script escribe el valor recibido; comando touch crea archivo en cwd autorizado. Revocación y cambio rw→ro dejan pendiente `denied`; reemplazo de ruta tras admisión produce `failed` sin archivo en reemplazo.
- Validación rechaza directorios en notificaciones y cwd no registrado; capacidad cambia con modo de acceso. Preflight rechaza identidad sustituida sin conceder permisos. Después de ampliar preflight a código de script por memfd, suite completa host/race pasa (core 43,507 s).
- `python scripts/test-directory-actions.py`: pasa sobre binarios recompilados. Perfil temporal sin HTTP, API real de preparación de directorio/script, config.save/preview/activate, simulación muestra mounts/cwd sin archivo de salida, emit produce contenido esperado mediante Python y unidad aislada.
- Compilación, vet, diff-check y sintaxis Python pasan. Core usa el ejecutable de prueba únicamente en TestMain para despachar el modo privado del auxiliar durante pruebas; el recorrido adicional usa quatrrod compilado real.
- UI de directorios y presentación de sus permisos siguen pendientes; no se atribuye a estas pruebas el recorrido gráfico completo ni se cierra F1C.3/F3.1. Instalación permanente sin cambios.

## 2026-09-28 — Formulario de directorios y scripts completos en desarrollo

- `python scripts/test-ui-flow.py` pasa: preparación real por API desde componente QML, modo rw, identidad visible, cwd `/work/data`, rechazo de ruta inexistente sin alterar el montaje y conservación después de editar/guardar acción.
- La revisión incluye origen, acceso y explicación del árbol. Tras conceder siete capacidades, el script comprueba cadena del evento/entero/booleano, escribe `result` con valor `mounted` en el directorio aprobado y el paso posterior entrega un HTTPS. Se mantienen pruebas de idiomas, monitores, programación, cola/cancelación y diagnóstico.
- Captura `.dev/ui-script-action.png` revisada: origen→destino, identidad y modo de acceso visibles. Es offscreen, no evidencia de integración estética nativa. Registro sin ERROR/ReferenceError/TypeError.
- Once pruebas de instalador pasan (1,118 s), con Directories.qml y 20 archivos registrados. Manifiesto y diff-check pasan. No hubo cambios Go desde la suite host/race anterior (core 43,507 s).
- Evidencia acumulada permite cerrar F3.1 funcional: revisiones/argumentos/invalidación, ejecución exacta y UI, incluidos directorios aprobados. F1C.3 ampliada, QA nativa y entrega instalada permanecen pendientes.

## 2026-09-28 — Cancelación y timeout con montajes rw

- `QUATRRO_HOST_TEST=1 go test -race ./internal/core -run 'TestHost(MountedScript|CancelledMounted)' -count=1 -v`: pasa, 7,369 s.
- Python ejecutado por auxiliar/memfd crea hijo que ignora SIGTERM, entra en su propia sesión y escribe cada 20 ms. Pruebas timeout/cancel capturan PID del cgroup después de observar escritura, esperan terminación acotada y comprueban que todos desaparecen. Archivo estable después de terminar, durante 150 ms adicionales.
- Recorrido worker con revisión aprobada: cancela durante escritura, espera salida, confirma `uncertain` en SQLite y ejecuta otro tick sin nuevos bytes. Demuestra ausencia de repetición automática en ese escenario, no rollback de la escritura previa.
- Vet y diff-check pasan. Solo se añadieron pruebas y documentación; la suite completa de 43,507 s anterior continúa siendo la última completa, complementada por estas pruebas nuevas. Instalación permanente intacta.

## 2026-09-28 — Perfiles de comandos

- Pruebas dirigidas host/race pasan (8,121 s): resolución exacta de mkdir y test, rechazo de argv/ejecutable alternativos, traversals, ruta anidada sin montaje del padre, plantillas, perfil desconocido y escritura sobre permiso ro. Cambiar ruta cambia capacidad; simulación devuelve argv sin crear directorio.
- Ejecución host de `make-directory` produce directorio 0700 con espacios/metacaracteres literales y falla al repetir sobre existente. `file-exists` funciona sobre montaje ro y falla al retirar el archivo. Worker completa ambos flujos.
- `python scripts/test-ui-flow.py`: pasa con ocho capacidades y un flujo notificación→perfil make-directory→script→HTTPS. Directorio preparado desde UI, resultado 0700 y escritura del script confirmados; controles anteriores de idiomas/cola/monitores/diagnóstico conservados.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 56,996 s. Compilación, vet, manifiesto y diff-check pasan. F1C.3 cerrada con perfiles concretos; el modo avanzado fixed sigue requiriendo revisar el programa y argv completos. Entrega instalada/QA nativa permanecen pendientes.

## 2026-09-28 — Transporte de adaptadores externos

- Pruebas dirigidas adapters/sandbox con host/race pasan (1,066 s / 2,056 s). Protocolo verifica versión, correlación, objeto de respuesta, campos desconocidos, documentos concatenados, profundidad, cuotas y capacidades exactas. El salto de línea cuenta en la cuota de entrada.
- Auxiliar recibe JSON por stdin y código por memfd separado. Python transforma el mensaje a mayúsculas; emitir 70000 bytes con cuota 64 produce error y no devuelve más de 64 bytes. Cuota cero descarta salida, manteniendo el comportamiento previo.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa (adapters 1,019 s, core 56,700 s, sandbox 1,149 s). Binarios recompilados, vet y diff-check pasan.
- `python scripts/test-adapter-transport.py`: pasa. Manifiesto/código de ejemplo status-normalizer ejecutados con quatrrod compilado en unidad temporal limitada; respuesta correlacionada y severidad error esperada, dentro de cuota. Sin perfil permanente ni conexión externa.
- Esta evidencia cubre protocolo/transporte; no registro, revocación, persistencia ni worker/UI de adaptadores. F3.2 permanece abierta.

## 2026-09-28 — Preparación y registro de adaptadores

- `TestAdapterPreparationPinsLocalCodeAndManifest` pasa con race (1,080 s): usa Engine.Handle/adapters.prepare sobre fuente temporal, verifica copia intacta tras editar original, revisión nueva, rechazo de código/manifiesto alterados con hash viejo, fuente simbólica y versión incompatible.
- SQLite confirma cero grants/ejecuciones tras preparar; config.save/config.get conservan código/revisión; IDs repetidos rechazados. Esquema JSON actualizado para manifiesto y recurso preparado.
- `go test -race ./internal/core ./internal/adapters`: pasa (core 10,222 s) con permiso de sockets; pruebas host optativas no activadas. Primer intento restringido falló en httptest al abrir loopback, sin atribuir ese fallo al cambio.
- Vet y diff-check pasan. Registro no equivale todavía a acción ejecutable: integración de permisos, worker, persistencia del resultado y UI permanece pendiente; F3.2 abierta.

## 2026-09-28 — Contextos privados y esquema 5

- Cuatro pruebas dirigidas con race pasan (1,466 s): contexto sobrevive reapertura y afecta solo al flujo correspondiente; evento original intacto; trigger SQL aborta el guardado sin dejar contexto, step ni avance parcial; cuota y conteo de bytes verificados; migración 4→5 conserva pendiente/contexto vacío.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 58,102 s. Casos de recuperación, firmas, montajes, procesos hijos, perfiles y scripts conservados.
- `python -m unittest discover -s tests -p test_install.py -q`: once pruebas pasan (1,209 s). `python scripts/test-install-runtime.py`: pasa sobre binarios recompilados con esquema 5, incluyendo backup, restauración pausada, permisos revocados, cola incierta y retirada conservando datos.
- Installer acepta versiones hasta 5 y mantiene restauración de fixtures versión 4. Compilación, vet y diff-check pasan. El perfil permanente conserva su esquema/binario anterior.
- Esta etapa valida la persistencia aislada y lectura por worker; todavía no ejecuta un paso adapter completo ni cierra F3.2.

## 2026-09-28 — Paso adapter completo

- Pruebas dirigidas host/race iniciales pasan (25,014 s): referencia/revisión y simulación diferida, transformación real, resultado confirmado y leído por siguiente paso después de reabrir, revocación/cambio de revisión denegados, respuesta inválida, cuota de salida y timeout sin avance/contexto.
- Se añaden cuota de entrada y de contexto: exceso detiene paso como failed, sin aplicar datos ni dejar running indefinidamente. Prueba io.Copy confirma que el buffer padre no permite eludir su límite mediante ReaderFrom promovido.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, core 88,697 s, con todos los casos nuevos y recuperación previa. Compilación, vet y diff-check pasan.
- `python scripts/test-adapter-flow.py`: pasa sobre binarios compilados, API de preparación y concesión reales, simulación sin resultados ficticios, adaptador status-normalizer seguido de script con binding data.adapter.severity. Archivo contiene error y estado final completed en perfil temporal.
- F3.2 cerrada funcionalmente en motor. F3.6 conserva UI de adaptadores y administración pendientes, junto con el resto de fases abiertas. Perfil permanente sin cambios.

## UI de adaptadores — recorrido integrado

- `python scripts/test-ui-flow.py` pasa con nueve capacidades y una entrega HTTPS: prepara un adaptador, exige seleccionar su revisión y verifica `data.adapter.severity` en el destino.
- Capturas `.dev/ui-adapter-review.png` y `.dev/ui-adapter-action.png` inspeccionadas: código, revisión y cuotas visibles en controles de desarrollo. No acreditan estética nativa ni validación en el shell instalado.

## OAuth2 Google — intercambio de renovación

- Pruebas `TestGoogleOAuthRefreshContract` y `TestGoogleOAuthFailureRedaction`
  pasan con race (1,027 s): form encoding, endpoint fijo, token conservado/rotado,
  expiración, respuestas inválidas/acotadas, invalid_grant/invalid_client, 429/503,
  redirecciones y errores de transporte sin filtrar datos sensibles.
- Transporte simulado: no prueba cuentas reales ni integración worker/secretos/UI.
  F3.3 permanece abierta. Diff-check pasa.

## OAuth2 — persistencia y concurrencia

- `go test -race ./internal/core -run 'TestGoogle|TestCredential' -count=1`
  pasa (1,427 s). Doce consumidores renuevan una sola vez; una nueva instancia
  reutiliza el token persistido y la renovación próxima usa el refresh rotado.
- Reemplazo y borrado durante renovación impiden guardar/devolver el resultado
  obsoleto. Prueba sobre backend file temporal, sin credenciales reales.
- Suite completa `go test -race ./internal/core` pasa (11,595 s), con sockets
  locales permitidos; pruebas host optativas no activadas. Diff-check pasa.

## OAuth2 — API y estados de conexión

- `go test -race ./internal/core -run 'TestGoogle|TestCredential' -count=1`
  pasa (1,746 s), incluyendo dispatcher put/status, cero grants al provisionar,
  respuestas redactadas, renovación requerida/disponible/expirado, errores
  temporales/cliente/respuesta/reconexión y persistencia tras reapertura.
- Invalid_grant solo provoca una solicitud de renovación: el estado persistido
  evita repetirla; sustituir la conexión restablece el estado. Borrarla devuelve
  unavailable. Pruebas usan perfiles temporales y transporte simulado.
- Diff-check pasa. No valida consentimiento Google real ni integración de envíos/UI.

## OAuth2 — destinos y envío

- `TestGoogleDestinationBoundary` comprueba hosts/puertos, dominios parecidos,
  userinfo/fragmentos, LAN y cambio del hash de permiso al cambiar conexión.
- `TestGoogleOutboundUsesPrivateTokenAndFailsClosed` verifica Authorization con
  solo access token, identidad de entrega, bloqueo tras reconnect_required,
  rechazo de destino ajeno y de uso del conjunto como Bearer genérico.
- Suite completa `go test -race ./internal/core` pasa (12,205 s), con sockets
  temporales habilitados y sin pruebas host optativas. Diff-check pasa.
- Transporte Google simulado; no prueba cuenta real, UI ni recuperación de 401.

## OAuth2 — rechazo HTTP 401 y worker

- Suite dirigida `TestGoogle|TestCredential` con race pasa (2,003 s) tras integrar
  estado token_rejected y comparación de revisión privada al recibir 401.
- `TestGoogleRemoteRejectionDoesNotReplayOrInvalidateReplacement`: una petición,
  401 terminal, estado actualizado, refresh conservado; sustitución durante envío
  permanece disponible. Renovar después no repite la entrega anterior.
- `TestGoogleDispatchRevisionDetectsChangedCredentials`: impide usar referencia
  privada obsoleta antes de enviar.
- `TestGoogleWorkerDeliveryAndTerminalRejection` con race pasa (1,221 s): admisión,
  permisos, render y emisor OAuth2 a través de tick; 200 completa, 401 falla, cinco
  ticks no repiten ninguna de las dos entregas. Transporte HTTP simulado.
- Diff-check pasa; no llamadas a cuentas Google reales ni actualización instalada.

## OAuth2 — formulario QML y estado

- Binarios recompilados; `python scripts/test-ui-flow.py` pasa con perfil temporal:
  importa conexión desde formulario, consulta renewal_required y comprueba que
  el estado no contiene client ID ni refresh token. Conserva el flujo previo:
  nueve capacidades y una entrega HTTPS completada, cola/diagnóstico verificados.
- Captura inicial no incluía el diálogo: se corrigió selección de contentItem y
  se repitió el recorrido completo, que pasa. `.dev/ui-oauth-credential.png`
  revisada: formulario inglés, valores sensibles enmascarados y selección file
  explícita. Render de desarrollo, sin cerrar estética nativa R1.UI.
- Diff-check pasa. No se probó consentimiento en navegador ni cuentas reales;
  no se actualizó la instalación permanente.

## OAuth2 — protocolo PKCE de escritorio

- `TestGoogle|TestCredential` con race pasa (2,218 s) tras compartir transporte
  de canje/renovación e incorporar sesión PKCE.
- `TestGoogleConsent*` pasa (1,022 s), incluido caso añadido sin refresh token:
  challenge S256, URL sin secretos/verifier, state independiente, callback
  duplicado/malformado/host/ruta/expiración, denegación y canje de un solo uso.
- Transporte simulado; no listener ni navegador real todavía. Diff-check pasa.

## OAuth2 — listener loopback real

- `go test -race ./internal/core -run TestGoogleConsentListener -count=1` pasa
  (2,132 s), con permiso para sockets temporales: callback inválido conserva
  sesión, código/denegación correctos completan y cierran puerto, cancelación
  impide nuevas conexiones, padre ya cancelado no abre listener.
- Caso adicional de deadline con petición HTTP incompleta confirma resultado
  cancelado y cierre forzado de la conexión, sin esperar a completar cabeceras.
- Cabeceras no-store y ausencia de códigos reflejados verificadas. Sin contactos
  Google, consentimiento real, UI ni instalación permanente. Diff-check pasa.

## OAuth2 — sesiones integradas

- Pruebas dirigidas sesión con race pasan (1,229 s): loopback real, canje simulado,
  guardado de tokens, cero grants, protección frente a credencial concurrente,
  estados redactados, exclusión de sesión simultánea y cancelación/cierre.
- Se añade cancelación durante canje: el contexto interrumpe el transporte y
  ninguna credencial se guarda; estado final cancelled.
- `go test -race ./internal/core` pasa (13,163 s), con sockets locales permitidos
  y pruebas host optativas desactivadas. Diff-check pasa.
- Sin navegador/cuenta real ni instalación permanente; faltan scopes y UI.

## OAuth2 — scopes concedidos y renovación

- Pruebas nuevas con race pasan (1,107 s): canje rechaza scopes ausentes/parciales,
  acepta permisos completos/adicionales, conserva el campo recibido y rechaza
  sintaxis inválida. Renovación con reducción termina scopes_missing, no devuelve
  token y no reintenta hasta restablecer la conexión.
- `go test -race ./internal/core` pasa (13,186 s), con loopback temporal permitido;
  pruebas host optativas no activadas. Fixtures de consentimiento incluyen scopes
  explícitos. Diff-check pasa.
- Metadatos provienen del token endpoint simulado; no hubo autorización con cuenta
  real. UI de inicio/navegador/seguimiento permanece pendiente.

## OAuth2 — inicio y cancelación desde QML

- Binarios recompilados; `python scripts/test-ui-flow.py` pasa. Formulario inicia
  una sesión real loopback con credenciales ficticias; estado waiting, URL Google
  sin client secret, cancelación con estado cancelled y URL eliminada. Consulta
  posterior confirma que no se creó la credencial.
- Flujo previo conserva nueve capacidades, una entrega HTTPS completada y
  verificaciones de cola/diagnóstico. Sin apertura de navegador/cuenta real.
- `.dev/ui-oauth-consent.png` inspeccionada: texto inglés, scopes y cliente visibles,
  client secret enmascarado y backend explícito. Render de desarrollo, R1.UI abierta.
- Diff-check pasa. Apertura real mediante Qt.openUrlExternally y consentimiento
  Google no verificados en escritorio instalado; pendiente recuperación de UI.

## OAuth2 — recuperación del panel y reinicio

- `TestGoogleConsentDiscoveryAndRestart` con race pasa (1,099 s): current inicial
  none, descubrimiento de sesión/link sin client secret, cancelación elimina URL,
  nueva instancia sin sesión y rechazo de identificador anterior.
- Build pasa. `python scripts/test-ui-flow.py` pasa: se borra estado OAuth2 del
  panel, se recuperan la misma sesión y URL desde current, luego se cancela y
  comprueba que no existe credencial. Flujo anterior completa una entrega HTTPS
  con nueve capacidades. No se abre navegador ni se contacta a Google.
- Guía consolidada sin atribuir pruebas simuladas a cuenta real/keyring. Diff-check pasa.

## LAN — autoridad exacta e IPv4/IPv6

- `TestLAN|TestHTTPSDelivery|TestAddressPolicy` con race pasa (1,144 s). TLS real
  IPv4 e IPv6 con certificados verificados: un envío autorizado; solicitudes sin
  permiso, con excepción ajena y con destino cambiado no llegan al servidor.
- Validación/pruebas cubren autoridad/puerto, duplicados, comodines, hash de permiso
  y clases de IP locales frente a multicast/unspecified/documentación/transición.
- Suite core/race pasa (13,188 s), sockets temporales permitidos, host optativo
  desactivado. Diff-check pasa. UI de revisión LAN editada, recorrido aún pendiente.

## LAN — revocación y recorrido QML

- `go test -race ./internal/core -run TestLAN -count=1`: pasa (1,264 s).
  Caso queued: revocar antes del primer tick deja denied con cero peticiones.
  Caso retry: un 503 crea reintento pendiente; revocar y hacer elegible la cola
  deja denied sin segundo envío, incluso con tres ticks. Receptor TLS real.
- Build y `python scripts/test-ui-flow.py` pasan: excepción exacta configurada
  desde formulario, texto de permiso incluye autoridad LAN, nueve capacidades y
  una entrega HTTPS completada. Pruebas OAuth2 anteriores permanecen verdes.
- Diff-check pasa. Evidencia completa F3.4 localizada en esta sección y la anterior;
  instalación permanente sin actualizar, ningún servicio LAN del usuario contactado.

## Broker administrativo — contrato y política

- `go test -race ./internal/broker` pasa (1,018 s). Política por UID/unidad/operación,
  argumentos fijos, sujeto Polkit completo sin interacción, rechazo de otra unidad,
  operación/UID, versión, comodín, ruta, template vacío y campos privilegiados.
- `go vet ./internal/broker` y diff-check pasan. Ningún comando administrativo,
  instalación o política real ejecutados. No acredita todavía la frontera root.

## Broker — carga de política por descriptores

- `go test -race ./internal/broker` pasa (1,019 s): archivo seguro, propietario
  incorrecto, escritura grupo en raíz/directorio/archivo, symlink final/intermedio,
  fichero >64 KiB, FIFO sin bloqueo, rutas no canónicas/traversal y recarga tras
  sustitución atómica por una política vacía.
- Helper privado usa UID del test en árbol temporal; `LoadPolicy` de producción
  fija root y rechaza esa ruta. No equivale a prueba privilegiada del servicio.
- Vet y diff-check pasan. Ningún archivo del sistema modificado.

## Broker — identidad del peer con pidfd

- `go test -race ./internal/broker` pasa (1,039 s). Kernel de esta máquina admite
  SO_PEERPIDFD: socketpair devuelve PID/UID actuales; revalidación pasa en vivo y
  falla tras cerrar referencia. Cliente hijo real devuelve su PID y se rechaza
  después de Kill/Wait, sin reinterpretar su PID como otro sujeto.
- Parser stat probado con comm complejo; PID distinto, campos incompletos y UID
  real/efectivo/saved/fs inconsistentes se rechazan. Lecturas acotadas.
- Vet y diff-check pasan. No hay comprobación Polkit real ni efecto privilegiado
  todavía; pruebas usaron sockets/procesos temporales del usuario.

## Broker — ejecutor y secuencia de autorización

- `go test -race ./internal/broker` pasa (1,085 s), vet/diff-check pasan.
- Runners simulados verifican: política denegada no llama Polkit; Polkit denegado,
  revocación en segunda lectura y peer terminado no llaman systemctl. Éxito usa
  solo pkcheck/systemctl fijos; fallo de mutación conserva incertidumbre.
- Status rechaza propiedades extra, duplicadas, unidad ajena y valores inválidos.
- Procesos reales no privilegiados: sleep termina por timeout de 30 ms (límite
  observado <2 s), head de 5000 bytes excede cuota sin devolver salida parcial,
  printf funciona. io.Copy no elude el límite de 4096 bytes.
- No se invocaron pkcheck/systemctl reales ni se probaron permisos root todavía.

## Broker — transporte Unix y binario

- Suite broker/race pasa (1,082 s). Sockets temporales y peer kernel reales,
  ejecutor simulado: JSON concatenado, >4 KiB o UID declarado no ejecutan;
  solicitud válida ejecuta una vez; error interno no se refleja. Cancelación
  cierra solicitud incompleta y finaliza el servidor.
- Compilación quatrro-broker y --version pasan; ejecución sin root devuelve exit 1
  y mensaje esperado. No se probó aún activación systemd/Polkit como root.
- Vet para broker/binario y diff-check pasan. Ninguna instalación de sistema.

## Broker — artefactos systemd/Polkit

- `TestGeneratedPolkitRulesExactGrants` con race pasa (1,363 s), ejecutando JS en
  Node: usuario/unit/action exactos, denegación de variantes y no intervención en
  org.freedesktop.systemd1. Política vacía no concede acceso, mapping root rechazado.
- `python scripts/test-broker-packaging.py` pasa: DTD Polkit local, cuatro acciones
  defaults no, sin implicaciones y política vacía; systemd-analyze verify sin arranque.
- Primer verify restringido falló por socket interno de systemd; permiso ampliado
  para consulta local permitió terminar. Se corrigió resolución DTD vía catálogo
  local para evitar intentos de red. Vet y diff-check pasan.
- No hay unidad/política instalada, comprobación polkitd real ni efecto root aún.

## Broker — cliente y resultados inciertos

- `go test -race ./internal/broker` pasa (1,156 s). Servidores Unix temporales:
  respuesta correcta, denegación, versión/ID/unidad ajenos, resultado contradictorio,
  documentos concatenados, salida excesiva/vacía y error interno desconocido.
- UID esperado incorrecto recibe cero bytes. Cancelar tras recibir la solicitud
  devuelve outcome_unknown. No hubo systemctl/Polkit real ni servidor root.
- Vet de broker/binario y diff-check pasan. Ruta pública fija y comprobación UID 0
  permanecen independientes del helper privado usado por las pruebas.

## Broker — acción administrativa en worker

- Pruebas dirigidas TestAdministrative con race pasan (1,540 s): validación de
  unidad/operación y rechazo de overrides, hash del permiso cambia, simulación
  declara broker sin invocarlo. Resultados del cliente simulados cubren completed,
  denied, uncertain y failed; revocación previa deja denied con cero llamadas.
  Cuatro ticks no repiten ninguna acción terminal.
- `go test -race ./internal/core ./internal/broker` pasa (core 14,063 s, broker
  cacheado), con sockets temporales permitidos. Diff-check pasa.
- No se ejecutaron operaciones root; preflight/UI/instalación y efecto real aislado
  permanecen pendientes. Perfil permanente sin cambios.

## UI administrativa e importación

- Build y `python scripts/test-ui-flow.py` pasan: formulario guarda únicamente
  id/kind/unit/operation, revisión incluye status + unidad y root policy/Polkit,
  diez capacidades aprobadas en perfil temporal. El flujo administrativo no recibe
  eventos; una entrega HTTPS del flujo ordinario completa sin cambios.
- `.dev/ui-admin-action.png` inspeccionada: aviso inglés, unidad exacta y selector
  de operación legibles. Render de desarrollo, no prueba estética nativa.
- `TestAdministrativeImportDoesNotGrantOrEnable` con race pasa (1,096 s): conserva
  acción, desactiva flujo, cero grants/ejecuciones y simulación sin llamar broker.
- Diff-check pasa. No hubo instalación administrativa ni efecto sobre servicios reales.

## F3.5 — Polkit real aislado

- `QUATRRO_POLKIT_TEST=1 go test ./internal/broker -run '^TestIsolatedPolkit$' -count=1 -v`
  pasa (0,239 s) con permiso para namespaces. Usa cuentas subordinadas, raíz de
  solo lectura, bus/run/políticas/red privados; no modifica el host.
- Renderizador de producción + polkitd/pkcheck reales: status permitido, unidad
  ajena y restart denegados; sustitución atómica por reglas vacías revoca status.
- Dos iteraciones iniciales detectaron comprobación de disponibilidad inadecuada
  y /proc con traducción de UID incorrecta. Introspección D-Bus y namespace PID
  propio de bwrap resuelven ambas; prueba final pasa sin relajar autorizaciones.
- No hay logind ni systemd manager en esta prueba. No valida efectos administrativos,
  restricciones de la unidad ni política D-Bus del sistema. F3.5 sigue pendiente.

## F3.5 — UID ajeno y política root dentro del aislamiento

- Prueba dirigida TestIsolatedPolkit pasa (0,295 s) con raíz tmpfs, /usr y archivos
  seleccionados del host de solo lectura. UID 102 no recibe autorización concedida
  a UID 1000. No se modifica ni se conecta con la instalación Polkit del host.
- LoadPolicy público usa su UID 0 fijo sobre /etc/quatrro/broker.json: acepta root,
  rechaza chown a 1000 y modo 0660, observa reemplazo atómico por reglas vacías.
  Se ejercita el namespace subordinado; no supone privilegios root del host.
- La prueba aún no ejecuta un manager systemd ni el servicio completo del broker.
- Suite completa `QUATRRO_POLKIT_TEST=1 go test -race ./internal/broker -count=1`
  pasa (2,468 s), incluida integración Polkit real; `git diff --check` pasa.

## F3.5 — systemd aislado y servicio efímero real

- `python3 scripts/test-broker-systemd.py` pasa: systemd 261.2-1-arch arranca como
  PID 1, D-Bus privado activado por socket, fixture start/running/stop/inactive y
  salida del contenedor con código 0. Reporte JSON confirma los tres recorridos.
- Namespace de cgroups dentro de scope transitorio delegado; raíz y /etc efímeros,
  /usr solo lectura, /sys privado/protegido, consola PTY y namespaces red/IPC/UTS.
  El script detiene su scope en finally. No instala servicios o políticas del host.
- Intentos previos identificaron /sys sin premontar y conexión del manager al bus
  dependiente de dbus.socket; ambos corregidos. Marcador se comprueba desde consola
  y solo se emite después de verificar el estado real del fixture.
- `py_compile` y `git diff --check` pasan. El broker/Polkit todavía no están
  enlazados en este script; F3.5 sigue abierta, sin atribuirle prueba integral.

## F3.5 — Broker, Polkit y systemd completos en aislamiento

- Compilación del broker y helper TestSystemdBrokerFixture; script
  test-broker-systemd.py pasa con JSON de siete comprobaciones verdaderas y ningún
  servicio de sistema del host modificado. Log local: .dev/broker-systemd.log.
- Unidades del broker copiadas sin cambios, socket activado por systemd, política
  root y reglas del generador de producción. Call real desde UID 1000 consulta,
  inicia, reinicia y detiene la unidad. UID 102 y otra unidad reciben denied.
- Revocación Polkit impide status; restauración vuelve a permitirlo. Sustitución
  de allowlist por vacía deniega start y la unidad permanece inactive.
- /proc confirma CapEff=0 y NoNewPrivs=1; cgroup confirma memory.max=134217728 y
  cpu.max=25000 100000. Primer intento detectó controladores sin habilitar por
  supervisor en raíz; subgrupo supervisor lo corrige y la aserción final pasa.
- Host sysinit/sockets wants ocultos en el contenedor para evitar servicios ajenos.
  Bus privado permisivo, sin logind; no equivale a validar políticas D-Bus del host
  o presión de recursos. Detiene broker antes de salir, finally retira el scope.
- Suite QUATRRO_POLKIT_TEST=1 go test -race ./internal/broker -count=1 pasa (2,468 s),
  py_compile y diff-check pasan. F3.5 sigue abierta por instalación/ciclo de vida,
  recuperación e integración final. No se instala el broker en el host.

## F3.5 — Preparación de artefactos administrativos

- Nuevo --prepare-policy no privilegiado: límite 64 KiB, ParsePolicy y
  RenderPolkitRules de producción; salida únicamente tras validar. Tests rechazan
  duplicados anidados, aliases de mayúsculas/escapes, desconocidos y JSON concatenado.
- scripts/test-broker-preparation.py pasa con binario real: paquete de seis archivos
  con política vacía y activation=disabled, hashes/modos correctos, no sobrescribe
  y no crea paquete con política inválida. Preview en .dev/broker-install-preview.
- Primera suite sin permisos de sockets falla por restricciones del entorno.
  Repetición QUATRRO_POLKIT_TEST=1 go test -race ./cmd/quatrro-broker ./internal/broker
  -count=1 con permisos apropiados pasa (1,019 s y 2,477 s), incluido Polkit real.
- py_compile y diff-check pasan. Aplicación privilegiada del paquete, actualización
  y desinstalación pendientes; ningún privilegio instalado ni activado en el host.

## F3.5 — Instalación administrativa, actualización y retirada

- test-broker-installer.py pasa: rutas fijas, hashes, apply/remove, rollback tras
  daemon-reload fallido, journal conservado si falla restauración y recover posterior;
  rechaza destinos de recuperación ajenos, archivos ajenos y symlinks. systemd simulado.
- test-broker-systemd.py pasa contra systemd/Polkit/broker reales en raíz efímera:
  apply deshabilitado, activación explícita y Call, update mantiene permisos aunque
  el paquete nuevo esté vacío, --replace-policy revoca, remove retira archivos
  propios/recibo/journal y conserva /etc/quatrro/unrelated. JSON install_update_remove=true.
- Para probar propietarios reales del namespace, /usr y /usr/share ahora son
  directorios efímeros; bin/lib se montan de solo lectura. Binario/política se copian
  a destinos efímeros antes de las comprobaciones. No se escriben rutas del host.
- Log local .dev/broker-systemd.log. El cierre reporta mounts bin/lib ocupados antes
  de terminar procesos restantes; contenedor sale 0 y scope se retira en finally.
- py_compile y diff-check pasan. Recuperación por interrupción real aún pendiente;
  prueba de fallos simulados no equivale a corte del instalador entre escrituras.

## F3.5 — Recuperación real y cierre del componente

- Build actual de quatrro-broker y go test -c de broker-integration.test;
  test-broker-systemd.py pasa. Reporte final incluye installer_sigkill_recovery=true
  junto a broker/Polkit/systemd/cgroups y ciclo de instalación verificados.
- test-broker-install-crash.py se ejecuta dentro de la raíz efímera: SIGSTOP por
  perfil al retornar operaciones duraderas del instalador sin modificar; SIGKILL
  por padre y recover mediante CLI pública. No ejecuta el bloque de rollback.
- Seis fronteras actualización/remove: journal, binario, reglas, recibo, retirada
  de socket y retirada de recibo. Dos fronteras primera instalación: binario y
  recibo. Restaura exactamente bytes, modos y propietario o ausencia originales.
- Cada caso prueba segundo instalador rechazado mientras el lock sigue retenido;
  después del kill, apply exige recover. Tras recover no queda journal ni socket
  habilitado/activo. Consulta real del cliente pasa con instalación restaurada.
- La consola se drena en paralelo a la ejecución con límite 2 MiB/plazo 55 s;
  permite registrar todos los cortes sin atascar el contenedor por buffer lleno.
- test-broker-installer.py, py_compile y diff-check pasan. Consulta systemctl de
  quatrro-broker-test-*.scope vacía tras finalizar. Log: .dev/broker-systemd.log.
- Evidencia suficiente para F3.5: componente privilegiado opcional, autorización
  estrecha, instalación/ciclo de vida y prueba aislada real. F3.6 disponibilidad UI,
  F3.7 compatibilidad y R1 continúan pendientes; no acredita cortes eléctricos.

## F3.6 — Consulta administrativa de permisos

- Tests dirigidos con race pasan broker (2,346 s) y core (1,362 s): check_only
  jamás invoca systemctl, doble comprobación/revocación conservadas, respuesta
  authorized no aceptada como completed y viceversa; rechazo antiguo se identifica.
- administration.check probado vía dispatcher: fuerza check_only, unidad/operación
  fijas, redacción de errores, no grants/jobs y rechazo local de unidad inválida.
- Broker/helper recompilados; script integral con systemd/Polkit reales pasa,
  permission_checks_without_effects=true: start no activa, restart/stop conservan
  PID/estado y revocación deniega. Incluye regresión de instalación y ocho SIGKILL.
- UI y activación todavía no consumen la API; no se declara F3.6 terminada.
- Suite completa QUATRRO_POLKIT_TEST=1 go test -race ./internal/broker ./internal/core
  -count=1 pasa (2,524 s y 14,448 s). Diff-check pasa; scopes temporales ausentes.

## F3.6 — Editor y activación con autorización reciente

- Pruebas administrativas con race pasan (2,186 s): preflight exige check_only,
  denegación/ausencia/incompatibilidad impiden activar y no escriben grants;
  flujos deshabilitados no llaman al broker. Tests existentes adaptan solo su
  fixture de activación para simular autorización, nunca el worker real.
- Build de tres binarios y test-ui-flow.py pasan: editor muestra unavailable al
  consultar broker ausente; textos inglés/español; respuesta authorized de revisión
  anterior se ignora tras cambiar unidad. Revisar muestra diez capacidades; activar
  falla por preflight. Deshabilitar flujo administrativo reduce a nueve, activa y
  completa una entrega HTTPS. Scripts, directorios, adaptadores e historial pasan.
- Captura ui-admin-action.png inspeccionada: botón y estado legibles, inglés por
  defecto; todavía controles genéricos, R1.UI pendiente. No se atribuye QA nativa.
- Resultados admin pendientes se correlacionan FIFO con revisión del editor; editar,
  cerrar o reabrir invalida resultado. Backend conserva cola y errores existentes.
- Pendiente QA F3.6: caso positivo UI/broker real y comprobaciones finales de estados.
- Suite completa go test -race ./internal/core -count=1 pasa (14,676 s).
  Diff-check pasa. El perfil permanente no se reinstaló en esta iteración.

## F3.6 — UI positiva contra broker real

- scripts/test-broker-systemd.py --ui pasa: QML y quatrrod UID 1000 dentro del
  contenedor, entorno explícito sin conexión al escritorio/servicios del host,
  perfil temporal. /project de solo lectura; solo capturas en carpeta dedicada.
- Editor muestra authorized real, español e inglés, luego denied para otra unidad.
  Cambiar recurso invalida el resultado. Revisión contiene una capacidad start
  quatrro-fixture.service, activación positiva no inicia servicio. Evento CLI
  posterior atraviesa worker/broker/Polkit/systemd y ejecución queda completed.
- Reporte ui_real_authorization/ui_real_denial/english_spanish/
  activation_without_effect/event_started_fixture=true. Requiere reporte y PNG;
  no confunde seleccionar --ui con haber ejecutado correctamente su recorrido.
- .dev/broker-admin-en.png y broker-admin-es.png inspeccionadas: textos legibles
  y estado positivo bilingüe; controles genéricos offscreen, no QA nativa R1.UI.
- Consola completa .dev/broker-ui-integration.log, incluye regresión de broker,
  límites, ciclo de instalación y recuperación de ocho SIGKILL. py_compile y
  diff-check pasan. F3.6 cerrada; F3.7 y gates finales permanecen abiertos.

## F3.7 — Negociación antes de operaciones

- TestClientCompatibilityGateDoesNotSendOperation prueba motores viejos, contrato
  distinto y saludo inválido: reciben solo hello sin payload privado/operación.
- TestServerRejectsUnnegotiatedAndChangedContracts prueba cliente viejo, ausencia
  de saludo y cambio de contrato entre saludo/operación: cero llamadas al handler;
  pareja compatible despacha exactamente una. Incluye frames concatenados.
- Cancelación de contexto sin deadline interrumpe handshake en menos de un segundo.
  Cliente verifica SO_PEERCRED antes de enviar; plazo se limita a 15 s o menor.
- Suite QUATRRO_POLKIT_TEST=1 go test -race ./... -count=1 pasa: local 1,053 s,
  hooks 2,038 s, core 14,603 s, broker 2,534 s y demás paquetes. No habilita la
  suite host completa QUATRRO_HOST_TEST. Diff-check pasa.
- Versión del broker/configuración permanece separada. UI/CLI y verificación de
  versiones de artefactos aún pendientes; F3.7 no se marca completa.
- Tres binarios recompilados y test-ui-flow.py pasa con protocolo 2: nueve
  capacidades activadas, una entrega HTTPS y recorrido QML completo. Perfil
  permanente no actualizado; la actualización conjunta queda en F3.7/R1.

## F3.7 — UI/CLI versionada

- cmd/quatrroctl tests: envelope compatible preserva payload; contrato faltante/
  ajeno, claves duplicadas o case alternativo, campos desconocidos, concatenación
  y operaciones reservadas rechazados. Validación precede paths.Resolve y Call.
- scripts/test-cli-contract.py con binarios reales pasa: --version describe daemon
  y CLI sin crear estado/config; tres contratos inválidos no conectan al socket
  controlado; válido envía hello sin data y luego emit con payload intacto.
- Build de tres binarios y test-ui-flow.py pasan con la nueva entrada ui: nueve
  capacidades activadas, una entrega HTTPS, scripts/adaptadores/admin/preflight,
  traducción y configuración desde QML conservados.
- Backend conserva correlación de la operación original para sus callbacks, aunque
  la CLI reciba ui. Mensaje de componentes incompatibles/unknown operation: ui se
  muestra con pasos de actualización en inglés/español. Metadatos compartidos en
  internal/release; verificación de instalación aún pendiente.
- Diff-check pasa. No se actualizó el perfil permanente ni se cierra F3.7.

## F3.7 — Snapshot y verificación de versiones al instalar

- validate_components ejecuta snapshots privados de bytes y contrasta identidad,
  contrato local/UI y release con requisitos QML compartidos; los mismos bytes se
  instalan. Descriptores inválidos/no verificables y tamaño excesivo se rechazan.
- Trece tests tests/test_install.py pasan: cada eje incompatible, componente ajeno,
  bool en lugar de versión numérica, requisitos incompletos y actualización rechazada
  preservando archivos/recibo, sin backup nuevo. Suite conserva actualización,
  backups SQLite, restauración sin grants y desinstalación previa.
- Recibo y JSON de instalación contienen compatibility; formatos anteriores de
  recibo se siguen admitiendo para actualizar. Las pruebas usan staging temporal,
  no modifican el perfil permanente ni habilitan servicios del host.
- test-ui-flow.py pasa con Compatibility.js: nueve capacidades, entrega HTTPS,
  historial/configuración/idiomas/admin preflight. py_compile y diff-check pasan.
- QA visual de incompatibilidad y recuperación aún pendiente; F3.7 sigue abierta.

## F3.7 — Rechazo gráfico y recuperación (2026-09-28)

- `python3 scripts/test-ui-compatibility.py`: pasa con motor/CLI/QML reales,
  perfil y copia de UI temporales; contrato 999 rechazado en EN/ES, borrador
  conservado, configuración del motor sin cambios; contrato 1 restaura guardado.
  `.dev/ui-compatibility-en.png` y `-es.png` inspeccionadas, mensajes completos.
  No es prueba de mezcla instalada ni de estilo nativo: contratos inyectados solo
  en harness efímero; producción permanece intacta. Procesos terminados al salir.
- `python3 -m unittest discover -s tests -p test_install.py`: 13 pruebas pasan
  (2,097 s). `python3 scripts/test-cli-contract.py`: metadatos sin arranque,
  rechazo sin conexión y contrato válido con negociación pasan.
- Pruebas race filtradas Compatibility/Contract/Adapter: local 1,039 s,
  adapters 1,024 s, core 1,542 s. CLI no coincidía con filtro; ejecutado de nuevo
  sin filtro, pasa (1,013 s). No se atribuye cobertura host al comando filtrado.
- `QUATRRO_HOST_TEST=1 go test -race ./internal/core -run
  '^TestHostAdapterRefusalDoesNotAdvance$/invalid-response$' -count=1 -v`:
  pasa (4,544 s), adaptador real aislado mediante systemd/Bubblewrap. Respuesta
  ahora usa ID correcto para demostrar rechazo debido a versión 2 por sí sola;
  el flujo no avanza al efecto siguiente. F3.7 cerrada; otros gates siguen abiertos.

## F1B.6 — Reenvío manual desde QML (2026-09-28)

- `python3 scripts/test-ui-flow.py`: pasa recorrido completo, diez capacidades,
  tres peticiones HTTPS. Fixture local con certificado temporal y excepción de
  autoridad exacta: 204 previo, 400 terminal, 204 después de confirmar reenvío.
- Botones y harness comparten inspectExecution/retryDelivery; confirmación usa
  señal accepted del diálogo real. Cancelar en EN/ES conserva estado failed y
  número de peticiones. Confirmar conserva ID de ejecución, cuerpo y clave de
  idempotencia; historial termina completed y detalle delivered/status 204.
- Detalle del fallo muestra 400/un intento, sin cuerpo ni secreto de entrada.
  Captura `.dev/ui-delivery-failed.png` inspeccionada; contenido de advertencias
  `.dev/ui-retry-en.png` y `-es.png` legible. Estas últimas muestran solo contenido,
  no botones/título; título y apertura se verifican mediante propiedades QML.
- Primera ejecución detectó que el contentItem implícito no se podía capturar.
  Label explícito como contentItem lo corrige; segunda ejecución completa pasa.
  Sin errores QML ERROR/ReferenceError/TypeError; avisos offscreen/máscara y QUrl
  del harness siguen sin atribuirles validación nativa. R1.UI permanece abierta.
- `git diff --check` y compilación Python pasan. No se modificó la instalación
  permanente; perfiles/procesos/receptor de prueba se limpian al finalizar.

## F1D.3 — Simulación explicada y prueba real (2026-09-28)

- TestSimulationExplainsRejectedFlowsWithoutEffectsOrSecrets: flujo coincidente,
  deshabilitado, origen ajeno, condición falsa y campo ausente; destino bearer con
  referencia inexistente se puede simular, cuerpo resuelto y cero jobs/outbox/grants.
  Evaluación de condiciones compartida con matches, sin cambiar semántica.
- Pruebas Simulation/Rules con race pasan (1,307 s); suite core completa con race
  pasa (15,397 s). No se habilitó QUATRRO_HOST_TEST en esta suite.
- Build de tres binarios pasa. test-ui-flow.py pasa: UI simula EN/ES y comprueba
  informe effects_executed=false, cuerpo previsto, contadores iguales y ninguna
  petición adicional. Cancelar confirmación real también conserva esos valores;
  aceptar emite la prueba que recibe 400, luego reenvío manual termina con 204.
- Capturas `.dev/ui-simulation-en.png` y `-es.png` inspeccionadas: explicaciones
  traducidas y parámetros legibles. Son contenido de diálogo offscreen; pendiente
  pasada estética R1.UI. Borrador divergente/condiciones fallidas desde UI todavía
  pendientes antes de cerrar F1D.3. Guía en docs/operations.md.

## F1D.3 — Cierre: borrador divergente y condición falsa (2026-09-28)

- `python3 scripts/test-ui-flow.py`: pasa recorrido completo. Flujo retry-flow
  activado con condición data.adapter.severity eq error; sample info produce
  evaluación source_matched/enabled true, matched false y condición falsa.
  Cero coincidencias, contadores y receptor sin cambios, explicación EN/ES.
- `.dev/ui-simulation-rejected-en.png` y `-es.png` inspeccionadas: razón y condición
  visibles, mensaje de ausencia de coincidencias. Render offscreen, no QA nativa.
- UI edita acción send, deja dirty y simula: borrador guardado sin error, preview
  muestra Draft only/draft. Confirmación real envía Retry fixture/error del activo;
  descarta que el test estuviera comparando dos revisiones idénticas. Cancelación
  previa en ambos idiomas permanece sin efectos. Reenvío HTTP conserva cuerpo/clave.
- Resultado final: diez capacidades, tres peticiones HTTPS; simulation_conditions
  y draft_active_separation true. Compilación Python y git diff --check pasan.
  F1D.3 cerrada por evidencia combinada con pruebas anteriores sin secretos/efectos.

## F1D.1 — Estado de listener real (2026-09-28)

- TestIngressStatusTracksListener con race pasa (1,092 s): empieza disabled,
  ServeIngress abre 127.0.0.1:0 y reporta puerto asignado, responde HTTP, status
  coincide, cancelación termina stopped sin dirección. Bind 0.0.0.0 rechazado
  queda failed; public_exposure permanece unverified. Primer intento del test
  esperaba 404; corregido a 405 porque el receptor rechaza GET antes de la ruta.
- Build motor/CLI/broker pasa. test-ui-flow.py pasa URL exacta del listener y
  advertencia EN/ES, además del recorrido completo previo (tres peticiones HTTPS).
- test-ui-compatibility.py pasa estado disabled en ambos idiomas, conserva pruebas
  de incompatibilidad y luego termina motor: consulta real fallida deja connected
  false y texto de receptor desconocido. No se infiere disponibilidad de proxy.
- Documentación de API/exposición actualizada. Pruebas con perfiles temporales;
  no se reinicia ni cambia la instalación permanente. F1D.1 todavía abierta para
  comprobación de gestión de credenciales/conexiones desde UI.

## F1D.1 — Ciclo de credencial inbound desde QML (2026-09-28)

- `python3 scripts/test-ui-flow.py`: segunda ejecución pasa con
  credential_rotation_deletion=true. Credencial incoming rotada por formulario;
  secret anterior produce 401 y contadores iguales, nuevo produce 202. Diálogo de
  eliminación visible/titulado EN/ES; cancelar mantiene referencia. Aceptar elimina
  credencial de lista; firma nueva devuelve 503 y contadores iguales. Restaurar
  referencia original desde formulario recupera 202 sin reactivar configuración.
- Primera ejecución detectó que secretID sin calificar en onAccepted entraba en
  conflicto con ID QML del campo de edición. Referencia explícita al diálogo
  corrige operación; prueba confirma efecto mediante HTTP, no solo estado visual.
- credentialInputEmpty true; secretos ausentes del estado JSON del harness y
  detalles de las dos ejecuciones retenidas. Efectos pausados durante probes y
  cancelados al final; tres peticiones HTTPS originales, ninguna adicional.
- Backend file explícito en perfil efímero. No afirma cobertura del keyring del
  usuario ni del ciclo outbound; F1D.1 mantiene ese próximo pendiente.
- Eliminado aviso de captura a QUrl nulo: Timer de Development ahora exige valor
  truthy en QUATRRO_CAPTURE. Nueva ejecución conserva solo avisos esperados de
  plataforma offscreen/máscara. Python compile y git diff --check pasan.

## F1D.1 — Cierre: autenticación outbound desde UI (2026-09-28)

- test-ui-flow.py crea destino HTTPS bearer por formulario, referencia outgoing,
  acción/flujo y once capacidades revisadas/activadas. Fixture /authenticated
  responde 401 salvo Authorization con token esperado. Primera entrega completed;
  rotar credencial por QML y cambiar expectativa del receptor completa segunda
  entrega sin reactivar configuración: demuestra uso del valor nuevo.
- Eliminar desde diálogo deja siguiente ejecución pendiente. Detalle confirma
  al menos un intento, next_at positivo, estado pending y status 0; contador de
  peticiones no aumenta, no se envía sin autenticación. Restaurar por QML completa
  automáticamente el mismo ID dentro del plazo. Tres solicitudes autenticadas
  aceptadas y seis HTTPS totales; cuerpos esperados, secretos ausentes de estado
  del harness y detalle consultado. Campos de edición vacíos tras guardar.
- Primer intento del test esperaba failed y venció; código de producción ya
  reintenta credenciales ausentes con espera. Expectativa corregida a pending y
  recuperación automática, sin modificar la política de ejecución.
- Segunda ejecución completa pasa outbound_credential_cycle=true y todos los
  recorridos anteriores. Python compile/git diff --check pasan. Perfiles y secretos
  efímeros con backend file; no prueba de proveedor externo ni keyring personal.
  F1D.1 cerrada; revocación UI F1D.5 y resto de gates permanecen pendientes.

## F1D.5 — Permiso revocado antes del envío (2026-09-28)

- WorkerPauseRevocationAndSequentialSteps pasa (race 1,125 s): primer paso ejecutado,
  permiso revocado antes del segundo, registro del segundo denied con mensaje
  estático, tick adicional sin repetir acción. Core/race completo pasa (14,867 s),
  sin QUATRRO_HOST_TEST; tres binarios recompilados.
- test-ui-flow.py pasa: entrega credential-output emitida desde UI con efectos
  pausados; grant visible en Seguridad, revokePermission compartido con botón,
  grant desaparece. Reanudar hace denied la misma ejecución, seis HTTPS totales
  sin incremento y ninguna entrega outbox creada para ese trabajo.
- Detalle consultado desde UI contiene paso denied y action permission missing or
  changed, sin secretos. `.dev/ui-revoked-step.png` inspeccionada, render offscreen.
  Historial del paso ahora se escribe atómicamente con estado global mediante
  finishJob; antes la rama de falta de grant no registraba detalle.
- F1D.5 no cerrada: pendiente corregir etiqueta pending de pasos ya completados,
  conservando interpretación de datos existentes. R1.UI nativa sigue abierta.

## F1D.5 — Cierre: historial de pasos (2026-09-28)

- finishJobContext persiste stepState completed cuando avanza la ejecución a
  pending. El estado de ejecución no cambia de semántica y los reintentos HTTP
  pendientes siguen registrados en outbox como pending.
- history.detail normaliza etiqueta legacy de pasos al consultar. Nueva prueba
  TestHistoryNormalizesLegacyFinishedStepsOnly: legacy pending -> completed,
  failed/denied/uncertain/completed intactos, outbox pending intacto y dato antiguo
  sin reescribir. WorkerPauseRevocation comprueba completed antes del paso denied.
- Core/race completo pasa (15,353 s; sin activar tests host), build de tres binarios
  pasa. test-ui-flow.py pasa: detalle tras reenvío tiene paso completed, mantiene
  revocación UI con paso denied y todas las comprobaciones anteriores. Once
  capacidades, seis peticiones HTTPS. Python compile y git diff --check pasan.
- F1D.5 cerrada sumando evidencia previa de secretos/redacción/reintentos. Esto no
  cierra la presentación nativa final R1.UI ni las pruebas de ciclo de sesión.

## F1D.6 — Pausa, cancelación y cierre de panel (2026-09-28)

- test-ui-flow.py pasa usando togglePause compartido con botón real: paused true
  comprobado por QML/status; entrada firmada retenida sin salida HTTP. Diálogo
  Stop jobs/Detener trabajos abierto en EN/ES, rechazo conserva un pending y
  aceptación cancela ese trabajo. Cola vacía y número de peticiones intacto.
- Cerrar panel mediante close deja status CLI disponible y paused true; reabrir
  y togglePause deja paused false en UI y API. No mata motor ni cambia admisión
  al cerrar. No se atribuye esta prueba a deshabilitar plugin en barra real.
- Recorrido completo pasa once capacidades/seis HTTPS junto con gates previos.
  Diff-check/Python compile pasan. Widget/estética e integración real del shell
  permanecen pendientes; F1D.6 no se cierra con controles del panel solamente.

## F1D.6 — Widget en harness con motor real (2026-09-28)

- `python3 scripts/test-widget.py` pasa live_states/english_spanish/shell_contract/
  unload_preserves_engine/reload. Motor efímero sin listener HTTP; trabajo retenido,
  permiso revocado antes de reanudar, ejecución denied sin acción externa.
- Widget horizontal muestra pendiente/pausa; vertical 28 px; capturas widget-en/es
  inspeccionadas. Tooltip/resumen explican que deshabilitar widget no detiene motor.
  Contrato activate probado con shell directo y fallback bar.shell; anfitrión simulado
  registra módulo/payload, por lo que esto no demuestra apertura en barra real.
- Proceso QML termina, CLI/motor siguen; QML nuevo recupera failed=1. Terminar motor
  muestra desconexión y oculta contadores en resumen. Backend/estados son reales.
- Apertura por teclado y Accessible implementadas, eventos de teclado pendientes
  de QA. `omarchy plugin validate .`, Python compile y git diff --check pasan.
  Log QML sin errores; avisos de entorno offscreen/máscara esperados.
- Sesión consultada sin mutarla: loginctl self no existe para agente; list-sessions
  identifica sesión 2 UID1000, show-session informa Wayland active LockedHint=no.
  Próximo QA en shell real, sin darlo por realizado con este harness.

## F0.5 — Instalación actualizada y shell real (2026-09-28)

- omarchy-shell lock status confirma locked/requested/pending/sessionLocked false;
  lastEvent unlocked. listPlugins enabled=true, debugBarGeometry widget visible
  sección right (116×28). El campo active del listado solo aplica a opciones bar,
  no a widgets; no se usó como prueba de carga fallida.
- Preflight del perfil instalado: actions/destinations/entries/flows/monitors 0,
  counts vacío, ningún flujo habilitado. Detener solo quatrrod; install.py actualiza
  47 archivos con snapshots compatibles 2/1/UI1, respaldo backup-sjbev36i; recargar
  systemd, iniciar servicio y rescanPlugins. Sin reiniciar shell ni instalar broker.
- Status real protocolo2, schema5/integrity ok mediante SQLite read-only, grants0,
  idioma en y receptor127.0.0.1:8791. Manifiesto instalado validado, revisión UI
  043340a97b6a5c7129eefa3ba1a9fbfd980b1e41f5906b18639f2eeea4364d10.
- shell summon devuelve ok; consulta posterior Hyprland confirma ventana Quatrro
  Automations mapped true/hidden false en [690,38], tamaño[664,718]. Captura grim
  recortada a ese rectángulo `.dev/native-panel-before.png`, inspeccionada. Muestra
  receptor real, EN, secciones nuevas; confirma UI nueva tras recarga de componentes.
- shell hide seguido de status demuestra motor independiente, sin trabajos/efectos.
  F0.5 cerrada. Interacción mouse/teclado del widget aún no validada en barra real;
  geometría (widget28/bar26 y panel tiled) y apariencia requieren F1D.7/R1.UI.

## F1D.6 — Eventos de entrada y tamaño nativo (2026-09-28)

- Referencia instalada Bar.qml moduleTargetClickable/pressModuleClickTarget exige
  triggerPress; Widget.qml lo implementa y filtra botón izquierdo. bar.barSize
  reemplaza altura28 fija. test-widget.py pasa con motor real, triggerPress y26px.
- test-widget-input.py usa Widget.qml real con Backend/Panel no ejercidos (backend
  stub), qmltestrunner Qt6.11.2 offscreen: pointer izquierdo, derecho sin activación,
  Space/Return/Enter y foco. 4PASS/0FAIL. Fallos de preparación (GTKtheme, TestCase
  invisible) corregidos; no se simulan eventos en escritorio del usuario.
- Instalación64 archivos respaldada en backup-bkqildw9; servicio iniciado sinjobs,
  plugin recargado y daemon-reload aplicado tras aviso de unidad. Geometría real
  widget116×26 y0 visible. Manifiesto válido, lock.isLocked false, motor saludable.
- Navegación por posición real aún falla para Quatrro: panelNavigationSlots requiere
  widget.open/close/opened. togglePanelAt right0 devolvió unknown (índice inválido),
  right1 abrió Agents y se cerró inmediatamente. No se afirma validación de apertura
  nativa por posición. Integrar ese contrato es próximo trabajo F1D.6.

## F1D.6 — Contrato de panel y navegación real (2026-09-28)

- Inspección de shell.isBarWidgetPanelPlugin confirma que kind panel conserva
  propiedad del loader; Widget delega open/close a summon/hide y opened a
  isPluginOpen del PluginShellApi. No instancia otro Panel ni cambia manifiesto.
- test-widget.py pasa: open true, cierre externo reflejado, open/close false,
  además de contrato triggerPress, estado del motor y unload/reload. QtTest input
  vuelve a pasar cuatro comprobaciones; diff-check/manifest/Python compile pasan.
- Instalador actualiza81 archivos con backup-92hk9yu3, daemon-reload/start y
  rescanPlugins; preflight sin pending/running/uncertain y lock false. Shell real
  togglePanelAt right1 retorna quatrro.automations, Hyprland confirma una ventana
  mapped/no hidden; repetir deja cero. summon/hide confirma mismo ciclo sin duplicado.
- Finalmente panel cerrado, status motor ready/counts{}, sin flujos permanentes.
  Se verifica navegación IPC del shell real y eventos de ratón/teclado en QtTest;
  no se atribuye un clic físico en el escritorio a la prueba IPC. Evidencia
  combinada cierra F1D.6; F1D.7 y R1.UI mantienen QA y apariencia pendientes.

## F1C.1 — Notificación visible y límites (2026-09-28)

- TestNotificationArgumentsAndLimits captura argv con ejecutable fixture: -- antes
  del título, título literal incluso --help y etiquetas, cuerpo escapado con enlace
  inerte. Acepta 200/4096 bytes; rechaza 201/4097 y título UTF-8 de202 bytes antes de
  ejecutar. Tests TestNotification* con race pasan (1,243s), incluyendo bus ausente
  y fallo terminal sin reintento. No se confunde fixture de argv con entrega D-Bus.
- Build completo pasa. test-notification-native.py envía por motor/worker reales
  en perfil temporal al daemon de Omarchy. DNDoff/lockfalse consultados; snapshot
  nuevo del propio aviso prueba título literal/cuerpo escapado. counts completed1.
- `.dev/notification-native.png` inspeccionada: <b>literal</b> y comillas/& visibles
  como texto. Primera captura contenía área inferior adicional; repetida con región
  ajustada 981,34 377x57. Dos avisos de prueba en total, cada uno descartado por
  marcador único; no se limpió historial ni notificaciones ajenas.
- F1C.1 cerrada. Guía explica aceptación por servicio frente a visibilidad/DND y
  política sin sesión. Cambio del título validado con build temporal, pendiente
  siguiente actualización permanente. Diff-check y Python compile pasan.

## F2.7 — Límites de commit local y recuperación de SQLITE_FULL (2026-09-28)

- TestLocalStorageBoundariesKeepEffectsUncertain: triggers ABORT en claim, steps
  INSERT y executions advance. Sin efecto en claim fallido; un efecto en fallos
  posteriores; cero steps confirmados y step0 conservado tras rollback. Quitar
  fallo no repite running. Close/Open/Run confirma uncertain después de efecto,
  completed para trabajo antes no reservado, integrity_check ok.
- TestSQLiteFullDoesNotAcknowledgeOrKeepPartialEvent exige modernc sqlite.Error
  Code&255==13; eventos/executions/seen_deliveries vacíos. Ampliar max_page_count
  permite ingerir el mismo full-event y crea una sola ejecución.
- `go test -race ./internal/core -run 'TestLocalStorageBoundaries|TestSQLiteFull|
  TestHTTPStorageBoundaries|TestAbruptProcessCrashRecovery' -count=1` pasa4,459s.
  Incluye pruebas previas HTTP y procesos SIGKILL. Fallos locales son inyectados;
  efectos modelados por contador, SQLITE_FULL proviene del límite real de páginas.
- Pendiente ENOSPC filesystem/WAL y lifecycle sesión/suspensión; no se llenó disco
  ni cerró/suspendió sesión del usuario. F2.7 permanece abierta.

## F2.7 — ENOSPC real en filesystem aislado

- TestFilesystemFullRecovery usa bwrap con host de solo lectura y tmpfs privado
  limitado a 16 MiB. Comprueba tipo/tamaño antes de llenar y exige ENOSPC real.
- Base en WAL, checkpoint TRUNCATE antes del llenado; entrada rechazada con
  SQLite FULL/IOERR, sin evento, ejecución ni deduplicación parciales.
- Liberar espacio permite reintentar el mismo ID; reapertura verifica integridad,
  deduplicación durable y exactamente una ejecución. No ejecuta efectos externos.
- Prueba con QUATRRO_HOST_TEST=1 y race pasa (2,223 s). El primer montaje de
  ejecutable falló sobre raíz readonly; corregido usando un destino existente
  dentro del namespace. Ningún archivo del host fue sustituido.
- F2.7 continúa abierta para ciclo de sesión/suspensión; no se cerró ni suspendió
  la sesión real. No se cambió producción ni instalación permanente.

## F2.7 — Target de sesión y congelación del cgroup

- `scripts/test-service-lifecycle.py` carga una copia temporal de la unidad
  empaquetada, cambiando solo ejecutable/perfil y el target gráfico por uno privado.
- Detener/arrancar ese target conserva PID y admisión de eventos. Congelar el
  cgroup 12 segundos detiene el cursor; al descongelar, coalesce emite una vez
  y skip omite el vencimiento atrasado. No se cambia el reloj del sistema.
- Stop/start del servicio conserva pausa, eventos e integridad SQLite. Unidades
  y perfil temporales retirados al terminar. Prueba real systemd pasa.
- El primer intento usó /tmp, invisible para el cliente por PrivateTmp; corregido
  situando el perfil en .dev. No se cambió la unidad empaquetada ni la instalación.
- Esto demuestra dependencia de targets, congelación y reinicio del motor;
  no demuestra logout completo del gestor de usuario ni suspensión física.
  F2.7 conserva esos límites pendientes de validación integral.

## R1.NAME — Nombre público actualizado

- Manifiesto, panel, barra (nombre corto Automations), notificaciones, valores
  predeterminados, retorno OAuth, descripciones de unidades/Polkit y documentación
  ahora usan Omarchy Automations. El instalador y los mensajes de incompatibilidad
  también usan el nombre nuevo; la UI reconoce además el mensaje antiguo.
- Identificadores técnicos se conservan deliberadamente para no romper selección
  del plugin, datos, secretos, servicios ni firmas de webhooks. Decisión detallada
  en docs/product-name.md; no hay migración de esquema ni de credenciales.
- Binarios compilados y manifiesto válido. Pruebas dirigidas con race pasan
  (local 1,026 s; core 2,901 s), incluidos argumentos notify-send y retorno OAuth.
  La primera ejecución sin permisos de sockets rechazó listeners; la comprobación
  host posterior pasó. test-cli-contract.py y test-ui-compatibility.py pasan:
  inglés/español, rechazo de componentes incompatibles y recuperación del borrador.
- Cambio preparado en fuentes/build; falta desplegar y verificar nombre visible
  en instalación real junto con R1.UI, icono y artefactos finales. R1.NAME no se
  marca completa todavía. No se publicaron cambios en GitHub.

## R1.7 — Paquete de desarrollo reproducible y staging

- scripts/package-release.py prepara tar.gz Linux amd64 con nombre público
  omarchy-automations, contratos/versiones comprobados y hashes por archivo.
  Orden, modos, propietarios y timestamps normalizados; rechaza symlinks de entrada.
- Incluye runtime, UI, instaladores, packaging, ejemplos, esquemas y documentación;
  excluye perfiles, caches, Git y credenciales. dist queda ignorado por Git.
- make build añade -buildvcs=false junto a -trimpath para no incorporar estado
  variable del checkout. No se cambia todavía 0.1.0-dev por una versión estable.
- test-release-package.py pasa: dos tar.gz idénticos, checksums y contenido
  verificados; instalar/reinstalar/desinstalar desde extracción en staging funciona.
  No activa servicios ni modifica la instalación del usuario.
- docs/releases.md documenta construcción, verificación e instalación. R1.7 sigue
  abierta: compilación independiente reproducible, licencias y notas finales;
  R1.UI/nombre desplegado y demás gates siguen pendientes. No hay publicación.

## R1.UI — Botones nativos y confirmaciones

- ActionButton.qml usa el componente instalado qs.Ui.Button y sus tokens de
  qs.Commons: foco por teclado, hover, selección, tipografía y colores del tema.
  Sustituye botones de panel y formularios; detener/eliminar/revocar usan Color.urgent.
- LocalizedDialog conserva aceptación/cancelación y etiquetas EN/ES mediante
  botones nativos; superficie de diálogo usa BorderSurface y tokens del shell.
  El adaptador standardButton conserva la inspección de los recorridos QA.
- scripts/native_ui.py expone módulos instalados a perfiles temporales mediante
  QML_IMPORT_PATH. Integrado en pruebas de UI general, compatibilidad y broker.
  No se copian ni modifican componentes de /usr/share/omarchy.
- test-ui-compatibility.py pasa. test-ui-flow.py pasa completo: 11 capacidades,
  seis entregas HTTPS, simulación, borrador/revisión activa, pausa/cancelación,
  credenciales inbound/outbound, reintento manual y revocación. Sin errores QML.
- Primera prueba completa interrumpida por recarga al editar/formatear archivos
  durante ejecución; repetida con fuentes estables pasó. Se retiraron symlinks
  auxiliares del workspace que impedían validar el manifiesto; validación pasa.
- Captura .dev/ui-timers.png inspeccionada: botones nativos visibles sin recortes
  en panel de desarrollo. No equivale todavía a QA del shell instalado.
- R1.UI sigue abierta: campos/selectores, tipografía y paleta del panel, widget e
  icono, geometrías/temas/teclado y despliegue nativo. Archivo dist anterior queda
  obsoleto frente a estas fuentes; regenerar después de terminar la UI.

## R1.UI — Campos, selectores, interruptores y tema

- InputField usa qs.Ui.TextField; ChoiceField usa Dropdown con adaptación de
  índice/modelo y entrada libre mediante campo nativo. ToggleField usa Toggle
  y conserva clicked/toggled y actualización de estado antes de las señales.
- MultiLineField mantiene Qt TextArea con BorderSurface y tokens del shell;
  ThemedLabel usa fuente/tamaño del tema. Panel/formularios ya no fijan colores
  hexadecimales; jerarquía tipográfica usa Style.font.
- Captura detectó invasión del selector en la columna de contenido y contraste
  insuficiente en textos secundarios del tema actual. Corregidos ancho adaptable
  y texto secundario derivado del foreground; captura final ui-timers.png revisada.
- test-native-controls.py ejecuta QtTest dentro de Quickshell con marcadores de
  éxito explícitos por caso: teclado/ratón reales en texto, dropdown, opción libre
  e interruptor. Detectó pérdida de foco tras clic en Toggle y se corrigió.
  qmltestrunner independiente carece del plugin Quickshell; no se toma su fallo
  como evidencia funcional. Tampoco se usa salida cero sin marcadores como PASS.
- test-ui-compatibility.py y recorrido completo test-ui-flow.py pasan; este último
  repetido después de ajustes visuales, con seis entregas HTTPS y 11 capacidades.
  La primera carga señaló onToggled faltante: señal añadida antes del recorrido válido.
- run-ui.py y smoke-native.py preparan imports nativos en directorio temporal;
  README actualizado. Manifiesto/Python/diff-check pasan. Instalación aún sin cambios.
- R1.UI conserva icono/widget, pestañas y QA de geometría/temas/accesibilidad
  en shell real. El paquete dist anterior requiere regeneración al cerrar cambios.

## R1.UI — Icono y despliegue nativo

- Widget pasa de rectángulo/rayo a qs.Ui.BarIconButton: icono Nerd Font de
  conexiones, pausa, enlace desconectado y advertencia según estado, tamaño óptico
  nativo y colores del tema. Contadores disponibles en tooltip/resumen accesible;
  conserva fachada open/close/opened y activación solo por botón izquierdo.
- Pestañas inbound/outbound sustituidas por botones nativos seleccionables.
- test-widget.py pasa estados reales, EN/ES, apertura/cierre, independencia del
  motor y recarga. test-widget-input.py ahora ejecuta QtTest en Quickshell con
  marcadores de éxito: mouse y Space/Return/Enter pasan. Adaptadas aserciones
  del host de prueba al contexto QtTest; iconos normal/pausa capturados y revisados.
- Instalación actualizada con 104 archivos y respaldo backup-kkjbkj40. Servicio
  reiniciado sin trabajos; manifiesto instalado válido. Shell recargado sin reinicio.
- Barra real mide icono 27×26, alineado con widgets vecinos. Panel visible único
  664×718; capturas .dev/native-panel-current.png y native-panel-es.png inspeccionadas
  en inglés/español, sin recortes ni solapamientos. Primera captura ES fue anterior
  al polling de 5 s; repetida tras 7 s confirma español. Inglés restaurado.
- Toggle nativo right1 abre/cierra exactamente un panel; motor ready, counts vacío,
  idioma en. Nombre Omarchy Automations e icono ya desplegados en instalación real.
- R1.UI/F1D.7 siguen abiertas para QA de formularios/diálogos reales, más tamaños,
  temas y accesibilidad. El artefacto dist previo no representa todavía esta UI.

## R1.UI/F1D.7 — Matriz de formularios, tamaños y temas

- Nuevo test-ui-layout.py carga copias temporales del panel con motor real,
  HOME privado y temas instalados Tokyo Night / Catppuccin Latte. No cambia
  tema/configuración del escritorio ni crea recursos en el perfil permanente.
- 48 combinaciones pasan: ocho clases de recurso, EN 1100×800, ES 664×718 y
  EN 540×600, en ambos temas. Comprueba tamaño efectivo y límites positivos
  del pie, guardar y cancelar dentro de la ventana. Verifica paleta cargada.
- Quickshell no cambia geometría visible solo al modificar implicitWidth;
  harness fija minimumSize/maximumSize para probar tamaños efectivos. Solo en
  copia temporal: no reduce requisitos de producción para hacer pasar pruebas.
- Captura del contenedor interno de ventana no soportada; se capturan formulario
  y pie por separado. Formulario usa fondo explícito igual a Color.popups.background
  únicamente para el render QA, evitando interpretación incorrecta de alfa.
- Capturas ES 664 de acciones en ambos temas inspeccionadas: textos y controles
  legibles, sin recortes. Render offscreen; no demuestra todos los formularios
  dentro del shell real ni revisión con lector de pantalla. Sin errores QML.
- Python compile/diff-check pasan. R1.UI/F1D.7 conservan accesibilidad completa,
  estados interactivos y revisión nativa de diálogos. Instalación sin cambios.

## Accesibilidad nativa y ejemplos documentados (2026-09-28)

- `scripts/test-native-controls.py`: pasa ratón/teclado de campos, selectores y
  toggle; además roles ComboBox/CheckBox, etiqueta EN/ES, acción accesible de
  toggle habilitado/deshabilitado y descripción de contraseña sin valor secreto.
- `scripts/test-ui-compatibility.py`: pasa rechazo de contrato incompatible,
  preservación del borrador, mensajes EN/ES y recuperación al contrato correcto.
- `scripts/test-ui-flow.py`: pasa configuración/activación, simulación, revocación,
  rotación/borrado de credenciales, reenvío manual y seis entregas HTTPS locales.
- `go test ./internal/core -run TestDocumentedUseCases -count=1`: pasa las tres
  configuraciones distribuidas y cuatro escenarios (incluye alerta/recuperación),
  con validación estricta, plantillas resueltas y sin permisos/ejecuciones/outbox.
- Las pruebas de interfaz con motor necesitaron sockets locales fuera del sandbox;
  el primer intento confinado de compatibilidad falló por `setsockopt` prohibido,
  no por un fallo del producto. La repetición autorizada pasó.
- No se ha realizado lectura asistida con un lector de pantalla, ni se han
  desplegado estos últimos cambios de accesibilidad en la instalación permanente.
  Las guías iniciales no cierran la referencia exhaustiva exigida en R1.DOCS.

## Referencia bilingüe de opciones (2026-09-28)

`python3 scripts/render-option-reference.py --check` pasa: referencia EN/ES y
matriz vigentes, sin campos ausentes dentro de entries/destinations/flows/timers/monitors.
El catálogo contiene 37 opciones, cada una con dos valores de ejemplo distintos.
El alcance excluye todavía actions/scripts/adapters y controles globales.
Revisión semántica realizada contra Schema.js, model.go, providers.go, lan.go,
rules.go, timers.go y monitors_extended.go; el generador verifica estructura, no semántica ni red.

## Referencia de acciones, scripts y adaptadores (2026-09-28)

`render-option-reference.py --check` incluye ahora los ocho tipos de recurso
Schema.js. El catálogo tiene 73 opciones; añade parámetros, argumentos,
revisiones y directorios. Se verificaron límites contra command_profiles.go,
scripts.go, adapters/protocol.go y sandbox/directories.go/runner.go. El checker
sigue siendo estructural; no acredita que cada ejemplo se haya ejecutado.

## Credenciales y retención documentadas (2026-09-28)

Catálogo de 85 opciones EN/ES; doce controles del panel ligados explícitamente
por ID a la referencia. Comprobación de generación y enlaces local pasa.
`go test ./internal/core -run 'TestCredentialRotationDeletionAndBackendIsolation|TestDocumentedUseCases' -count=1`
y `scripts/test-native-controls.py` pasan. Se cambia solo texto de error/etiqueta
para representar el límite existente en bytes, sin modificar aceptación de secretos.
Los scopes de ejemplo se contrastaron con la documentación oficial de Google
Calendar; no supone una prueba de cuenta Google real ni habilita GET outbound.

## Controles generales documentados (2026-09-28)

Referencia EN/ES de 119 entradas, con índice por grupo y dos ejemplos por entrada.
Generación `--check`, enlaces/anclas y compilación Python pasan. Se revisaron
handlers del panel para distinguir operaciones inmediatas de las que exigen
confirmación y separar edición/borrador/activación. No se ejecutaron efectos de
producción ni se considera esto una auditoría exhaustiva de cada ejemplo.

## Scripts/adaptadores de la documentación (2026-09-28)

- `go test ./internal/core -run 'TestDocumented(CodeExamples|UseCases)' -count=1`
  pasa las configuraciones anteriores y las nuevas de código: validación exacta,
  hashes, correspondencia de fuente, valores tipados y simulación sin trabajos.
- `QUATRRO_HOST_TEST=1 go test ./internal/core -run '^TestHostDocumentedCodeExamples$' -count=1 -v`
  pasa con acceso autorizado a systemd/Bubblewrap: dos ejecuciones del script y
  dos transformaciones del adaptador. No se monta directorio del usuario, contacta
  servicio externo ni muestra notificación; se comprueba su texto resuelto.
- Guías EN/ES enlazan cinco configuraciones importables en total. Estos casos
  no acreditan todavía todo el recorrido de importar/activar desde la UI final.

## Ejemplo firmado GitHub (2026-09-28)

`go test ./internal/core -run '^TestDocumentedGitHubRelease$' -count=1 -v` pasa.
Carga JSON distribuido, valida que esté deshabilitado, simula dos tags y usa
handler HTTP en memoria, almacén temporal y worker con notificación sustituida.
Entrega válida y duplicado responden 202; alteración 401; otro repositorio/acción
persisten sin ejecución coincidente. Total tres eventos válidos, una ejecución.
No equivale a conexión real de cuenta GitHub, proxy publicado ni popup nativo.

## Servicio de usuario documentado (2026-09-28)

`TestDocumentedServiceSimulation` y `QUATRRO_HOST_TEST=1 ... -run '^TestHostDocumentedService$'`
pasan. El segundo activa configuración distribuida con únicamente el nombre de
unidad sustituido por uno temporal único, y usa el worker real. Verifica PID
sin cambio para status, PID nuevo para restart, bloqueo por revocación, estados
persistidos (dos completed, un denied) y parada/retirada al finalizar. No toca
servicios preexistentes ni requiere instalar broker administrativo.

## Recuperación HTTP documentada (2026-09-28)

`TestDocumentedHTTPRecovery` pasa cargando el JSON distribuido. Simula ambos
mensajes y controla resultados del emisor 503/400/204 sobre worker/outbox reales.
Comprueba espera persistida, transición pending/failed, reenvío manual que respeta
pausa, cuerpo/clave originales pese a edición de borrador y rechazo de completed.
No usa receptor externo; adelanta explícitamente next_at en su base temporal.
No prueba por sí solo TLS, autenticación o idempotencia de un receptor remoto.

## Ejemplo Google OAuth documentado (2026-09-28)

`TestDocumentedGoogleOAuth` pasa validación del JSON importable, dos cuerpos
simulados sin credenciales, transporte controlado con URL/método/cabecera/cuerpo
esperados y ausencia de segunda llamada tras borrar credencial. No contacta Google
ni prueba consentimientos reales, renovación, DNS/TLS o disponibilidad de calendario.
Documentos EN/ES distinguen estado de entrega de contenido API descartado.

## Campos numéricos nativos (2026-09-28)

Los cuatro límites de Seguridad usan `qs.Ui.NumberField` del Omarchy instalado,
con adaptador de sincronización/clamp y nombre accesible en el SpinBox interno.
`test-native-controls.py` pasa flechas, edición textual, límites dinámicos y disabled.
`test-ui-flow.py` pasa flujo completo con seis entregas HTTPS; compatibilidad UI
pasa rechazo/recuperación de contratos y EN/ES. La instalación permanente sigue
con la revisión anterior; capturas y despliegue final permanecen pendientes.

## Capturas bilingües y corrección de Seguridad (2026-09-28)

`capture-docs.py` genera ocho PNG de Conexiones/Seguridad en EN/ES, Tokyo Night y
Catppuccin Latte, desde QML real fuera de pantalla a 1100×800. Usa motor/perfil
privados, ejemplo distribuido sin activar y receptor HTTP deshabilitado. No cambia
el tema del host. Se revisaron las ocho composiciones; se corrigió ancho del
ScrollView y se comprobó después la posición de los cuatro campos de Seguridad.
También se corrigieron indicadores claros sobre fondo claro vinculando la paleta
numérica al foreground nativo; se inspeccionó de nuevo Seguridad con tema claro.
Los controles nativos vuelven a pasar teclado/edición/límites/accesibilidad.
Las guías EN/ES enlazan las imágenes y explican límites de esta evidencia.

La comprobación posterior de formularios (`test-ui-layout.py`) pasa 48 combinaciones
con dos idiomas, dos temas y tamaños 1100×800, 664×718 y 540×600. Esta prueba cubre
formularios y sus botones, no una lectura integral por tecnología asistiva.
`go test ./...` pasa todos los paquetes; pruebas host opt-in no se habilitaron en
esa ejecución. La referencia generada de 119 opciones coincide con su catálogo.
Diez capítulos por idioma tienen nombres equivalentes y 160 enlaces/anchors
locales válidos. Se añadió guía equivalente de hooks y administración opcional,
con ejemplos de eventos y políticas; no se instaló ningún hook/broker para escribirla.
`test-release-package.py` pasa reproducibilidad, hashes, presencia de ocho PNG e
instalación/actualización/retirada extraídas en staging. No se actualizó el plugin
permanente con esta revisión UI.

## Auditoría de nombre y etiquetas accesibles (2026-09-28)

Se comprobó R1.NAME en fuentes, manifiesto instalado, QML apuntado por sus
entryPoints y manifiesto/release.json del tar.gz. Todos usan Omarchy Automations.
La búsqueda de nombres públicos antiguos no encuentra apariciones en el código
actual; los registros históricos no se reescriben. Se conserva el ID técnico
quatrro.automations sin migración. Esta evidencia complementa la comprobación
anterior del nombre dentro del shell real y cierra R1.NAME, no toda R1.UI.

Los campos anidados y de simulación/importación tienen ahora nombres accesibles;
ActionButton admite etiqueta distinta del símbolo visible. test-native-controls.py
comprueba en QML vivo nombres EN/ES para condiciones, parámetros y pares, además
de las pruebas previas de teclado/ratón/contratos. Validación de plugin y diff-check
pasan. No se ha realizado una lectura con lector de pantalla ni desplegado esta
última revisión en el perfil permanente.

## Compilación independiente y publicación documentada (2026-09-28)

`test-reproducible-build.py` pasa: Go 1.26.8 linux/amd64, dos rutas fuente distintas,
cachés de compilación separadas inicialmente vacías, flags trimpath/buildvcs=false.
Comparte caché de módulos y herramienta; GOPROXY/GOSUMDB=off y GOTOOLCHAIN=local
impiden descargas automáticas de dependencias/herramienta. No se afirma aislamiento
completo de red ni verificación independiente de la procedencia del compilador.
SHA256 idénticos en ambas compilaciones:

- quatrrod: `2587685a3cb83d77ecec8f2155fad11e299842a6c85ebef39a768ba2a952fb82`
- quatrroctl: `7daf5f996e2e341fdefded65e7a154428d06a59b46f3d1fae9d629980fd0f4d5`
- quatrro-broker: `6f54c9a5ddd505dcd76d470fb94f118295a4c7ab5bea083623f67413c16dfb80`

Guías de primera publicación EN/ES describen autenticación, identidad local,
revisión de archivos, commit, remoto vacío/existente y comprobación del push.
Comprobado localmente: rama main, origin correcto, sin HEAD ni identidad local
configurada. No se autenticó ni publicó. Se solicitó al propietario elección de
licencia; permanece pendiente y no se atribuyen permisos de distribución elegidos.

## Admisión y recursos de instalación limpia/reinstalada (2026-09-28)

`test-installed-load.py` pasa con binarios instalados en HOME temporal, rutas XDG
reales y listener loopback. Dos fases (limpia y reinstalación misma versión), cada
una con 5 s de reposo, 400 POST JSON/~1 KiB y cuatro clientes/entradas. No hay flujos
ni efectos: mide admisión y persistencia, no throughput de acciones.

- Limpia: 0,406 s por ráfaga, p50 3,789 ms/p95 4,672 ms; CPU idle 0,01 s y carga
  0,58 s; RSS idle 19856 KiB, carga/pico 23784 KiB.
- Reinstalada: 0,581 s, p50 5,571 ms/p95 6,767 ms; CPU idle 0,02 s y carga 0,76 s;
  RSS idle 21940 KiB, carga 24460 KiB/pico 24852 KiB.
- Cada fase añade 20 solicitudes a una entrada y verifica 429 en la 121 de esa
  ventana. Eventos 420/840, cero ejecuciones, integrity_check ok. El idioma parte
  en inglés y conserva español tras reinstalar. Retirada conserva base de datos.
- Cifras de una observación, no SLA/tasa sostenida. Límites efectivos: 16 handlers,
  120 intentos/entrada/min y 600 global/min; guías EN/ES explican diferencia.

Los 13 tests de instalación pasan (propiedad/rutas, preferencias, backups,
restauración, compatibilidad y retirada). `test-install-runtime.py` pasa motor
instalado, actualización, restauración pausada con cola incierta y permisos
revocados, idioma restaurado y datos conservados al desinstalar. No toca el servicio
permanente. Los cinco escenarios integrales de R1.5 siguen separados de esta prueba.

## Cierre R1.2: esquema anterior con runtime instalado (2026-09-28)

`test-install-runtime.py` reconstruye una fixture v4 sin executions.context tras
parar el motor. Reinstala, arranca binario instalado y comprueba versión 5, trabajo
pending/context vacío, configuración e idioma conservados. Restaura el backup v4
y vuelve a arrancar: pausa, cola uncertain, permisos vacíos, borrador e idioma
originales. Pasa y elimina únicamente la instalación temporal conservando datos.
No afirma haber ejecutado una distribución histórica: prueba su layout anterior.

También pasan TestMigrationPreservesRowsAndRejectsFutureSchema,
TestJournalMigrationPreservesTimerAndEvent, TestVersionFourMigrationPreservesPendingJob
y TestTimerValidationAndMigration. Con las pruebas de backups/restauración y guías
operativas se cierra R1.2, sin atribuirle los cinco escenarios integrales de R1.5.

## R1.5 — A1 y A4 instalados antes/después de migración (2026-09-28)

`python3 scripts/test-installed-acceptance.py --native-notifications` pasa las dos
etapas: instalación limpia y actualización desde fixture de esquema 4 a 5. Usa
binarios colocados por el instalador, HOME/XDG privados, historial/configuración/
credenciales/permisos conservados y TLS verificado contra certificado temporal.
El esquema anterior se reconstruye sin executions.context; no ejecuta un binario
histórico. La segunda etapa no vuelve a guardar configuración ni conceder permisos.

- A1: HTTP loopback con firma HMAC GitHub del cuerpo; firma inválida devuelve 401.
  Reenvío del mismo cuerpo con otro delivery header no duplica. Otros repositorio
  y entorno firmados se aceptan como eventos pero no generan ejecución. Por etapa:
  tres eventos de entrada, una ejecución, una notificación recibida por el daemon
  nativo y un JSON seleccionado recibido por HTTPS con bearer e idempotency key.
  Se comprueban contenido y cantidad de notificaciones con un marcador único.
- A4: instalador y runner de hooks reales de Omarchy dentro del HOME privado,
  theme-set → worker → receptor HTTPS autenticado; historial completed. Hook ajeno
  ejecutado y conservado. Con motor detenido el runner termina en 0,016/0,015 s,
  sin entrega nueva ni bloqueo del hook ajeno. Claves de entrega distintas.
- No se cambia tema ni perfil permanente del plugin. Las únicas interacciones
  con el escritorio son dos avisos identificados, retirados por su marcador en
  finally. Se requiere sesión desbloqueada y DND apagado; no se modifican esos
  ajustes para forzar la prueba. Listener/certificado/secretos son temporales.
- Desinstalación del staging conserva el hook ajeno. Los hooks añadidos por separado
  quedan bajo el HOME temporal y desaparecen con su limpieza.

A1 y A4 cumplen su recorrido instalado en ambas etapas. A2, A3 y A5 todavía deben
pasar esa misma matriz; R1.5 no se cierra. No se contactó GitHub ni un receptor remoto.

## R1.5 — A3 con unidades de usuario reales (2026-09-28)

`python3 scripts/test-installed-acceptance.py --user-services` pasa A3 y A4 en
instalación limpia y tras migrar fixture de esquema 4 a 5. Crea dos unidades
transitorias sleep con nombres únicos, retiradas al finalizar. A3 usa entrada
bearer, unidad exacta en acción, pasos restart → status → reporte HTTPS. Comprueba
cambio de PID autorizado, tres pasos completed y cuerpo autenticado con resultado
active. El reporte solo se emite después de los dos pasos de servicio.

La segunda unidad tiene un flujo separado cuyo permiso se revoca antes de enviar
ningún evento. Su webhook válido produce estado denied, conserva ambos PID y no
emite reporte. El permiso permanece revocado tras actualización. Bearer inválido
se rechaza con 401. No se actúa sobre unidades preexistentes ni servicios root.

La primera ejecución no pasó: XDG_RUNTIME_DIR privado impedía que systemctl --user
localizara al gestor (confirmado con consulta de solo lectura). Se corrigió el
harness, no el producto: conserva runtime real para systemctl y usa QUATRRO_PROFILE
privado para socket/configuración/estado del motor instalado. En esta modalidad el
instalador actualiza binarios y el motor migra ese perfil; no se prueba un backup
automático del perfil alternativo. El backup/restauración XDG estándar está cubierto
por R1.2. Las etapas no vuelven a conceder permisos ni a guardar configuración.

A3/A4 pasan con perfil alternativo; A1/A4 ya pasaron con XDG privado estándar.
A2 y A5 continúan pendientes en la matriz instalada/actualizada de R1.5.

## R1.5 — A5: respaldo programado con runtime instalado (2026-09-28)

`python3 scripts/test-installed-schedule.py` pasa A5 limpio y tras actualización
de fixture v4 a v5. Binarios instalados por el instalador; QUATRRO_PROFILE privado
con runtime real del gestor de usuario para el aislamiento systemd/Bubblewrap.
El temporizador real (10 s, coalesce) genera eventos scheduled, sin emitirlos
manualmente ni adelantar su next_at. Script Python preparado por API y fijado por
hash copia bytes desde montaje ro a montaje rw mediante archivo temporal/replace.
Paso posterior informa backup-completed y scheduled_at por HTTPS autenticado.

- Cada fase produce un respaldo cuyos bytes coinciden con el documento de esa
  fase, una ejecución completed y un reporte con clave de idempotencia distinta.
- Tras preparar/activar se reemplaza el archivo fuente por código que falla. Las
  dos ejecuciones usan la revisión aprobada copiada, no reabren esa ruta.
- El script comprueba que no puede escribir en el montaje ro, que no ve un archivo
  del host sin montar y que no recibe la variable de entorno secreta del motor.
  Credencial fixture no aparece en cuerpos de eventos, mensajes de pasos ni outbox.
- Flujo local adicional ejecuta modo lento con timeout de 1 s: termina failed en
  1,323/1,475 s observados (incluye espera del worker). Tras esperar más allá de su
  escritura prevista no aparece late-effect. No genera un reporte de éxito.
- Migración conserva scripts, grants, directorios, temporizador e historial sin
  volver a guardar/activar. Esta modalidad no prueba backup automático del perfil
  alternativo; eso pertenece al recorrido XDG estándar de R1.2.
- Solo archivos, certificados, credenciales y receptores temporales; no avisos ni
  cambios sobre documentos reales. No afirma aquí medición de CPU/memoria o prueba
  completa de red del sandbox: se comprueban los límites descritos.

A1/A3/A4/A5 acreditados en instalación limpia y después de migración. A2 queda
pendiente en esa matriz; R1.5 continúa abierta.

## Cierre R1.5 — A2 y matriz instalada completa (2026-09-28)

`python3 scripts/test-installed-monitor.py` pasa instalación limpia y actualización
de fixture esquema 4 a 5. Ejecuta instalador y binarios en un namespace Bubblewrap:
host de solo lectura, /tmp privado limitado a 256 MiB y filesystem observado
independiente tmpfs de 32 MiB. Antes de escribir verifica tipo y tamaño. Solo llena
30 MiB de ese filesystem, nunca el disco del usuario. statfs real mide 6,25 % libre.

Monitor con umbral 10/recuperación 20, confirmación 5 s en ambos sentidos, intervalo
5 s y cooldown 10 s: estado pending sin entrega antes de confirmar, una alerta,
dos muestras adicionales bajas sin repetir, liberación del archivo, confirmación
de recuperación y un evento recovered distinto. Cada etapa produce dos entregas
HTTPS autenticadas, estados completed y eventos persistidos. Cuatro claves de
idempotencia distintas. Segunda etapa conserva estado/configuración/permisos y
migra sin reactivar. No sustituye la métrica ni manipula reloj/next_at.

Matriz que cierra R1.5:

| Requisito | Prueba que pasó | Alcance |
|---|---|---|
| A1 despliegue firmado | test-installed-acceptance.py --native-notifications | Firma inválida, duplicados, filtros, aviso nativo y HTTPS; limpio/v4→v5 |
| A2 disco bajo | test-installed-monitor.py | statfs real, confirmación, ausencia de repetición, recuperación y HTTPS; limpio/v4→v5 |
| A3 mantenimiento | test-installed-acceptance.py --user-services | Webhook, PID de unidad autorizada, status, reporte, unidad sin permiso denegada; limpio/v4→v5 |
| A4 hook de tema | test-installed-acceptance.py | Runner real, HTTPS, conservación de hook ajeno, motor apagado sin bloqueo; limpio/v4→v5 |
| A5 respaldo programado | test-installed-schedule.py | Temporizador real, revisión fija, respaldo por bytes, montajes/timeout/secreto, HTTPS; limpio/v4→v5 |
| Reposo/carga | test-installed-load.py | CPU/RSS, 400 solicitudes por etapa, límite por entrada y persistencia; limpio/reinstalación |

No afirma ejecución de un binario histórico: las actualizaciones parten del layout
anterior reconstruido. Servicios/scripts usan QUATRRO_PROFILE privado y runtime de
usuario real; pruebas de backup XDG estándar pertenecen a R1.2. Las cifras de carga
miden admisión sin acciones y no constituyen SLA. Receptores HTTPS locales y fixtures
del proveedor sustituyen cuentas externas según el criterio del roadmap. R1.5 se
cierra; OAuth real, recuperación de sesión/suspensión, revisión UI y demás gates
siguen evaluándose por separado.

## Cierre R1.4 — auditoría y regresiones de seguridad (2026-09-28)

Se contrastaron los requisitos con código/pruebas de ingress/providers, egress/LAN,
config/events/scripts, secretos/diagnóstico, protocolo local, sandbox y broker.
La matriz bilingüe docs/en/security.md y docs/es/security.md delimita evidencias,
modelo de amenaza y límites; no afirma auditoría independiente ni protección
contra malware del mismo usuario.

La auditoría encontró una apertura bloqueante de FIFO en lectura de credenciales:
O_NOFOLLOW impedía symlinks pero fstat ocurría después de open. Se añadió
O_NONBLOCK antes de validar archivo regular/UID/modo y tamaño. El nuevo
TestCredentialFileBoundaries comprueba permisos 0700/0600 y rechazo sin divulgar
valor de permisos inseguros, symlink, directorio, 8193 bytes y FIFO; espera máxima
1 s impide que una regresión quede silenciosamente colgada.

TestDNSChangesRevalidatedWithoutSecondLookup usa resolver controlado y TLS local:
primera dirección permitida se conecta con una sola resolución; una segunda
respuesta multicast se rechaza antes de conectar. La excepción privada es exacta.
Solo la fixture fija el ServerName del certificado local; producción conserva
validación automática del hostname URL. Complementa DNS mixto, IPv4/IPv6,
redirecciones y revocación LAN; no contacta un DNS público ni direcciones externas.

Pasaron go test -race ./... y las nuevas pruebas dirigidas con race. También
QUATRRO_HOST_TEST=1 go test ./internal/core -run '^TestHostIsolation' -count=1 -v:
CPU/cgroup, memoria, tareas, filesystem/red/entorno y timeout/cancelación de
hijos reales. Pruebas host adicionales de broker/Polkit constan en sus secciones.

make check detectó un patrón de cancelación que go vet no podía probar: un defer
cerraba sobre cancelAdmin asignado después. Se movió defer cancelAdmin() al punto
de creación del contexto, conservando inicio perezoso y presupuesto compartido
8 s. No se atribuye una fuga confirmada al patrón anterior. Pasan pruebas Admin/
Preflight con race, go vet, validación de plugin y compilación.

Se cierra R1.4 para estos requisitos comprobados. Integración real Google/keyring,
sesión/suspensión y actualización final del plugin permanente siguen en sus gates.
El nuevo arreglo está compilado; no se ha desplegado todavía al perfil permanente.

## Cierre R1.1 — instalación final de correcciones y revisión nativa (2026-09-28)

Estado previo ready, counts vacío, idioma en y sin pausa. Se detuvo el motor,
se ejecutó instalador normal, daemon-reload y rescanPlugins y se arrancó en finally.
Backup: /home/macondo/.local/state/quatrro-install/backup-q8scdsyx. Recibo de 128
archivos con todos sus hashes comprobados; quatrrod/quatrroctl coinciden con build.
La comprobación inicial incluyó por error quatrro-broker en ~/.local/bin: el
instalador normal no instala el broker. Se corrigió el verificador, sin repetir
instalación ni añadir privilegios. El servicio ya estaba de nuevo activo.

Antes de ese error del verificador se comprobó igualdad exacta de config.get,
permissions.list, contenido de shell.json, idioma/pausa y enablement del servicio.
Se completó validación del manifiesto instalado y de la revisión UI
0647ef60075a92318c3bb3328da0de95b90d5b3fa6d0e14423a0146df1fefe50, con NumericField,
ancho de Seguridad y etiquetas accesibles. El arreglo de lectura de credenciales
FIFO y simplificación de cancelación administrativa están ahora desplegados.

Shell real: panel único 664×718, captura final-native-panel-en.png; navegación
Tab observada hasta New y Enter abre formulario. Capturas finales EN/ES del área
exacta del panel con grim muestran campos y pie Cancel/Save completos. Capturar
solo contentItem QML daba fondo transparente y no se usó como evidencia visual;
las capturas de compositor se incorporaron a docs/images/native-entry-{en,es}.png.
Escape descarta formulario; inglés restaurado, panel cerrado, configuración vacía,
counts vacío y motor ready. No se modificó el tema ni se guardaron recursos.

R1.1 se cierra con esta instalación real y las pruebas previas de preflight,
compatibilidad, XDG, backups y conservación de archivos/preferencias. R1.UI/F1D.7
mantienen su auditoría final global; no se extiende una captura de entrada a todos
los diálogos o estados interactivos.


## R1.3 — desinstalación (2026-09-28)

`python3 -m unittest discover -s tests -v`: 15 pruebas pasan. Las pruebas nuevas
comprueban orden de controles antes del borrado y fallo conservador de archivos.
`scripts/test-uninstall-service.py`: servicio systemd temporal real parado y sin
enablement runtime, binarios retirados, datos/credencial/hook ajeno conservados.
El primer fixture en /tmp no pudo arrancar por PrivateTmp de la unidad; se movió
el fixture a .dev sin debilitar el aislamiento y la prueba pasó. La llamada de
control del plugin en esta prueba está registrada, no ejecutada.

Comprobación independiente posterior al ciclo nativo disable/enable: plugin
habilitado, shell.json semánticamente idéntico al respaldo, quatrrod activo,
ready, idioma en, sin pausa y counts vacío. No se atribuye a esta comprobación
posterior la continuidad de PID durante el ciclo, cuyo resultado no se recuperó.
No se retiró la instalación permanente. Las guías EN/ES aclaran conservación de
hooks manuales y procedimiento separado del broker administrativo.


## Documentación bilingüe — navegación y comprobación (2026-09-28)

- Índices EN/ES organizan las 13 guías temáticas: inicio, interfaz, opciones,
  ejemplos, scripts/adaptadores, proveedor firmado, servicios, recuperación,
  OAuth, integración, seguridad, rendimiento y publicación.
- 14 documentos emparejados por idioma (incluido índice), enlace recíproco de
  idioma y 224 enlaces locales/anchors comprobados. Referencia generada de 119
  opciones y matriz de cobertura coherentes con su catálogo.
- `go test ./internal/core -run '^TestDocumented' -count=1` pasa con GOCACHE del
  workspace: configuraciones importables, código, proveedor, servicio, recuperación
  y OAuth local. El primer intento usó por error caché de HOME de solo lectura;
  falló antes de ejecutar pruebas, corregido sin cambios al producto.
- Estas comprobaciones no equivalen a una revisión semántica exhaustiva de todos
  los valores ni a autenticación Google real. R1.DOCS conserva esas limitaciones
  y la revisión final pendiente; no se declara documentación final completa aún.


## Auditoría de variantes de monitoreo (2026-09-28)

La revisión de Schema.js, validación y transición de monitores detectó una brecha
en los ejemplos: se listaban las doce métricas pero faltaban configuraciones por
variante. Nuevas guías EN/ES monitor-recipes.md y monitor-variants.json proporcionan
dos por métrica (24), deshabilitadas, sin flujos ni acciones. Incluyen unidades,
rutas, sensores, destinos, journal, confirmación y recuperación. Los placeholders
se identifican expresamente. La referencia aclara comparaciones estrictas.

TestDocumentedMonitorVariants valida el archivo distribuido y comprueba igualdad
en alerta/recuperación sin muestrear el equipo. La suite TestDocumented completa
pasa. Se verifican 15 documentos emparejados por idioma y todos sus enlaces locales.
Continúa la revisión semántica del resto de variantes; esto no cierra R1.DOCS.


## Auditoría de acciones y programación (2026-09-28)

Nuevas recetas EN/ES cubren los ocho tipos de acción, dos ejemplos por tipo,
perfiles de comandos file-exists/make-directory y preparación de montajes. Se
contrastaron con Schema.js, actions.go, command_profiles.go y preflight.go; se
aclaran stdout no propagado, límites de status/start/restart con servicios oneshot,
efectos de lock/toggle y alcance de montajes. Corregida nota obsoleta de falta de
despliegue en directory-access.md. Etiquetas de directorios cotejadas con QML.

Guías de casos añaden dos intervalos y dos calendarios, zona horaria, DST, atrasos,
pausa, revocación y diferencia entre horario mostrado y UTC del evento. Pasan
pruebas documentadas/perfiles/Omarchy y pruebas de timers/calendario/programación;
referencia generada consistente. Hay 16 documentos emparejados por idioma con
enlaces locales comprobados. R1.DOCS sigue pendiente de la auditoría restante.


## Artefactos actuales — reproducibilidad y documentación extraída (2026-09-28)

`scripts/test-reproducible-build.py` pasa con Go 1.26.8 linux/amd64, dos árboles
fuente distintos, dos cachés inicialmente vacías y descargas deshabilitadas.
Hashes idénticos para las fuentes actuales, incluidas las correcciones recientes:

- quatrrod: `f3c625878cf4fd6631b3bfbfc58fe56465b35465d8be79d1d18aab5be9650401`
- quatrroctl: `7daf5f996e2e341fdefded65e7a154428d06a59b46f3d1fae9d629980fd0f4d5`
- quatrro-broker: `6f54c9a5ddd505dcd76d470fb94f118295a4c7ab5bea083623f67413c16dfb80`

La prueba de paquete ahora exige todas las guías EN/ES y configuraciones de casos
de uso del árbol fuente, además de resolver los enlaces locales de las guías
desde el archivo extraído. Pasa determinismo byte a byte, hashes y ciclo de
instalación/actualización/desinstalación de staging. Guías EN/ES distinguen
checkout compilable de paquete de ejecución con binarios ya incluidos.
`make check` pasa go vet y validación nativa del plugin.

Sigue siendo 0.1.0-dev: no se anuncia versión estable ni otra arquitectura.
R1.7 mantiene pendientes versión/notas/licencia definitivas y cierre de entrega.


## Controles nativos — comprobación ampliada (2026-09-28)

`scripts/test-native-controls.py` pasa estados de ActionButton mediante eventos
Qt reales en offscreen: teclado, ratón, foco, selección, urgente y disabled.
También pasan etiquetas de orden de pasos en ambos idiomas tras corregirlas.
Referencias y alcance documentados en ui-audit.md; no se presenta como recorrido
del panel instalado ni contraste visual completo.


## Recorrido UI y despliegue de etiquetas (2026-09-28)

Test-ui-flow pasa edición, revisión/activación (11 capacidades), seis entregas
HTTPS, rotación/eliminación de credenciales, reintento manual y revocación de
trabajo encolado. Test-ui-layout pasa 48 formularios con dos temas y tres casos
de idioma/tamaño; el primer arranque sin permisos de socket no inició el motor
temporal y se repitió con acceso local.

Corrección de etiquetas instalada: 152 archivos del recibo verificados, backup
/home/macondo/.local/state/quatrro-install/backup-xotohmw_. Configuración, permisos,
idioma/pausa y shell.json conservados; motor ready. Durante el arranque hubo una
lectura transitoria de socket ausente; la espera posterior confirmó disponibilidad.

Inspección visual de ui-flow.png encontró texto truncado en filtro de Historial
por ancho nativo del Toggle; pendiente corregir. Se cierra solo repetición del
recorrido funcional, no R1.UI global.


## Corrección del filtro de Historial (2026-09-28)

ToggleField admite descripción nativa y la incluye en su etiqueta accesible.
Historial usa título breve EN/ES, descripción completa, ancho disponible y
paginación debajo. La prueba ampliada valida límites y ausencia de elisión del
filtro con cola activada/desactivada en ambos temas y tres tamaños/idiomas.

Los primeros intentos identificaron un desbordamiento real: mínimos de layouts
no bastaban porque el contenedor crecía por otros controles. Limitar el máximo
del área de contenido al ancho de su padre resolvió el filtro. Pasan los 48
formularios y controles nativos. Capturas inspeccionadas: oscuro EN 540×600 y
claro ES 664×718. En 540×600 quedan recortadas acciones del encabezado; registrado
como siguiente corrección, no se cierra R1.UI. Este cambio aún no está instalado.


## Encabezado adaptable y área de navegación (2026-09-28)

El encabezado usa GridLayout adaptable y las cinco acciones principales un Flow
que distribuye filas según ancho. Se mantiene Button nativo y orden de acciones.
La barra lateral usa ScrollView; el área principal queda limitada a la altura
y anchura disponibles. La matriz ampliada prueba 64 formularios, incluyendo
ahora español a 540×600, límites de los siete botones, filtro sin elisión y
límite vertical del viewport lateral. Todo pasa, además de make check.

Captura clara ES 540×600 inspeccionada: botones superiores completos, filtro
legible. El contenido lateral más largo requiere desplazamiento; falta comprobar
interacción de scroll/foco y recorrer el plugin instalado antes del cierre UI.
Los cambios de layout permanecen en fuente; instalación y paquete aún anteriores.


## Foco lateral y despliegue adaptable (2026-09-28)

La barra lateral revela el control al recibir foco, sin cambiar la selección.
Prueba ampliada enfoca Refresh y selector de idioma y comprueba sus límites
dentro del viewport, en temas/idiomas/tamaños de la matriz de 64 formularios.
No equivale a simular toda la secuencia Tab ni gestos de rueda. Pasan controles
nativos y make check. Test-ui-flow vuelve a pasar completo con seis entregas
HTTPS, 11 capacidades y revocación de trabajo pendiente.

Instalada interfaz adaptable y filtro: 176 archivos verificados por hash, backup
/home/macondo/.local/state/quatrro-install/backup-v78xmw4z. Configuración, permisos,
shell.json, idioma/pausa conservados; motor ready. El número de archivos incluye
revisiones UI anteriores conservadas por el instalador. Guías EN/ES explican
las filas adaptables y el desplazamiento lateral. Sigue pendiente revisión visual
en shell real y actualización del paquete después de cerrar cambios UI.


## Revisión instalada de Historial EN/ES (2026-09-28)

Panel real del shell a 664×718 capturado únicamente por región de ventana.
Inspección de Connections: encabezado y acciones completos. Navegación con
Tab/Shift-Tab y foco observado hasta Monitors, después Tab+Return abre History.
Capturas finales native-history-en.png y native-history-es.png incorporadas a
las guías: filtro completo, descripción legible y acciones sin recortes.

Idioma inglés restaurado y panel cerrado; status ready, counts vacío, sin pausa
ni last_error. Esta comprobación no activa flujos ni cambia configuración.
No se extrapola esta revisión a todos los diálogos; R1.UI sigue pendiente del
resto del recorrido visual.


## Matriz de diálogos y variantes (2026-09-28)

Test-ui-layout ahora abre y rechaza, sin aceptar operaciones, revisión de
capacidades, cancelación, reintento, eliminación de credencial, credencial genérica,
simulación, confirmación real, resultado, reset journal, detalle e importación.
Añade importación OAuth, autorización OAuth, exportación y diagnóstico.

Pasan 120 disposiciones de diálogo (15 variantes × 2 temas × 4 casos de
idioma/tamaño) y 64 formularios. Cada diálogo y su pie quedan dentro de la
ventana con dimensiones positivas. Perfiles temporales, sin secretos ni efectos
confirmados. Referencia de opciones consistente. Esta evidencia verifica geometría;
no sustituye inspección de todo el contenido, lectura de contraste ni captura
en shell real de cada diálogo. La revisión R1.UI permanece abierta para eso.


## Capturas de diálogos y contraste del encabezado (2026-09-28)

El comprobador genera 60 capturas completas de PopupItem, incluido fondo/pie.
Capturar window.contentItem falló porque ProxyWindowContentItem no tiene motor
QML; se corrigió el mecanismo sin cambiar producción. Inspección de permisos,
OAuth, simulación, cancelación y diagnóstico detectó encabezado genérico gris
oscuro con texto de bajo contraste en tema claro.

LocalizedDialog ahora define encabezado ThemedLabel con tipografía, espaciado
y color nativos, título multilínea. Capturas posteriores claras ES y oscuras
EN/ES muestran encabezado legible y campos/pie completos. Pasan nuevamente
64 formularios, 120 diálogos, controles nativos y make check. Corrección fuente
pendiente de instalación/capturas reales y actualización del paquete.


## Encabezado instalado y diálogos reales EN/ES (2026-09-28)

Instalada corrección de LocalizedDialog: recibo de 200 archivos verificado,
backup /home/macondo/.local/state/quatrro-install/backup-8nludv_7, configuración,
permisos y shell.json conservados. Motor ready. Capturas reales a 664×718 de
confirmación de detener y formulario de simulación en ambos idiomas: títulos,
textos y botones completos. Apertura con Tab/Return tras observar foco; Escape
descarta sin aceptar ni ejecutar efectos. Inglés restaurado y panel cerrado,
counts vacío. Cuatro capturas añadidas a las guías EN/ES.

La evidencia nativa complementa 120 geometrías y capturas claras/oscuras; se
marca integración de componentes, tipografía y estados de controles como
comprobada. R1.UI global mantiene auditoría final restante y widget/capturas
comparativas antes del cierre definitivo.


## Widget y comparación de controles (2026-09-28)

Test-widget y test-widget-input pasan estados vivos, EN/ES, apertura/cierre,
recarga sin detener motor, ratón y Space/Return/Enter. Geometría real confirma
27×26 en sección derecha, igual que vecinos. Captura reducida de barra inspeccionada.
Auditoría fuente: 50 usos de ActionButton; único Button encontrado es Native.Button
en su adaptador, sin Button/ToolButton/CheckBox/ComboBox genéricos en panel/widgets.
Correspondencias con componentes del shell documentadas en ui-audit.md.

Comparación histórica real antes/después preservada en docs/images e incluida
en guías EN/ES. Captura anterior conserva nombre histórico solo como evidencia.
Inspeccionadas capturas actuales de monitores, programación y cola con sus
acciones nativas legibles. Se cierran criterio de icono y auditoría de botones;
R1.UI global aún requiere consolidar el resto de evidencia visual.


## Cierre R1.UI y F1D.7 (2026-09-28)

Matriz requisito→evidencia consolidada en docs/ui-audit.md. Revisión de fuentes
Network/Tailscale, componentes nativos, 50 botones, widget/icono, estados Qt,
64 formularios/120 diálogos, capturas claras/oscuras EN/ES y comparación histórica
complementan pruebas funcionales y recorridos reales de escritorio.

Test-ui-compatibility pasa rechazo incompatible, conservación de borrador,
recuperación y desconexión; capturas EN/ES legibles. Inspección adicional de
reintento, eliminación de credencial, reset journal, importación/exportación,
credencial genérica y confirmación real no muestra recortes. Correcciones de
filtro, toolbar, foco lateral y encabezados están instaladas y comprobadas.

Se cierran R1.UI/F1D.7 con alcance explícito: temas y tamaños probados, recorridos
nativos representativos y matriz completa de geometrías. No se atribuye prueba
de lector de pantalla ni todos los temas/escales de terceros. F2.7, F3.3 y gates
de documentación, artefactos/publicación y cierre general permanecen abiertos.


## Auditoría de formatos y condiciones (2026-09-28)

Guías payload-recipes EN/ES cubren dos cuerpos por cada entrada JSON/form/raw/XML/
multipart, dos ejemplos por operador eq/ne/gt/lt/contains y dos por salida JSON/
form/raw. Se cotejaron parsePayload, normalizadores, rutas y condiciones en rules.go
y serialización outbound. Se aclaran tipos, campos ausentes, índices, claves con
puntos, límites y simulación frente a autenticación/parsing real.

examples/payload-formats.json contiene diez peticiones con Content-Type, cuerpo
exacto y campo esperado; no es configuración importable. TestDocumentedPayloadFormats
las interpreta con el parser real y verifica resultados tipados. Suite
TestDocumented pasa. Nuevas guías enlazadas desde índices y casos EN/ES; enlaces
y referencias de idioma comprobados. Continúa auditoría final R1.DOCS.


## Contratos de autenticación y ejemplos EN/ES (2026-09-28)

Guías authentication-recipes explican HMAC genérico, Slack, GitHub y bearer,
con dos casos por variante, cabeceras exactas, bytes firmados, reloj, identidad,
reintentos y límites. Se cotejó providers.go; no se presenta la prueba local
como integración con una cuenta externa. Corregida referencia obsoleta a
adaptadores pendientes en providers.md.

Ocho vectores públicos generados con Python hmac/sha256, reloj fijo y valor
explícitamente no secreto se distribuyen en examples/authentication-vectors.json.
TestDocumentedAuthenticationVectors pasa por el verificador Go real y comprueba
rechazo de cuerpo modificado o bearer incorrecto. Suite TestDocumented pasa;
no se realizaron peticiones externas ni se registraron credenciales reales.


## Cierre documental (2026-09-28)

R1.DOCS/R1.6 aceptadas con mapa completo en documentation-audit.md.
check-docs.py pasa 18 pares, 296 enlaces locales y referencia de 119 opciones.
Suite TestDocumented pasa sobre ejemplos distribuidos. Prueba de paquete con
las guías actualizadas pasa determinismo, hashes, enlaces y ciclo staging.
No se cierran gates externos o de entrega por estas comprobaciones.


## Notas de desarrollo y avisos de dependencias (2026-09-28)

Notas 0.1.0-dev EN/ES describen funciones, contratos, actualización, evidencia y
gates externos pendientes. No se cambia versión ni se declara 1.0. Corregida
nota obsoleta de QA pendiente en compatibility.md. check-docs pasa 19 pares y
306 enlaces. Paquete actualizado pasa determinismo/hashes/enlaces/ciclo staging.

Avisos de Go LICENSE/PATENTS y LICENSE raíz de ocho módulos enlazados copiados
textualmente desde toolchain/caché a docs/third-party-notices.md, incluido en
paquete. Inventario deriva de go version -m de los tres binarios. Intento inicial
go list -m all requería metadatos no disponibles en caché y se abandonó sin
descargas; la lectura del build evita esa dependencia. Falta revisión final de
avisos adicionales de componentes derivados antes de cerrar R1.7.

Solicitada elección de licencia del proyecto y participación para consentimiento
Google/suspensión/logout. Sin respuesta todavía; no se ejecutan acciones de
sesión disruptivas ni se inventa licencia/credenciales. R1.7/F2.7/F3.3/R1.8 abiertos.
