# Omarchy Automations — roadmap de implementación

Documento operativo del proyecto. Diseño de referencia: [Diseño de arquitectura](docs/DESIGN.md).

Actualizado: 2026-09-28.
Estado: implementación en curso; núcleo F1A implementado y probado, F1B–F1D avanzadas. Integración permanente y fases avanzadas pendientes.
Siguiente tarea: **publicar el proyecto de desarrollo en PuroDelphi/omarchy-automations. Por petición del usuario, F2.7 (suspensión/cierre de sesión) y la validación real Google de F3.3 se posponen. Permanecen registradas para retomarlas después; no impiden subir el código de desarrollo ni se consideran superadas.**

## Objetivo de entrega

Entregar un plugin nativo para Omarchy que permita configurar webhooks de entrada y salida, flujos, notificaciones, servicios, comandos y monitores. La configuración debe poder hacerse desde la interfaz; CLI y formatos exportables complementan esa experiencia. La interfaz usa inglés por defecto y ofrece español con preferencia persistente (requisito del usuario).

La versión 1.0 incluye las fases 0–3 y la validación de entrega R1. El broker administrativo se implementa como componente opcional, desactivado por defecto. Su uso no será necesario para usar el resto del producto.

Compatibilidad universal significa capacidad de añadir adaptadores, no soporte inmediato para cada protocolo o proveedor existente. La matriz de compatibilidad enumerará formatos y proveedores realmente probados.

## Cómo ejecutar este roadmap

1. Leer el estado y tomar la primera tarea pendiente cuyas dependencias estén completas.
2. Consultar instrucciones aplicables del repositorio y la skill Omarchy al intervenir en la integración del escritorio.
3. Implementar una porción verificable; mantener operativos los casos de uso ya completados.
4. Ejecutar las comprobaciones correspondientes al cambio. Registrar comandos, resultados y limitaciones en `docs/validation.md`.
5. Marcar una tarea como terminada solo con código o entregable y evidencia suficiente. Un archivo vacío, mock o prueba unitaria no demuestra integración real.
6. Actualizar este documento, el registro de ejecución y la próxima tarea antes de cerrar cada sesión de trabajo.
7. Continuar sin solicitar confirmaciones rutinarias. Una dependencia externa bloqueada no impide avanzar en tareas independientes; no marcar como aprobada una comprobación que no pudo ejecutarse.
8. Para reanudar, leer este roadmap y el último registro; no repetir fases cerradas salvo cambios que invaliden su evidencia.

Las casillas indican finalización. El estado de trabajo o bloqueo se anota en la tabla de seguimiento. Las decisiones de arquitectura se registrarán en `docs/decisions/`; los cambios de alcance deben actualizar el diseño y el roadmap.

## Secuencia y dependencias

| Hito | Dependencia | Resultado comprobable |
|---|---|---|
| F0 — Viabilidad | Ninguna | Plugin, motor y CLI se comunican en Omarchy |
| F1A — Núcleo | F0 | Estado duradero, contratos y permisos |
| F1B — Webhooks | F1A | Entrada autenticada y salida persistente |
| F1C — Acciones y monitores | F1A; F1B para escenarios completos | Acciones autorizadas y alertas |
| F1D — Interfaz y MVP | F1B + F1C | Usuario configura un flujo completo sin editar archivos |
| F2 — Integraciones y resiliencia | F1D | Hooks, proveedores y recuperación operativa |
| F3 — Funciones avanzadas | F2 | Scripts, extensiones, OAuth2 y administración opcional |
| R1 — Entrega 1.0 | F0–F3 | Instalación, actualización y desinstalación verificadas |

## Fase 0 — Viabilidad y contratos

- [x] **F0.1 Entorno:** comprobar Go, herramientas QML, Omarchy, `systemctl --user`, D-Bus, SQLite previsto, Secret Service y mecanismos de sandbox. Distinguir host real de entorno de ejecución restringido; registrar capacidades y versiones sin recoger secretos.
- [x] **F0.2 Contrato Omarchy:** inspeccionar esquema de manifiesto y plugins de referencia; decidir entrypoints de widget/panel e interacción permitida con el shell.
- [x] **F0.3 Base del proyecto:** crear módulo Go, motor, CLI, estructura QML, comandos de desarrollo y dependencias fijadas. Crear `README.md`, `docs/validation.md`, `docs/compatibility.md` y primeras decisiones.
- [x] **F0.4 Comunicación local:** implementar `quatrrod`, socket Unix y `quatrroctl status`; permisos del socket, versión del protocolo, verificación del peer y errores legibles. Sin API administrativa HTTP.
- [x] **F0.5 Prototipo nativo:** widget y panel muestran conexión/desconexión y versión mediante CLI/socket; tratar errores sin congelar el shell. Validar manifiesto con `omarchy plugin validate`.
- [x] **F0.6 Integración mínima:** unidad systemd de usuario, notificación de prueba y reconexión de UI. Comprobar qué restricciones del ejecutor pueden aplicarse realmente.

**Cierre:** compilación reproducible, CLI conectada, plugin válido y evidencia de funcionamiento gráfico y recarga sin detener el motor. Si el entorno no permite la prueba gráfica, mantenerla pendiente y avanzar en componentes independientes. No habilitar ejecución avanzada sobre aislamiento supuesto.

## Fase 1A — Núcleo, configuración y permisos

- [x] **F1A.1 Modelos:** esquemas versionados de entrada, destino, evento, acción, flujo, monitor y ejecución; errores de validación útiles.
- [x] **F1A.2 Persistencia:** SQLite, migraciones, inbox/outbox, estados por paso, reservas de trabajos, retención y cuotas. Confirmar entrada solo después de commit.
- [x] **F1A.3 Reglas:** condiciones declarativas, mapeo tipado, límites de evaluación y flujos secuenciales. Sin evaluación de código recibido.
- [x] **F1A.4 Revisiones:** borrador, simulación, activación y revisión inmutable por ejecución; importación desactivada y exportación sin credenciales.
- [x] **F1A.5 Autorización:** capacidades por acción, destino y recurso; revalidación antes de cada efecto y revocación aplicable a trabajos pendientes. El payload nunca define permisos.
- [x] **F1A.6 Secretos:** referencias y backend de almacenamiento; detectar Secret Service bloqueado/ausente. Alternativa de archivo 0600 explícita, sin presentarla como cifrado.
- [x] **F1A.7 Control:** pausar admisión/efectos según política, cancelar trabajos, controlar concurrencia y registrar decisiones sin cuerpos sensibles.

**Cierre:** un evento local simulado recorre un flujo persistido; una revisión nueva no altera ejecuciones existentes y una revocación impide su siguiente efecto. Reiniciar conserva estados coherentes.

## Fase 1B — Webhooks de entrada y salida

- [x] **F1B.1 Receptor:** loopback por defecto, rutas por entrada, POST JSON/formulario/raw, tamaño y tiempos máximos, cuotas y respuestas de error.
- [x] **F1B.2 Autenticación:** HMAC sobre bytes originales, comparación constante, Bearer, rotación y deduplicación. Timestamp firmado cuando el contrato lo contemple.
- [x] **F1B.3 Emisor:** destinos registrados, métodos/cabeceras permitidos, cuerpos mapeados, referencias a secretos y límites de respuesta. Impedir sobrescritura remota de destino o credencial.
- [x] **F1B.4 Red:** HTTPS, política IPv4/IPv6, validación DNS y conexión a IP validada; bloquear destinos internos por defecto. Redirecciones desactivadas; ignorar proxy ambiental no autorizado o someterlo a política explícita para evitar eludir validaciones.
- [x] **F1B.5 Entrega:** outbox duradera, reintentos con jitter, clasificación de errores, `Retry-After` acotado, vencimiento e idempotencia estable cuando sea soportada.
- [x] **F1B.6 Fallos:** inspección y reenvío de entregas fallidas; pruebas con receptor local controlado y excepciones de red limitadas al entorno de prueba.

**Cierre:** evento firmado válido se persiste y llega al receptor de prueba; firma inválida y exceso de tamaño son rechazados; el reinicio conserva envíos; los destinos prohibidos se bloquean también mediante redirección y cambios DNS.

## Fase 1C — Acciones y monitoreo

- [x] **F1C.1 Notificaciones:** D-Bus de sesión, texto acotado, tratamiento seguro de markup y política cuando no hay sesión gráfica.
- [x] **F1C.2 Servicios:** consultar/iniciar/detener/reiniciar solo unidades de usuario autorizadas; verificar resultado real y evitar operaciones simultáneas sobre la misma unidad.
- [x] **F1C.3 Ejecutor:** perfiles con ejecutable fijo, validación semántica de argumentos, entorno mínimo, directorios autorizados, timeout, salida acotada y terminación de procesos hijos.
- [x] **F1C.4 Aislamiento:** aplicar y probar límites de memoria/CPU, archivos y red según perfil. Rechazar la activación si falta una garantía requerida; no degradar silenciosamente.
- [x] **F1C.5 Métricas:** CPU, RAM, disco, batería y estado de unidades. Mostrar «no disponible» cuando el hardware o los permisos no permitan medir.
- [x] **F1C.6 Alertas:** ventana temporal, histéresis, cooldown, recuperación y límites de frecuencia; consolidación tras suspensión.
- [x] **F1C.7 Acciones Omarchy:** catálogo inicial de operaciones concretas verificadas con la CLI instalada; permisos y validación propios, sin pasarela genérica a cualquier comando.

**Cierre:** notificación visible, servicio de prueba reiniciado con autorización, comando registrado limitado y monitor que emite alerta y recuperación sin duplicaciones continuas. Acciones no autorizadas e inyección de argumentos se rechazan.

## Fase 1D — Interfaz completa y cierre del MVP

- [x] **F1D.1 Conexiones:** formularios para entradas/salidas, credenciales, prueba y estado de exposición; nunca mostrar secretos guardados en el historial.
- [x] **F1D.2 Flujos:** asistente «cuando → si → ejecutar → enviar», pasos ordenados, validación contextual y revisión de capacidades antes de activar.
- [x] **F1D.3 Simulación:** vista de condiciones, parámetros y efectos previstos sin ejecutar acciones ni resolver secretos; prueba real diferenciada.
- [x] **F1D.4 Monitores:** configurar métrica, umbral, tiempos y recuperación; mostrar valor actual y última evaluación.
- [x] **F1D.5 Historial y seguridad:** estados por paso, reintentos, errores redactados, revocación y gestión de referencias a secretos.
- [x] **F1D.6 Widget:** salud, pendientes/fallos, abrir panel, pausar y cancelar con semántica clara; comunicar que deshabilitar la UI no detiene el backend.
- [x] **F1D.7 QA nativa:** probar legibilidad, navegación por teclado, tamaños, temas, errores del backend y recarga QML; revisar capturas de las pantallas principales.

- [x] **F1D.8 Idiomas:** inglés predeterminado, selector español y preferencia persistente; formularios, estados, ayudas y botones traducidos sin modificar datos del usuario.

**Cierre del MVP:** desde la UI y sin editar archivos, crear una entrada autenticada, un flujo que notifica y envía resultado, y un monitor con recuperación. Demostrar pausa, revocación, persistencia tras reinicio y manejo de desconexión del backend. CLI funcional por sí sola no cierra esta fase.

## Fase 2 — Integraciones y operación continua

- [x] **F2.1 Hooks:** adaptadores propios para `battery-low`, `font-set`, `post-boot`, `post-update`, `pre-refresh-pacman` y `theme-set`; timeout corto, identidad de evento y convivencia con hooks existentes.
- [x] **F2.2 Proveedores:** contrato de adaptadores y primer verificador GitHub con fixtures oficiales; mecanismos específicos de desafío/respuesta y documentación para añadir otros proveedores.
- [x] **F2.3 Formatos:** XML con entidades externas desactivadas y multipart con límites de archivos/campos; rutas y almacenamiento temporales seguros.
- [x] **F2.4 Programación:** intervalos/calendario, zona horaria y política explícita para tareas atrasadas; reloj controlable en pruebas.
- [x] **F2.5 Monitores ampliados:** procesos seleccionados, archivos en rutas aprobadas, conectividad y temperatura cuando esté disponible; eventos de journald solo con permisos existentes y selección explícita.
- [x] **F2.6 Operación:** rotación completa de secretos, importación/exportación, diagnóstico redactado, retención configurable y herramientas de inspección de cola.
- [ ] **F2.7 Recuperación:** caída durante cada frontera de persistencia/efecto, disco lleno, servicios externos caídos, suspensión y cierre de sesión. Marcar acciones inciertas sin repetirlas ciegamente.
- [x] **F2.8 Exposición:** guía y configuración verificable de proxy/túnel TLS para LAN/Internet sin publicar administración ni activar exposición por defecto.

**Cierre:** hooks no bloquean operaciones de Omarchy si el motor está caído; proveedores y formatos implementados figuran en la matriz de compatibilidad; pruebas reproducibles de recuperación y ausencia de secretos en exportaciones/diagnósticos.

## Fase 3 — Extensibilidad y funciones avanzadas

- [x] **F3.1 Scripts:** registro de revisiones locales aprobadas, parámetros tipados e invalidación al cambiar contenido; ejecutar la revisión aprobada sin carrera entre comprobación y uso.
- [x] **F3.2 Adaptadores externos:** protocolo versionado en procesos separados, manifiesto de capacidades, timeout, cuotas y revocación. Nunca instalar código a partir del payload.
- [ ] **F3.3 OAuth2:** conector inicial concreto, almacenamiento y renovación segura de tokens, estados de expiración/revocación y redacción de errores; seguir el contrato del proveedor seleccionado.
- [x] **F3.4 LAN:** permisos por destino/puerto para servicios privados; mantener bloqueos globales para el resto y probar IPv4/IPv6.
- [x] **F3.5 Broker administrativo:** servicio separado con operaciones/unidades permitidas, identificación del solicitante y políticas Polkit estrechas. Instalar de forma opcional; ninguna ejecución shell root genérica.
- [x] **F3.6 Integración UI:** exponer scripts, adaptadores y operaciones administrativas con su estado y capacidades; no habilitar permisos ampliados por importación o actualización.
- [x] **F3.7 Actualizaciones:** comprobar compatibilidad entre motor, CLI, UI y adaptadores; ante incompatibilidad, impedir efectos y mostrar cómo recuperar.

**Cierre:** script aprobado ejecuta su revisión exacta, adaptador incompatible se rechaza, renovación OAuth2 se prueba y el broker rechaza recursos/operaciones no autorizados. Las pruebas privilegiadas se realizan primero en entorno aislado; escribir las políticas no sustituye ejecutarlas y verificarlas.

## R1 — Empaquetado, validación y entrega 1.0

- [x] **R1.1 Instalación:** paquete o instalador explícito de backend y mecanismo estándar de plugin Omarchy; preflight, versiones, rutas XDG y backups. Instalar no debe reemplazar preferencias existentes.
- [x] **R1.2 Actualización:** migraciones, backup/restauración y procedimiento de recuperación; pruebas desde la versión anterior de datos y configuración.
- [x] **R1.3 Desinstalación:** detener/deshabilitar servicio, retirar solo hooks/archivos propios y permitir conservar estado. No eliminar secretos o datos ajenos.
- [x] **R1.4 Pruebas de seguridad:** firmas, replay según protocolo, SSRF/DNS/redirecciones, inyección, revocación, permisos de archivos/socket, límites efectivos y fuga de secretos.
- [x] **R1.5 Pruebas funcionales:** ejecutar los cinco escenarios de la sección siguiente sobre instalación limpia y tras actualización; medir recursos en reposo y con carga controlada, y documentar límites.
- [x] **R1.6 Documentación:** instalación, primeros pasos, modelo de permisos, ejemplos importables sin secretos, resolución de fallos, contratos de extensión y matriz de compatibilidad real.
- [ ] **R1.7 Artefactos:** compilaciones reproducibles para arquitecturas verificadas, checksums, versiones fijadas y notas de cambios. Preparar artefactos locales; publicación remota es una acción separada.
- [x] **R1.UI Estética nativa (pasada final):** sustituir el aspecto genérico actual de botones y controles por una interfaz coherente con los plugins instalados de Omarchy Quatrro. Inspeccionar sus componentes y patrones reales; adaptar botones, campos, selectores, interruptores, diálogos, tipografía, espaciado, iconos y estados hover/foco/deshabilitado. Integrar colores del tema y verificar contraste, teclado, tamaños y textos en inglés/español. Realizar esta pasada después de completar las funciones y antes de la revisión final, con capturas comparativas del panel, formularios, diálogos y widget.
- [ ] **R1.8 Revisión final:** requiere R1.NAME, R1.GITHUB y R1.DOCS completadas, además de R1.UI completada y comprobada dentro del shell real; los botones genéricos actuales no cumplen el criterio de entrega. Ninguna tarea requerida pendiente, ningún fallo crítico abierto y evidencias localizables. Actualizar estado a «1.0 completada» y registrar limitaciones conocidas.

### R1.UI — Criterios de aceptación visual

Esta fase es obligatoria al terminar las funcionalidades, antes de R1.8. El objetivo es que el plugin se perciba como parte del escritorio Omarchy Quatrro, con especial atención a los botones actuales.

