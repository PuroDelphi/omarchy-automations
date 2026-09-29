# Instalación en desarrollo

Referencia avanzada del instalador para desarrollo y recuperación. Para instalar, actualizar o retirar el plugin normalmente, usa la guía de instalación: [English](en/installation.md) · [Español](es/installation.md). No necesitas Go para la instalación precompilada.

La instalación permanente y los ciclos de instalación, actualización y retirada
se han comprobado para la versión de desarrollo; las evidencias están en el
roadmap. Quedan pendientes las validaciones externas aplazadas y el cierre de
versión estable. No se anuncia todavía como entrega 1.0.

Para trabajar desde fuentes, sigue primero la [guía de desarrollo](es/development.md).
El instalador de bajo nivel descrito aquí requiere binarios ya preparados; no
descarga paquetes ni solicita privilegios administrativos.

## Prueba temporal

```bash
python scripts/install.py --staging-root /ruta/absoluta/temporal
python scripts/install.py --staging-root /ruta/absoluta/temporal --uninstall
```

El directorio representa un HOME independiente y nunca activa servicios. Las
pruebas reproducibles están en `python -m unittest discover -s tests -v`.

## Rutas y propiedad

Sin staging usa HOME, XDG_CONFIG_HOME y XDG_STATE_HOME. Copia los binarios en
`~/.local/bin`, el plugin en `omarchy/plugins/quatrro.automations` bajo la
configuración del usuario y la unidad en `systemd/user/quatrrod.service`.
La UI instalada referencia la ruta absoluta de la CLI, sin depender de PATH.
No instala archivos de desarrollo, pruebas ni credenciales.

Un recibo privado en `quatrro-install/receipt.json`, bajo XDG_STATE_HOME,
registra hashes y rutas de los archivos propios. Rechaza enlaces simbólicos,
archivos ajenos y cambios locales sobre archivos registrados. No sobrescribe
shell.json durante la copia.

Al actualizar guarda los archivos previos y un índice en `backup-*` junto al
recibo. La copia incluye binarios/UI/unidad y, si existe, una instantánea SQLite en
`state.sqlite`, tomada mediante la API de backup para incluir páginas WAL
confirmadas. `state.json` registra versión del esquema y SHA256; la copia se
verifica con integrity_check y tiene permisos 0600. Puede contener eventos y
configuración sensibles: consérvala como dato privado. No copia valores del
almacén de credenciales ni Secret Service.

El instalador mantiene el mismo flock de estado que el motor durante la copia
y sustitución de archivos. Exige detenerlo, incluso si se ejecuta manualmente
fuera de systemd, y libera el bloqueo antes de activarlo. La restauración de datos se describe a continuación; no reemplaces manualmente
un archivo SQLite mientras el motor está activo.

`--activate` solicita validar el manifiesto, recargar systemd, habilitar el
motor y habilitar el widget mediante `omarchy plugin enable`. Si la activación
falla, los archivos ya instalados permanecen disponibles para diagnóstico;
no se afirma una reversión de los efectos de systemd/Omarchy.

## Retirada

`--uninstall` usa el recibo, verifica hashes y restringe rutas al paquete. En
una instalación real solicita detener/deshabilitar el servicio y deshabilitar
el plugin antes de retirar archivos. Conserva SQLite, credenciales, backups y
archivos no registrados. Solo borra directorios del plugin si están vacíos.
Sin recibo o ante modificaciones locales, se detiene sin borrarlas.

## Restaurar datos

Con el motor detenido, utiliza la ruta `backup` que imprimió la actualización:

```bash
python scripts/install.py --restore-data /ruta/absoluta/quatrro-install/backup-IDENTIFICADOR
```

Para una prueba temporal, añade el mismo `--staging-root` utilizado al instalar.
Solo admite directorios de respaldo de esa instalación y archivos privados del
usuario. Comprueba checksum, integridad y versión de esquema; rechaza versiones
futuras, enlaces simbólicos y esquemas inesperados con triggers o vistas.

Antes de sustituir la base guarda otra copia `backup-before-restore-*` del estado
actual. Usa la API SQLite para reemplazarla, conservando la coherencia con WAL.
El motor debe permanecer detenido y se mantiene su bloqueo durante la operación.
No arranca servicios automáticamente.

La copia restaurada queda pausada, rechaza nuevas entradas y no conserva permisos
activos. Trabajos pendientes/en curso pasan a inciertos; no se repiten al arrancar.
Revisa configuración, credenciales y resultados, activa una revisión con permisos
explícitos y después reanuda. El historial completado se conserva. La operación
queda registrada en auditoría.

Esta opción restaura datos, no binarios ni valores de credenciales. Las referencias
a secretos pueden requerir recreación si el valor ya no existe. Las copias pueden
contener información sensible de eventos y deben permanecer privadas.

`python scripts/test-install-runtime.py` verifica instalación, actualización,
respaldo, restauración y retirada usando los binarios reales en un HOME temporal;
necesita permiso para crear sockets Unix locales. No instala servicios permanentes.

## Revisiones de interfaz

La UI instalada usa rutas `ui-revisions/<SHA256>/` y el manifiesto apunta a esa
revisión. Evita que una recarga del shell reutilice componentes QML anteriores.
Las revisiones previas siguen registradas y se conservan mientras puedan estar
cargadas; la desinstalación retira sus archivos verificados. No se elimina código
ajeno ni se reinicia el shell para actualizar el plugin.

## Comprobación de compatibilidad antes de instalar

El instalador lee los requisitos de `qml/Compatibility.js`, usados también por
Backend.qml. Captura los bytes de quatrrod/quatrroctl y ejecuta copias temporales
privadas con --version. Exige componente correcto, protocolo/contrato local y
contrato UI exactos, además de igual versión de release para ambos binarios.
Las mismas capturas verificadas son las que se escriben en los destinos.

Una versión ausente/incorrecta o un binario sin descriptor verificable rechaza la
instalación antes de reemplazar archivos, crear backups de actualización o cambiar
el recibo. El bloqueo operativo puede crear sus directorios/lock habituales.
El recibo y el resultado JSON incluyen los contratos y versión comprobados.
No es una firma de procedencia: se instala código de una compilación confiable.

Si falla esta comprobación, recompila el proyecto completo con make build y usa
la UI de la misma revisión. Para una instalación ya mezclada, detén quatrrod,
reinstala conjuntamente motor/CLI/plugin con el instalador y vuelve a iniciarlo;
recarga el plugin mediante rescanPlugins. No borres perfiles ni credenciales para
resolver un error de versión. Los backups y las comprobaciones de archivos ajenos
siguen aplicándose al actualizar.
