# Recetas de monitoreo

[English](../en/monitor-recipes.md) · [Guía de usuario](user-guide.md) · [Opciones](options.md#group-monitors)

Importa [monitor-variants.json](../../examples/use-cases/monitor-variants.json) en
un borrador de prueba después de exportar el actual. Contiene dos monitores
deshabilitados por métrica, sin flujos ni acciones. Importar reemplaza el borrador.
Sustituye USER, rutas de archivos y sensores, unidades y direcciones HTTPS de
ejemplo por recursos de tu equipo. Selecciona solo los monitores necesarios antes
de activar.

Cada pareja indica **umbral de alerta → umbral de recuperación**, más el recurso
cuando corresponde. Usan muestreo cada 10 segundos, confirmación de alerta de
60 segundos, recuperación de 30 segundos y cooldown de 300 segundos. Journal usa
solo intervalo y prioridad; no aplica umbrales, confirmación ni cooldown.

| Métrica | Ejemplo A | Ejemplo B |
|---|---|---|
| `cpu` | 90 → 75 | 80 → 65 |
| `memory` | 90 → 75 | 85 → 70 |
| `disk` | 10 → 15 · path=`/` | 15 → 25 · path=`/home` |
| `battery` | 15 → 25 | 25 → 40 |
| `service` | 50 → 75 · unit=`backup.service` | 25 → 90 · unit=`worker.service` |
| `process` | 50 → 75 · path=`/usr/bin/python3` | 25 → 90 · path=`/usr/bin/bash` |
| `file_exists` | 50 → 75 · path=`/home/USER/reports/ready.json` | 25 → 90 · path=`/home/USER/backups/latest.tar` |
| `file_age` | 3600 → 300 · path=`/home/USER/reports/ready.json` | 86400 → 3600 · path=`/home/USER/backups/latest.tar` |
| `file_size` | 104857600 → 52428800 · path=`/home/USER/reports/output.log` | 10485760 → 5242880 · path=`/home/USER/reports/output.json` |
| `connectivity` | 50 → 75 · destination=`health-primary` | 25 → 90 · destination=`health-secondary` |
| `temperature` | 80 → 65 · path=`/sys/class/thermal/thermal_zone0/temp` | 75 → 60 · path=`/sys/class/hwmon/hwmon0/temp1_input` |
| `journal` | unit=`backup.service`, priority=`3` | unit=`worker.service`, priority=`4` |

## Interpretar los resultados

CPU y memoria miden porcentaje usado; disco y batería, porcentaje disponible.
La antigüedad se mide en segundos, el tamaño en bytes y la temperatura en grados
Celsius. Servicio, proceso, existencia y conectividad usan 0 o 100. Sus dos parejas
de umbrales producen intencionalmente el mismo comportamiento binario.

Las comparaciones son estrictas: disco A alerta por debajo de 10, no en 10;
recupera por encima de 15, no exactamente en 15. CPU A alerta por encima de 90
y recupera por debajo de 75. La confirmación exige observaciones sucesivas válidas.
Los huecos de muestreo pueden reiniciarla. Cooldown limita nuevas alertas; una
alerta persistente no genera recordatorios periódicos.

Un archivo regular ausente da 0 para file_exists; archivos ausentes o ilegibles
en antigüedad/tamaño quedan no disponibles. Proceso comprueba el usuario actual
y la ruta exacta del ejecutable. Servicio observa una unidad de usuario, nunca
de sistema. Conectividad usa HEAD, credenciales y política de red del destino;
HTTP 2xx da 100, fallo HTTP/transporte da 0 y credenciales ausentes o rechazo de
política dejan la lectura no disponible. Las URL de ejemplo no son receptores reales.

Journal A incluye prioridades 0–3; B incluye 0–4. Sus eventos tienen tipo `journal`
y contienen `unit`, `priority` y `timestamp`, sin texto del mensaje. La observación
inicial empieza ahora; reiniciar descarta el cursor y comienza en el siguiente
muestreo autorizado. Journal no emite eventos `recovered`.

## Conectar un monitor a un flujo

Crea una acción de notificación y un flujo con fuente `monitor:cpu-a`. Usa cuerpo
`CPU {{data.value}}: {{type}}`. Habilita el flujo en el borrador y simula `alert`
con `{"metric":"cpu","value":95,"state":"alert"}`, después `recovered` con
`{"metric":"cpu","value":70,"state":"recovered"}`. Espera el texto correspondiente
sin efectos en el escritorio. Para journal A usa fuente `monitor:journal-a`, tipo
`journal`, cuerpo `{{data.unit}} prioridad {{data.priority}}` y datos
`{"unit":"backup.service","priority":3,"timestamp":"2026-09-28T12:00:00Z"}`.

Para monitoreo real, habilita el monitor elegido, guarda y revisa permisos del
monitor y la acción antes de activar. Consulta estado e Historial. Simular prueba
coincidencias y plantillas; no muestrea hardware ni comprueba permisos para leer
archivos, sensores, journal o destinos HTTPS.