**Condición de entrega solicitada por el usuario:** el aspecto actual de los botones es provisional. La entrega final debe usar el lenguaje visual de los demás plugins del sistema operativo; completar las funcionalidades no permite omitir esta pasada. Conservar una captura del estado anterior y compararla con el resultado integrado en el shell.

- [x] Comparar el panel con los plugins instalados de Network y Tailscale; revisar `qs.Ui` y `qs.Commons` de la versión instalada antes de elegir componentes compatibles.
- [x] Usar los componentes nativos disponibles o adaptar sus patrones comprobados: `Button`, `PanelActionButton`, `TextField`, `Dropdown`, `ToggleSwitch`, `ConfirmDialog` y `BarIconButton`. Verificar sus contratos de integración; no basta con cambiar colores de los controles genéricos.
- [x] Unificar tipografía, dimensiones, separación, bordes e iconografía con `Style` y `Color` del shell, respetando cambios de tema y escala.
- [x] Dar tratamiento consistente a acciones principales, secundarias y destructivas; comprobar estados normal, hover, pulsado, foco, deshabilitado y operación en curso.
- [x] Auditar todos los botones actuales y registrar su equivalente en los plugins de referencia; sustituir su apariencia genérica, incluidas barras de acciones y botones dentro de formularios y diálogos. La aceptación exige coherencia con la interfaz real del sistema, no solo una nueva paleta de colores.
- [x] Revisar panel, widget, formularios, selectores y diálogos en inglés y español: sin recortes, saltos incómodos ni pérdida de foco; inglés sigue siendo el idioma predeterminado.
- [x] Verificar teclado, contraste, tamaños de pantalla y temas claros/oscuros disponibles. Guardar capturas comparativas y registrar qué se comprobó en el shell real y qué únicamente en render de desarrollo.
- [x] Repetir el recorrido funcional de configuración después del cambio visual; cerrar R1.UI solo con evidencia visual y funcional.

Referencias locales de lectura: `/usr/share/omarchy/shell/Ui/`, `/usr/share/omarchy/shell/Commons/`, `/usr/share/omarchy/shell/plugins/panels/network/Panel.qml` y `/usr/share/omarchy/shell/plugins/panels/tailscale/Panel.qml`. No modificar archivos del sistema para adaptar este plugin.

## Escenarios obligatorios de aceptación

| Caso | Preparación | Resultado esperado |
|---|---|---|
| A1 — Despliegue fallido | Fixture firmado del proveedor, filtro de repositorio/entorno | Una notificación y una salida con datos seleccionados; firma inválida no produce efectos |
| A2 — Disco bajo | Métrica controlada que cruza umbral y recuperación | Alerta tras duración configurada, sin tormenta; evento de recuperación distinto |
| A3 — Mantenimiento | Webhook autorizado y unidad de prueba | Reinicia solo esa unidad, comprueba estado y emite resultado; otra unidad es rechazada |
| A4 — Cambio de tema | Hook real o invocación fiel al contrato | Evento entregado al destino autorizado; caída del motor no bloquea el hook |
| A5 — Respaldo programado | Script local aprobado y temporizador | Ejecuta revisión autorizada, respeta límites y reporta resultado sin exponer secretos |

Los receptores externos se sustituyen por servidores de prueba cuando no hay cuentas o credenciales. Eso valida el protocolo local; una integración real pendiente permanece identificada en la matriz, sin atribuirle una validación inexistente.

## Fuera del compromiso 1.0

- Editor visual de grafos, marketplace y sincronización entre equipos.
- MQTT/WebSocket y otros transportes no HTTP, salvo que se amplíe expresamente el alcance.
- Garantía de exactamente una vez sobre acciones externas o rollback universal.
- Aislamiento frente a malware del mismo usuario para todos los componentes; la frontera y limitaciones se documentan.
- Publicación pública, compra de dominios, túneles contratados o conexión a cuentas de terceros sin datos y autorización correspondientes.

## Seguimiento actual

| Elemento | Estado | Evidencia / próxima acción |
|---|---|---|
| Diseño | Completado | `docs/DESIGN.md` |
| Roadmap | Completado | Este documento |
| Fase 0 | Completada | Motor/CLI instalados y panel nativo nuevo verificados; barra registra widget visible y manifiesto válido |
| Fase 1A–1D | En curso | F1A probada; formularios, monitores, entrada/salida y recuperación implementados; quedan QA y cierre de integración |
| Fase 2 | En curso | Hooks, proveedores, formatos, programación, monitores ampliados, operación y exposición comprobados; recuperación F2.7 en curso |
| Fase 3 | En curso | F3.1–F3.2 y F3.4–F3.7 comprobadas; F3.3 mantiene validación externa OAuth/keyring pendiente |
| R1 | En curso | R1.NAME, R1.1, R1.2, R1.4 y R1.5 comprobadas; continúan UI, seguridad, documentación y controles finales |

## Registro de ejecución

### 2026-09-28 — Preparación

- Se revisó el diseño y se convirtió en tareas identificadas con dependencias y criterios de cierre.
- Se creó el directorio de trabajo `quatrro-automations/`.
- No se ha creado todavía código de producto ni instalado componentes del plugin.
- Siguiente paso: F0.1. Las versiones y capacidades consultadas durante el diseño deben verificarse con los comandos adecuados al entorno de implementación.

Formato para futuras entradas: fecha; IDs trabajados; archivos o entregables; verificaciones y resultados; bloqueos concretos; siguiente tarea.

### 2026-09-28 — Primera implementación

- F0.1–F0.4 completadas: compilador local verificado, módulo Go, dependencias fijadas, repositorio Git local, protocolo Unix con control de UID y singleton.
- Panel QML cargado en Quickshell y render interno revisado: muestra motor conectado. Falta integración del widget en la barra real y servicio permanente del motor para cerrar F0.
- Núcleo implementado: borrador/revisión, capacidades por hash, revocación, secretos por referencia, reglas, inbox/outbox SQLite, paso a paso, pausa, cancelación, HTTP firmado, HTTPS con validación de red y reintentos. Estas fases aún requieren completar sus controles y evidencias.
- Pruebas `QUATRRO_HOST_TEST=1 go test -race ./...` pasan: revisiones, revocación, persistencia, deduplicación, firmas, SSRF, JSON seguro, notificación real, servicio temporal e intento de escritura bloqueado en comando aislado.
- Próximo trabajo: ampliar los tests de caídas/reintentos y cancelación HTTP; completar estatus/historial/importación/exportación, monitores, UI F1D y fases 2–3/R1.
- Revisión pendiente explícita: defensa adicional de replay GitHub frente a cambio de cabecera delivery no firmada; política y aplicación de retención/cuotas de disco; validación de capacidades del ejecutor antes de activar; migraciones reales; eliminación/rotación completa de secretos; métricas y pausa visibles en status.

- Continuación del mismo bloque: se añadió worker persistente, outbox con claves estables y reintentos, ejecución de notificación/servicio/comando real, y monitores de CPU/RAM/disco/batería/servicio. Estos últimos tienen prueba de duración/histéresis/cooldown; faltan recorridos reales y formularios.
- No quedan motores o paneles de prueba intencionadamente activos. Herramientas locales en `/home/macondo/Work/.quatrro-tools/`; usar `GOCACHE=/home/macondo/Work/.quatrro-tools/cache` y el binario Go de ese directorio al retomar.
- Próxima implementación concreta: estatus operativo e historial detallado, formularios QML de conexiones/acciones/flujos/monitores, completar F1A/F1B y pruebas de recuperación. Revisar atomicidad de transición de monitores y comportamiento de cancelación HTTP antes de cerrar fases.

### 2026-09-28 — Formularios, operación y recuperación

- Añadidos formularios QML de entradas/destinos, acciones, flujos (condiciones y pasos ordenados), monitores, credenciales y retención. Revisión de capacidades antes de activar; importación desactivada y exportación exclusiva a un archivo nuevo.
- `scripts/test-ui-flow.py` crea recursos mediante las señales reales de los formularios en Quickshell, registra una credencial de prueba, revisa tres capacidades (dos acciones y un monitor), activa y verifica notificación + entrega HTTPS local + muestra real de disco + historial completado. Usa render Qt fuera de pantalla porque la sesión está bloqueada. No equivale a comprobar la barra de Omarchy ni toda la navegación por teclado.
- Capturas nuevas revisadas: `.dev/ui-flow.png` y `.dev/ui-form.png`. El test ahora elimina la captura anterior antes de renderizar para impedir evidencia obsoleta. La navegación lateral tiene ancho fijo.
- Nuevos controles: estado operativo, historial por paso, reenvío HTTP con clave estable, rotación/eliminación de credenciales, permisos de monitores revocables y política de cuotas/retención.
- SQLite migra de esquema 1 a 2 preservando filas y deduplicación; rechaza esquemas futuros. Cuotas de eventos/payload y máximo de páginas; limpieza conserva trabajos pendientes e inciertos y deduplicación separada. Prueba de base llena confirma que no se acepta ni persiste parcialmente el evento.
- Corregidos: transición de monitor y evento en una sola transacción; cancelación HTTP sin reprogramación automática; deduplicación GitHub ligada al cuerpo firmado; las reglas ya no usan cabeceras de tipo sin firmar; salida JSON preserva tipos escalares y evita inyección; salida formulario y raw implementadas.
- Recuperación probada: comandos inciertos no se repiten; entregas confirmadas sobreviven una caída antes de confirmar el paso; reintentos conservan cuerpo, clave, intentos y plazo al reabrir la base.
- Host: rotación y borrado de una credencial temporal en Secret Service, métricas CPU/RAM/disco, notificación, servicio temporal y comando aislado. Catálogo Omarchy limitado a tema actual, estado/cambio de luz nocturna y bloqueo; se probó la operación de lectura de tema, sin cambiar preferencias ni bloquear la sesión.
- F1C.4 sigue pendiente de comprobar cada límite de CPU/memoria/red y cancelación de descendientes; la activación de comandos ahora prueba el perfil de sandbox antes de conceder permisos.
- F1D pendientes: completar pruebas de simulación/prueba real/pausa/revocación/reconexión desde UI, conexión de widget con barra real, accesibilidad y diferentes tamaños/temas. F1B.6: ampliar evidencia de reenvío manual y gestión de fallos desde UI.
- Próximo bloque: integración nativa F0, aislamiento restante y cierre MVP; luego F2/F3/R1 completos según las tareas originales. No se han descartado broker administrativo, OAuth2, scripts, extensiones ni la guía final de GitHub.

### 2026-09-28 — Hooks y separación de permisos

- F2.1 completada: seis adaptadores en `packaging/hooks/`, normalización tipada en `internal/hooks` y comando `quatrroctl hook`. El socket espera un máximo de un segundo; los wrappers acotan además el proceso y no transmiten fallos a Omarchy.
- `scripts/test-hooks.py` usa el ejecutor real de Omarchy con HOME y perfil temporales: seis eventos persistidos, argumentos preservados, otro hook intacto y operativo; los seis hooks con motor apagado finalizaron en 0,070 s en conjunto. No se instalaron hooks en la configuración personal.
- Prueba de socket que acepta conexión pero no responde: la llamada del hook termina por timeout. Los eventos no confirmados no se reintentan automáticamente; comportamiento documentado en `docs/hooks.md`.
- Corregida una colisión de nombres entre capacidades de acciones y monitores. Ahora las acciones usan `flow:<flujo>:action:<acción>`; los permisos antiguos fallan de forma cerrada y requieren nueva revisión/activación. Prueba de regresión incluida.
- Próxima tarea: cerrar integración F0 y las verificaciones pendientes del MVP; continuar F2.2–F2.8 y F3/R1 sin reducir su alcance.

### 2026-09-28 — XML y multipart

- F2.3 completada: receptores XML y multipart, validación de configuración, esquema JSON y opciones del formulario de conexiones. Normalización documentada en `docs/formats.md`.
- XML rechaza DTD/entidades personalizadas e instrucciones de procesamiento; límites de profundidad, tokens, elementos y atributos. Multipart limita partes, campos y archivos, rechaza duplicados y rutas de archivo; contenido en memoria/base64, sin archivos temporales.
- Reglas y plantillas pueden consultar índices de arrays, incluidos hijos XML. Se rechazan índices negativos, fuera de rango o no canónicos.
- Pruebas unitarias y HTTP autenticado verifican formatos válidos, entradas malformadas, límites y persistencia. `go test -race ./...` pasa.
- Recorrido QML actualizado configura ambas opciones de formato y conserva el flujo previo: revisión de tres capacidades, notificación, una entrega HTTPS e historial completado.
- Pendientes originales de F0/F1 permanecen abiertos; F2.2, F2.4–F2.8, F3 y R1 aún requieren implementación/verificación.

### 2026-09-28 — Contrato de proveedores y Slack

- F2.2 completada: `inboundAdapter` versión 1 separa firma, formatos, desafío y confirmación. Registro incorporado estático; un proveedor desconocido falla cerrado.
- GitHub conserva la verificación de su fixture oficial y deduplicación independiente de cabeceras no firmadas. Slack añade firma v0 con timestamp de cinco minutos, desafío autenticado sin crear flujos y HTTP 200 solo después del commit para eventos normales.
- Pruebas con vectores oficiales y reloj controlado; cuerpo alterado, firma/versión incorrectas, cabeceras duplicadas, timestamps ausentes/vencidos, desafío inválido y fallo de almacenamiento. Reintentos Slack con nuevo timestamp producen una sola ejecución.
- Selector Slack en el formulario y esquema JSON actualizados. Documentación del contrato y sus limitaciones en `docs/providers.md`; las integraciones de cuentas reales siguen sin validarse.
- `go test -race ./...`, `go vet ./...`, compilación y manifiesto pasan. Recorrido QML comprueba selección Slack y conserva notificación/entrega HTTPS/historial del flujo existente.
- Próximas tareas: F2.4 programación y las verificaciones pendientes del MVP; completar F2 restante, F3 y entrega R1 antes de declarar terminado el proyecto.

### 2026-09-28 — Intervalos y calendarios

- F2.4 completada: intervalos, hora diaria/semanal, zona IANA y política `coalesce`/`skip`; permiso `timer:<id>`, fuente registrada `timer:<id>` y eventos `scheduled` integrados con flujos. Importación deshabilitada.
- Migración SQLite 3 añade `timer_state`. Próximo vencimiento y evento/ejecuciones comparten transacción; reinicio conserva fase y un fallo de persistencia revierte ambos. Revocación comprobada antes de producir cada evento.
- Pruebas de reloj controlado cubren atraso consolidado, descarte, retroceso, pausa de admisión/efectos, reinicio real de Engine, cambio de revisión, permisos y calendario con cambio de horario. Solo se admite la primera ocurrencia de una hora repetida, incluso si el motor arranca durante la segunda.
- UI: sección Programación, formularios, origen seleccionable en flujos, revisión de capacidades y estado. `scripts/test-ui-flow.py` configura intervalo y calendario; espera un disparo real, revisa cuatro permisos y conserva el recorrido de webhook/notificación/salida HTTPS. Captura `.dev/ui-timers.png` revisada.
- Documentación de semántica, tolerancias y límites en `docs/scheduling.md`. Hasta 64 programaciones; intervalos mínimos de cinco segundos. No despierta equipos suspendidos ni promete precisión de tiempo real.
- `go test -race ./...`, pruebas adicionales de calendario, compilación, vet y validación Omarchy pasan. Quedan F0/F1 pendientes de integración/QA, F2.5–F2.8, F3 y R1 completos.

### 2026-09-28 — Aislamiento efectivo y terminación

- F1C.4 completada sobre el perfil actual de comandos: límites del kernel CPU 50 %, memoria 256 MiB, swap 0 y tareas 32; pruebas reales de throttling, OOM, procesos, archivos/red y entorno.
- Detectado y corregido un defecto: el gestor de usuario tenía 90 segundos de espera predeterminada al parar. Las unidades ahora fijan `TimeoutStopSec=1s`, `KillMode=control-group`, SIGKILL y limpieza explícita del cgroup tras cancelación/error. Prueba con descendiente que ignora SIGTERM y crea sesión propia: desaparecen sus PID.
- Activación reforzada: prueba del ejecutor real y lectura de controles cgroup antes de conceder permisos; rechaza controladores ausentes o ilimitados. Rutas de ejecutable canónicas y argumentos sin NUL exigidos.
- `QUATRRO_HOST_TEST=1 go test -race ./...`: pasa, incluyendo todas las pruebas anteriores, activación real, notificación, servicios, Secret Service, métricas y calendario. No quedaron unidades `quatrro-isolation-test-*` ni `quatrro-probe-*`.
- Detalles en `docs/command-isolation.md`. F1C.3 conserva pendientes los perfiles/validación semántica ampliada de argumentos y directorios autorizados; no se anuncia esa tarea completa solo por cerrar sus pruebas de cancelación. F3 de scripts/extensiones continúa íntegra.

### 2026-09-28 — Diagnóstico y cola operativa

