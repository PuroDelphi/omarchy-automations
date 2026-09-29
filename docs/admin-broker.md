# Broker administrativo opcional — F3.5

Referencia técnica del broker administrativo opcional. No es un requisito para instalar ni usar notificaciones o flujos normales. Instalación del plugin sin compilador: [English](en/installation.md) · [Español](es/installation.md). Los comandos de compilación de este documento corresponden al desarrollo y las pruebas del broker.

En desarrollo. No se ha instalado un servicio privilegiado ni se han modificado
políticas del sistema. El componente es opcional, desactivado por defecto y
separado de quatrrod. El resto del plugin no requiere root.

## Contrato implementado

`internal/broker` define protocolo 1 y una política declarativa acotada a 64 KiB,
128 reglas. Cada regla liga un UID no root, una unidad `.service` exacta y una
lista explícita de operaciones: status, start, stop, restart. Vacío deniega todo.
Se rechazan comodines, rutas, unidades template sin instancia, operaciones ajenas,
reglas duplicadas, operaciones duplicadas, campos desconocidos y claves JSON
duplicadas (incluidas variantes de mayúsculas o escapes equivalentes).

La solicitud JSON (máximo 4 KiB) contiene únicamente version, id, unit y operation.
PID, UID y tiempo de inicio no son campos controlables por el solicitante: deberán
obtenerse del socket y del proceso. El plan rechaza identidad incompleta/root,
operaciones sin regla y versiones incompatibles.

