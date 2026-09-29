# Monitores del sistema

Configura los monitores en **Monitores**, guarda el borrador y revisa sus
capacidades antes de activar. El permiso se liga a la configuración exacta;
cambiar el recurso requiere revisarlo. Importar deja los monitores desactivados.

| Métrica | Valor | Recurso |
|---|---|---|
| CPU / memoria | Porcentaje de uso | Equipo |
| Disco / batería | Porcentaje disponible | Ruta / batería disponible |
| Servicio | 100 activo, 0 inactivo | Unidad de usuario explícita |
| Proceso | 100 presente, 0 ausente | Ruta del ejecutable; procesos del usuario |
| Archivo presente | 100 presente, 0 ausente | Archivo regular con ruta absoluta |
| Antigüedad de archivo | Segundos desde la modificación | Archivo regular con ruta absoluta |
| Tamaño de archivo | Bytes | Archivo regular con ruta absoluta |
| Temperatura | Grados Celsius | Sensor térmico o hwmon explícito |
| Conectividad | 100 si HTTP 2xx, 0 si falla | Destino HTTPS registrado |
| Journald | Eventos de la última consulta | Unidad de usuario y prioridad máxima |

Los monitores de valores emiten `alert` y `recovered` desde `monitor:<id>`.
CPU, memoria, antigüedad, tamaño y temperatura alertan al subir; las otras
métricas alertan al bajar. El umbral de recuperación debe cruzarse en sentido
contrario. Duración, duración de recuperación y cooldown limitan las alertas.

Se admiten 64 monitores e intervalos de 5 a 3.600 segundos. El muestreo secuencial
puede retrasarse por consultas lentas: no es tiempo real ni despierta el equipo.
Una lectura fallida muestra «no disponible», sin interpretarse como cero. Un
hueco prolongado entre muestras reinicia la ventana de confirmación.

## Archivos, procesos y sensores

Los monitores de archivo leen únicamente metadatos del descriptor. Rechazan
enlaces simbólicos en cualquier componente, directorios, dispositivos y FIFO.
No leen contenido ni recorren directorios. Requieren `openat2`; si el kernel
no lo admite, fallan sin degradar la garantía. Una modificación futura produce
«no disponible».

La búsqueda examina hasta 8.192 entradas de `/proc` y compara la ruta del
ejecutable, incluyendo su ruta resuelta. No inspecciona argumentos ni procesos
de otros usuarios. Si hay procesos propios inaccesibles y no se encuentra una
coincidencia, no puede confirmar ausencia: muestra «no disponible». Encontrar
una coincidencia accesible sí confirma presencia.

Temperatura admite `/sys/class/thermal/thermal_zoneN/temp` y
`/sys/class/hwmon/hwmonN/tempM_input`, con lectura máxima de 64 bytes y conversión
de miligrados a Celsius. La numeración puede cambiar al reiniciar; verifica que
el sensor siga correspondiendo al componente esperado.

## Comprobación HTTPS

Envía **HEAD** a la URL exacta con las cabeceras y autenticación del destino,
con plazo de tres segundos. El servidor debe admitir HEAD. No sigue
redirecciones ni convierte el cuerpo de respuesta en datos del evento.
Errores de transporte o HTTP fuera de 2xx valen cero; credenciales ausentes
o bloqueos de política de red producen «no disponible».

Aplica la misma validación TLS, DNS, IPv4/IPv6 y excepciones privadas exactas
que las salidas. El permiso incluye el destino completo: cambiar URL,
autenticación, cabeceras o excepciones exige revisar la capacidad.

## Journald

Selecciona una unidad `.service` del usuario y prioridad máxima de 0 a 7
(0 emergencia, 3 error, 4 advertencia, 6 información, 7 depuración). Incluye esa
prioridad y las más urgentes. Solo usa permisos de lectura existentes.

La primera muestra observa desde ese momento, sin importar historia anterior.
Después consulta hasta 64 registros en orden, con plazo de tres segundos y
salida máxima de 1 MiB. Emite `journal` con `unit`, `priority` y `timestamp`;
no solicita ni transmite `MESSAGE`. Solo emite coincidencias exactas de unidad.

Cursor y eventos comparten transacción: un fallo no consume el lote. Reiniciar
conserva el cursor; la identidad del registro evita duplicados dentro de la
retención. Cambiar el monitor inicia un periodo nuevo. Revocar impide confirmar
un lote que se haya leído pero aún no persistido.

Si journald eliminó el cursor, muestra error sin saltar registros silenciosamente.
**Reiniciar cursor** pide confirmación y descarta la continuidad anterior: la
siguiente muestra autorizada empieza desde entonces. No concede permisos y
queda en auditoría. CLI: `quatrroctl monitors.reset '{"id":"mi-monitor"}'`.
