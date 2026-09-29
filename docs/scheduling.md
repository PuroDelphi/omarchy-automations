# Programación

La sección **Programación** permite crear intervalos y calendarios. Guarda el
borrador, revisa sus permisos y activa la revisión. Cada programación habilitada
requiere la capacidad `timer:<id>`, vinculada a su configuración exacta.
Un flujo puede seleccionar `timer:<id>` como origen; cada acción del flujo
conserva su autorización independiente.

## Intervalos

Intervalo entre 5 segundos y 365 días. El primer vencimiento se calcula desde
la primera evaluación de una programación nueva, aproximadamente un segundo
después de activarla. El intervalo se expresa en tiempo UTC, no en hora local.
Tras un atraso se conserva la fase original del intervalo.

## Calendarios

Una hora `HH:MM`, una zona horaria IANA explícita —por ejemplo
`America/Bogota` o `UTC`— y días opcionales de la semana. Usa `mon`, `tue`,
`wed`, `thu`, `fri`, `sat`, `sun`; sin días seleccionados se ejecuta a diario.
No se admite la zona implícita `Local`. La compilación incluye tzdata como
respaldo cuando no hay una base de zonas del sistema.

La programación comienza en la siguiente ocurrencia posterior a la primera
evaluación. Una hora que no existe por adelanto de reloj se omite. En una hora
repetida por cambio de zona/desfase, solo es elegible la primera ocurrencia.
No se ejecuta la segunda aunque el motor se inicie en la hora repetida.
No es sintaxis cron: para varias horas crea varias programaciones.

## Atrasos, pausa y reinicios

Selecciona una política explícita:

- `coalesce`: genera un solo evento por los vencimientos acumulados y calcula
  la siguiente ocurrencia futura. El evento identifica el primer vencimiento
  pendiente, no una lista de todas las ocurrencias perdidas.
- `skip`: descarta vencimientos con más de 5 segundos de atraso en intervalos
  o más de 60 segundos en calendarios. Dentro de esa tolerancia genera el evento.

La próxima ejecución y el evento se guardan en una misma transacción SQLite.
Un reinicio no reinicia el intervalo. Si falla la persistencia no se consume
la ocurrencia, y se vuelve a evaluar según la política de atrasos.
Un retroceso del reloj espera hasta alcanzar de nuevo el próximo vencimiento
guardado, sin repetir los anteriores.

Pausar efectos con admisión `retain` permite guardar los eventos para después.
Con admisión `reject`, el evento no se guarda y su ocurrencia queda pendiente;
al reanudar se aplica `coalesce` o `skip`. Revocar el permiso de una programación
detiene sus nuevos eventos; los eventos ya encolados conservan los permisos
independientes de sus acciones. Para impedir esos efectos, revoca las acciones
o utiliza pausa/cancelación. Reactivar la misma programación conserva su
vencimiento pendiente; cambiar su configuración crea un calendario nuevo.

## Evento y diagnóstico

El tipo es `scheduled`, y los datos incluyen:

```json
{
  "timer": "backup",
  "scheduled_at": "2026-09-28T14:00:00Z",
  "fired_at": "2026-09-28T14:00:01Z",
  "late_seconds": 1,
  "policy": "coalesce"
}
```

Las plantillas pueden usar `{{data.scheduled_at}}`. Los tiempos del evento se
expresan en UTC. `quatrroctl timers.status` muestra permiso, próximo vencimiento
y último evento de las programaciones activas. El panel muestra la próxima
ejecución en la zona local del escritorio.

Se admiten hasta 64 programaciones por configuración. La evaluación ocurre
aproximadamente cada segundo; no es un planificador de tiempo real y no
despierta un equipo suspendido. Los retrasos de suspensión/cierre de sesión se
tratan como los demás atrasos. Importar deshabilita todas las programaciones.
