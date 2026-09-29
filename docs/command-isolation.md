# Aislamiento de comandos

Las acciones `command` ejecutan un binario registrado bajo `/usr/bin` con hasta
32 argumentos fijos. La revisión aprobada contiene el ejecutable y cada
argumento; no se sustituyen campos del evento ni se hace expansión de shell.
Las rutas deben ser absolutas y canónicas, y se rechazan NUL, plantillas y
argumentos de más de 4096 bytes. Un intérprete registrado puede interpretar
sus propios argumentos dentro del aislamiento: hay que revisar su contenido
como código local antes de autorizarlo.

## Perfiles de comandos

`command_profile` selecciona un contrato. Ausente o `fixed` conserva el comando
avanzado descrito arriba: se autoriza el ejecutable y argv completos, sin afirmar
que el motor conoce la semántica de cualquier programa instalado.

Los perfiles concretos generan el ejecutable y los argumentos; no admiten
`executable` ni argumentos adicionales proporcionados por la configuración:

| Perfil | Parámetro | Ejecución generada | Acceso requerido |
|---|---|---|---|
| `file-exists` | `command_path` | `/usr/bin/test -f RUTA` | Montaje ro o rw del padre |
| `make-directory` | `command_path` | `/usr/bin/mkdir --mode=0700 -- RUTA` | Montaje rw del padre |

La ruta debe ser absoluta y canónica y nombrar un hijo directo de un montaje
aprobado, con nombre de hasta 255 bytes. Se rechazan `..`, rutas anidadas sin
montaje explícito de su padre, NUL y plantillas de evento. Los espacios y
metacaracteres se transmiten literalmente como parte de un único argumento.
No se añaden opciones desde el evento ni se permite cambiar el ejecutable del
perfil. Cambiar perfil, ruta o montajes modifica la capacidad que hay que aprobar.

`file-exists` comprueba un archivo regular; sigue enlaces según la semántica de
`test -f`, dentro de lo visible en el sandbox. No fija la identidad del archivo
ni garantiza que siga existiendo para un paso posterior. `make-directory` crea
un único directorio con modo 0700, sin crear padres; falla si ya existe y no lo
modifica. Contratos contrastados con el manual de [GNU Coreutils](https://www.gnu.org/software/coreutils/manual/coreutils.html).

La validación, simulación, preflight y ejecución comparten la misma resolución
del perfil. La simulación muestra el ejecutable y argv generados sin ejecutarlos.
En la UI se elige **Command profile**, se introduce **Profile path** y se prepara
el montaje del padre en **Approved directories**. La revisión de capacidades
muestra perfil, ruta y alcance del montaje. Estos perfiles no convierten los
comandos `fixed` en operaciones semánticamente verificadas.

## Perfil aplicado

Cada ejecución usa una unidad transitoria de systemd del usuario y Bubblewrap:

| Recurso | Límite o acceso |
|---|---|
| CPU | 50 % de un núcleo, aplicado por cgroup |
| Memoria | 256 MiB; swap adicional deshabilitado; OOM termina el grupo |
| Procesos/hilos | Máximo 32 tareas, incluidos los procesos del aislamiento |
| Duración | De 1 a 300 segundos según la acción |
| Archivos | `/usr` de solo lectura; `/tmp` y `/home` privados y temporales |
| Sesión | Sin HOME real, sockets de sesión ni bus D-Bus dentro del comando |
| Red | Namespace privado sin acceso a la red del host |
| Privilegios | `NoNewPrivileges=yes` |
| Salida | stdout/stderr descartados; no se guardan en el historial |

El motor de desarrollo permite añadir árboles concretos mediante `directories`
y elegir `working_directory`, sujetos a identidad, modo de acceso y capacidad
de la acción. Véase [directorios autorizados](directory-access.md); el formulario
está implementado y el despliegue sigue pendiente. Las operaciones de escritorio se realizan
mediante los tipos explícitos `notify`, `service` y `omarchy`.

## Activación y terminación

Antes de activar una revisión que contiene comandos, el motor ejecuta una
prueba inocua con el perfil real. Lee los controles `memory.max`,
`memory.swap.max`, `pids.max` y `cpu.max` de su cgroup y exige los valores
esperados. La prueba debe terminar correctamente. Si no puede verificarlo,
rechaza la activación; no ejecuta comandos sin aislamiento como alternativa.

Cada unidad usa `KillMode=control-group`, una parada de un segundo y SIGKILL
habilitado. Cuando se cancela el cliente o falla la ejecución, el motor
solicita además SIGKILL al grupo antes de detener la unidad. Esto cubre hijos
que ignoran SIGTERM o crean su propia sesión de procesos.

La cancelación no revierte efectos que ya ocurrieron. El motor conserva la
semántica de resultado incierto para acciones interrumpidas; no las repite
ciegamente.

## Evidencia

`QUATRRO_HOST_TEST=1 go test -race ./...` incluye pruebas reales de:

- Lectura de controles del kernel y aumento del contador de throttling de CPU.
- Asignación acotada por encima del límite y resultado `oom-kill` en el journal
  de la unidad temporal propia.
- Rechazo de creación de procesos al alcanzar el límite de tareas.
- Montaje `/usr` con bandera de solo lectura, archivo privado del host
  inaccesible y variables sensibles ausentes.
- Conexión bloqueada a un listener local del host y ausencia de rutas de red.
- Timeout y cancelación con un hijo que ignora SIGTERM y llama `setsid`;
  comprobación de que todos sus PID desaparecen.
- La misma terminación para scripts mediante el auxiliar con directorios rw:
  hijo escribe continuamente, ignora SIGTERM y llama `setsid`; todos los PID
  capturados desaparecen y el archivo deja de crecer al terminar el ejecutor.
- Cancelación de un trabajo de script desde el motor: estado persistido incierto,
  sin escrituras posteriores ni repetición al ejecutar el siguiente ciclo.

Las pruebas requieren systemd del usuario, Bubblewrap, Python 3 y acceso al
journal propio. Solo crean recursos temporales con nombres exclusivos. No
demuestran protección frente a fallos del kernel ni frente a un atacante que
ya controla la cuenta del usuario, límites fuera del compromiso del proyecto.
