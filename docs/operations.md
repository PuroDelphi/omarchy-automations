# Operación y diagnóstico

## Inspeccionar una cola

En **Historial**, activa «Solo cola pendiente / incierta». La vista muestra
trabajos pendientes, en ejecución o inciertos, con paginación de 100 filas.
El historial normal sigue mostrando las 100 ejecuciones más recientes.

CLI equivalente:

```bash
quatrroctl queue.inspect '{"state":"pending","offset":0,"limit":100}'
quatrroctl queue.inspect '{"state":"uncertain","offset":0,"limit":100}'
quatrroctl history.detail '{"id":"IDENTIFICADOR_DE_EJECUCION"}'
```

`queue.inspect` devuelve `items`, `total`, `offset`, `limit` y `has_more`.
Las filas incluyen ID, flujo, estado, paso, creación, intentos y próximo
reintento HTTP. No incluyen el evento, cuerpo de salida ni credenciales.
La paginación refleja una cola viva: si se procesan trabajos entre consultas,
pueden cambiar las filas de una página.

En los detalles, un paso `completed` ya terminó aunque la ejecución siga
pendiente del siguiente paso. Una entrega HTTP `pending` todavía espera un
intento. Los pasos exitosos de versiones anteriores, guardados internamente
como `pending`, se muestran como `completed` al consultarlos sin reescribir
los registros. Un paso bloqueado por permisos queda `denied` con su motivo;
revocar no deshace efectos de pasos anteriores.

Pausar efectos permite revisar trabajos antes de que continúen. La admisión
`retain` los conserva; `reject` rechaza eventos nuevos. Cancelar no revierte
acciones externas ya iniciadas. No reintentar a ciegas un resultado incierto.
El reenvío manual está limitado a pasos HTTP fallidos/cancelados y mantiene
la clave de idempotencia y el cuerpo originales.

En el panel, **Pause / Pausar** retiene eventos nuevos y detiene el despacho de
los siguientes trabajos; **Resume / Reanudar** vuelve a procesarlos.
**Stop executions / Detener ejecuciones** pide confirmación: cancelar el diálogo
no cambia los trabajos, mientras que aceptar cancela pendientes e intenta detener
procesos activos. No reemplaza la pausa ni impide admitir eventos posteriores.
Cerrar el panel no detiene el motor ni cambia su estado de pausa.

Para recuperar una entrega fallida desde la interfaz:

1. Abre **History / Historial** y desmarca el filtro de cola para ver fallos
   terminales. Pulsa **Details / Detalles** para revisar estado HTTP e intentos.
2. Corrige el problema del receptor. Pulsa **Retry HTTP / Reenviar HTTP** y
   revisa la advertencia de idempotencia; cancelar el diálogo no envía nada.
3. Confirma el reenvío. La misma ejecución vuelve a pendiente y, si el receptor
   acepta la entrega, termina completada. Abre sus detalles para comprobar el
   estado HTTP final. La pausa y los permisos siguen aplicándose al reenvío.

El contador de intentos se reinicia al solicitar un reenvío manual; no representa
un acumulado de todos los ciclos manuales. Reutilizar la clave no garantiza que
el receptor evite duplicados: este debe implementar idempotencia.

## Simular antes de ejecutar

**Simulate / test · Simular / probar** permite introducir un origen, tipo y datos
JSON de ejemplo. **Simulate without effects / Simular sin efectos** usa el
borrador (lo guarda si hay cambios), sin activar permisos, consultar credenciales
ni crear ejecuciones. El resultado explica qué flujos están deshabilitados,
cuáles tienen otro origen y qué condiciones coinciden. Muestra los parámetros
resueltos de los pasos que se ejecutarían; las condiciones de flujos deshabilitados
o de otro origen se evalúan solo para explicar el resultado.

Los adaptadores no se ejecutan durante la simulación. Los pasos que dependen de
su salida se identifican como pendientes de ese resultado; no se inventan valores.

**Run real test / Ejecutar prueba real** abre una confirmación separada. Al
aceptar, emite el evento sobre la revisión activa con sus permisos y puede
producir efectos reales. Cancelar no emite el evento. Esta prueba admite orígenes
locales/hook; no sustituye la validación de firma de un webhook de entrada.

## Diagnóstico para compartir

En **Seguridad**, «Ver diagnóstico» muestra versiones y contadores;
«Exportar diagnóstico» crea un archivo JSON nuevo con permisos 0600.

