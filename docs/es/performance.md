# Recursos medidos y límites de admisión

[English](../en/performance.md) · [Guía de usuario](user-guide.md)

La compilación de desarrollo se midió en esta máquina Linux amd64 con
`python3 scripts/test-installed-load.py`. La prueba instala los binarios reales
en un HOME temporal, los ejecuta, los detiene, reinstala y repite con los datos
conservados. Es reinstalación de la misma versión, no migración de esquema anterior.

Cada fase mide cinco segundos de reposo y envía 400 solicitudes JSON autenticadas
por HTTP loopback, con cuatro clientes concurrentes y cuatro entradas. Cada
solicitud incluye un mensaje de 1024 caracteres y campos de secuencia/fase.
No hay flujos, monitores, scripts, notificaciones ni entregas salientes. Solo se
mide el proceso del motor; se excluyen CLI, instalador e interfaz de escritorio.

| Medición | Instalación limpia | Tras reinstalar |
|---|---:|---:|
| Tiempo CPU en 5 segundos de reposo | 0,01 s | 0,02 s |
| Memoria residente en reposo | 19856 KiB | 21940 KiB |
| Memoria residente tras la ráfaga | 23784 KiB | 24460 KiB |
| Pico de memoria residente | 23784 KiB | 24852 KiB |
| Tiempo para admitir 400 solicitudes | 0,406 s | 0,581 s |
| Mediana de latencia | 3,789 ms | 5,571 ms |
| Percentil 95 de latencia | 4,672 ms | 6,767 ms |
| Tiempo CPU del motor durante la ráfaga | 0,58 s | 0,76 s |

Son observaciones de una ejecución, no capacidad garantizada ni tasa sostenida.
La fase actualizada empieza con datos existentes; el tiempo CPU puede superar al
tiempo transcurrido cuando varios hilos usan distintos núcleos. Hardware, disco,
cuerpos y trabajo activo cambian los resultados. Cinco segundos de reposo no
acreditan estabilidad de memoria a largo plazo.

Tras cada ráfaga, se envían 20 solicitudes más a una entrada, alcanzando su límite
de 120 por ventana; la siguiente debe devolver 429. Se verifican 420 y después
840 eventos conservados, cero ejecuciones e integridad SQLite. El idioma inicial
es inglés; la preferencia española elegida sobrevive a reinstalar. Desinstalar
conserva la base. No se cambia ningún servicio ni preferencia del escritorio real.

## Límites operativos

La entrada admite actualmente hasta 16 manejadores simultáneos, 120 intentos por
entrada por minuto y 600 intentos globales por minuto. Exceder concurrencia devuelve
503; exceder tasa devuelve 429 con `Retry-After: 60`. Los intentos cuentan antes
de autenticar correctamente, por lo que los fallos pueden consumir la ventana.
Son límites fijos del motor, no opciones configurables por entrada. Los cuerpos
se limitan a 262144 bytes. Las cuotas de almacenamiento pueden rechazar eventos
nuevos que serían válidos por lo demás.

Por ejemplo, 100 solicitudes a una entrada sin uso previo caben en su ventana;
121 solicitudes dentro de esa ventana la superan aunque sobre CPU. Repartir entre
entradas no elimina el límite global. Respeta respuestas de reintento y usa
identificadores de entrega estables cuando el contrato del proveedor permita
deduplicar; los eventos rechazados por tasa no han sido aceptados.

Despacho de cola, receptores lentos, concurrencia de comandos y recuperación
requieren sus propias pruebas funcionales. Los cinco escenarios completos del
roadmap siguen siendo un control separado; esta medición no cierra R1.5 por sí sola.

Por separado, `scripts/test-installed-acceptance.py --native-notifications` pasó
A1 (despliegue firmado → notificación nativa y HTTPS) y A4 (hook de tema Omarchy
→ HTTPS), en instalación limpia y tras actualizar una fixture del esquema 4 a 5.
Verifica firmas, deduplicación, filtros repositorio/entorno, permisos conservados
y hook sin bloqueo con motor apagado. Muestra y retira dos notificaciones del
escritorio identificadas de forma única. La matriz instalada/actualizada completa ya está verificada como se describe más abajo. Consulta alcance y evidencia en el [registro de validación](../validation.md).

El modo `--user-services` pasó además A3 antes/después de migrar: un webhook
autenticado reinicia un servicio de usuario temporal, comprueba estado activo y
reporta por HTTPS. Otra unidad con permiso revocado queda denegada sin cambiar
ningún PID; se rechaza bearer inválido. Este modo necesita el runtime real del
gestor de usuario y aísla el motor instalado mediante `QUATRRO_PROFILE`. No prueba
backups automáticos del instalador para perfiles alternativos. Ambas unidades
temporales se detienen al salir; no se tocan servicios preexistentes.

`scripts/test-installed-schedule.py` pasó también A5 en instalación limpia y tras
migrar esquema. Un temporizador real de 10 segundos ejecuta un script fijado que
copia un documento temporal de un montaje de solo lectura a otro autorizado para
escritura y reporta por HTTPS autenticado. Los bytes del respaldo coinciden.
Modificar el script original no cambia su revisión aprobada. Una ejecución lenta
separada falla con timeout de un segundo y no realiza la escritura posterior.
También se comprueba que un archivo del host sin montar queda oculto y que un
secreto fixture no aparece en el entorno del script ni cuerpos de eventos/pasos/
outbox. Usa perfil y datos temporales; no respalda documentos reales del usuario.

Finalmente, `scripts/test-installed-monitor.py` pasó A2 en ambas etapas con
lecturas reales de disco sobre un tmpfs aislado de 32 MiB. Ocupar 30 MiB reduce
el espacio libre al 6,25 %. Verifica cinco segundos de confirmación antes de
alertar, dos muestras bajas adicionales sin duplicados y recuperación confirmada
al liberar espacio. Cada fase entrega alerta y recuperación por HTTPS autenticado.
El filesystem del host está montado de solo lectura; la prueba no puede llenarlo.

Estas pruebas instaladas y la medición separada de reposo/carga cierran R1.5.
No cierran OAuth con cuenta real, recuperación de sesión/escritorio ni revisión
visual final. Las actualizaciones reconstruyen el esquema anterior, sin ejecutar
un binario histórico. El [registro de validación](../validation.md) relaciona
cada requisito con su prueba.