- F2.6 completada: reporte por lista de campos permitidos, vista y exportación 0600 en Seguridad; consulta paginada de cola con detalle/reintentos existentes, accesible también desde Historial. No incluye cuerpos ni secretos.
- `queue.inspect` permite filtrar pendientes, en curso e inciertos y recorrer páginas de hasta 100 filas. `diagnostics` / `diagnostics.export-file` incluyen versiones, estado, cuotas y contadores, sin nombres de recursos, URLs, rutas ni errores arbitrarios.
- Importación reforzada: abre y verifica el mismo descriptor, rechaza enlace simbólico final/FIFO y limita la lectura aunque el archivo crezca. Exportación nunca reemplaza y mantiene el límite portable de 1 MiB.
- Rotación verificada de extremo a extremo: token previo rechazado inmediatamente después de reemplazarlo; token nuevo aceptado; eliminación hace fallar autenticación. Las pruebas de Secret Service, archivos y retención anteriores se conservan.
- `go test -race ./...`, vet, compilación y manifiesto pasan. QML muestra un trabajo retenido por pausa, exporta diagnóstico sin datos de prueba sensibles y observa su salida de la cola tras cancelar; no se produjo un segundo envío HTTPS. Captura `.dev/ui-queue.png` revisada.
- Contratos y pasos operativos en `docs/operations.md`. Permanecen pendientes monitores ampliados F2.5, recuperación completa F2.7, exposición F2.8, integración/QA del MVP y las fases F3/R1.

### 2026-09-28 — Proxy TLS opcional

- F2.8 completada: plantilla Caddy con certificados explícitos y loopback predeterminado; solo POST `/hooks/<id>`, sin API administrativa de Caddy, sin listener HTTP adicional ni autoexposición. Guía LAN/Internet/túnel SSH en `docs/exposure.md`.
- Caddy 2.11.4 oficial descargado al directorio local de herramientas; checksum SHA512 coincide con publicación oficial. No se instaló un paquete global ni se habilitó ningún servicio público.
- `scripts/test-tls-proxy.py` valida y ejecuta la plantilla real con certificados temporales, comprueba confianza TLS, firma sobre bytes originales, deduplicación, bloqueo administrativo, 401 ante firma inválida, 413 por tamaño y 502 al caer el motor. Todos los listeners de prueba son loopback.
- Plantilla de unidad opcional incluida. Sintaxis comprobada con una copia temporal que apunta al binario local; el archivo definitivo espera `/usr/bin/caddy`, todavía no instalado. Instalación/arranque permanente se verifican en R1, sin atribuirles evidencia del test en primer plano.
- Comando de túnel SSH contrastado con documentación oficial y `ssh -G`, sin conexión a servidor remoto ni cuenta externa. Se documentan comprobación de listener efectivo, GatewayPorts y límites de disponibilidad.
- Pendientes: integración/QA F0/F1, monitores ampliados F2.5, recuperación F2.7 y fases F3/R1; ninguna se sustituye por la entrega de la guía.

### 2026-09-28 — Monitores ampliados y recuperación de resultados terminales

- F2.5 completada: procesos propios seleccionados por ejecutable, metadatos de archivos sin enlaces simbólicos, temperatura y comprobación HEAD de destinos HTTPS. Permisos ligados a rutas y, para conectividad, a toda la configuración del destino.
- Journald de unidad de usuario y prioridad explícitas; solo metadatos, sin MESSAGE. Cursor y eventos atómicos, lotes acotados, deduplicación y reinicio de cursor confirmado/auditado. Migración 3→4 preserva eventos y programaciones.
- Host: sensor real leído, proceso propio identificado, servicio temporal genera evento de journal, reinicio conserva cursor sin duplicar. Lecturas inaccesibles aparecen como no disponibles, sin afirmar ausencia de procesos.
- Formularios QML incluyen todas las métricas; recorrido de UI revisa seis capacidades, observa archivo y HTTPS en 100 y conserva notificación, entrega, temporizadores, cola y diagnóstico. Contratos en `docs/monitoring.md`.
- F2.7 avanzada: corregido reenvío indebido cuando outbox ya había persistido fallo o cancelación incierta pero faltaba cerrar el paso. Pruebas de las tres salidas terminales con fallo inyectado y reapertura real pasan.
- Cinco pruebas con SIGKILL en proceso independiente verifican WAL, integridad, acciones inciertas y cuerpo/clave HTTP conservados. Notificación sin bus de sesión falla sin reintento. Semántica y límites en `docs/recovery.md`.
- Suite host con race pasa (core 13,746 s); nuevas pruebas SIGKILL/sin sesión pasan (3,024 s). Compilación, vet, manifiesto y recorrido QML pasan. No se instaló ni expuso un servicio permanente.
- Próximo: completar recuperación F2.7 y cerrar integración/QA pendiente F0/F1; después F3 y R1 íntegros. F1C.1 sigue abierta por verificación visual de notificación; la prueba de bus ausente no la sustituye.

### 2026-09-28 — Cancelación duradera y presupuesto de envíos

- F2.7: intento HTTP reservado antes de despachar, para que una caída después del efecto no borre el intento consumido. Se mantiene la clave y el cuerpo persistidos.
- Cancelación guarda en una transacción los pendientes cancelados y las salidas HTTP en curso inciertas antes de confirmar y señalar trabajadores. Un fallo de commit devuelve error y no confirma cancelación.
- Sexto escenario SIGKILL: caída inmediatamente después de confirmar cancelación dentro de un envío; reapertura conserva incertidumbre y no reenvía. Los escenarios HTTP comprueban también el intento persistido.
- Fallos inyectados sobre creación de outbox, reserva del trabajo, reserva del intento y guardado del resultado verifican las fronteras de despacho. Prueba adicional de cancelación con commit fallido.
- Suite completa con host/race pasa (core 16,952 s), vet y diff-check pasan. F2.7 permanece abierta por matriz restante y ciclo de vida instalado; F0/F1, F3 y R1 conservan sus pendientes.

### 2026-09-28 — Instalador y retirada en staging

- R1.1–R1.3 iniciadas: `scripts/install.py` instala binarios, UI y unidad en rutas del usuario/XDG; modo staging impide activación. CLI referenciada por ruta absoluta en la copia instalada.
- Recibo con hashes, rechazo de destinos ajenos/modificados y symlinks, copias anteriores al actualizar y reversión de archivos ante fallo de escritura. No modifica preferencias durante la copia.
- Retirada solo de archivos registrados y sin modificaciones; conserva estado, credenciales, copias y archivos ajenos. Validación de rutas impide utilizar un recibo para borrar fuera del paquete.
- Seis pruebas en directorios temporales pasan, incluyendo actualización y retirada. Documentación provisional `docs/installation.md`.
- No se marcan R1.1–R1.3 completas: faltan activación permanente, hooks, backup/restauración de SQLite y recuperación integral de actualización. Próximo paso: completar esos contratos y probar servicio/plugin instalados para cerrar F0 y F2.7.

### 2026-09-28 — Respaldo SQLite previo a actualizar

- R1.2: instalador obtiene el mismo flock que el motor durante la sustitución y snapshot; rechaza motor activo incluso fuera de systemd. Libera antes de activar servicio.
- Backup SQLite mediante API, incluyendo WAL confirmado, integrity_check, archivo independiente sin WAL requerido, permisos 0600 y metadatos con esquema/SHA256. No copia valores del almacén de credenciales.
- Ocho pruebas de instalador pasan (0,616 s), incluidas cola pendiente en WAL y exclusión por bloqueo real. Actualizada documentación provisional.
- Restauración guiada, actualización integral e integración permanente siguen pendientes; R1.2 permanece abierta.

### 2026-09-28 — Restauración segura y recorrido de instalación real en staging

- R1.2: `--restore-data` verifica origen privado, checksum, esquema/integridad y bloqueo del motor; prepara imagen independiente, copia primero el estado sustituido y restaura mediante API SQLite.
- Estado restaurado pausado, admisión rechazada, permisos revocados y pendientes/en curso inciertos. No se reactivan efectos antiguos ni se restauran valores de credenciales; auditoría registra la operación.
- Once pruebas del instalador pasan, incluidas restauración de configuración, conservación de completados, respaldo del estado nuevo, rechazo de backup alterado y motor activo.
- `scripts/test-install-runtime.py` pasa con binarios compilados: instala en HOME temporal, crea flujo y cola pausada, rechaza actualizar motor activo, respalda, cambia borrador, restaura, arranca y comprueba pausa/revocación/incertidumbre, finalmente retira binarios conservando SQLite.
- R1.2 sigue abierta por reversión integral de binarios y pruebas de actualización/versiones; la prueba en HOME temporal no cierra integración permanente F0/F1 ni ciclo de sesión F2.7. Próximo: integración del servicio/plugin y completar funcionalidades F3 pendientes.

### 2026-09-28 — Servicio y plugin instalados; requisito de idiomas

- Unidad temporal basada en plantilla empaquetada pasa en systemd real: socket, activación con sandbox comprobado, notificación/comando aislado, cola conservada tras SIGKILL y Restart=on-failure, stop/start y propiedades efectivas.
- Instalación permanente autorizada realizada: binarios en ~/.local/bin, plugin en ~/.config/omarchy/plugins/quatrro.automations, unidad habilitada y running. No hay flujos ni trabajos configurados en el perfil permanente; escucha predeterminada loopback.
- Se respalda shell.json antes de habilitar mediante Omarchy. Geometría del shell confirma widget visible en barra derecha (116×28). Panel nativo abre y su captura muestra conectado; detener motor muestra desconexión sin bloquear shell; arrancar reconecta.
- Recarga por rescan conserva PID del motor. Una actualización real crea backup de archivos/SQLite y devuelve servicio a running. Captura posterior aún muestra texto anterior de cabecera pese al archivo actualizado: investigar caché de componentes antes de cerrar F0.5/QA de recarga. F0.6 cerrada; F0.5 permanece abierta.
- Usuario añade inglés por defecto y selector español persistente. Backend preferences.get/set implementado con idioma en/es y default en; prueba de persistencia/rechazo de valores inválidos pasa. Traducción completa y selector QML son la siguiente prioridad; todavía no se anuncian implementados.

### 2026-09-28 — Inglés predeterminado y español persistente

- F1D.8 completada: catálogo inglés/español, singleton QML compartido, selector, preferencia de motor y actualización del widget por estado. Formularios, ayudas, estados, revisiones y botones de diálogos localizados. Identificadores/datos del usuario mantienen sus valores.
- Prueba QML cambia en→es→en con diálogo abierto, comprueba secciones/título/Guardar-Save y JSON de ejemplo válido en ambos idiomas. Recorrido completo conserva seis capacidades, notificación, HTTPS, timers, monitores, cola y diagnóstico. Capturas en/es revisadas; corregido ancho de formularios.
- Detectada caché real del panel anterior: rescan dejaba el método nuevo desconocido. Instalador ahora usa URLs QML por hash de revisión, conserva archivos anteriores registrados y cambia entrypoints del manifiesto. Panel instalado ya responde al método nuevo y devuelve revisión de permisos en ambos idiomas.
- Verificación nativa: cambiar a es guarda preferencia y revisión de capacidades en español; volver a en devuelve texto inglés y persiste. Instalación queda en inglés. No se reinició shell ni se desbloqueó sesión. Captura de la nueva ventana nativa no se produjo bajo bloqueo; renders actuales offscreen sí se revisaron.
- Suite host/race pasa (core 16,542 s), vet, manifiesto instalado y once pruebas de instalador pasan. Recorrido de binarios instalados verifica idioma conservado tras reinicio y restaurado con SQLite.
- Próxima prioridad: funciones avanzadas F3, cierre de QA pendiente F0/F1 y ciclo de sesión/recuperación F2.7; R1 sigue incompleta. El requisito de idiomas no sustituye ninguno de esos entregables.

### 2026-09-28 — Requisito visual del usuario

- Se añade R1.UI como fase final obligatoria antes de entrega: apariencia adaptada a los plugins reales del sistema, en particular reemplazar los botones genéricos actuales.
- La referencia serán componentes/patrones de Omarchy Quatrro instalado; no basta con recolorear controles Fusion. Se conserva el requisito inglés por defecto/español y la navegación por teclado.

### 2026-09-28 — Contrato inicial de scripts (F3.1)

- `scripts.prepare` copia archivo local regular UTF-8 acotado mediante descriptor sin symlinks; devuelve contenido y revisión SHA256 ligados a intérprete/contrato. No ejecuta ni concede permisos.
- Configuración y esquema JSON admiten hasta ocho revisiones; valida hash y hasta 16 parámetros tipados, cadenas acotadas/patrones/opciones, enteros exactos con límites y booleanos estrictos. Edición del original no altera la copia preparada.
- Pruebas dirigidas de contenido/tipos pasan; suite core con race pasa (9,774 s, pruebas host optativas no activadas). Contrato en docs/scripts.md.
- F3.1 permanece abierta: falta conectar revisión exacta a la acción y al sandbox, directorios autorizados, invalidación de capacidades y UI. Ningún script queda ejecutable por esta preparación.
- R1.UI agregado por petición del usuario: pasada final de estética nativa, posterior a las funciones y anterior a entrega.

### 2026-09-28 — Ejecución y simulación de scripts aprobados

- Acciones `script` ligadas a revisión exacta, valores literales o campos de evento con contrato tipado. Worker consume contenido de configuración fijada, sin reabrir la fuente original; capacidades cambian junto con la revisión.
- Bash/Python ejecutan contenido transmitido por stdin y montado en solo lectura por bubblewrap, bajo límites de systemd. Preflight comprueba también la entrada de código para cada intérprete requerido.
- Corregida expansión de argumentos por systemd con `--expand-environment=no`; se preservan literalmente metacaracteres tanto en scripts como en comandos fijos.
- Pruebas host verifican modificación posterior del archivo original, argumentos literales, código de solo lectura, revocación, sustitución de revisión y rechazo de parámetros ausentes/tipos/longitudes inválidos. Suite completa con race pasa (core 27,092 s).
- Simulación incorpora revisión y argumentos resueltos usando la misma validación que ejecución, sin lanzar código ni crear trabajos; prueba específica añadida.
- F3.1 sigue abierta: directorios autorizados, UI y validaciones restantes. No se han desplegado estos cambios sobre la instalación permanente. Próximo: completar esos contratos y continuar F3; pendientes previos F0/F1/F2/R1 se mantienen.

### 2026-09-28 — Formulario de preparación de scripts

- Sección Scripts añadida al panel; formulario de identificador, ruta, intérprete y parámetros con tipos, límites, patrón y opciones. Preparación mediante API, revisión del código/hash y guardado sin ruta original. Cambiar campos invalida la preparación.
- Inglés predeterminado y textos españoles incorporados al catálogo. La revisión aparece antes de los campos para que el contenido copiado quede visible; código solo lectura y texto plano.
- Recorrido QML con perfil temporal comprueba rechazo de guardar sin preparar, preparación real del archivo y conservación exacta de la revisión. Captura offscreen `.dev/ui-script-review.png` revisada; no equivale a validación estética nativa R1.UI.
- Corregida dependencia circular entre Field/ScriptParameters y acceso a filas durante desmontaje. Recorrido completo pasa sin errores QML: seis capacidades, entrega HTTPS, monitores, programación, historial, cola y diagnóstico. Once pruebas del instalador pasan con el nuevo componente incluido.
- F3.1/F3.6 permanecen abiertas: falta vincular parámetros/revisión a acciones desde el formulario, directorios autorizados y el resto de funciones avanzadas. No se actualizó la instalación permanente.

### 2026-09-28 — Acciones de script desde la interfaz

- Formulario de acción `script`: selección de recurso, revisión disponible/seleccionada, código de solo lectura, selección explícita de revisión y parámetros literales o vinculados al evento. Cadenas, enteros y booleanos conservan sus tipos.
- Cambiar recurso invalida revisión/valores. Guardar una revisión obsoleta se rechaza en el editor; la revisión de capacidades incluye identificador, hash y timeout. Texto nuevo en inglés/español.
- Recorrido QML crea revisión con tres parámetros, configura acción, la inserta entre notificación y salida HTTPS y activa siete capacidades. Script real comprueba los tres argumentos; solo después llega una entrega HTTPS y la ejecución finaliza.
- Captura detectó controles ocultos al cambiar tipo de acción; corregida dependencia reactiva y añadida comprobación de visibilidad/estado. Captura final `.dev/ui-script-action.png` revisada. Recorrido completo y once pruebas del instalador pasan; instalación permanente sin cambios.
- Próximo: directorios autorizados y validaciones restantes F1C.3/F3.1, después continuar extensiones y resto de F3. La UI de scripts no cierra F3.6, que incluye también adaptadores y administración opcional. R1.UI sigue pendiente.

### 2026-09-28 — Montajes por descriptor y auxiliar interno

