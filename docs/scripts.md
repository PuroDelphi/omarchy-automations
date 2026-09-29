# Scripts locales aprobados

F3.1 está implementada y probada en desarrollo. Incluye preparación, validación,
acciones con revisión exacta y ejecución aislada de Bash/Python. El panel permite
preparar y revisar scripts, vincularlos a acciones y asignar sus parámetros.
El motor y su formulario admiten directorios autorizados. Preparar o guardar una
revisión no ejecuta código ni concede permisos. La entrega completa sigue sujeta
a los controles finales del roadmap.

En el panel de desarrollo, abrir **Scripts → New**, introducir identificador,
ruta absoluta e intérprete, y añadir los parámetros en el orden que espera el
programa. Para cadenas se configuran longitud máxima, patrón y opciones; para
enteros, mínimo y máximo. **Prepare revision** copia el archivo y muestra su
código y hash. **Save** conserva únicamente la revisión preparada, sin la ruta
de origen. Un cambio en los campos invalida la preparación y exige repetirla.
El editor de una revisión existente muestra el contenido conservado; para
reemplazarlo hay que indicar nuevamente la ruta del archivo local.

Para ejecutarlo, abrir **Actions → New**, elegir `script` y el script registrado.
El formulario muestra el código, la revisión disponible y la seleccionada.
**Use this revision and reset parameters** selecciona explícitamente esa revisión
y reinicia los parámetros; después elegir **Literal value** o **Event field**
para cada uno. Las cadenas y enteros se introducen en un campo; los booleanos
literales usan una casilla. El formulario muestra los límites del contrato.
Las rutas de evento se escriben como `data.message`, `source` o `type`.

Guardar la acción exige que su revisión seleccionada siga siendo la disponible.
Cambiar de script vacía la selección de revisión y sus valores anteriores. Una
revisión existente no se sustituye silenciosamente al abrir el editor. Añadir la
acción a un flujo, guardar el borrador y revisar capacidades completa el proceso:
la aprobación muestra identificador, hash y tiempo máximo del script. El motor
valida valores y contrato al guardar y los campos del evento antes de ejecutar.

`scripts.prepare` recibe `id`, `path` absoluto, `interpreter` (`bash` o `python3`)
y `parameters`. Lee un archivo regular UTF-8 de hasta 32 KiB sin NUL y rechaza
enlaces simbólicos en cualquier componente. Devuelve `id`, `interpreter`, `code`,
`parameters` y `revision`. La ruta original no forma parte de la revisión.

La revisión SHA256 liga identificador, intérprete, código y contrato de parámetros.
Cambiar alguno exige calcular y revisar una nueva revisión. El código se guarda
como contenido de configuración; exportarlo incluye ese código, por lo que no
debe contener credenciales incrustadas. El límite global del protocolo continúa
siendo 1 MiB; se admiten hasta ocho scripts por configuración.

Hasta 16 parámetros obligatorios, ordenados por su contrato:

- `string`: longitud máxima explícita de 1 a 4096 bytes, patrón RE2 opcional
  aplicado al valor completo y lista opcional de hasta 32 opciones.
- `integer`: mínimo y máximo explícitos dentro del rango entero exacto de JSON.
  No convierte texto ni números fraccionarios.
- `boolean`: requiere booleano JSON, sin convertir texto como `"true"`.

No se admiten valores adicionales, ausentes ni NUL. La conversión produce una
lista de argumentos; los valores no se interpolan en código shell. Esta propiedad
del contrato no sustituye la revisión del script: un script puede interpretar
sus argumentos como código por decisión de su autor.

Una acción de tipo `script` referencia `script` (identificador) y
`script_revision` (hash exacto), con `timeout_seconds` obligatorio de 1 a 300 segundos.
Cada parámetro recibe exactamente un valor literal en `script_values` o una
ruta del evento en `script_bindings`. Las rutas admitidas son `source`, `type`
y campos bajo `data`; se validan nuevamente los valores resueltos antes de
ejecutar. Por ejemplo, para un contrato con una cadena llamada `message`:

```json
{
  "id": "run-local-script",
  "kind": "script",
  "script": "my-script",
  "script_revision": "<revision devuelta por scripts.prepare>",
  "script_bindings": {"message": "data.message"},
  "timeout_seconds": 30
}
```

El marcador del ejemplo debe sustituirse por el hash real. Añadir el script a
`scripts` y la acción a `actions` en el borrador permite validarlos; el flujo
requiere revisión y concesión explícita de su capacidad al activar. Cambiar
código, intérprete o contrato obliga a actualizar la referencia de la acción y
su permiso. La concesión para la revisión nueva no autoriza trabajos pendientes
de la anterior. Importar configuración no activa flujos ni concede permisos.

La simulación muestra `script`, `script_revision` y `arguments` resueltos en
orden de contrato. Usa la misma validación que la ejecución y rechaza valores
ausentes o incompatibles, sin lanzar procesos ni crear trabajos. Estos valores
pueden contener datos del evento de prueba: no pegar credenciales en ellos.

El ejecutor consume el código de la configuración fijada al admitir el evento,
sin reabrir el archivo original. Lo transmite por entrada estándar a bubblewrap,
que lo monta en solo lectura en `/quatrro/script`; no crea un archivo de código
en el host. Bash arranca con `--noprofile --norc`; Python con `-I -S`. Los
parámetros siguen al nombre del archivo como argumentos separados. systemd se
invoca con `--expand-environment=no` para preservar también valores con `$`.

La unidad transitoria aplica los mismos límites que el ejecutor de comandos:
256 MiB de memoria, sin swap, cuota CPU del 50 %, hasta 32 tareas, timeout y
terminación del grupo de procesos. El sandbox tiene `/usr` de solo lectura,
directorios temporales privados, entorno mínimo y red aislada, sin acceso al
directorio personal por defecto. La acción puede añadir árboles autorizados
mediante `directories` y elegir `working_directory`; véase
[el contrato de directorios](directory-access.md). No se puede habilitar red IP
para scripts. La activación
comprueba tanto el sandbox como una prueba de entrada de código con cada
intérprete requerido; falla si estas garantías no están disponibles.

Un fallo de script no se reintenta automáticamente. Cancelación después del
despacho o caída con resultado sin persistir dejan una ejecución incierta para
inspección, según el contrato general del motor. La salida del proceso se
descarta; el historial conserva estado y errores acotados, no stdout/stderr.

Las pruebas host ejecutan ambos intérpretes, modifican el archivo original tras
activar, comprueban argumentos con metacaracteres y rechazo de escritura sobre
el código montado. Consultar `docs/validation.md` para resultados y pendientes.
