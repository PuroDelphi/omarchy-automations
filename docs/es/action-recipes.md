# Recetas de acciones

[English](../en/action-recipes.md) · [Guía de usuario](user-guide.md) · [Opciones](options.md#group-actions)

Crea acciones en el borrador, referéncialas desde un flujo, simula y después revisa
y activa sus permisos exactos. Guardar una acción no la ejecuta. La tabla presenta
configuraciones alternativas; usa tus propios recursos registrados.

| Tipo | Ejemplo A | Ejemplo B | Resultado esperado |
|---|---|---|---|
| `notify` | Título `Build`, cuerpo `{{data.message}}` | Título `Disco`, cuerpo `Libre: {{data.value}}%` | Notificación de escritorio; falla si faltan campos de plantilla. |
| `http` | Destino JSON, cuerpo `{"state":"{{type}}"}` | Destino raw, cuerpo `Build: {{data.message}}` | Entrega HTTPS registrada en Historial/outbox. |
| `service` | `backup.service`, `start` | `worker.service`, `stop` | Inicia o detiene esa unidad exacta de usuario. |
| `command` | Fijo `/usr/bin/true`, sin argumentos, timeout 5 | Fijo `/usr/bin/test`, argumentos `-d` y `/tmp`, timeout 5 | Proceso aislado exitoso, sin conservar stdout. |
| `omarchy` | `theme.current` | `nightlight.status` | Comprueba éxito del comando; su salida no se devuelve como datos del evento. |
| `script` | `check-project`, project `demo-project`, count 3 | Misma revisión, project `release-2`, count 10 | Argumentos tipados validados y entregados al código aprobado. |
| `adapter` | Normalizador con `{"status":"failed","message":"Build failed"}` | `{"status":"passed","message":"Build passed"}` | Resultado privado `data.adapter` disponible para pasos posteriores. |
| `system-service` | Unidad exacta autorizada, `status` | Unidad exacta autorizada, `restart` | Exige broker separado, política, Polkit y permiso del flujo. |

Scripts y adaptadores exigen seleccionar explícitamente una revisión preparada.
Sigue los [ejemplos completos de código](code-examples.md) para contratos, activación
y resultados esperados. Para salida de formulario usa un destino `form` con cuerpo
JSON `{"project":"{{data.project}}"}`; el motor codifica los campos resueltos.
No pegues texto URL-encoded en una acción de formulario. Consulta
[recuperación de entregas](recovery-example.md).

## Operaciones de servicios de usuario

`status` comprueba si el servicio está activo; inactivo significa acción fallida.
`start` y `restart` también exigen estado activo tras el comando. Esto importa para
servicios oneshot breves que terminan correctamente y quedan inactivos.
`stop` comprueba éxito del comando sin exigir que la unidad siga activa.
Estas acciones no habilitan/deshabilitan unidades al iniciar sesión. El
[ejercicio de servicios](service-example.md) usa una unidad temporal y verifica
estado, reinicio y revocación. Cambiar unidad u operación exige revisar permisos.

## Perfiles de comando y directorios

Prepara un directorio existente y dedicado del host en **Directorios autorizados**.
Por ejemplo, crea tú mismo `/home/USER/automation-demo`, introduce su ruta absoluta
real como origen, destino `/work/data` y elige **Solo lectura**. Selecciona
**Preparar y añadir directorio**. Sustituye USER; no puedes reutilizar una identidad
de inode/dispositivo de ejemplo. Preparar identifica el directorio, pero no concede
acceso ni lo crea.

| Perfil | Ejemplo A | Ejemplo B | Acceso y resultado |
|---|---|---|---|
| `file-exists` | `/work/data/report.json` | `/work/data/ready.txt` | Basta `ro`; tiene éxito solo si el hijo directo es archivo regular. |
| `make-directory` | `/work/data/new-backup` | `/work/data/export` | Exige `rw`; crea un hijo directo con modo 0700 y falla si existe. |

Usa cada ruta como **Ruta del perfil**, timeout 5 y conserva el montaje preparado.
Los perfiles derivan ejecutable/argumentos; no admiten reemplazarlos.
Para un comando fijo que lea ese directorio, elige `/usr/bin/test`, con un argumento
por línea: `-f`, después `/work/data/report.json`. Otro ejemplo usa `-d`, después
`/work/data`. Son argumentos literales, sin expansión de shell.

El directorio de trabajo puede ser `/tmp` o un destino autorizado como `/work/data`.
`/tmp` es privado y temporal. Un montaje escribible del host es real: sus cambios
persisten y cancelar no los deshace. El permiso abarca el árbol montado, incluidos
archivos especiales y submontajes. Un socket Unix allí puede comunicar con un
servicio del host aunque la red IP esté aislada. Prefiere un directorio dedicado.
Cambiar identidad, acceso, destino o directorio de trabajo exige nueva revisión.
Simular muestra montajes y argumentos, sin comprobar existencia ni crear directorios.
Una prueba real sí realiza la operación del sistema de archivos.

## Efectos en el escritorio

`nightlight.toggle` cambia el estado actual de luz nocturna. `system.lock` bloquea
la sesión. A diferencia de las dos consultas anteriores, producen efectos visibles.
Simula primero; activa y envía una prueba real solo cuando quieras ese efecto.
Ninguna operación ofrece stdout a pasos posteriores. En particular, `theme.current`
no rellena `data.theme`; usa el evento del hook de tema descrito en
[integración del sistema](system-integration.md) para obtener esa información.