- Paquete `internal/sandbox`: directorios con ruta/identidad dispositivo-inode, destino `/work/id`, acceso ro/rw y directorio de trabajo autorizado. Apertura sin symlinks y comprobación en el mismo descriptor pasado a Bubblewrap; máximo ocho montajes.
- Auxiliar privado `quatrrod --sandbox-runner` recibe JSON acotado/versionado, sin abrir perfil ni API; código por memfd sellado, montado en solo lectura. Mantiene entorno mínimo y namespaces; los límites de recursos los impone la unidad invocadora.
- Pruebas sustituyen la ruta después de abrirla: el sandbox sigue leyendo el directorio original; ro rechaza escritura y rw escribe solo allí. Ejecución siguiente rechaza identidad sustituida. Prueba del binario compilado pasa dentro de systemd con directorio temporal y parámetros literales.
- Suite host/race completa pasa (core 27,107 s; sandbox 1,067 s), compilación, vet y diff-check pasan. Contrato y límites en `docs/directory-access.md`, incluidos datos vivos y alcance de recursos dentro del árbol.
- Pendiente inmediato: conectar estos montajes a acciones/capacidades, API de preparación, preflight y UI; todavía no son configurables en el motor. F1C.3/F3.1 siguen abiertas y no se reinstaló el perfil permanente.

### 2026-09-28 — Directorios integrados en el motor

- API `directories.prepare`; acciones command/script admiten `directories` y `working_directory`, validados y ligados al hash de capacidad. Otros tipos no admiten esos recursos. Esquema JSON actualizado; simulación enumera montajes/cwd sin ejecutarlos.
- Worker transmite la configuración fijada al auxiliar dentro de la misma unidad limitada. Preflight comprueba identidad y montaje real; para scripts prueba intérprete/código por el auxiliar. Un recurso sustituido se rechaza antes de activar o al ejecutar.
- Pruebas host ejecutan comandos y scripts con escritura autorizada, revocación, cambio rw→ro y sustitución de ruta después de encolar. Revocación/cambio de capacidad deniegan pendientes; sustitución falla sin escribir en el reemplazo.
- Suite host/race completa pasa (core 43,507 s). Binarios compilados + API local pasan en perfil temporal: preparación, revisión/concesión, simulación sin efectos y script Python escribiendo en el directorio aprobado. Compilación, vet y diff-check pasan.
- Próximo: formulario de directorios y revisión gráfica de alcance/permisos, más validaciones restantes de cancelación/aislamiento F1C.3/F3.1. No se actualizó instalación permanente; F3/R1 y otros pendientes siguen abiertos.

### 2026-09-28 — Directorios en UI y cierre funcional de scripts

- Formulario compartido por command/script: origen, destino, ro/rw (ro predeterminado), preparación por API, identidad visible, retirada y cwd entre destinos preparados. Conserva montajes al editar; ruta inválida no reemplaza entradas válidas. Preparaciones de un editor anterior se descartan por generación.
- Revisión de capacidades muestra ruta, identidad, acceso y cwd; explica alcance del árbol, modificación/borrado y sockets. Textos en inglés/español; guardar espera preparación pendiente y no concede permisos.
- Recorrido QML pasa: prepara directorio temporal, rechaza ruta inexistente conservando el válido, vuelve a editar y guarda el montaje, revisa siete capacidades y ejecuta notificación→script con tres parámetros→HTTPS. El script escribe resultado en cwd autorizado. Captura del formulario revisada offscreen; once pruebas de instalador pasan con 20 archivos.
- F3.1 cerrada funcionalmente con evidencia acumulada: copia de código/revisión inmutable, Bash/Python reales, argumentos tipados/literales, invalidación y revocación, descriptor contra sustitución de ruta, simulación sin efectos y recorrido gráfico autorizado. El despliegue y QA estética nativa permanecen en F0/F1D/R1; no se consideran cerrados por esta marca.
- Próximo: comprobaciones restantes del ejecutor F1C.3 y funciones F3.2–F3.7. F3.6 conserva adaptadores/administración pendientes. No se reinstaló el perfil permanente.

### 2026-09-28 — Terminación con directorios de escritura

- Nuevas pruebas host usan el auxiliar real dentro de systemd: Python crea hijo que ignora SIGTERM, llama setsid y escribe cada 20 ms en un directorio aprobado. Timeout y cancelación terminan todos los PID observados en el cgroup y detienen el crecimiento del archivo.
- Prueba de worker: cancelar durante escritura deja el trabajo `uncertain`; siguiente tick no lo repite ni produce más escrituras. Cambios ya realizados no se presentan como revertidos.
- Pruebas dirigidas con race pasan (7,369 s), vet y diff-check pasan. No se cambió el ejecutor ni se desplegó otra versión; se amplió evidencia de su comportamiento.
- F1C.3 conserva el pendiente concreto de perfiles/validación semántica ampliada de comandos. Directorios, límites y terminación ya tienen evidencia; no se cierra la tarea omitiendo ese requisito. F3.2–F3.7 y entrega permanecen abiertas.

### 2026-09-28 — Perfiles semánticos y cierre del ejecutor

- Catálogo inicial `file-exists` y `make-directory`: ejecutables/argv generados por el motor, ruta canónica a hijo directo de montaje aprobado, acceso rw obligatorio para crear y modo 0700. Rechaza opciones adicionales, cambio de ejecutable, traversals, rutas no aprobadas y plantillas de evento.
- Resolución compartida por validación, simulación, preflight y despacho. Perfil/ruta ligados a capacidad. `fixed` conserva comando avanzado explícitamente aprobado; no se presenta como semánticamente validado para todos los programas.
- Formularios y revisión de permisos incluyen perfil/ruta, con traducciones. Recorrido QML activa ocho capacidades, crea directorio 0700 mediante perfil, ejecuta script autorizado y entrega HTTPS. Pruebas reales verifican ambos perfiles, nombres con metacaracteres literales y fallo ante recurso existente/ausente según contrato.
- Pruebas dirigidas pasan (8,121 s); suite completa host/race pasa (core 56,996 s), incluyendo límites, montajes, revocación y cancelación previamente añadidos. Compilación, vet, manifiesto y diff-check pasan.
- F1C.3 cerrada con catálogo acotado documentado y evidencia acumulada de aislamiento/terminación. Próximo: F3.2 adaptadores externos. QA nativa, recuperación pendiente, otras funciones avanzadas y R1 conservan sus tareas; no se reinstaló versión permanente.

### 2026-09-28 — Contrato y transporte de adaptadores externos

- F3.2 iniciada con protocolo de transformación versión 1: manifiesto de capacidades event.read/data.write, runtime Python3, cuotas de entrada/salida y timeout. Respuesta ligada al ID de invocación, objeto de datos acotado; no admite operaciones de sistema/configuración en el sobre.
- Auxiliar aislado admite stdin de datos separado del código en memfd y captura de stdout limitada; exceso falla y la salida parcial no debe aplicarse. Cuota cero conserva salida descartada de comandos/scripts existentes. No se registra stderr.
- Pruebas de protocolo rechazan versiones, IDs, capacidades, sobres, tamaño y profundidad inválidos. Python aislado transforma JSON y falla al exceder stdout. Ejemplo status-normalizer y prueba reproducible del binario compilado dentro de systemd pasan.
- Suite host/race completa pasa: adapters 1,019 s, core 56,700 s, sandbox 1,149 s; compilación, vet y diff-check pasan. Contrato y límites en docs/external-adapters.md.
- F3.2 permanece abierta: falta registro local de revisiones, permisos/revocación, integración del worker y persistencia del resultado por ejecución, además de UI en F3.6. Próxima tarea es integrar esos contratos, sin aceptar código del evento. No se cambió instalación permanente.

### 2026-09-28 — Revisiones locales de adaptadores

- `adapters.prepare` valida manifiesto y copia código local con el mismo contrato de archivo seguro de scripts: regular, sin symlinks, UTF-8/NUL y tamaño acotados. Devuelve manifiesto/código/hash sin ruta de ejecución ni permisos.
- Configuración y esquema JSON admiten hasta ocho adaptadores, incluidos en 500 recursos. Revisión liga código, protocolo, capacidades, cuotas y timeout; se recomputa al validar. Borrador conserva exactamente la copia preparada.
- API probada: edición posterior del original conserva copia; hash anterior rechaza código/cuotas cambiados, symlink y protocolo incompatible rechazados, ID duplicado rechazado, preparación no crea grants ni ejecuciones.
- Prueba dirigida pasa (1,080 s); core/adapters con race pasan (core 10,222 s, sin pruebas host optativas). Un primer intento sin permiso de sockets falló al arrancar receptor HTTPS; repetido con permiso loopback, pasó. Vet y diff-check pasan.
- Próximo: referencias de acción, permisos/revocación, worker y persistencia por ejecución; UI después. F3.2 sigue abierta y no se reinstaló versión permanente.

### 2026-09-28 — Contexto privado y transaccional por ejecución

- Migración SQLite 5 añade contexto de evento por ejecución, vacío para datos previos. El worker lee la vista privada confirmada o el evento original. Resultado de adaptador se sitúa en data.adapter, preservando metadatos y otros datos; no altera inbox compartido.
- Contexto, registro del paso y avance se guardan en una transacción. Límite de 256 KiB por vista; cuota global incluye eventos/contextos y cuenta bytes UTF-8. Actualizar contexto sustituye su consumo anterior y retención lo elimina junto con ejecución.
- Pruebas confirman separación entre dos flujos, persistencia tras reabrir, rollback de contexto/step ante fallo SQL, cuota y migración 4→5 preservando pendientes. Fixtures históricos actualizados para no incluir la columna futura.
- Suite completa host/race pasa (core 58,102 s); once pruebas de instalador y recorrido de binarios instalados/backup/restauración temporal pasan con esquema 5. Compilación, vet y diff-check pasan.
- F3.2 sigue abierta: el guardado está preparado pero falta conectarlo a referencia/permiso y despacho real del adaptador. UI después. No se migró ni reinstaló el perfil permanente.

### 2026-09-28 — Adaptadores conectados al flujo y cierre del motor F3.2

- Acción adapter referencia ID/revisión exacta; manifiesto determina runtime/cuotas/timeout y no admite ejecutable alternativo ni montajes. Hash de capacidad incluye referencia. Worker despacha copia fijada, valida respuesta y confirma contexto/avance atómicos antes de continuar.
- Preflight realiza intercambio inocuo con código propio y cuotas del manifiesto. Captura de salida acotada tanto en auxiliar como en padre; el buffer del padre no expone ReaderFrom que pueda eludir Write. Exceso de contexto termina failed sin avance; fallo SQL conserva política de incertidumbre.
- Simulación no ejecuta adaptadores ni inventa resultados: muestra referencia y difiere vistas de pasos posteriores. No toma data.adapter entrante como una salida calculada.
- Pruebas host confirman resultado tras reapertura, revocación/revisión nueva denegadas, respuesta inválida, cuotas de entrada/salida/contexto y timeout sin avance. Flujo con binarios reales normaliza estado y pasa resultado a script que escribe en directorio aprobado.
- Suite completa host/race pasa (core 88,697 s); compilación, vet y diff-check pasan. F3.2 cerrada en motor con contrato inicial de transformación Python3. Próximo: UI de adaptadores en F3.6, luego continuar OAuth2/LAN/broker/compatibilidad y QA/R1 pendientes. No se actualizó instalación permanente.

## UI de adaptadores — validación del recorrido

- Preparación local, revisión de código/cuotas y selección explícita de revisión disponibles en el panel, con textos inglés/español.
- `python scripts/test-ui-flow.py` pasa: nueve capacidades, flujo con transformación y entrega HTTPS del resultado. Guardar una acción sin seleccionar la revisión se rechaza.
- Capturas de revisión y acción inspeccionadas en render de desarrollo; conservan controles genéricos. R1.UI continúa pendiente y obligatoria, antes de R1.8. Instalación permanente sin actualizar.

## F3.3 — Primer tramo del conector OAuth2

- Proveedor inicial: Google, contrato oficial enlazado en `docs/oauth2.md`.
- Renovación implementada en `internal/core/oauth_google.go`: endpoint fijo,
  transporte público existente, cuotas, expiración, conservación/rotación de
  refresh token y errores redactados. No integrado todavía con destinos ni UI.
- `go test -race ./internal/core -run TestGoogleOAuth -count=1` pasa (1,027 s).
  Transporte simulado comprueba contrato y fallos sin cuentas reales.
- Próximo: almacenamiento/estado de conexión, renovación serializada y enlace
  a destinos autorizados; luego interfaz y pruebas completas de F3.3.

## F3.3 — Persistencia y concurrencia de renovación

- Helper del motor guarda tokens/expiración juntos en el backend de secretos
  seleccionado, reutiliza tokens vigentes y renueva con margen de 60 segundos.
- Renovaciones serializadas; eliminación/sustitución durante la red no se bloquea
  ni se sobrescribe al recibir la respuesta. Sin caché de credenciales separada.
- Pruebas dirigidas con race pasan (1,427 s): doce consumidores/una renovación,
  refresh token rotado persistente, reapertura, expiración próxima y carreras de
  sustitución/eliminación. No usa cuentas reales ni prueba keyring del usuario.
- F3.3 sigue abierta: faltan API de conexión/estado, destinos, permisos y UI.

## F3.3 — API de conexión y estados

- `oauth.google.put` guarda una autorización existente sin permisos ni llamadas
  externas; `oauth.google.status` consulta estado/expiración sin exponer secretos.
- Fallos de renovación persistentes y redactados; reconexión/cliente rechazado
  detienen nuevos intentos hasta reemplazar la conexión. Borrado local usa secrets.delete.
- Pruebas cubren dispatcher, ausencia de grants, redacción, estados disponibles,
  expirados y fallidos, reapertura y restablecimiento al sustituir credenciales.
- Pendiente: enlace de destinos y permisos, emisor HTTP, consentimiento inicial,
  UI inglés/español y cierre funcional F3.3. Instalación permanente sin cambios.

## F3.3 — Destinos y autenticación saliente

- Modo `oauth2-google` en configuración/esquema; referencia a conexión vinculada
  al hash del permiso existente. Autenticación usa token privado del almacén.
- Hosts Google iniciales explícitos, HTTPS/443 y sin excepciones LAN; validación
  tanto al guardar como al emitir. Rechazo de uso genérico del conjunto OAuth2.
- Reintentos solo para errores temporales/cambio concurrente; errores terminales
  redactados sin efectuar el envío. Pruebas dirigidas iniciales pasan (1,924 s).
- F3.3 abierta: rechazo remoto/estado, recorrido worker, consentimiento y UI.

## F3.3 — Rechazo remoto y recorrido de cola

- HTTP 401 termina la entrega sin repetirla. Se marca token_rejected y se elimina
  el access token solo si coincide la revisión privada usada al despachar.
  Reemplazos/renovaciones concurrentes quedan intactos; se conserva refresh token
  para una solicitud posterior. Fallos al guardar estado se indican sin secretos.
- Pruebas OAuth2 dirigidas pasan (2,003 s): rechazo, reemplazo durante envío,
  renovación posterior y revisión obsoleta. Prueba adicional worker pasa (1,221 s):
  éxito completado y 401 fallido, cinco ticks y una sola petición en cada caso.
- Pendiente F3.3: consentimiento inicial y UI, revisión de permisos visible y
  verificación integrada restante. Resto del roadmap sigue abierto.

## F3.3 — Formulario y estado OAuth2 en el panel

- Formulario Google para importar autorización existente, secretos enmascarados,
  selección de backend y limpieza al guardar/cancelar. Estado consultable sin red.
- Editor de destinos ofrece oauth2-google; revisión visible muestra autenticación
  y referencia. Etiquetas y estados inglés/español, inglés predeterminado.
- Consentimiento por navegador y comprobaciones finales F3.3 permanecen abiertos;
  importación de refresh token no sustituye ese recorrido. R1.UI sigue pendiente.

## F3.3 — Protocolo de consentimiento de escritorio

- Contrato oficial Google revisado: authorization code, PKCE S256, navegador
  externo y callback loopback para cliente Desktop.
- Sesión privada de diez minutos, state/verifier aleatorios, scopes explícitos,
  callback validado y canje de un solo uso. Requiere refresh token inicial.
- Suite OAuth2 dirigida pasa (2,218 s), sin cuentas externas. Listener, API de
  sesión, persistencia/scopes y apertura desde UI siguen pendientes.

## F3.3 — Receptor loopback de consentimiento

- Listener efímero limitado a 127.0.0.1, vinculado a sesión PKCE, con límites de
  cabeceras/tiempos y respuesta estática sin reflejar parámetros sensibles.
- Retorno/denegación válidos terminan la sesión; solicitudes inválidas no la
  consumen. Cancelación/caducidad cierran puerto y conexiones pendientes.
- Pruebas reales de loopback pasan inicialmente (1,031 s), sin Google/navegador.
- Pendiente: API de inicio/estado/cancelación, conexión con canje/almacenamiento,
  scopes concedidos y UI del consentimiento. F3.3 continúa abierta.

## F3.3 — Sesiones de consentimiento en el motor

- API begin/session/cancel conectada a listener, canje y almacenamiento; una
  autorización en curso, plazo de diez minutos y cierre ligado al motor.
- Guardado condicionado a credencial sin cambios; cancelación serializada con
  persistencia final y sin concesión de permisos a flujos. Estados redactados.
- Pruebas dirigidas pasan (1,229 s): éxito, conflicto durante canje, cancelación,
  cierre del motor y exclusión de segunda sesión. Sin proveedor real.
- Pendiente: scopes concedidos, UI de inicio/navegador/seguimiento y QA F3.3.

