# Recuperación y resultados inciertos

El motor confirma una entrada solo después de guardar evento y ejecuciones.
Cada ejecución conserva su revisión de configuración. Reiniciar no reemplaza
sus acciones por las de un borrador posterior; los permisos se vuelven a
comprobar antes del siguiente efecto.

| Frontera de interrupción | Comportamiento al reiniciar |
|---|---|
| Entrada sin commit | No se confirma; el emisor debe reintentar |
| Entrada persistida, paso pendiente | Continúa desde el paso pendiente |
| Acción local marcada en ejecución, resultado sin commit | Queda `uncertain`; no se repite automáticamente |
| Acción local y avance de paso persistidos | Continúa en el paso siguiente |
| HTTP enviado, respuesta sin persistir | Reintenta con el mismo cuerpo y `Idempotency-Key` |
| HTTP entregado, avance de paso sin persistir | Avanza sin reenviar |
| HTTP fallido o cancelación incierta persistidos, paso sin cerrar | Conserva fallo/incertidumbre sin reenviar |
| Próximo reintento HTTP persistido | Respeta su fecha después del reinicio |

Un efecto externo y la base de datos no comparten transacción. Por eso no se
promete exactamente una vez. El receptor HTTP debe implementar idempotencia;
conservar la clave por sí solo no evita repetir efectos en ese receptor.
La ventana entre reservar una acción local y despacharla puede producir un
resultado incierto aunque todavía no haya ocurrido el efecto.

Cada intento HTTP se reserva antes del despacho. Una caída consume ese intento
aunque ocurra antes de enviar; reiniciar no permite eludir el límite de ocho.
Una cancelación confirmada sobre HTTP en curso permanece incierta incluso si
el motor muere antes de que termine la llamada de red.

## Intervención del usuario

**Cancelar** guarda primero la cancelación de pendientes y la incertidumbre de
HTTP en curso en una transacción; después solicita detener trabajos activos.
Si falla ese commit, devuelve error sin confirmar ni señalar la cancelación. Una
acción despachada puede haber ocurrido; se muestra incierta. Inspecciona el
servicio o receptor correspondiente antes de crear un nuevo trabajo.

**Reintentar entrega** solo admite HTTP fallido o cancelado pendiente de envío;
no admite ejecuciones completadas ni inciertas. Conserva cuerpo y clave, y
reinicia el presupuesto de intentos. El permiso sigue siendo obligatorio.

Si falla la persistencia después de un efecto, el motor conserva el trabajo
en ejecución y muestra un error de almacenamiento. No lo vuelve a ejecutar
en ese proceso. Tras resolver el almacenamiento y reiniciar, se aplican las
reglas de la tabla. No edites estados directamente en SQLite.

Sin servicio de notificaciones de sesión, la acción falla con un mensaje
genérico y no se reintenta automáticamente al abrir sesión. Un resultado
correcto de D-Bus acredita aceptación por el servicio, no que el usuario haya
visto la notificación.

## Evidencia y límites actuales

Las pruebas matan un proceso independiente mediante SIGKILL antes del despacho,
después de reservar, después del efecto, después del commit del paso, después del envío HTTP y después de confirmar su
cancelación. Reabren el WAL sin cierre limpio y comprueban integridad,
estado e identidad del reintento. Un marcador de archivo modela el efecto
externo; no equivale a cortar la alimentación del equipo.

Se inyectan fallos al crear outbox, reservar ejecución, reservar intento y
persistir el resultado, verificando que no se envíe antes de guardar las
reservas ni se repita trabajo en curso. También se inyecta un fallo al guardar
pasos después de persistir cada resultado HTTP terminal. La prueba de base llena usa el límite real de páginas
SQLite. Temporizadores y monitores tienen pruebas de reloj discontinuo y de
rollback de cursor/estado junto con eventos.

Las acciones locales tienen pruebas en la reserva, inserción del paso y avance
de ejecución: fallar antes del efecto permite continuar; fallar después conserva
estado running sin paso parcialmente confirmado y al reabrir queda uncertain.
La prueba SQLITE_FULL comprueba el código SQLite 13, ausencia de evento,
ejecución y deduplicación parciales, y permite reintentar el mismo evento al
ampliar el límite de páginas. No equivale a agotar el filesystem del usuario.

`TestFilesystemFullRecovery` llena un tmpfs privado de 16 MiB hasta recibir
ENOSPC del kernel, con el filesystem del host montado en solo lectura. Trunca
el WAL antes del llenado para forzar nuevas escrituras. Exige error SQLite de
almacenamiento y ausencia de evento, ejecución y deduplicación parciales;
libera espacio, reintenta el mismo ID y reabre la base para comprobar integridad
y una única ejecución persistida. Se ejecuta con `QUATRRO_HOST_TEST=1`.
Esta prueba cubre falta de espacio durante la entrada, no todas las fronteras
de efectos externos ni fallos físicos del dispositivo.

La prueba `scripts/test-service-lifecycle.py` ejecuta una unidad temporal con
la configuración empaquetada y un target gráfico privado. Detener ese target
conserva el motor y la admisión; congelar su cgroup durante 12 segundos y
reanudar verifica consolidación y omisión de temporizadores. Stop/start conserva
pausa e historial. Esta congelación no equivale a suspensión física y el target
privado no reproduce el cierre completo del gestor de usuario.

F2.7 sigue abierta: falta validar integralmente sesión/suspensión.
No se ha suspendido ni cerrado la sesión del usuario para estas pruebas.
