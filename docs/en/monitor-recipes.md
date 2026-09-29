# Monitor recipes

[Español](../es/monitor-recipes.md) · [User guide](user-guide.md) · [Options](options.md#group-monitors)

Import [monitor-variants.json](../../examples/use-cases/monitor-variants.json) into
a disposable draft after exporting your current draft. It contains two disabled
monitors for each metric, no flows and no actions. Import replaces the draft.
Replace USER, file paths, sensor paths, units and example HTTPS addresses with
resources on your machine. Select only the monitors you need before activation.

Each pair below gives **alert threshold → recovery threshold**, plus the resource
when needed. All use a 10-second sample interval, 60-second alert confirmation,
30-second recovery confirmation and 300-second cooldown. Journal uses only the
sample interval and priority; threshold, confirmation and cooldown do not apply.

| Metric | Example A | Example B |
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

## Reading the results

CPU and memory measure percent used; disk and battery measure percent available.
File age is seconds, file size is bytes and temperature is degrees Celsius.
Service, process, file existence and connectivity use 0 or 100. Their two threshold
pairs above intentionally produce the same binary behavior.

Comparisons are strict: disk A alerts below 10, not at 10; recovery requires more
than 15, not exactly 15. CPU A alerts above 90 and recovers below 75. Confirmation
requires successive eligible observations. Sampling gaps can reset confirmation.
Cooldown limits new alerts; a persistent alert does not emit periodic reminders.

A missing regular file gives 0 for file_exists; missing/unreadable age or size
resources become unavailable. Process checks the current user and exact executable
path. A service monitor observes a user service, never a system service. Connectivity
uses HEAD, credentials and network policy of its destination; HTTP 2xx gives 100,
HTTP/transport failure gives 0, while missing credentials or policy denial is
unavailable. Example URLs are placeholders, not working receivers.

Journal A includes priorities 0–3; B includes 0–4. Events have type `journal` and
contain `unit`, `priority` and `timestamp`, without log message text. Initial
observation starts now; reset discards the cursor and starts at the next authorized
sample. There is no journal `recovered` event.

## Connect a monitor to a flow

Create a notification action and a flow with source `monitor:cpu-a`. Use body
`CPU {{data.value}}: {{type}}`. Enable the flow in the draft and simulate `alert`
with `{"metric":"cpu","value":95,"state":"alert"}`, then `recovered` with
`{"metric":"cpu","value":70,"state":"recovered"}`. Expect the corresponding
notification text without desktop effects. For journal A, use source
`monitor:journal-a`, type `journal`, body `{{data.unit}} priority {{data.priority}}`
and data `{"unit":"backup.service","priority":3,"timestamp":"2026-09-28T12:00:00Z"}`.

For real monitoring, enable the selected monitor, save and review monitor/action
permissions before activating. Inspect monitor status and History. Simulation tests
flow matching and templates; it does not sample hardware or establish permissions
to read a file, sensor, journal or HTTPS endpoint.