## F3.3 — Scopes concedidos

- Canje exige todos los scopes solicitados antes de guardar; consentimiento
  parcial termina scopes_missing. Listas solicitada/concedida persisten como
  metadatos de la conexión y aparecen en status sin tokens.
- Reducción explícita durante renovación bloquea uso/reintentos hasta reconectar;
  ausencia del campo conserva metadatos previos. Importaciones no fingen scopes.
- Pruebas dirigidas pasan (1,107 s): consentimientos parcial/ausente/completo,
  scopes adicionales, sintaxis/cuotas y reducción durante renovación.
- Pendiente inmediato: UI de inicio, apertura en navegador y seguimiento de sesión.

## F3.3 — Inicio y seguimiento de consentimiento desde QML

- Nuevo modo autorizar en navegador en formulario: cliente Desktop, scopes,
  backend e ID; inicio local y botón explícito para navegador externo.
- Seguimiento de sesión, cancelación, estados inglés/español y eliminación de
  URL temporal al terminar. Éxito actualiza lista de credenciales.
- `scripts/test-ui-flow.py` pasa: inicia/cancela desde formulario, URL sin client
  secret, estado cancelled, ninguna credencial creada, flujo previo completado
  con nueve capacidades y una entrega HTTPS. Captura de formulario inspeccionada.
- Resta revisión de cierre F3.3 (incluida recuperación de sesión/UI y documentación
  consolidada). No hay cliente/cuenta Google real configurados; esa validación no
  se sustituye por el transporte simulado. Instalación permanente sin actualizar.

## F3.3 — Recuperación de sesión y guía consolidada

- `oauth.google.current` recupera sesión desde una UI recargada; URL disponible
  solo durante espera, eliminada al canjear/cancelar/finalizar. Reinicio del motor
  devuelve none y rechaza identificadores previos.
- Panel consulta current al actualizar Seguridad. Fallo de consulta detiene sondeo
  y permite recuperar estado, sin insistir indefinidamente sobre sesión perdida.
- Prueba dirigida con race pasa (1,099 s); binarios recompilados y recorrido QML
  completo pasa, incluyendo pérdida/recuperación del estado local y cancelación.
- Guía OAuth2 consolidada. F3.3 conserva pendientes explícitos de navegador/cuenta
  real y verificación keyring del recorrido; no se declaran cubiertos por mocks.

## F3.4 — Excepciones LAN por autoridad exacta

- Validación exige una única excepción host:puerto que coincida con URL del destino,
  puerto explícito/canónico (443 implícito), sin comodines/entradas ajenas/duplicadas.
- Transporte comprueba excepción también al resolver: solo unicast local admitido;
  multicast/unspecified y rangos de transición/documentación permanecen bloqueados.
- Pruebas TLS reales IPv4/IPv6 pasan (1,144 s): éxito con excepción, bloqueo sin ella,
  con host ajeno o reutilizada por otro destino. Hash del permiso cambia con excepción.
- Suite core/race pasa (13,188 s). Formulario aclara sintaxis y revisión muestra LAN.
- F3.4 pendiente de comprobar recorrido UI actualizado y revocación desde worker;
  guía `docs/lan.md` documenta límites, TLS y compatibilidad de configuraciones previas.


## F3.4 — Cierre con revocación y panel

- `TestLAN*` con race pasa (1,264 s), incluyendo revocación antes del primer envío
  y antes de reintentar un 503. TLS local real; varios ticks dejan la ejecución
  denied sin nueva petición al receptor.
- Binarios recompilados. Recorrido QML completo pasa y exige que el texto de
  revisión muestre Private network exceptions y la autoridad exacta autorizada.
  El flujo configura excepción desde formulario y completa una entrega HTTPS.
- Evidencia combinada: validación de host/puerto, hash del permiso, transporte TLS
  IPv4/IPv6, DNS/bloqueos conservados, revocación y revisión UI. F3.4 completada.
  No se modificó la instalación permanente ni se afirma acceso a dispositivos LAN
  reales o soporte verificado de IPv6 link-local con zona.
- Próxima fase independiente: F3.5 broker administrativo opcional, con pruebas
  privilegiadas primero en entorno aislado. Pendientes F3.3/F3.6/F3.7 y R1 siguen.

## F3.5 — Contrato inicial del broker administrativo

- Nuevo paquete internal/broker: política acotada por UID/unidad/operación exactos,
  protocolo separado, rechazo de campos desconocidos y ausencia de shell/argv libre.
- Plan fija systemctl y comprobación Polkit con PID/inicio/UID y detalle unit,
  sin interacción. Todavía no ejecuta ni concede privilegios.
- Pruebas race pasan (1,018 s), vet y diff-check pasan: identidad incompleta/ajena,
  recursos/operaciones no permitidos, inputs de privilegios y políticas inválidas.
- F3.5 sigue abierta: carga segura, peer real, servicio, Polkit/ejecutor, instalación,
  UI y pruebas privilegiadas aisladas. Guía en docs/admin-broker.md.

## F3.5 — Lectura segura de política

- Loader abre ruta canónica desde raíz mediante descriptores y O_NOFOLLOW;
  exige root-owner y ausencia de escritura grupo/otros en todos los niveles,
  archivo regular y máximo 64 KiB. No cachea reglas ni expone contenido en errores.
- Pruebas del paquete con race pasan (1,019 s): propietario/permisos, symlinks,
  FIFO, exceso de tamaño, traversal y reemplazo atómico observado al releer.
- Pruebas usan helper privado con UID de prueba; producción fija UID 0. Vet y
  diff-check pasan. No se escribieron políticas de sistema ni se usó root.
- Próximo: identidad del peer y servicio con comprobación Polkit/ejecución acotada;
  validación privilegiada aislada e instalación/UI continúan pendientes.

## F3.5 — Identidad de socket y proceso

- Peer obtenido de SO_PEERCRED + SO_PEERPIDFD, con referencia estable del kernel,
  lectura descriptor-relative de proc y comparación de UID/inicio; sin fallback
  inseguro a PID no fijado. API Revalidate/Close para el futuro ejecutor.
- Pruebas race del paquete pasan (1,039 s): proceso actual, cliente hijo real,
  terminación del cliente, referencia cerrada y metadatos inválidos/UID distinto.
  Parser tolera comm con espacios, saltos y paréntesis. Vet/diff-check pasan.
- Sin root ni cambios de sistema. Próximo: transportar solicitudes acotadas y
  ejecutar comprobación Polkit antes de operaciones; F3.5 sigue abierta.

## F3.5 — Ejecutor con doble comprobación de política

- Execute enlaza identidad kernel, política fija, Polkit no interactivo, nueva
  lectura de política y revalidación antes de systemctl. No admite argumentos,
  timeout, entorno ni rutas suministrados en la solicitud.
- Plazos de 5/15 segundos, grupo separado, salida de 4 KiB, stderr descartado y
  status con propiedades permitidas. Fallo de mutación queda incierto, sin retry.
- Pruebas race pasan (1,085 s): secuencia simulada, denegación, revocación, peer
  terminado, redacción, parser status; timeout/cuota con procesos inocuos reales.
  Vet y diff-check pasan.
- F3.5 abierta: daemon/transporte, Polkit/políticas reales y aislamiento privilegiado,
  instalación opcional e integración motor/UI. Ningún servicio de sistema operado.

## F3.5 — Transporte y binario del broker

- Servidor Unix acotado (8 conexiones, JSON 4 KiB, half-close, deadlines), identidad
  kernel antes de ejecutar y errores estáticos. Cancelación cierra listener/conexiones.
- Binario quatrro-broker compilado: requiere UID root y un socket de activación
  systemd en ruta fija, comprueba política y enlaza al ejecutor. Build lo produce,
  pero instalador de usuario no lo instala/inicia.
- Pruebas broker/race pasan (1,082 s); concatenación, tamaño y campos de identidad
  no llegan al ejecutor simulado. Solicitud válida/redacción/cierre verificados.
  Binario rechaza arranque del usuario; version, compilación, vet y diff-check pasan.
- Pendiente: unidades/políticas reales, cliente e integración, instalación opcional
  y pruebas de efecto privilegiado en entorno aislado. No se activó ningún privilegio.

## F3.5 — Unidades y reglas de instalación preparadas

- Artefactos socket/service endurecidos, acciones Polkit con defaults no y política
  vacía; sin instalación ni activación. Generador de reglas liga usuario/unidad/acción
  exactos y no interviene en namespaces ajenos. UID sigue comprobado por broker.
- Prueba JS en Node pasa (race 1,363 s): autorizaciones exactas, denegación de sujeto,
  unidad/operación ajenos o detalle ausente; namespace externo queda intacto.
- XML valida contra DTD local y systemd-analyze verify pasa con copia temporal y
  ejecutable local. Primera ejecución restringida falló por sockets internos de
  verificación; repetida con permiso local pasa. Ninguna unidad iniciada.
- Próximo: validar privilegios en entorno aislado e integrar cliente/instalación/UI.
  F3.5 sigue abierta; archivos escritos no equivalen a autorización real comprobada.

## F3.5 — Cliente del broker

- Call fija socket del sistema y exige UID root vía SO_PEERCRED antes de enviar;
  valida solicitudes/respuestas y acota conexión, plazo, tamaño y cancelación.
- Respuestas deben coincidir con versión/ID/unidad. Fallos posteriores al envío
  quedan inciertos, sin reintento automático; UID ajeno no recibe bytes.
- Suite broker/race pasa (1,156 s), vet y diff-check pasan. Tests con servidores
  Unix temporales del usuario y helper privado de UID, sin privilegios reales.
- Pendiente: integración acción/worker/UI, instalador opcional y prueba completa
  de systemd/Polkit/efecto root aislado. F3.5 permanece abierta.

## F3.5 — Acción administrativa integrada en motor

- Tipo system-service en modelo/esquema: unidad y operación fijas, sin argv/shell,
  timeout ni scripts/directorios. Capacidad del flujo ligada a acción completa.
- Worker usa cliente del broker; denied/failed/uncertain se conservan y no generan
  reintento automático. Revocación previa impide llamar al broker. Simulación no
  ejecuta y muestra recurso/operación/requisito administrativo.
- Pruebas dirigidas con race pasan (1,540 s): validación, hash, simulación, éxito,
  denegación, incertidumbre, ausencia del broker y revocación; cuatro ticks sin repetir.
- Pendiente: UI/preflight opcional, instalador y sistema/Polkit/efecto root aislado.
  Integración no modifica permisos ni instala servicios; F3.5 continúa abierta.

## F3.6 — Formulario administrativo y revisión

- Selector system-service, unidad/operación exactas y aviso de broker opcional,
  política administrativa y ausencia de instalación/concesión implícita.
- Revisión muestra unidad/operación y requisitos de flujo/root/Polkit. Textos en
  inglés/español. Captura de desarrollo inspeccionada, sin cerrar R1.UI.
- Build y recorrido QML pasan con diez capacidades (acción administrativa revisada
  en flujo sin eventos) y una entrega HTTPS del flujo existente. No llama al broker.
- Importación administrativa con race pasa (1,096 s): flujo desactivado, cero grants
  y ejecuciones. F3.6 conserva estado/disponibilidad administrativa y QA pendientes.

## F3.5 — Primeras autorizaciones con Polkit real

- Prueba opt-in TestIsolatedPolkit: polkitd, bus y cuentas subordinadas aislados,
  filesystem host de solo lectura y políticas efímeras. Pasa (0,239 s).
- Autoriza status exacto, rechaza unidad/operación ajenas y observa revocación al
  sustituir reglas. Usa acciones empaquetadas y renderizador de producción.
- Próximo: sujetos ajenos, broker/transporte y efecto sobre systemd real aislado.
  Instalador opcional, disponibilidad UI y cierre F3.5 siguen pendientes.

## F3.5 — Identidades y loader de producción aislados

- TestIsolatedPolkit ahora usa raíz efímera con /usr de solo lectura y archivos
  concretos de cuentas; /etc y la política del broker pertenecen al root aislado.
- Polkit real rechaza un segundo UID (102). LoadPolicy de producción acepta la
  política root, rechaza propietario ajeno/escritura de grupo y observa revocación
  por reemplazo atómico. Prueba dirigida pasa (0,295 s).
- Siguiente tarea F3.5: servicio systemd y recorrido completo dentro del aislamiento;
  no se operaron unidades administrativas del host ni se cierra la fase.

## F3.5 — Base systemd aislada disponible

- Nuevo scripts/test-broker-systemd.py pasa: PID 1 real, D-Bus privado por socket,
  inicio/consulta/parada de fixture y cierre del contenedor. Scope delegado temporal
  se retira al finalizar; ningún servicio administrativo del host se modifica.
- Próximo: enlazar binario/unidades del broker y Polkit en este entorno para
  verificar operaciones autorizadas, denegaciones, revocación y límites reales.

## F3.5 — Recorrido administrativo real comprobado

- Cliente de producción → socket systemd → broker empaquetado → Polkit → systemctl
  funciona en raíz efímera con UID subordinados. status/start/restart/stop pasan.
- Usuario/unidad ajenos, revocación Polkit y allowlist denegados; restauración
  Polkit comprobada. Límites kernel 128 MiB/25 % CPU, CapEff=0 y NoNewPrivs=1.
- Script integral y suite broker/race pasan; evidencia y límites documentados.
- Próximo: instalación administrativa opcional con actualización/desinstalación
  verificadas en aislamiento, disponibilidad UI y recuperación. F3.5 permanece abierta.

## F3.5 — Preparación del instalador administrativo

- --prepare-policy valida política y genera reglas sin privilegios, con parser y
  renderizador de producción. Claves JSON duplicadas, también aliases por case o
  escapes, rechazadas antes de producir artefactos.
- prepare-broker-install.py genera paquete nuevo con seis archivos, destinos,
  modos, tamaños y hashes; cero reglas por defecto y activation=disabled. No instala.
- Prueba de preparación pasa: hashes/modos, denegación predeterminada, rechazo de
  sobrescritura y política inválida sin crear paquete. Suite broker/CLI con race
  pasa, incluido Polkit aislado. Vista local en .dev/broker-install-preview.
- Próximo: aplicador administrativo con recibo/rollback y ciclo de vida probado en
  aislamiento. F3.5 sigue abierta; preparación no equivale a instalación funcional.

## F3.5 — Aplicador administrativo y ciclo de vida inicial

- install-broker.py apply/remove/recover: destinos fijos, hashes/modos, traversal
  por descriptores root/no-symlink, recibo, lock, journal y restauración ante fallos.
- Actualización conserva política/reglas salvo --replace-policy. Cada operación
  deja el socket sin activar; rechaza archivos ajenos o cambiados respecto al recibo.
- Prueba systemd aislada pasa instalación, llamada real, actualización conservando
  grants, reemplazo explícito por política vacía y retirada sin borrar archivo ajeno.
- Prueba filesystem pasa rollback y recuperación pendiente con fallos simulados.
- Próximo: interrupción real/recuperación administrativa, preflight y disponibilidad
  UI; revisión F3.5/F3.6. No se instala ni se habilita el broker en el host.

## F3.5 — Cierre con recuperación por SIGKILL

- Broker/helper recompilados desde fuentes actuales; recorrido systemd/Polkit real
  pasa operaciones, denegación/revocación, límites y ciclo de instalación completo.
- Seis cortes SIGKILL de actualización/retirada y dos de primera instalación:
  restauración exacta de bytes/modos/propietarios o ausencia original, bloqueo de
  apply mientras hay journal y exclusión de instalador concurrente. Socket queda
  deshabilitado; el cliente real vuelve a funcionar tras recuperar instalación previa.
- Perfil de prueba detiene al volver de operaciones duraderas; no se añadieron
  flags de fallo al instalador de producción. Consola ahora se drena durante la
  ejecución, con cuota y plazo; evita bloquear al aumentar salida de diagnósticos.
- F3.5 cerrada con evidencia en docs/validation.md y guía administrativa consolidada.
  No se instaló el broker en el host; no quedaron scopes temporales.
- Siguiente: disponibilidad y preflight en F3.6. Compatibilidad F3.7 y gates R1,
  incluida estética nativa final, permanecen abiertos. No simula corte eléctrico.

## F3.6 — Consulta administrativa sin efectos

- check_only en protocolo broker: identidad + política + Polkit + revalidación,
  devuelve authorized sin invocar systemctl. No sustituye la autorización al ejecutar.
- API administration.check con plazo ocho segundos, estados authorized/denied/
  unavailable/unsupported y effects_executed=false. No concede grants ni crea jobs.
- Prueba real aislada pasa: start no inicia unidad; restart/stop conservan PID y
  estado; unidad ajena y Polkit revocado denegados. Cliente no confunde autorización
  con ejecución. Brokers anteriores rechazan el campo sin fallback con efectos.
- Pendiente inmediato: botón/estado en editor y comprobación de activación; F3.6 abierta.

## F3.6 — Comprobación visible y requisito al activar

- Editor incorpora botón y estados administrativos en inglés/español; resultado
  ligado a revisión local, invalidado al editar/cerrar, respuestas obsoletas descartadas.