Tras comprobar esa política local, el plan requiere la acción Polkit
`org.quatrro.automations.service.<operación>`, detalle unit y sujeto
`pid,start-time,uid`. No incluye allow-user-interaction ni agente interno.
[Referencia oficial pkcheck](https://polkit.pages.freedesktop.org/polkit/pkcheck.1.html).
La comprobación local **no sustituye** la autorización Polkit.

El ejecutable es fijo: `/usr/bin/systemctl`, con modo system,
no-ask-password, no-pager y unidad tras `--`. status usa show con propiedades
Id, LoadState, ActiveState y SubState. El protocolo no admite shell, ejecutable,
argumentos arbitrarios, instalación, enable/disable, daemon-reload ni archivos.
Generar el plan no ejecuta nada.

## Estado de integración

El broker, su instalador opcional y el ciclo administrativo están implementados y
probados con Polkit y systemd reales dentro de un namespace subordinado. Las pruebas
cubren autorización, denegación, revocación, recursos, instalación/actualización/
retirada y recuperación tras SIGKILL. No se ha instalado un broker en el host.

F3.6 incorpora disponibilidad, comprobación previa desde la UI, revisión de
capacidades y activación verificadas con el broker real. F3.7 incorpora contratos
de compatibilidad de versiones verificados. Los gates R1 incluyen validación
final de la instalación completa y carga. Esta cobertura no cierra todo el proyecto.

## Carga de política implementada

`LoadPolicy` parte de `/` y abre cada componente de una ruta absoluta canónica
mediante descriptores y O_NOFOLLOW. Directorios y archivo final deben pertenecer
a UID 0 y no admitir escritura de grupo/otros. El archivo final debe ser regular;
la apertura no bloqueante permite rechazar un FIFO sin esperar a un escritor.
La lectura queda acotada a 64 KiB y valida el contrato antes de devolver reglas.
Ruta de instalación: `/etc/quatrro/broker.json`.

La función no mantiene caché: una sustitución atómica se observa en la siguiente
lectura. El ejecutor la invoca antes de Polkit y vuelve a comprobar permisos
antes del efecto.
Errores se devuelven como política inválida, sin contenido del archivo.

Las pruebas ejercitan los mismos descriptores en árboles temporales con el UID
del proceso de prueba mediante un helper privado. La entrada de producción fija
UID 0 y rechaza la ruta temporal. Se probaron permisos en cada nivel, propietario
incorrecto, symlink final/intermedio, FIFO, tamaño, traversal y sustitución atómica.
La prueba aislada posterior también verifica LoadPolicy con UID 0 dentro de un
namespace de usuario, propietario ajeno, permisos de grupo y sustitución atómica.
No se crearon políticas root-owned en el host ni se ejecutó allí el broker.

## Identidad del peer implementada

`IdentifyPeer` usa SO_PEERCRED y SO_PEERPIDFD del socket Unix aceptado. El pidfd
queda abierto durante la comprobación; no hay fallback a un PID sin referencia
estable si el kernel no admite esa opción. Ese requisito deberá comprobarse al
instalar/iniciar el broker.

Abre `/proc/<pid>` una vez y lee stat/status mediante ese descriptor. Comprueba
PID, tiempo de inicio no nulo y que UID real/efectivo/saved/fs coincidan con el UID
del socket. La lectura es acotada, el parser de stat tolera espacios/paréntesis en
comm y no incluye metadatos del proceso en errores. Se comprueba vida del pidfd
antes/después de leer. `Revalidate` exige que permanezcan identidad y tiempo de
inicio; `Close` invalida la referencia. El resultado público Peer es una copia.

El ejecutor mantiene esa referencia y la revalida después de Polkit
antes del efecto. La conexión es atribuida a su creador por el kernel; no se acepta
identidad declarada en JSON. La allowlist sigue rechazando UID root como solicitante.

Pruebas reales no privilegiadas: socketpair del proceso actual y cliente hijo
con socket Unix. El segundo se termina antes de revalidar y el broker lo rechaza.
La combinación con Polkit y servicio root subordinado se verifica además en
el recorrido integral descrito más abajo.

## Ejecutor implementado

`Execute` recibe una PeerIdentity derivada del kernel, nunca una identidad JSON.
Comprueba vida/identidad, lee la política fija, construye el plan, ejecuta pkcheck
con plazo de cinco segundos, revalida el peer y vuelve a leer la política antes
del efecto. Una revocación mientras Polkit responde bloquea systemctl. La última
revalidación ocurre inmediatamente antes de lanzar la operación.

Systemctl dispone de quince segundos. Entorno mínimo fijo, directorio `/`, stdin
cerrado, stderr descartado, stdout limitado a 4096 bytes y grupo de procesos
separado; cancelar mata ese grupo. No conserva variables DBus/proxy/pager del
solicitante. Los errores originales no se devuelven.

Status devuelve solo ID, unidad, LoadState, ActiveState y SubState validados; rechaza
propiedades adicionales y una unidad Id diferente. Para mutaciones, un fallo tras
invocar el ejecutor devuelve resultado incierto. No se debe reintentar automáticamente:
terminar systemctl no deshace un trabajo que systemd ya haya aceptado.

Las pruebas del orden política/Polkit/efecto usan runners simulados. Timeout y cuota
se verifican con sleep/head/printf del usuario, sin systemctl ni pkcheck reales.
El recorrido integral complementa esas pruebas con el daemon, instalación,
Polkit y efectos systemd reales en aislamiento.

## Transporte y binario

`quatrro-broker` compila como tercer binario, pero no se instala ni inicia con el
instalador de usuario. Admite `--version` y `--prepare-policy` sin privilegios; el arranque del
servicio requiere activación systemd: UID real/efectivo
0, LISTEN_PID del proceso, un único descriptor y socket Unix en
`/run/quatrro-broker/control.sock`. Valida política antes de servir. No crea ni
reemplaza rutas recibidas por solicitudes. El instalador administrativo separado aplica las unidades sin activarlas.

El transporte permite ocho conexiones concurrentes. Cada una identifica al peer
mediante el kernel y admite un único JSON de hasta 4096 bytes. El cliente debe
cerrar su mitad de escritura para delimitar la petición; JSON concatenados se
rechazan antes de ejecutar. Lectura de tres segundos, operación de 22 y escritura
de dos. Cerrar el servicio cancela operaciones y cierra conexiones pendientes.

Respuestas de protocolo 1 contienen OK y resultado tipado o un código estático:
denied, invalid_request, outcome_unknown, status_unavailable, broker_unavailable.
No se serializa el error original. El transporte de prueba usa ejecutor simulado;
la entrada de producción conecta exclusivamente Execute. Cliente, políticas, unidades e instalador se ejercitan también en la prueba
privilegiada aislada descrita más abajo.

## Artefactos de instalación preparados

`packaging/broker/` contiene socket/service systemd, acciones Polkit y política
JSON vacía. No se instalan con el plugin de usuario. El socket es root-owned y
admite conexiones locales (0666), pero ninguna operación sin superar identidad,
allowlist y Polkit. El protocolo limita conexiones/tiempos; el socket no es permiso
para ejecutar. Las acciones status/start/stop/restart deniegan para cualquier
sujeto por defecto, sin annotations que impliquen otros privilegios.

La unidad ejecuta `/usr/local/libexec/quatrro-broker` como root, sin capacidades
adicionales ni elevación, con filesystem protegido, home oculto, dispositivos/tmp
privados, familias de socket AF_UNIX y límites de CPU/memoria/tareas/descriptores.
Conserva acceso a proc para comprobar peers y al bus de sistema. La prueba integral
usa esta unidad sin relajar restricciones y comprueba límites kernel; la validación
estática se conserva como comprobación adicional del empaquetado.

`RenderPolkitRules` genera reglas desde la misma allowlist. Resuelve nombres de
cuentas localmente para subject.user, codifica los datos como JSON y compara usuario,
unidad y action ID exactos. El broker conserva la comprobación independiente del
UID del kernel. Instalar/actualizar reglas y allowlist debe realizarse con broker
parado; nombres de cuenta cambiados requieren regeneración. Fuera del namespace
Omarchy Automations no interviene. No concede manage-units, pkexec ni autorizaciones generales.

`scripts/test-broker-packaging.py` valida XML con DTD local y unidades mediante
systemd-analyze verify; sustituye solo ExecStart por el binario local existente
en una copia temporal. No instala ni inicia nada. Las reglas JS se ejecutan en
Node con una interfaz Polkit simulada; las pruebas aisladas adicionales cubren
polkitd real y no se sustituyen por esa simulación.

## Cliente del motor preparado

`broker.Call` usa exclusivamente `/run/quatrro-broker/control.sock`, comprueba
SO_PEERCRED y exige UID 0 del servidor antes de escribir. No acepta overrides de
socket/UID en producción. La identidad puede corresponder al creador root del
socket de activación systemd; no exige que su PID sea el proceso servidor actual.

Valida la solicitud antes de conectar, limita conexión a dos segundos y respuesta
a 8192 bytes, con plazo global de 25 segundos y cancelación del contexto. Envía una
sola solicitud y cierra su mitad de escritura. Valida versión, resultado, ID y
unidad exactos, estado completed y propiedades permitidas para status.

Fallo de conexión/identidad antes del envío devuelve unavailable. Error de envío,
respuesta truncada/malformada/desconocida o cancelación después de despachar devuelve
outcome_unknown. No hay reintentos automáticos. El motor integra esos
resultados como estados terminales/inciertos de ejecución.

Pruebas usan servidores Unix del usuario y un helper privado que espera su UID;
la entrada pública fija UID 0. Se comprueba que no se envía ningún byte al UID
incorrecto, y que respuestas ajenas y desconexiones no simulan éxito.

## Integración inicial con el motor

La acción `system-service` configura unidad y operación fijas. El motor rechaza
comodines/templates de payload, operaciones no admitidas y campos de ejecución
como executable, args, timeout, script o directorios. El permiso del flujo liga
la acción completa; no sustituye la allowlist root ni la autorización Polkit.

El worker llama al cliente del broker, sin usar shell ni systemctl del motor de
usuario. Una denegación termina denied, broker no disponible termina failed y
resultado desconocido termina uncertain. No hay reintento automático de estas
acciones. Cancelación tras despachar conserva la política de incertidumbre.
Para status, el paso guarda únicamente los tres estados validados del servicio.

Simular muestra recurso, operación y requisito de broker administrativo, sin
conectar a él. El formulario permite configurar la acción, pero no instala el broker. Activación del borrador no prueba que el servicio opcional esté
instalado o autorizado; al ejecutar, el cliente falla cerrado si no está disponible.
Las pruebas de worker sustituyen la llamada al broker para verificar los estados;
no prueban efectos administrativos del sistema.


## Formulario y revisión

En Acciones, elegir system-service presenta unidad de sistema y operación
administrativa (status/start/stop/restart). El aviso aclara que guardar no instala
broker ni concede privilegios y que un resultado incierto no se reintenta.
La revisión muestra operación/unidad y los permisos de flujo, política root y
Polkit requeridos. Textos inglés/español; estética nativa R1.UI pendiente.

Importar una configuración conserva acciones administrativas pero desactiva sus
flujos, sin crear grants ni ejecuciones. La prueba QML configura y revisa una
acción de status en un flujo que no recibe eventos; no contacta al broker ni
opera servicios del sistema.

## Prueba de Polkit real en namespace aislado

`QUATRRO_POLKIT_TEST=1 go test ./internal/broker -run '^TestIsolatedPolkit$' -count=1 -v`

Requiere Linux, subuid/subgid, newuidmap/newgidmap, unshare, bwrap, dbus-daemon,
polkitd, busctl y setpriv. La prueba crea una raíz efímera y espacios privados de usuario/PID/red. Expone
/usr y archivos concretos de cuentas/configuración del host en solo lectura;
/run, /tmp y los directorios de políticas son montajes efímeros. No instala reglas ni conecta con el bus del host. Usa un bus
privado permisivo exclusivamente para comprobar el motor de autorización Polkit;
no valida la política D-Bus de producción.

Se verificó con polkitd/pkcheck reales: autorización exacta status para UID 1000,
rechazo de UID 102, rechazo de unidad ajena, rechazo de restart no concedido y revocación mediante
sustitución atómica de reglas con recarga automática. Las reglas proceden del
renderizador de producción. No hay logind dentro del aislamiento; las reglas no
requieren sesión activa. LoadPolicy también se verifica con propietario UID 0,
rechazo de propietario ajeno/escritura de grupo y reemplazo por política vacía.
El transporte completo y los efectos systemd se cubren por separado en la prueba
integral siguiente; esta prueba aislada de Polkit no los representa por sí sola.

## Arranque de systemd para la prueba integral

`python3 scripts/test-broker-systemd.py` crea un scope transitorio de usuario con
Delegate=yes, namespaces de usuario/PID/red/IPC/UTS/cgroup y raíz tmpfs. /usr/bin
y /usr/lib se exponen de solo lectura; /usr/share y /usr/local son efímeros.
/sys es privado y de solo lectura salvo el cgroup montado
dentro del namespace. La consola PTY es propia. El scope se detiene en finally.
El host no recibe unidades de sistema ni políticas; sí se crea ese scope temporal.

La comprobación ya arranca systemd como PID 1, activa un D-Bus privado por socket
y verifica start, estado running, stop y estado inactive de quatrro-fixture.service.
El manager sale y los montajes desaparecen al finalizar. Ahora incluye el broker y Polkit reales, las unidades empaquetadas sin relajar
sus restricciones y el cliente Go de producción ejecutado como UID 1000.
Su bus de pruebas permite mensajes locales; no valida las políticas D-Bus del host.

La preparación sigue la [interfaz de contenedores de systemd](https://systemd.io/CONTAINER_INTERFACE/):
scope delegado, detección por container, /sys protegido y propagación de montajes
compartida únicamente después de separar el namespace del host.

### Recorrido del broker verificado

Antes de ejecutar el script, compilar desde la raíz del repositorio:

```sh
go build -o build/quatrro-broker ./cmd/quatrro-broker
go test -c -o build/broker-integration.test ./internal/broker
python3 scripts/test-broker-systemd.py
```

El helper prepara la allowlist y genera reglas con RenderPolkitRules dentro de la
raíz efímera. Rechaza su ejecución fuera del namespace subordinado. El cliente
Call usa el socket fijo y su comprobación UID 0, sin overrides de prueba.

El recorrido comprueba status/start/restart/stop, usuario ajeno, unidad ajena,
revocación/restauración de Polkit y revocación atómica de la allowlist. Comprueba
CapEff=0, NoNewPrivs=1, memory.max=134217728 y cpu.max=25000 100000 en el proceso y
cgroup reales. El supervisor reside en un subgrupo separado para que PID 1 pueda
habilitar controladores. El broker se detiene antes de cerrar el contenedor.

El bus es permisivo exclusivamente dentro del aislamiento y no hay logind. El
arranque mínimo oculta wants de sysinit/sockets del host; no modifica las unidades
del broker. Instalación, actualización, desinstalación y recuperación del instalador se
comprueban en el mismo recorrido. La presión de recursos y la revisión integral
de la instalación completa permanecen en los gates R1.
El perfil de Polkit propio ya tiene prueba separada de recarga sin reinicio;
el recorrido integral reinicia únicamente el Polkit aislado para sincronizar cambios.

## Preparación revisable de instalación opcional

```sh
make build
python3 scripts/prepare-broker-install.py --output .dev/broker-install-preview
```

El directorio debe ser nuevo. La política predeterminada contiene cero reglas;
`--policy archivo.json` permite preparar una allowlist explícita. El binario
`quatrro-broker --prepare-policy` valida stdin (64 KiB), resuelve cuentas locales
y devuelve política canónica y reglas Polkit mediante el código de producción.
No requiere root, no usa D-Bus y no concede permisos.

El paquete contiene seis archivos y manifest.json con destinos, modos, tamaños y
SHA-256. El manifiesto marca activation=disabled y muestra las reglas para revisión.
Los hashes detectan cambios de contenido; no constituyen una firma de procedencia.
La preparación rechaza paquetes existentes y no escribe archivos del sistema.

El aplicador scripts/install-broker.py ya implementa destinos fijos, validación de
propietarios/symlinks, recibo root, actualización con broker detenido, recuperación
y retirada de archivos propios. Revalida hashes, tamaños y modos antes de escribir;
no confía en destinos arbitrarios del manifiesto. Los artefactos son código de
administración que debe proceder de una compilación revisada y confiable. Los
hashes del propio paquete no autentican su procedencia.

## Aplicación, actualización y retirada opcionales

Desde una terminal administrativa, después de revisar el paquete:

```sh
sudo python3 scripts/install-broker.py apply --bundle .dev/broker-install-preview
```

No se invoca desde el instalador normal del usuario. Requiere UID real/efectivo 0.
Todas las rutas de destino tienen componentes root y sin escritura de grupo/otros;
se abren mediante descriptores y O_NOFOLLOW. El paquete se captura en memoria con
límites y se comprueba contra destinos/modos fijos. Archivo ajeno o cambio respecto
al recibo impide continuar; no hay opción force para sobrescribirlo silenciosamente.

La instalación y cada actualización dejan socket y servicio detenidos y socket
sin habilitar. Para activarlo explícitamente tras revisar reglas:

```sh
sudo systemctl enable --now quatrro-broker.socket
```

Repetir apply con un paquete nuevo actualiza binario/unidades, conservando la
política y reglas Polkit instaladas. Solo `--replace-policy` sustituye esos permisos
por los del paquete revisado. El recibo y bloqueo de concurrencia se guardan en
/var/lib/quatrro-broker-install. Los archivos se reemplazan atómicamente y se
sincronizan junto con sus directorios.

```sh
sudo python3 scripts/install-broker.py remove
sudo python3 scripts/install-broker.py recover
```

remove retira exclusivamente los seis archivos del recibo y el recibo; conserva
archivos ajenos y directorios. Antes de mutar se persiste pending.json con el
estado anterior. Un fallo intenta restaurarlo dejando el broker detenido; si
falla la restauración o se interrumpe el proceso, recover restaura el estado
anterior con el mismo control de rutas. Una transacción pendiente impide apply y
remove. recover nunca reactiva el socket. No ejecutar recover salvo que exista una
transacción pendiente: sin ella rechaza la operación.

Prueba real en el systemd aislado: instalación deshabilitada, activación explícita,
llamada del cliente, actualización conservando reglas, reemplazo explícito por
política vacía y retirada conservando un archivo ajeno. Las pruebas de filesystem
simulan fallos de daemon-reload y restauración para verificar rollback y recuperación.
La interrupción real también se verifica mediante test-broker-install-crash.py,
invocado solamente dentro del contenedor: seis cortes de actualización/retirada
y dos de primera instalación. Un perfil de Python detiene el instalador sin
modificarlo al volver de operaciones duraderas; el padre envía SIGKILL. Después
se ejecuta recover por su CLI pública, se comparan bytes/modos/propietarios, se
comprueba socket deshabilitado y se repite una llamada del cliente real. También
se rechaza un segundo instalador mientras el proceso detenido conserva el lock.
Es una prueba de caída de proceso; no simula un corte eléctrico del almacenamiento.

## Consulta de autorización sin efectos

Request admite el campo opcional check_only=true. Valida la misma operación,
identidad, allowlist y Polkit; vuelve a cargar la política y revalidar al peer,
pero devuelve state=authorized antes de invocar systemctl. Tampoco ejecuta show
para una consulta status. No concede ni conserva autorización para otra solicitud.
El cliente exige authorized para checks y completed para ejecuciones, rechazando
respuestas que confundan ambos recorridos o incluyan propiedades de servicio en
un check. Sin check_only se conserva el contrato anterior.

Es una extensión del protocolo 1: un broker anterior con su parser estricto
rechaza el campo desconocido antes de ejecutar y administration.check informa
unsupported. No hay fallback a una operación real para probar disponibilidad.
La UI y la activación consumen esta API como se describe más abajo; su integración
positiva con el broker real se verifica en el entorno aislado.

La prueba integral comprueba start sobre servicio inactivo (sigue inactivo),
restart/stop sobre servicio activo (conserva PID y estado), unidad ajena y rechazo
tras revocación de Polkit. La API del motor devuelve estados redactados sin crear
grants o ejecuciones. Los permisos se comprueban nuevamente al ejecutar realmente.

## Integración de editor y activación

El editor de acciones system-service permite comprobar permisos sobre la unidad y
operación actuales. Muestra sin comprobar/comprobando/autorizado/denegado/no
disponible/no compatible, en inglés o español. Al cambiar el recurso o cerrarlo,
invalida el resultado; las respuestas pendientes se asocian a la revisión local
y no pueden autorizar visualmente un recurso modificado.

Activar una configuración vuelve a consultar las acciones administrativas usadas
por flujos habilitados. Comparte un plazo de ocho segundos y evita repetir la
misma unidad/operación. Una denegación, ausencia o respuesta no verificable impide
activar y guardar grants. Los flujos deshabilitados y acciones sin uso no requieren
el broker. Guardar borradores sigue siendo posible; ejecutar revalida otra vez.

Prueba QML: componente ausente visible, traducción español, respuesta obsoleta
rechazada, activación administrativa impedida y activación del resto tras deshabilitar
ese flujo. Captura de desarrollo .dev/ui-admin-action.png; aún no es QA estética nativa.
La comprobación positiva completa también pasa con broker real:

```sh
python3 scripts/test-broker-systemd.py --ui
```

La opción añade Development.qml y el motor al contenedor. Los ejecuta con UID 1000,
entorno explícito y perfil temporal, sin conexión a la sesión del host. Quickshell
renderiza offscreen. La UI consulta autorización/denegación reales, cambia idioma,
revisa una capacidad y activa un flujo sin efectos. Después un evento enviado por
CLI inicia la unidad mediante el worker/broker y queda completed. El servicio se
detiene antes de continuar el resto de pruebas. No se usan respuestas simuladas
del broker en este recorrido.

Capturas: .dev/broker-admin-en.png y .dev/broker-admin-es.png. Se inspeccionaron los
textos y controles; esto valida la integración funcional bilingüe, no la estética
nativa final R1.UI. El test exige el reporte de éxito y ambas imágenes PNG.
