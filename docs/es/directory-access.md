# Directorios autorizados

[English](../en/directory-access.md) · [Documentación](README.md)

El motor de desarrollo implementa montajes por identidad y descriptor para
acciones `command` y `script`, con capacidades, preflight y formulario.
Disponible en la instalación de desarrollo verificada en R1.1. Consulta las
[recetas en inglés](../en/action-recipes.md) o [español](../es/action-recipes.md) para uso.
Las comprobaciones funcionales del ejecutor F1C.3 están completadas en desarrollo.

En el editor de una acción de comando o script, **Approved directories** permite
introducir la ruta del host, un destino `/work/id` y el modo de acceso. El valor
predeterminado es **Read only**. **Prepare and add directory** verifica y añade
su identidad al borrador; un destino repetido reemplaza su entrada. Una ruta
inválida muestra error y conserva las entradas anteriores. **Remove** quita el
montaje y devuelve el directorio de trabajo a `/tmp` si lo estaba usando.
**Working directory** ofrece `/tmp` y los destinos preparados.

Guardar la acción conserva los montajes, también al editarla. No se puede guardar
mientras está pendiente una preparación. La revisión de capacidades muestra
origen, destino, identidad, acceso, directorio de trabajo y alcance del árbol;
añadir al borrador no concede esos permisos. Los textos están disponibles en
inglés y español.

La API local `directories.prepare` recibe `source`, `target` y `access`; devuelve
el objeto descrito abajo. No guarda configuración, concede permisos ni crea
directorios. Añadir los objetos preparados al campo `directories` de una acción
y establecer opcionalmente `working_directory` permite validar el borrador.
Guardar y activar sigue exigiendo revisar y conceder las capacidades del flujo.
La simulación incluye los montajes y el directorio de trabajo previstos.

El hash de capacidad incluye todos esos campos: cambiar acceso, identidad,
origen, destino o directorio de trabajo exige un permiso nuevo. Un trabajo
pendiente con otro hash queda denegado. El preflight verifica identidad y
ejecuta una prueba inocua de los montajes en el sandbox; para scripts prueba
también su intérprete y el transporte de código por memfd. Al ejecutar se vuelve
a abrir y verificar la identidad en el auxiliar, antes de montar por descriptor.

Cada permiso de directorio describe:

- `source`: ruta absoluta canónica del host, sin enlaces simbólicos.
- `target`: punto de montaje `/work/identificador` dentro del sandbox.
- `access`: `ro` (solo lectura) o `rw` (lectura/escritura), obligatorio.
- `device` e `inode`: identidad observada al preparar el directorio, como
  cadenas decimales para no perder precisión al pasar por JSON/QML.

Se admiten hasta ocho montajes con destinos distintos; el directorio de trabajo
es `/tmp` o uno de los destinos aprobados. No se admiten la raíz del host ni
interfaces bajo `/proc`, `/sys`, `/dev` y `/run`. No se pueden sustituir `/usr`,
`/quatrro/script` u otros puntos del sandbox mediante destinos personalizados.

La apertura usa `openat2` con `O_PATH | O_DIRECTORY` y
`RESOLVE_NO_SYMLINKS`. La identidad se comprueba con `fstat` sobre ese mismo
descriptor. Se pasa a Bubblewrap mediante `--ro-bind-fd` o `--bind-fd`; no se
vuelve a abrir la ruta entre validación y montaje. Estos mecanismos están
documentados en [openat2](https://man7.org/linux/man-pages/man2/openat2.2.html)
y el [código de Bubblewrap](https://github.com/containers/bubblewrap/blob/main/bubblewrap.c).

Una ruta que ahora apunta a otro inode/dispositivo se rechaza hasta preparar y
aprobar de nuevo. Si cambia después de abrir el descriptor, el montaje sigue
refiriéndose al objeto abierto. Esto no congela el contenido del directorio:
los archivos siguen siendo datos vivos y `rw` autoriza cambios y borrados
según los permisos normales del usuario. No es una copia de seguridad ni una
garantía de identidad criptográfica frente al control de la cuenta o del sistema
de archivos. Una restauración o cambio de sistema de archivos puede requerir
preparar de nuevo las identidades.

El permiso abarca el árbol montado, incluidos submontajes y recursos especiales
que contenga. En particular, un socket Unix dentro de un árbol autorizado puede
comunicar con un servicio del host: aislar la red IP no convierte ese árbol en
un conjunto exclusivo de archivos ordinarios. La UI y revisión de capacidades
explican ese alcance antes de permitir concederlo. Los perfiles sin montajes
conservan su aislamiento anterior.

## Transporte interno

`quatrrod --sandbox-runner` es una rama privada del ejecutable que recibe un
único documento JSON de hasta 1 MiB por stdin y ejecuta Bubblewrap. No abre el
perfil, socket ni base de datos del motor. No es una API de autorización ni un
servicio privilegiado: el motor debe comprobar las capacidades antes de
invocarla dentro de la unidad transitoria con límites.

El protocolo interno versión 1 contiene ejecutable, argumentos, código opcional,
montajes y directorio de trabajo. Rechaza campos desconocidos, versiones
incompatibles, datos adicionales y límites excesivos. Código de script se copia
a un memfd sellado, que Bubblewrap monta en solo lectura; no se crea un archivo
de código en el host. Los descriptores se cierran al terminar el auxiliar y
Bubblewrap los consume antes de ejecutar el programa.

## Evidencia disponible

Las pruebas host del paquete `internal/sandbox` cambian la ruta después de
preparar el comando y antes de arrancar Bubblewrap. Comprueban que lee el árbol
original, que `ro` rechaza escritura y que `rw` escribe solo en el árbol original.
Una segunda ejecución con la identidad antigua rechaza la ruta sustituida.
También se prueban rutas/destinos inválidos, symlinks y límites del transporte.

`scripts/test-sandbox-runner.py` ejecuta el binario compilado dentro de una unidad
transitoria de usuario con límites: Python recibe argumentos literales, lee y
escribe en un directorio temporal aprobado y no puede modificar su código.
El recorrido `scripts/test-ui-flow.py` prepara un montaje desde el formulario,
lo conserva al editar, revisa su capacidad y ejecuta un script que escribe en
ese directorio antes de completar la salida HTTPS.