- Activación exige autorización fresca de acciones de flujos habilitados, sin efectos,
  plazo compartido ocho segundos y deduplicación unidad/operación. Fallo no guarda grants.
- Recorrido QML pasa ausencia, traducción, obsolescencia y activación rechazada;
  deshabilitar flujo administrativo permite activar nueve capacidades y entregar HTTPS.
- Próximo: QA positiva UI con broker real y cierre F3.6; F3.7 y demás gates siguen abiertos.

## F3.6 — Cierre de integración administrativa positiva

- test-broker-systemd.py --ui añade QML + motor como UID 1000 con entorno/perfil
  efímeros al systemd/Polkit/broker reales. No usa una respuesta administrativa simulada.
- Autorización visible, denegación de unidad ajena, invalidación al editar, textos
  inglés/español y capturas verificadas. Revisión muestra una capacidad; activar
  no cambia servicio, evento posterior inicia fixture y ejecución queda completed.
- Capturas broker-admin-en/es inspeccionadas. Test exige reporte y ambos PNG; además
  mantiene pruebas de instalación, recursos y ocho cortes SIGKILL del instalador.
- F3.6 cerrada sumando evidencia previa scripts/adaptadores, importación desactivada,
  revisión de capacidades y estados administrativos. Estética nativa sigue en R1.UI.
- Próximo F3.7: compatibilidad entre componentes y recuperación ante incompatibilidad.

## F3.7 — Compatibilidad del transporte local

- Protocolo local 2/contrato 1: saludo obligatorio en la misma conexión antes de
  enviar operación/payload, doble comprobación y sin fallback. Clientes antiguos
  y contratos distintos no alcanzan el handler. Cliente verifica también UID servidor.
- Cancelación interrumpe saludo, plazo máximo conservado y lector persistente por
  conexión. Error de incompatibilidad incluye pasos de actualización/recarga EN/ES.
- Pruebas de mezclas versión/contrato, payload no enviado, operación sin saludo y
  cancelación pasan; suite Go completa con race y Polkit aislado pasa.
- Pendiente: contrato independiente UI/CLI, metadatos y comprobación de instalación,
  compatibilidad de adaptadores y recuperación visible. F3.7 sigue abierta.

## F3.7 — Contrato UI/CLI y metadatos de componentes

- Backend QML usa quatrroctl ui --stdin con contrato 1, operación y data. CLI
  valida antes de rutas/conexión; rechaza desconocidos, duplicados y contrato ajeno.
- Nueva UI no hace fallback al envío directo si CLI vieja rechaza ui; mensajes de
  incompatibilidad/operación ui desconocida orientan recuperación EN/ES.
- quatrrod/quatrroctl --version producen metadatos compartidos sin iniciar nada.
- Test binario verifica contrato inválido sin conexión y válido con saludo/payload;
  recorrido QML completo pasa (nueve capacidades, una entrega HTTPS).
- Pendiente inmediato: usar metadatos para verificar artefactos/actualización y
  completar recuperación visible/matriz de adaptadores. F3.7 continúa abierta.

## F3.7 — Verificación de artefactos antes de instalar

- Compatibility.js define requisitos usados por Backend y por el instalador.
  Verifica snapshots exactos de motor/CLI con --version: componente, protocolo,
  contrato API/UI y release iguales. Instala esos bytes, sin releer el ejecutable.
- Recibo/resultado incluyen metadatos. Incompatibilidad rechazada antes de sustituir
  archivos, respaldos de actualización o recibo; conserva validaciones previas.
- Trece tests de instalación pasan con versión mezclada, ejes de contrato y
  actualización rechazada sin cambios. Recorrido QML completo pasa tras centralizar contrato.
- Pendiente F3.7: QA gráfica de incompatibilidad/recuperación y auditoría de cierre;
  matriz de contratos y guía de recuperación añadidas.

## F3.7 — Cierre con rechazo y recuperación visible

- Nuevo test-ui-compatibility.py ejecuta QML, CLI y motor reales en perfil temporal.
  Solo la copia temporal del harness permite cambiar el contrato: rechaza el 999
  en inglés/español, conserva borrador y no modifica configuración persistida.
  Restaurar contrato 1 permite guardar sin perder borrador ni preferencia de idioma.
- Capturas EN/ES inspeccionadas: error completo con instrucciones de actualización,
  reinicio y recarga. Render offscreen; no demuestra todavía estética nativa R1.UI.
- Trece tests de instalación y test-cli-contract.py pasan. Contratos local/adaptador
  y CLI pasan con race. Test host del adaptador ahora devuelve ID correcto y versión
  incompatible, aislando ese motivo: proceso real rechazado sin avanzar el flujo.
- F3.7 cerrada con matriz y recuperación documentadas. No se cambió la instalación
  permanente. Próximo: gestión de fallos F1B.6 desde UI y gates restantes.

## F1B.6 — Cierre de recuperación manual desde interfaz

- Recorrido QML completo ampliado: receptor HTTPS local devuelve 400, historial
  muestra fallo y detalles registran estado/intento sin cuerpo ni credenciales.
- Confirmación EN/ES: cancelar conserva fallo y no envía; confirmar tras recuperar
  receptor completa la misma ejecución, con cuerpo y clave de idempotencia iguales.
- Funciones compartidas por botones y harness abren detalle/confirmación; no se
  sustituye el reenvío por una llamada CLI en la prueba. Captura de contenido del
  diálogo corregida; advertencias EN/ES inspeccionadas (no cierre estético R1.UI).
- test-ui-flow.py pasa con diez capacidades y tres peticiones HTTPS (éxito previo,
  fallo y reenvío); conserva recorrido de scripts/adaptadores, cola y diagnóstico.
- Guía de operación explica inspección, confirmación y contador reiniciado.
  F1B.6 cerrada. Próximo: F1D.3 simulación/prueba real y demás gates abiertos.

## F1D.3 — Explicación de simulación y confirmación real

- Motor añade evaluaciones por flujo (habilitado, origen, condiciones, coincidencia)
  manteniendo matches. Condiciones usan la misma función que el despacho real;
  campos ausentes siguen sin coincidir, incluso con ne.
- Resultado UI EN/ES explica exclusiones y parámetros previstos; mantiene explícita
  la dependencia de resultados de adaptadores, que no se ejecutan para simular.
- Prueba de motor verifica cinco flujos, credencial referenciada inexistente y cero
  ejecuciones/outbox/grants. Suite core/race pasa (15,397 s; sin activar tests host).
- Recorrido QML pasa simulación/confirmación/cancelación EN/ES con contadores y
  receptor intactos. Aceptar prueba real produce HTTP 400 y posterior reenvío 204.
  Capturas de simulación inspeccionadas; R1.UI sigue pendiente.
- F1D.3 sigue abierta hasta comprobar en UI borrador distinto de revisión activa
  y condiciones no coincidentes. Guía operativa actualizada; instalación sin cambios.

## F1D.3 — Cierre con revisión activa y condiciones desde UI

- test-ui-flow.py incorpora condición data.adapter.severity eq error en flujo
  activo. Simulación con info explica el rechazo EN/ES, devuelve cero matches y
  conserva contadores y receptor; capturas inspeccionadas.
- Edición del cuerpo HTTP desde formulario crea borrador distinto. Simular guarda
  y muestra ese cuerpo sin activarlo; confirmar prueba real envía el cuerpo de la
  revisión activa anterior. Reenvío posterior conserva ese mismo cuerpo y clave.
- Recorrido completo pasa con diez capacidades, tres peticiones HTTPS y reportes
  simulation_conditions/draft_active_separation verdaderos. Diff-check/Python pasan.
- F1D.3 cerrada. Próximo F1D.1: estado real de receptor; actualmente el panel solo
  muestra una ayuda estática, sin distinguir HTTP deshabilitado de escucha activa.
  Otros gates y estética nativa R1.UI siguen abiertos; instalación sin cambios.

## F1D.1 — Estado real de recepción

- status incorpora ingress con ciclo disabled/starting/listening/stopped/failed;
  dirección obtenida del socket realmente abierto, retirada al terminar. No se
  deduce exposición externa: public_exposure siempre unverified.
- Panel Entrada muestra dirección y limitación del proxy EN/ES; sin conexión al
  motor deja de presentar la escucha como comprobada. No modifica firewall/DNS.
- Test de ciclo real loopback con race pasa (1,092 s): puerto efímero, respuesta
  HTTP, estado API, cierre, rechazo de bind no loopback. Build completo pasa.
- test-ui-flow.py pasa dirección real/advertencia EN/ES y recorrido previo;
  test-ui-compatibility.py pasa HTTP deshabilitado EN/ES y desconexión tras cerrar
  motor. Perfiles temporales, sin modificar instalación permanente.
- F1D.1 sigue abierta: completar recorrido de gestión de credenciales/conexiones.
  Documentados estados y límite de exposición en guía y protocolo.

## F1D.1 — Credenciales inbound: rotación, eliminación y recuperación

- Prueba integral ampliada con credencial de archivo efímera: rotación desde
  formulario, firma anterior 401 sin crear ejecución, nueva firma 202; cancelar
  eliminación EN/ES conserva referencia, aceptar la elimina y devuelve 503 sin
  crear ejecución. Restaurar desde formulario recupera 202.
- Detectado y corregido conflicto QML: secrets.delete usaba secretID ambiguo
  frente al TextField homónimo. Ahora usa deleteCredential.secretID explícito.
  Primera ejecución falló; segunda pasa con eliminación real y lista actualizada.
- Campos de entrada quedan vacíos, estado público/detalles no contienen valores.
  Efectos pausados, dos probes autenticados retenidos y cancelados; permanecen las
  tres peticiones HTTPS originales. No se usa keyring personal para esta prueba.
- Harness deja de intentar capturas cuando QUATRRO_CAPTURE está ausente; desaparece
  el aviso QUrl nulo en la ejecución nueva. Diff-check/compilación Python pasan.
- F1D.1 sigue abierta: siguiente verificación de credenciales outbound desde UI.

## F1D.1 — Cierre con credenciales outbound

- Destino bearer, acción y flujo creados desde formularios; credencial preparada,
  rotada, eliminada y restaurada desde diálogos. Receptor TLS local exige el token
  esperado: primera y segunda entrega autenticadas con valores distintos.
- Al eliminar credencial, ejecución queda pendiente con intento y próximo plazo,
  status 0 y ninguna petición al receptor. Restaurarla permite reintento automático
  y finaliza la misma ejecución; estado UI/detalles no muestran valores secretos.
- Primera prueba esperaba fallo terminal inmediato; corregida según política real
  de reintentos sin alterar producción. Segunda ejecución completa pasa: once
  capacidades y seis peticiones HTTPS, tres con autenticación exigida por fixture.
- F1D.1 cerrada con evidencia acumulada de formularios, receptor real/exposición,
  inbound/outbound y credenciales file explícitas. Validación externa Google/keyring
  permanece en F3.3; no se atribuye a estos fixtures. Próximo: revocación UI F1D.5.

## F1D.5 — Revocación desde Seguridad y detalle del bloqueo

- Botón y harness comparten revokePermission. Prueba gráfica retiene entrega con
  pausa, revoca permiso desde Seguridad, comprueba desaparición del grant y reanuda:
  mismo trabajo queda denied sin petición adicional al receptor.
- Motor ahora registra paso denied y razón estática en la misma transacción que
  termina ejecución; antes solo cambiaba estado global. Prueba dirigida verifica
  el detalle y ausencia de reintento del paso bloqueado.
- Core/race completo pasa (14,867 s), build y recorrido QML completo pasan con
  once capacidades, seis HTTPS y ui_revocation_blocks_pending=true. Captura de
  detalle inspeccionada: denied, razón explícita, sin entregas ni secretos.
- F1D.5 permanece abierta: pasos realizados aparecen con etiqueta interna pending
  porque la ejecución avanza al siguiente paso. Corregir representación/registro
  sin confundirla con entregas pendientes ni invalidar historiales anteriores.

## F1D.5 — Cierre con estados de paso correctos

- Al avanzar al siguiente paso, el registro del paso terminado guarda completed;
  la ejecución conserva pending hasta continuar. Lectura de registros antiguos
  normaliza solo steps.pending a completed, sin reescribir datos ni alterar outbox.
- Prueba de compatibilidad conserva failed/denied/uncertain/completed y entrega
  pendiente; verifica que lectura no muta registro antiguo. Prueba secuencial
  comprueba primer paso completed, segundo denied y ausencia de repetición.
- Suite core/race pasa (15,353 s); build de tres componentes y test-ui-flow.py pasan.
  QML comprueba paso completed tras entrega recuperada y mantiene pruebas de
  credenciales, revocación y detalles sin secretos. Once capacidades/seis HTTPS.
- F1D.5 cerrada con evidencia acumulada de pasos, reintentos, redacción, gestión
  de referencias y revocación. Próximo: widget/pausa/cancelación F1D.6; QA nativa,
  recuperación y R1.UI continúan pendientes. Instalación permanente sin cambios.

## F1D.6 — Controles del panel y motor independiente

- Botón Pausar/Reanudar y harness comparten togglePause. Recorrido real QML pausa
  con admisión retain, retiene entrada firmada y verifica estado en UI y motor.
- Diálogo Detener trabajos EN/ES: rechazar mantiene pendiente, aceptar elimina
  trabajo de cola sin envío HTTP. Panel cerrado conserva motor y pausa; reapertura
  permite reanudar desde mismo control. Guía distingue cancelar de pausar/admitir.
- test-ui-flow.py pasa completo con once capacidades y seis HTTPS; comprobaciones
  anteriores de simulación/credenciales/revocación se mantienen. Diff-check y
  compilación Python pasan; no se modifica instalación permanente.
- F1D.6 abierta para widget: estado/contadores, apertura y legibilidad. Integración
  con barra real y pasada estética R1.UI siguen siendo gates separados pendientes.

## F1D.6 — Widget: estados y ciclo de vida comprobados

- Widget ajusta ancho al texto, incluye denied en fallos, ofrece resumen accesible
  con pendientes/fallos y aviso de independencia del motor. Añadida apertura por
  teclado/acción accesible, foco visible y tooltip. Orientación vertical distingue
  desconexión/fallo, pausa y operación normal; aspecto nativo final sigue en R1.UI.
- test-widget.py usa motor real y shell-harness temporal: pausa/cola, grant revocado
  sin efecto, contador denied, EN/ES, orientación horizontal/vertical y contrato
  toggle por shell o bar.shell. Capturas de pausa EN/ES inspeccionadas, sin recortes.
- Terminar proceso QML deja motor disponible; recargar recupera estado; terminar
  motor produce desconexión sin presentar contadores viejos como vigentes.
- Manifiesto valida; QML sin errores y solo avisos esperados offscreen/máscara.
  Python compile/diff-check pasan. Teclado implementado, no aún probado con eventos.
- F1D.6/F0.5 continúan abiertos para integración real de barra. Consulta loginctl:
  agente sin sesión propia; sesión 2 del usuario Type=wayland, State=active,
  LockedHint=no. Revalidar shell antes de QA gráfica; no se desbloqueó ni reinició.

## F0.5 — Cierre con actualización instalada y panel real

- Consultado lock.status del shell: unlocked, sin solicitud/bloqueo activo. No se
  desbloqueó ni reinició escritorio. listPlugins confirma plugin habilitado;
  debugBarGeometry muestra widget visible en sección right (116×28, barra 26 px).
- Antes de actualizar: perfil sin recursos, trabajos ni grants. Servicio quatrrod
  detenido para instalador, 47 archivos actualizados con respaldo
  ~/.local/state/quatrro-install/backup-sjbev36i; daemon-reload/start y rescanPlugins.
  Nuevo protocolo 2, ingress loopback8791, idioma en, sin trabajos. Broker no instalado.
- Manifiesto instalado apunta a UI inmutable 043340a97b6a5c7129eefa3ba1a9fbfd980b1e41f5906b18639f2eeea4364d10.
  Valida; DB migrada a esquema5, integrity_check ok, grants0. Preferencia preservada.
- shell summon abre panel real mapped/not hidden; captura recortada solo del panel
  `.dev/native-panel-before.png` inspeccionada, muestra receptor y nuevas secciones.
  shell hide cierra panel y motor permanece activo. F0.5 cerrada, fase0 completada.
- F1D.6/F1D.7 abiertos para interacción de widget/teclado y QA. Geometría real de
  panel 664×718 (tiled), controles genéricos y widget más alto que barra quedan para
  QA/R1.UI; no se declara cerrada estética con esta captura. Instalación permanente
  ahora actualizada; documentación anterior sin actualizar solo describe pruebas previas.

## F1D.6 — Entrada real Qt y contrato de clic de barra

- Código instalado del shell revela que barra reenvía clics a triggerPress. Añadido
  al widget (solo botón izquierdo); MouseArea reutiliza ese método. Usa bar.barSize:
  altura horizontal y ancho vertical respetan dimensiones reales, fallback28.
- test-widget.py pasa contrato triggerPress izquierdo/derecho y ancho vertical26.
  Nuevo test-widget-input.py pasa eventos Qt reales mouseClick y Space/Return/Enter
  con backend stub; primera configuración falló por tema GTK y TestCase oculto,
  corregida aislando tema y haciendo visible el caso. 4 comprobaciones pasan.
