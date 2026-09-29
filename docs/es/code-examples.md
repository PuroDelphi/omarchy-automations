# Scripts y adaptadores: ejemplos completos

[English](../en/code-examples.md) · [Guía de usuario](user-guide.md) · [Otros ejemplos](use-cases.md)

Estos ejemplos requieren motor, systemd de usuario operativo, Bubblewrap y Python 3.
Importar reemplaza el borrador: exporta lo que quieras conservar. Cada archivo es
una configuración completa con flujo deshabilitado. Contiene código y revisión
validada; importar no lo ejecuta ni concede permisos.

## Validar parámetros tipados con un script

Importa [typed-script.json](../../examples/use-cases/typed-script.json). Su fuente
legible es [check-project.py](../../examples/scripts/check-project.py). Valida
argumentos y termina; no crea archivos, contacta servicios ni muestra notificaciones.
Una ejecución exitosa aparece en Historial.

El contrato tiene tres parámetros obligatorios en este orden argv:

| Parámetro | Contrato | Valor de la acción |
|---|---|---|
| `project` | String, máximo 64 bytes, patrón completo `[a-z][a-z0-9-]*` | Campo `data.project` |
| `count` | Integer, rango inclusivo 1–10 | Campo `data.count` |
| `enabled` | Boolean | Literal `true` |

1. Abre Scripts e inspecciona `check-project`, código y revisión preparada.
2. Abre Acciones e inspecciona `check`. Su revisión debe coincidir; el timeout
   es de 5 segundos y no tiene directorios del host autorizados.
3. Habilita `validate-project` en el borrador y guarda.
4. Abre Simular / probar con origen `local:project`, tipo `test` y:

   ```json
   {"project":"demo-project","count":3}
   ```

   Simula sin efectos. Debe resolver argumentos `demo-project`, `3`, `true`, en
   ese orden. La simulación no ejecuta el script.
5. Repite con `{"project":"release-2","count":10}`: debe resolver `release-2`,
   `10`, `true`. Prueba después `{"project":"demo-project","count":11}`:
   debe fallar el contrato sin ejecución. Texto `"3"` no es el entero `3`.
6. Revisa y activa la capacidad del script. Ejecuta prueba real con cualquiera
   de los datos válidos y confirma. Debe completarse en Historial. No se espera
   archivo ni notificación; stdout del proceso no se conserva en Historial.

Para recrearlo en formularios sin importar, crea script `check-project`, con la
**ruta local absoluta** al Python suministrado, intérprete `python3` y los tres
parámetros anteriores. Elige Preparar revisión, inspecciona y Guarda. Crea acción
`check` de tipo `script`, selecciona el script, pulsa Usar esta revisión y reiniciar
parámetros y configura los dos campos del evento y literal booleano. Añádela al
flujo habilitado `local:project`, guarda, simula, revisa y activa.

Si editas la fuente, prepara expresamente una revisión nueva. Selecciónala en la
acción y vuelve a introducir sus valores antes de guardar/revisar. Editar el
archivo no cambia por sí solo el código de una revisión activa.

## Normalizar un evento de compilación y después notificar

Importa [adapter-notification.json](../../examples/use-cases/adapter-notification.json).
Su fuente es [status-normalizer.py](../../examples/adapters/status-normalizer.py),
con este [manifiesto](../../examples/adapters/status-normalizer.json). Acepta una
solicitud JSON versión 1, transforma estado en severidad y devuelve datos para el
siguiente paso. No realiza por sí mismo efectos sobre el sistema operativo.

1. Inspecciona `status-normalizer`: Python 3, entrada máxima 262144 bytes, salida
   4096 bytes y timeout 5 segundos. La entrada incluye el sobre del protocolo.
2. Inspecciona `normalize`, que selecciona esa revisión exacta, y la notificación
   `notice`, cuyo cuerpo es
   `{{data.adapter.severity}}: {{data.adapter.message}}`.
3. Habilita `normalized-notice`, origen `local:build`, con pasos `normalize` y
   después `notice`. Guarda el borrador.
4. Simula tipo `build` con:

   ```json
   {"status":"failed","message":"Build failed"}
   ```

   Debe coincidir el flujo, indicar resultado del adaptador no disponible y
   notificación dependiente de ese resultado. Simular deliberadamente no calcula
   `error: Build failed`. Proporcionar tu propio `data.adapter` no evita esperar
   al resultado real.
5. Revisa capacidades de adaptador y notificación, activa y confirma prueba real.
   Debe mostrar `error: Build failed` y completarse en Historial.
6. Prueba `{"status":"passed","message":"Build passed"}`: debe mostrar
   `info: Build passed`. Las notificaciones dependen del servicio del escritorio;
   consulta Historial si está activo no molestar.

Para construirlo en formularios, crea adaptador desde ruta absoluta de fuente y
límites anteriores, Prepara revisión, inspecciona y Guarda. Crea acción de adaptador
y selecciona expresamente revisión disponible. Crea notificación y flujo ordenado
indicados. Un adaptador no ofrece montajes del host ni opciones de red. Guarda su
resultado en la vista privada del evento de esa ejecución bajo `data.adapter`;
no reescribe el evento original de otros flujos.

Timeout, respuesta malformada o exceso de cuota hacen fallar la transformación e
impiden usar su resultado. Los fallos de scripts/adaptadores no se reintentan
automáticamente. Inspecciona el fallo, cambia código/contrato si corresponde y
revisa nueva revisión antes de enviar otro evento.

## Qué se verificó

`TestDocumentedCodeExamples` carga configuraciones distribuidas, verifica hashes
con el motor, compara código incorporado con fuente legible, exige flujos inicialmente
deshabilitados y prueba simulación y rechazo de contador fuera de rango. Simular
no crea ejecuciones, permisos ni entradas de outbox HTTP.

`TestHostDocumentedCodeExamples` ejecuta ambos casos válidos del script y ambos del
adaptador dentro del aislamiento real systemd/Bubblewrap y comprueba texto resuelto
de notificación. **No** muestra notificaciones ni acredita todo el recorrido de
importar/activar en UI; esa es una comprobación separada.
