# Adaptadores externos — protocolo de transformación

[English](../en/external-adapters.md) · [Documentación](README.md)

Existen contrato, transporte aislado y preparación/registro local de revisiones.
La ejecución integra referencias exactas, capacidades, preflight, worker y
persistencia privada del resultado. El panel permite preparar y revisar el adaptador,
y seleccionar explícitamente su revisión en una acción. La entrega completa sigue
sujeta a los controles finales del roadmap.

El contrato inicial es de transformación de datos de evento en un proceso Python
separado, sin acceso a red ni directorios del usuario. El motor conserva la
responsabilidad de efectos autorizados mediante los pasos del flujo. El adaptador
no es una vía para solicitar comandos, instalar código, modificar configuración
ni obtener secretos. El registro toma código de un archivo local seleccionado
explícitamente y fija su revisión; nunca del evento recibido.

## Preparación y registro local

`adapters.prepare` recibe `path` absoluto y `manifest` con los campos descritos
abajo. Lee un archivo regular UTF-8 de hasta 32 KiB, sin NUL y sin enlaces
simbólicos en ningún componente. Devuelve los campos del manifiesto junto con
`code` y `revision`, sin conservar la ruta original. No guarda configuración,
concede permisos ni lanza procesos.

La revisión SHA256 liga código y manifiesto completos, incluidos protocolo,
capacidades, cuotas y timeout. El recurso puede guardarse en `adapters` dentro
del borrador; se admiten hasta ocho, incluidos en el límite global de 500
recursos. Validar una revisión recomputa su hash. Modificar la fuente no cambia
el código copiado; modificar código o manifiesto requiere una revisión nueva.
La configuración exportada incluye el código: no incrustar credenciales en él.

La preparación rechaza protocolos y capacidades no soportados antes de leer la
fuente. Para usar la revisión en un flujo se registra una acción:

```json
{"id":"normalize","kind":"adapter","adapter":"status-normalizer","adapter_revision":"<hash devuelto por adapters.prepare>"}
```

El marcador debe sustituirse por el hash real. La acción no puede elegir otro
ejecutable, argumentos, timeout ni directorios: rigen el código/manifiesto
aprobados y el perfil aislado sin montajes del usuario. Cambiar la revisión cambia
su capacidad. Activar exige concederla explícitamente; revocar o sustituir el
permiso impide ejecutar trabajos pendientes con la referencia anterior.

El preflight comprueba límites del sandbox y un intercambio JSON inocuo con código
propio del motor, usando el runtime y cuotas del manifiesto. No ejecuta el código
del adaptador registrado durante esa comprobación. El worker sí ejecuta la copia
fijada en la revisión al admitir el evento, valida la respuesta y confirma su
resultado junto con el avance del paso. Fallo, timeout, exceso de cuota o respuesta
inválida detienen la ejecución sin pasar datos a pasos posteriores.

La simulación no ejecuta el adaptador. Muestra su referencia con
`result_available: false` y marca pasos posteriores con
`requires_adapter_result: true`, sin fingir valores transformados ni utilizar
como resultado un campo `data.adapter` presente en el evento de prueba.

## Manifiesto versión 1

Ver `examples/adapters/status-normalizer.json`. Campos obligatorios:

- `id`: identificador local, hasta 64 caracteres.
- `protocol`: `1`; cualquier otro valor se rechaza.
- `runtime`: `python3`, ejecutado con `-I -S`.
- `capabilities`: exactamente `event.read` y `data.write`, sin duplicados.
- `input_limit`: 1–262144 bytes, incluido el salto de línea del mensaje.
- `output_limit`: 1–65536 bytes.
- `timeout_seconds`: 1–30 segundos.

`event.read` permite recibir el evento seleccionado; `data.write` permite devolver
un objeto de datos. No representan autorización de efectos sobre el host. La
integración liga manifiesto y código al hash aprobado, aplica
revocación y ejecuta bajo límites de systemd antes de aceptar resultados.

## Intercambio por proceso

Una invocación recibe un único JSON por stdin y termina después de escribir un
único JSON por stdout. El ID de invocación debe repetirse exactamente en la
respuesta. Ejemplo de solicitud:

```json
{"version":1,"id":"run-1","operation":"transform","event":{"source":"local:test","type":"build","data":{"status":"failed","message":"Build failed"}}}
```

Respuesta:

```json
{"version":1,"id":"run-1","data":{"severity":"error","message":"Build failed"}}
```

El resultado debe ser un objeto no nulo. La respuesta rechaza campos adicionales
en el sobre, documentos concatenados, otra versión/ID, más de 2048 nodos,
profundidad mayor de 16 o claves de más de 256 bytes. Ningún campo del sobre
puede pedir ejecución o cambiar permisos. Los campos dentro de `data` siguen
siendo datos sin autoridad.

El auxiliar recibe el código aprobado en un memfd sellado y el mensaje de entrada
por un canal separado. Devuelve como máximo la cuota indicada. Excederla hace
fallar el proceso; una salida parcial jamás debe aplicarse, ni cuando parezca un
JSON válido. stderr se descarta y una cuota cero conserva el comportamiento de
salida descartada de comandos y scripts existentes. La unidad invocadora sigue
siendo responsable del timeout y terminación del grupo de procesos.

El ejemplo `status-normalizer.py` convierte estados habituales en severidad y
acota el mensaje a 512 caracteres. No utiliza red, archivos de usuario ni APIs
del motor. Es una referencia del protocolo; aún no un plugin instalado.

## Contexto de ejecución

El esquema SQLite 5 añade una vista privada del evento por ejecución. El resultado
se guarda en `data.adapter`, conservando el resto de datos y metadatos del evento.
Otra transformación reemplaza ese campo; no se acumulan salidas sin límite. El
evento original de inbox, compartido por otros flujos, nunca se modifica.

El guardado de esa vista, el registro del paso y su avance usan una transacción.
Los pasos posteriores leen únicamente el contexto confirmado, incluso tras
reabrir la base. La vista completa tiene límite de 256 KiB y cuenta en la cuota
global de payload; una actualización reemplaza su consumo anterior. Un fallo de
escritura o cuota revierte contexto y avance juntos. La retención de ejecución
elimina también su contexto. Migrar desde esquema 4 conserva pendientes con
contexto vacío, que siguen leyendo el evento original.

Este mecanismo está conectado al worker. Las pruebas host verifican ejecución,
consumo del resultado tras reabrir, revocación/cambio de revisión, respuesta
inválida, timeout y cuotas. El exceso previsto de contexto termina como fallo;
un error de persistencia no confirma avance y conserva la política general de
incertidumbre tras caída. `scripts/test-ui-flow.py` verifica preparación y selección
de revisión en el panel, permisos y consumo del resultado por un paso HTTP.
`scripts/test-adapter-flow.py` prueba los binarios reales con un adaptador que
normaliza un estado y un script posterior que escribe la severidad resultante
en un directorio autorizado.