- Instalado con respaldo backup-bkqildw9, servicio reiniciado sin trabajos,
  rescan y daemon-reload. Barra real informa visible,116×26,y0; status conserva
  motor saludable, protocolo2, inglés y receptorloopback. Manifiesto instalado válido.
- Navegación shell.togglePanelAt es uno-based y exige open/close/opened en widget.
  Índice0 devolvió unknown; índice1 abrió Agents, cerrado inmediatamente. Quatrro
  es omitido por tener panel separado. Próximo integrar fachada nativa de panel,
  evitando recursión shell.toggle; F1D.6 sigue abierta, estética R1.UI pendiente.

## F1D.6 — Cierre con navegación nativa del panel

- Widget expone open/close/opened mediante PluginShellApi.summon/hide/isPluginOpen.
  Manifiesto conserva kind panel: el loader del shell sigue siendo dueño del panel,
  sin segundo Panel embebido ni recursión de toggle. opened sigue cierre externo.
- test-widget.py pasa apertura/cierre/sincronización, estados reales y recarga;
  test-widget-input.py mantiene cuatro PASS con eventos Qt mouse/teclado.
- Instalación actualizada con respaldo backup-92hk9yu3, servicio recargado sinjobs,
  revisión UI nueva rescan. Manifiesto instalado valida, sesión desbloqueada.
- Barra real: togglePanelAt right1 devuelve quatrro.automations, ventana mapped
  única; segunda activación la cierra. summon/hide repite apertura/cierre correcto.
  Motor permanece ready, counts vacío. No se configura ningún flujo permanente.
- F1D.6 cerrada sumando controles, independencia de UI, estados e integración.
  F1D.7/R1.UI mantienen QA/estética nativa final. Próximo F1C.1 notificación visible
  y política de sesión; recuperación F2.7 y validaciones externas siguen abiertas.

## F1C.1 — Cierre de notificaciones

- Corregido título: D-Bus/Omarchy lo trata como texto plano; ahora conserva literal
  en vez de mostrar entidades HTML. Cuerpo permanece escapado, límite título200/
  cuerpo4096 bytes UTF-8 antes de escape y separación -- impide opciones en título.
- TestNotificationArgumentsAndLimits verifica argv, HTML/enlaces literales, límites
  exactos, exceso y Unicode; TestNotificationWithoutSessionFailsWithoutRetry prueba
  bus ausente, fallo estático y ausencia de repetición. Ambas con race pasan1,243s.
- test-notification-native.py pasa con motor temporal, grants explícitos y daemon
  real de sesión: snapshot título literal/cuerpo escapado, ejecución completed y
  captura visible inspeccionada. DNDoff/lockfalse comprobados sin cambiarlos.
  Descarta solo su notificación; no modifica configuración permanente ni borra historial.
- F1C.1 cerrada. Tres binarios recompilados; corrección del título aún no desplegada
  en servicio permanente. Próximo F2.7 recuperación; F1D.7/R1.UI siguen abiertos.

## F2.7 — Matriz de persistencia local y SQLite lleno

- Nueva matriz local: falla reserva running, inserción de paso o avance de ejecución.
  Comprueba cero efectos antes de reserva, rollback del paso/avance y ausencia de
  reejecución cuando el almacenamiento vuelve. Reapertura+Run deja uncertain
  después del efecto; trabajo que falló antes de reservar continúa de forma segura.
- Prueba de base llena ahora exige SQLITE_FULL(13), ausencia de evento/ejecución/
  deduplicación parciales y admisión única del mismo ID después de ampliar páginas.
- Pruebas race dirigidas locales/HTTP/full/SIGKILL pasan (4,459s), integridad ok.
  No se modifica producción ni perfil permanente; efecto local modelado por helper.
- F2.7 abierta: siguiente ENOSPC de filesystem aislado/WAL, suspensión y ciclo
  de sesión. No se confunde límite de páginas SQLite con llenar disco del host.

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

### Avance: accesibilidad y primeras guías bilingües

- Controles nativos con roles, etiquetas EN/ES, estado y acciones accesibles.
  La etiqueta descriptiva del campo de contraseña no incluye su valor; se
  conserva `passwordEdit` y el nombre vacío que Qt aplica a ese modo.
- Pruebas reales de ratón/teclado y contratos accesibles pasan, así como
  compatibilidad UI EN/ES y el flujo integral de formularios (seis entregas HTTPS).
  No equivale a una revisión con lector de pantalla; F1D.7/R1.UI siguen abiertos.
- Guías de uso equivalentes en `docs/en/user-guide.md` y `docs/es/user-guide.md`,
  con instalación, primera automatización, conexiones, acciones, monitores,
  programación, operación, recuperación, diagnóstico y acceso a GitHub.
- Casos guiados EN/ES y tres JSON importables deshabilitados: webhook/notificación,
  monitor/outbound y recordatorio programado. `TestDocumentedUseCases` comprueba
  validación y cuerpos resueltos, alerta/recuperación y ausencia de efectos.
- R1.DOCS sigue abierto: falta referencia por opción con dos ejemplos verificables,
  matriz de cobertura, casos avanzados, capturas finales y auditoría de paridad.

### Avance: referencia bilingüe de conexiones, flujos, programación y monitores

- 37 opciones redactadas en EN/ES con valor inicial, restricciones y dos ejemplos
  independientes por opción: campos comunes, entradas, destinos, flujos,
  condiciones, programaciones y monitores. `docs/reference/options.json` es la fuente;
  `docs/en/options.md` y `docs/es/options.md` son documentos generados legibles.
- Matriz `docs/option-coverage.md` enlaza cada campo cubierto a ambos idiomas.
  `scripts/render-option-reference.py --check` comprueba campos de esas cinco
  secciones contra Schema.js, estructura bilingüe, ejemplos y archivos generados.
  No acredita validación semántica ni ejecución de todos los valores de ejemplo.
- Verificados los límites contra modelo, proveedores, red privada, reglas y
  programador: una excepción privada exacta por destino; Slack solo JSON/form;
  condiciones tipadas y semántica de atrasos. R1.DOCS sigue abierto para acciones,
  código local, credenciales, controles globales y casos avanzados.

### Avance: referencia de acciones y código local

- Se incorporan acciones, scripts y adaptadores, parámetros tipados, valores de
  invocación, revisiones y directorios autorizados. La referencia EN/ES contiene
  73 opciones con dos ejemplos cada una; todos los tipos de recurso de Schema.js
  entran ahora en la comprobación de cobertura.
- Límites contrastados con command_profiles.go, scripts.go, adapters/protocol.go
  y sandbox: comandos/scripts 1–300 s, adaptadores 1–30 s, parámetros acotados,
  identidades de directorios y revisiones fijadas al código/contrato preparado.
- El generador admite ejemplos traducidos además de valores técnicos compartidos.
  `--check` verifica su paridad estructural y la vigencia de los documentos.
- Sigue pendiente R1.DOCS para credenciales, controles generales, demostraciones
  completas avanzadas, capturas finales y auditoría semántica de cada ejemplo.

### Avance: credenciales, OAuth y retención documentados

- Referencia EN/ES ampliada a 85 opciones: tipos de credencial, referencias,
  almacenamiento, campos OAuth, scopes y cuatro límites de almacenamiento.
  Incluye restricciones reales de solicitud OAuth (1–16 scopes, hasta 256 bytes),
  diferentes de los límites del parser de respuestas del proveedor.
- Etiqueta de valor genérico y error corregidos de caracteres a 16–8192 bytes,
  conforme a la validación existente. No cambia la política de credenciales.
- Verificados nombres de scopes de ejemplo contra la referencia oficial Google;
  se distingue un scope de solo lectura de los métodos POST/PUT/PATCH ofrecidos.
- `render-option-reference.py --check` pasa, con correspondencia explícita de
  doce controles del panel a opciones documentadas. Pruebas de rotación/backend,
  ejemplos importables y controles nativos pasan. No se han desplegado todavía
  las últimas etiquetas en la instalación permanente.
- Siguen abiertos controles generales, casos avanzados y revisión final de
  documentación; R1.DOCS no se marca completo por esta ampliación.

### Avance: controles operativos documentados en ambos idiomas

- 119 opciones/controles en la referencia: añade toolbar, edición, importación/
  exportación, cola/historial, simulación/prueba real, operaciones de credenciales,
  diagnóstico, cursor journal, preparación y comprobación administrativa.
- Cada entrada tiene dos ejemplos EN/ES. Referencia agrupada con índice navegable;
  no presenta valor inicial ficticio para botones. Se conservan anclas de matriz.
- Cuatro campos adicionales del panel enlazados explícitamente al checker:
  origen, tipo, datos de prueba y ruta de archivo. Generación/estructura/enlaces
  verificados, sin inferir cobertura semántica completa de ese resultado.
- R1.DOCS sigue pendiente de casos avanzados completos, capturas finales,
  auditoría de todas las variantes y correspondencia visual final.

### Avance: casos reproducibles de scripts y adaptadores

- Configuraciones importables `typed-script.json` y `adapter-notification.json`,
  con código y revisión exacta, flujos deshabilitados y fuentes legibles separadas.
- Guías equivalentes `docs/en/code-examples.md` / `docs/es/code-examples.md`:
  importación y construcción desde formularios, contrato tipado, dos eventos
  válidos por ejemplo, rechazo fuera de rango y resultados esperados.
- `TestDocumentedCodeExamples` pasa validación/hash, correspondencia de fuentes,
  simulación de argumentos y dependencia de adaptador sin efectos persistidos.
- `QUATRRO_HOST_TEST=1 ... -run '^TestHostDocumentedCodeExamples$'` pasa:
  ejecución real de ambos casos de script y adaptador dentro de systemd/Bubblewrap.
  Se comprueba texto de notificación resuelto sin mostrar un popup en esta prueba.
- Pendientes documentales: servicio autorizado, proveedor firmado, OAuth y
  recuperación como casos completos, capturas y auditoría final de opciones.

### Avance: caso GitHub firmado reproducible

- `github-release.json` añade entrada GitHub y flujo de releases publicadas,
  filtrado por repositorio/cuerpo firmado y recursos inicialmente deshabilitados.
- Guías EN/ES `provider-example.md` separan simulación, configuración real,
  requisitos HTTPS, firma, deduplicación y diagnóstico. Pasos de GitHub
  contrastados con documentación oficial de firmas y creación de webhooks.
- `TestDocumentedGitHubRelease` verifica dos tags simulados y handler real:
  aceptación, deduplicación pese a cambiar ID de entrega, rechazo de alteración,
  descarte por condiciones y texto resuelto por el worker. Sin popup ni red externa.
- No se creó webhook ni se cambió repositorio real. Casos de servicios/OAuth/
  recuperación, capturas y auditoría documental final siguen pendientes.

### Avance: caso de servicio autorizado reproducible

- `user-service.json` define acciones fijas status/restart y dos flujos
  inicialmente deshabilitados. El evento selecciona flujo, nunca unidad/argv.
- Guías EN/ES `service-example.md`: crear unidad transitoria propia, simular,
  revisar, activar, comprobar PID, revocar y limpiar; distingue broker de sistema.
- `TestDocumentedServiceSimulation` pasa status/restart y rechazo por condiciones
  de stop sin ejecuciones. `TestHostDocumentedService` pasa con systemd real:
  status conserva PID, restart lo cambia, revocación impide otro reinicio;
  dos completados, un denegado y limpieza de la unidad temporal.
- Aclarado status de usuario en referencia: consulta is-active y falla si inactivo.
  Quedan casos OAuth/recuperación, capturas y auditoría documental final.

### Avance: caso de recuperación HTTP

- `http-recovery.json` y guías EN/ES `recovery-example.md` completan el caso
  de fallo temporal, fallo terminal y reenvío explícito, con dos mensajes de ejemplo.
- `TestDocumentedHTTPRecovery` pasa con worker/outbox reales y emisor controlado:
  503 pendiente y espera, 400 fallido, reenvío pausado, 204 completado, misma clave/
  cuerpo incluso tras editar borrador, y rechazo de reintento del completado.
- El test adelanta su propio vencimiento persistido; no acredita transporte remoto
  ni idempotencia del receptor. Las guías distinguen resultados inciertos y acciones
  no HTTP, e indican requisitos del receptor de prueba controlado por el usuario.
- Caso OAuth, capturas finales y auditoría documental completa siguen pendientes.

### Avance: caso Google OAuth y navegación de ejemplos

- `google-oauth.json` y guías EN/ES `oauth-example.md`: cliente Desktop propio,
  consentimiento, scopes, referencia local, simulación de dos calendarios y POST
  Freebusy sin crear eventos. Contrato y preparación contrastados con Google.
- Se explica una limitación relevante: el plugin descarta cuerpo de respuesta;
  completado/HTTP 2xx no demuestra ausencia de errores internos por calendario.
- `TestDocumentedGoogleOAuth` pasa configuración, simulación sin credenciales y
  transporte controlado con token privado ficticio; eliminarlo bloquea otro envío.
  No se confunde con consentimiento/renovación/TLS reales frente a Google.
- Índice de casos EN/ES simplificado para enlazar guías avanzadas. Se solicitó al
  usuario disponibilidad de cliente Desktop/cuenta de prueba para F3.3, indicando
  introducir secretos localmente. Esa validación externa continúa pendiente.
- Documentación todavía requiere capturas finales y auditoría de opciones/variantes;
  no se cierra R1.DOCS ni F3.3 por esta evidencia local.

### Avance: controles numéricos nativos de Seguridad

- Sustituidos los cuatro SpinBox genéricos restantes del panel por el componente
  real `qs.Ui.NumberField`, mediante `qml/NumericField.qml`. Conserva edición,
  límites y actualización de valor, y etiqueta accesible de cada política.
- Prueba real de teclado verifica incremento/decremento, edición directa,
  límite superior, aumento del mínimo y bloqueo al deshabilitar. El mínimo de
  deduplicación se ajusta al aumentar retención, sin enviar un valor desfasado.
- Pasan controles nativos, flujo UI completo (seis entregas HTTPS), compatibilidad
  EN/ES, validación del plugin y diff-check. No se ha actualizado la instalación
  permanente con estos cambios ni se da por cerrada revisión visual final.

### Avance: capturas bilingües y límites del panel

- Añadido recorrido visual EN/ES con ocho capturas reales del panel, dos temas
  nativos y las vistas Conexiones/Seguridad, perfil privado y HTTP deshabilitado.
- Las capturas detectaron desbordamiento horizontal en Seguridad: ScrollView y
  contenido ahora comparten availableWidth y recortan correctamente. Comprobación
  de geometría verifica los cuatro límites dentro de la ventana en ambos idiomas.
- La paleta de los indicadores numéricos hereda foreground nativo para conservar
  contraste en el tema claro. Se mantiene NumberField de Omarchy.
- El paquete incluye imágenes PNG y comprueba su presencia e integridad; pasa
  reproducibilidad e instalación/actualización/retirada en staging. La prueba de
  formularios pasa 48 combinaciones (dos idiomas/temas y tres tamaños).
- Diez capítulos por idioma, incluidos hooks y administración, con 160 enlaces
  locales verificados. Pasan go test ./... y referencia generada de 119 opciones.
  R1.UI/R1.DOCS continúan abiertos hasta cerrar auditoría semántica y validación de la instalación final.

### Auditoría: nombre final y etiquetas accesibles

- R1.NAME cerrado: manifiestos de fuente/instalado/paquete, títulos de panel y
  widget instalados y metadatos release usan Omarchy Automations. No aparecen
  nombres públicos antiguos en el código actual. Los identificadores técnicos
  se conservan sin migración según docs/product-name.md; las comprobaciones
  anteriores del instalador y UI acreditan continuidad de datos y preferencias.
- Condiciones, directorios, parámetros/argumentos, pares, simulación y ruta JSON
  reciben etiquetas accesibles EN/ES. Botones de quitar identifican el elemento.
  Prueba QML comprueba nombres anidados traducidos y contratos de entrada reales.
- Estas etiquetas no equivalen a una sesión de lector de pantalla. La revisión
  visual/teclado y despliegue de la última UI continúan dentro de R1.UI/F1D.7.

### Avance: reproducibilidad y guía de publicación

- Dos compilaciones independientes (rutas y cachés separadas) producen los tres
  binarios idénticos con Go 1.26.8 Linux amd64. Hashes en docs/validation.md.
- Guías GitHub EN/ES incluyen comandos de acceso, autor, commit y push, con camino
  separado para remoto existente. No se ha autenticado ni publicado.
- Elección de licencia solicitada al usuario; sigue pendiente. R1.7 requiere
  todavía licencias/notas finales y el cierre de la versión estable.

### Avance: carga instalada y ciclo de datos

- Medidos reposo y admisión HTTP en instalación limpia y tras reinstalar con estado
  conservado. Dos ráfagas de 400 solicitudes; límite por entrada comprobado, 840
  eventos persistidos, cero efectos, integridad de base e idioma conservado.
- Resultados y límites en guías EN/ES performance.md y docs/validation.md.
  No se confunde la ráfaga con una tasa sostenida ni con ejecución de acciones.