```bash
quatrroctl diagnostics
quatrroctl diagnostics.export-file '{"path":"/ruta/absoluta/diagnostico.json"}'
```

El reporte versión 1 contiene versiones del motor/protocolo/base de datos,
plataforma, estado de pausa, error operativo de una lista conocida, contadores
de recursos y cola, cuotas/retención y disponibilidad de herramientas en PATH.
La disponibilidad de un ejecutable no demuestra que su servicio o sesión
estén operativos; los permisos del ejecutor se comprueban por separado al
activar comandos.

El reporte se construye mediante campos permitidos. No recoge nombres de
recursos, configuración, URLs, rutas, entorno, cuerpos de eventos, cabeceras,
valores de secretos ni mensajes arbitrarios del journal. No se comparte
automáticamente con ningún servicio.

## Configuración portable

Exportar configuración guarda el borrador y referencias a credenciales.
No consulta sus valores. Los campos comunes que el usuario haya escrito
manualmente —por ejemplo una cabecera o el cuerpo de una acción— sí forman
parte de la configuración; usa referencias para las credenciales.

La exportación requiere una ruta absoluta `.json` nueva y nunca reemplaza
archivos existentes ni sigue un enlace simbólico final. Si la escritura falla,
puede quedar un archivo incompleto; el error lo indica y debe eliminarse antes
de volver a usar ese nombre. El límite portable es 1 MiB; se utiliza formato
compacto cuando la indentación superaría ese límite.

Importar lee un archivo regular, sin seguir un enlace simbólico final, con
límite durante la lectura. No admite FIFO/dispositivos. Sustituye solo el
borrador y deshabilita entradas, flujos, monitores y programaciones; no modifica
la revisión activa ni concede permisos. Después de importar, crea las
credenciales referenciadas, revisa los recursos y activa explícitamente.

## Credenciales y retención

«Crear o rotar credencial» reemplaza el valor de una referencia existente en
su mismo backend. Las siguientes consultas del receptor/emisor obtienen el
valor nuevo; una petición ya autenticada o en curso no puede deshacerse.
Eliminar una credencial hace fallar las conexiones que aún la referencian.
Para cambiar entre Secret Service y archivo, elimina primero la referencia y
créala con el backend nuevo: se evita dejar una copia antigua sin gestionar.

Si falta una credencial de salida, la entrega no se envía sin autenticación:
permanece pendiente mientras se aplica la política de reintentos. Restaurar la
misma referencia permite que el siguiente intento use su valor actual. Si ya
se agotaron los reintentos y la entrega está fallida, usa el reenvío manual.

Los archivos de credenciales son una alternativa explícita de texto plano con
permisos restrictivos. Secret Service puede estar ausente o bloqueado; el motor
devuelve error y no selecciona silenciosamente otro backend.

La sección Seguridad permite configurar cuotas y retención. La limpieza
conserva trabajos pendientes, en curso e inciertos; la deduplicación tiene su
propia ventana. `quatrroctl storage.status` muestra política y ocupación.

## Notificaciones y sesión gráfica

Las acciones de notificación usan el servicio D-Bus de la sesión del usuario.
El título es texto plano (máximo 200 bytes UTF-8); el cuerpo admite hasta 4096
bytes antes de escapar el markup. Las etiquetas proporcionadas por eventos se
muestran literalmente: no crean formato, imágenes ni enlaces activos.

Si el servicio no está disponible, la acción queda fallida con un mensaje
estático y no se reintenta automáticamente al iniciar sesión. No se guarda
una cola separada de notificaciones para reproducirla al regresar al escritorio.
Una ejecución completada significa que el servicio aceptó la notificación;
No molestar, la política del escritorio o el bloqueo pueden impedir verla.

## Vida del servicio y escritorio

El servicio de usuario pertenece a `default.target`. Su relación
`After=graphical-session.target` ordena el arranque; no lo detiene cuando termina
ese target. Cerrar el panel o detener el shell tampoco detiene el motor.
Al terminar el gestor de usuario, el motor recibe la parada del servicio; al
volver a iniciarlo, recupera la cola persistida. La duración del gestor después
de cerrar sesión depende de logind y de la configuración de linger del equipo.
El instalador no habilita linger ni promete ejecución continua sin sesión.
Las notificaciones y el almacén de credenciales siguen dependiendo de sus
servicios de sesión, aunque el motor esté activo. Durante la suspensión no hay
ejecución; los temporizadores aplican su política de atrasos al reanudarse.