- Pasan 13 pruebas del instalador y su recorrido con motor instalado/restaurado.
  R1.5 aún requiere los cinco escenarios completos en ambas etapas.

### Cierre R1.2: migración instalada y restauración

- El recorrido instalado parte ahora de una fixture de esquema 4 (sin context),
  conserva un trabajo pendiente/configuración/idioma y actualiza a esquema 5 con
  el binario instalado. No se presenta como ejecución de un binario histórico.
- El backup anterior se restaura y migra al abrir, conserva configuración/idioma,
  deja el motor pausado, revoca permisos y convierte trabajo previo en incierto.
  Rechazos de backup modificado y motor activo están cubiertos por los 13 tests.
- Pruebas de migraciones v1/v3/v4, conservación de dedup/temporizadores y rechazo
  de esquema futuro pasan. El procedimiento se encuentra en las guías EN/ES.
- Se cierra R1.2; el recorrido completo A1–A5 tras actualizar sigue en R1.5.

### Avance R1.5: A1 y A4 completos en ambas etapas

- Nuevo test-installed-acceptance.py: instalación limpia y actualización desde
  fixture esquema 4 a 5, con binarios instalados y estado/permisos conservados.
- A1 pasa firma inválida, duplicación y filtros repositorio/entorno; entrega válida
  produce una notificación nativa y una salida HTTPS autenticada por etapa.
- A4 pasa hook instalado/runner real, envío HTTPS autorizado, estado completed,
  conservación de hook ajeno y retorno rápido con motor detenido.
- Perfil/tema permanentes sin cambios; dos notificaciones temporales retiradas.
  Evidencia y límites en docs/validation.md. A2/A3/A5 permanecen pendientes en la
  matriz instalada/actualizada; no se cierra R1.5 todavía.

### Avance R1.5: A3 con servicios reales

- Webhook bearer → reinicio de unidad temporal exacta → comprobación de estado →
  reporte HTTPS pasa limpio y tras migración. PID cambia solo en unidad permitida.
- Otra unidad temporal con permiso revocado queda denied sin cambio de PID ni
  reporte; credencial inválida devuelve 401. Unidades retiradas al finalizar.
- Runtime real para systemctl y perfil de motor privado; límites de esta modalidad
  y primer fallo del harness documentados en docs/validation.md.
- A1/A3/A4 acreditados en ambas etapas. A2/A5 aún pendientes; R1.5 sigue abierta.

### Avance R1.5: A5 respaldo programado

- Temporizador real → script aprobado → respaldo verificado por bytes → reporte
  HTTPS pasa en limpio y tras migración v4/v5, sin reactivar configuración.
- Se verifica revisión fijada pese a modificar fuente, entrada ro, host no montado
  oculto, entorno secreto no heredado y timeout que evita escritura tardía.
- Perfil privado y directorios temporales; evidencia en docs/validation.md.
  A1/A3/A4/A5 acreditados; A2 sigue pendiente antes de cerrar R1.5.

### Cierre R1.5: A2 y matriz completa

- A2 pasa en limpio y tras migración: tmpfs privado 32 MiB, statfs real 6,25 % libre,
  confirmación previa, una alerta sin repetición, recuperación confirmada y HTTPS.
  Host montado de solo lectura; no se llena ni modifica el disco del usuario.
- A1–A5 pasan instalados y después de migrar fixture v4→v5. Reposo/carga medidos
  aparte; scripts, comandos y límites de cada prueba figuran en docs/validation.md.
- Se cierra R1.5 sin cerrar por extensión OAuth, sesión/suspensión ni QA UI.

### Cierre R1.4: seguridad y límites efectivos

- Matriz EN/ES de firmas/replay, red/DNS/SSRF, inyección, revocación, permisos,
  aislamiento y redacción, con evidencia y límites en docs/validation.md.
- Corregida apertura bloqueante de FIFO de credencial; regresión comprueba archivos
  especiales, symlinks, modos y tamaño. DNS cambiante se revalida por conexión.
- Pasan suite con race, pruebas reales de aislamiento y checks de compilación.
  Simplificado defer administrativo para que go vet pruebe su cancelación.
- R1.4 cerrada; OAuth/keyring real y despliegue final continúan separados.

### Cierre R1.1: actualización permanente verificada

- Instalador normal aplicado con motor sin trabajos; backup-q8scdsyx, 128 hashes
  verificados, servicio activo y manifiesto válido. Conserva configuración,
  permisos, shell.json, idioma y estado de habilitación.
- Correcciones de credenciales y UI desplegadas. Broker opcional no instalado.
- Panel real 664×718 y formulario abierto mediante Tab/Enter, revisados EN/ES;
  capturas de compositor en guías. Escape sin guardar; inglés y perfil vacío
  conservados. R1.UI/F1D.7 mantienen revisión global independiente.

## Requisitos finales — nombre, icono y repositorio del usuario

Estos requisitos forman parte de la entrega y se ejecutan antes de cerrar R1.8.

- [x] **R1.NAME Nombre definitivo:** el producto se llama **Omarchy Automations**.
  Actualizar nombre visible del plugin, panel, widget, notificaciones, manifiesto,
  documentación y artefactos de distribución. Auditar identificadores técnicos,
  rutas, servicios y datos existentes para que el cambio conserve la instalación
  y las preferencias; documentar y probar cualquier migración necesaria.
- [x] **R1.UI Icono:** sustituir el icono actual por uno coherente con la iconografía
  nativa de Omarchy. Comprobar legibilidad en la barra, escalado, temas y estados,
  junto con la revisión final de botones y controles. Mantener inglés por defecto
  y español seleccionable.
- [x] **R1.GITHUB Preparación de publicación:** repositorio indicado por el usuario:
  https://github.com/PuroDelphi/omarchy-automations . Preparar el remoto local,
  revisar archivos que se publicarán, excluir secretos y artefactos temporales,
  y entregar pasos concretos para autenticarse en esta máquina y subir el proyecto.
  La autenticación se hace en el navegador o herramienta local del usuario, sin
  compartir tokens, contraseñas ni claves privadas en el chat. La publicación
  queda pendiente de disponer de acceso y de la instrucción de subir los cambios.

## Requisito final — documentación completa en inglés y español

- [x] **R1.DOCS Guías completas equivalentes:** redactar al finalizar la interfaz
  y las funcionalidades, antes de R1.8, una guía de usuario en inglés y otra en
  español. Ambas deben cubrir el mismo contenido y enlazarse entre sí.
- [x] Explicar instalación/actualización/desinstalación, primer inicio, idioma,
  widget, navegación, configuración de cada recurso, borrador/activación,
  permisos/revocación, simulación/prueba real, pausa/cancelación, historial,
  reintentos, diagnóstico, importación/exportación y recuperación.
- [x] Documentar cada opción visible: finalidad, valores admitidos, valor
  predeterminado, requisitos, efectos y errores habituales. Cubrir entradas,
  destinos, autenticación/credenciales, formatos, acciones, condiciones/flujos,
  monitores, programación, scripts, adaptadores y administración opcional.
- [x] Incluir **varios ejemplos por opción** en los dos idiomas, con al menos
  dos ejemplos aplicables por opción y más cuando sus variantes lo requieran.
  Mantener una matriz de cobertura opción → sección EN/ES → ejemplos; no limitar
  la documentación a un resumen de cada pantalla.
- [x] Añadir casos de uso completos en ambos idiomas: webhook que notifica,
  alerta/recuperación de un monitor con entrega outbound, tarea programada,
  control autorizado de un servicio, comando/script con parámetros tipados,
  adaptador de eventos, proveedor firmado, credencial/OAuth y recuperación
  de entregas fallidas. Indicar requisitos y resultado verificable de cada caso.
- [x] Proporcionar configuraciones importables y comandos copiables sin secretos
  reales; diferenciar simulación y efectos reales. Comprobar los ejemplos contra
  la versión final y documentar las limitaciones reales de cada integración.
- [x] Incluir capturas de la interfaz final en ambos idiomas, solución de problemas,
  glosario y guía de GitHub para esta máquina. Verificar enlaces, paridad EN/ES y
  correspondencia entre etiquetas documentadas y las de la UI antes de entregar.


## Cierre R1.3 — retirada y conservación (2026-09-28)

- 15 pruebas del instalador pasan, incluidas orden de parada/deshabilitación antes
  del borrado y conservación de todos los archivos si falla uno de esos controles.
- `scripts/test-uninstall-service.py` verifica una unidad real temporal de usuario:
  parada, retirada de enablement runtime y binarios, conservación de SQLite,
  credencial de prueba y hook ajeno. El control del shell se registra en esa prueba.
- Tras la prueba independiente de deshabilitar/habilitar el widget nativo, se
  confirma plugin habilitado, preferencias del shell idénticas al respaldo y motor
  ready en inglés. No se desinstaló el plugin permanente para estas comprobaciones.
- Guías EN/ES explican hooks instalados por separado, broker opcional, fallos antes
  del borrado y recuperación. Los hooks manuales no forman parte del recibo.


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


## Auditoría de controles nativos — estados y pasos (2026-09-28)

Revisadas fuentes instaladas Button, PanelActionButton, Network y Tailscale;
correspondencias en docs/ui-audit.md. Se marca la comparación de componentes,
no la aceptación visual global. Prueba ampliada de controles Qt pasa foco,
Return/Enter/Space, hover, pulsación sin cambio de tamaño, selección, urgente
y bloqueo de clicks/teclado deshabilitados.

Corregidas etiquetas accesibles de flechas para reordenar pasos en EN/ES, con
número de paso; comprobadas por la prueba de etiquetas anidadas. Cambio fuente
pendiente de despliegue final junto con revisión UI global. R1.UI sigue abierto.


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


## Cierre R1.DOCS y R1.6 (2026-09-28)

Auditoría consolidada en docs/documentation-audit.md, con requisito→guías EN/ES
→fuentes/pruebas y alcance. 18 documentos emparejados por idioma, 119 opciones
con dos ejemplos mínimos, recetas por variante y casos importables, capturas
reales/de desarrollo, glosario, recuperación y publicación.

Nuevo scripts/check-docs.py reproduce paridad de capítulos, enlaces recíprocos,
296 enlaces locales/anchors y referencia generada. TestDocumented pasó tras
añadir formatos y autenticación. Test-release-package confirma guías/casos
incluidos, enlaces extraídos y ciclo de instalación. La revisión semántica se
registra explícitamente; no se confunde con estos controles estructurales.

Se actualizan introducciones y README para reflejar cierre documental de la
versión actual. Google real, logout/suspend, licencia/versión/publicación y cierre
general permanecen pendientes; documentar sus límites no los declara validados.


## Cierre R1.GITHUB — preparación local (2026-09-28)

origin verificado: https://github.com/PuroDelphi/omarchy-automations.git; rama
main, sin commits ni autor local. 314 archivos candidatos (3,416,443 bytes) sin
perfiles/generados/symlinks inesperados ni coincidencias de patrones de claves
privadas/tokens comunes. Es un análisis acotado, no prueba universal de ausencia
de secretos. Vectores públicos de prueba son intencionales.

Git diff --check y cached --check pasan. .gitignore ampliado para .env.* y
SQLite; guías EN/ES explican revisión de staging antiguo, login preparado,
autoría, commit, remoto vacío/con historia y verificación posterior. No se ha
iniciado sesión, creado commit ni publicado. Preparación local cerrada; elegir
licencia y cerrar versión estable siguen en R1.7, acceso/publicación a cargo del
usuario conforme a sus instrucciones.


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


## Licencia MIT y paquete actualizado (2026-09-28)

El usuario eligió MIT explícitamente. LICENSE contiene el texto completo con
copyright 2026 Omarchy Automations contributors. README, guías de publicación
y notas EN/ES enlazan la licencia; eliminada la instrucción vigente de elegirla.
Las solicitudes históricas anteriores quedan resueltas por esta decisión.

El paquete incluye LICENSE y la prueba compara sus bytes con el original.
Se agregaron textualmente los avisos adicionales de modernc libc, la dedicación
SQLite y LICENSE/PATENTS disponibles de los paquetes vendor Go crypto/net/sys/text
usados por el grafo de dependencias de cmd. go list -deps se ejecutó sin red;
no se incluyeron avisos de testdata ni se atribuyeron al producto sus licencias.

check-docs: 19 pares, 311 enlaces locales; test-release-package: archivos
deterministas, hashes y ciclo instalación/actualización/desinstalación en staging
correctos, sin activar servicios del host. git diff --check correcto.
Continúan pendientes consentimiento Google real y ciclo físico de sesión;
no se anuncia versión estable ni se ha publicado en GitHub.


## Preparación de pruebas con el usuario (2026-09-28)

Usuario dispuesto a hacer login Google. Panel y Google Cloud abiertos; motor
verificado ready, sin error, sin ejecuciones, idioma en y listener loopback.
Guías external-acceptance EN/ES describen consentimiento, consulta, renovación
tras expiración real, persistencia del almacén y transición suspend/logout.
Cliente Desktop propio aún por confirmar; no se han introducido credenciales ni
ejecutado consentimiento/consulta/suspensión. F3.3/F2.7 siguen abiertos.


## Publicación solicitada; pruebas externas pospuestas (2026-09-28)

El usuario solicita omitir por ahora las pruebas Google/sesión y subir al
repositorio; pedirá ayuda para retomarlas. Se mantiene 0.1.0-dev y no se presenta
como versión estable 1.0. Se inicia conexión GitHub por navegador, sin solicitar
contraseñas ni tokens en el chat.

Remoto inspeccionado antes de publicar: main contiene commit inicial ec561e2
con LICENSE MIT. Se conserva su historial y copyright JhonnySuarez junto al
aviso de contributors; no se usa force push.


## Revisión de presentación pública e instalación (2026-09-28)

README ampliado con casos de uso, capturas existentes inspeccionadas, requisitos,
instalación completa, primera automatización, actualización, desinstalación y
soporte. Referencias revisadas: READMEs de Reprise y OmaPilot, y comandos Omarchy
locales add/remove. Corregida nota obsoleta en docs/installation.md.

Mejoras de distribución detectadas (propuestas; no declaradas implementadas):
- Integración completa del backend con una instalación estándar Omarchy: add
  actualmente solo clona y entra en conflicto con la propiedad del instalador.
- Publicar artefactos versionados y checksums como GitHub Releases para evitar
  exigir Go al usuario final; el paquete local no equivale a release publicada.
- Automatizar verificaciones portables en CI; todavía no hay workflow .github.
- Traducir las notas técnicas históricas que siguen solo en español; las guías
  de usuario ya están emparejadas.
Se conservan las pruebas Google/sesión aplazadas por el usuario.

## Mejoras solicitadas de documentación y distribución (en curso)

El usuario solicita ejecutar las mejoras detectadas: instalación sencilla,
paquetes descargables, CI y traducciones técnicas. Google/suspensión/logout
permanecen aplazados según su instrucción anterior.

Documentación: 36 capturas nuevas reproducibles de nueve ejemplos, con valores
reales de los JSON públicos y formularios EN/ES, sin activar efectos. Se incorporan
recorridos visuales, procedimiento común paso a paso, resultados esperados y
comparación de pruebas positivas/negativas. Las capturas no acreditan conexión
real a proveedores. Generador: scripts/capture-tutorials.py.

CI: workflow portable con acciones oficiales fijadas por SHA, permisos de solo
lectura, Go 1.26.8, Python 3.13, vet, race, documentación, instalador y paquete.
No sustituye pruebas gráficas/host de Omarchy. Resultado remoto aún por verificar.

Instalación simplificada implementada: scripts/setup.py descarga una versión
precompilada seleccionada, valida archivo/miembros/hashes y usa el instalador
revisado local. Modo --git-plugin mantiene QML/.git bajo Omarchy y registra solo
motor/CLI/unidad. Rechaza UI distinta y mezcla de modos; conserva modo antiguo.
Documentadas instalación en dos comandos, actualización y retirada ordenada,
uso offline y fallback desde fuentes en EN/ES. Publicación y descarga real de
Preview 1 pendientes de verificar antes de considerar cerrada esta mejora.


### Preview 2 — activation onboarding correction

- [x] Preview 1 published and its public download/install verified; remote CI passed.
- [x] Explain saved draft versus active revision in the panel and both user guides.
- [x] Explicit Authorize and activate button, activation confirmation and source guidance.
- [x] Real tests open History, report execution counts and explain zero matches.
- [x] Replace empty tutorial History with actual completed notification captures in EN/ES.
- [x] Native regression: unactivated draft cannot run, activation enables real notifications, conditions producing zero matches are explained, both languages.
- [x] Layout checks: 64 forms, 120 dialogs, two themes and three sizes.
- [x] Add paired technical references and Git-managed native panel test.
- [ ] Google OAuth and physical suspend/logout acceptance remain deferred by the user.

### Documentation installation consistency

- [x] EN/ES user guides use prebuilt setup for installation, updates and removal.
- [x] Source compilation and isolated development workflows moved to paired development guides.
- [x] README and installation guides link to optional development instructions.
- [x] Legacy installer/package/broker references identify their advanced scope and link to normal installation.
- Historical validation records and third-party license notices remain unchanged.
