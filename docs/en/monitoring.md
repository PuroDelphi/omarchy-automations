# System monitors

[Español](../es/monitoring.md) · [Monitor recipes](monitor-recipes.md)

Configure in Monitors, save the draft and review capabilities before activation.
Permission binds to the exact configuration; resource changes require review.
Import leaves monitors disabled.

| Metric | Value | Resource |
|---|---|---|
| CPU / memory | Usage percent | Machine |
| Disk / battery | Available percent | Path / available battery |
| Service | 100 active, 0 inactive | Explicit user unit |
| Process | 100 present, 0 absent | Executable path; current user's processes |
| File presence | 100 present, 0 absent | Absolute regular-file path |
| File age | Seconds since modification | Absolute regular-file path |
| File size | Bytes | Absolute regular-file path |
| Temperature | Celsius | Explicit thermal/hwmon sensor |
| Connectivity | 100 for HTTP 2xx, 0 on failure | Registered HTTPS destination |
| Journald | Events from last query | User unit and maximum priority |

Value monitors emit `alert`/`recovered` from `monitor:<id>`. CPU, memory, age,
size and temperature alert upward; the others alert downward. Recovery crosses
the opposite threshold. Confirmation durations and cooldown limit alerts.

Up to 64 monitors, sampling every 5–3600 seconds. Sequential sampling can be
delayed by slow queries; it is not real-time and does not wake the machine.
Failed readings show unavailable, not zero. A prolonged gap resets confirmation.

## Files, processes and sensors

File monitors read descriptor metadata only. They reject symlinks in every path
component, directories, devices and FIFOs. They neither read content nor traverse
directories. `openat2` is required; unsupported kernels fail without weakening the
boundary. A modification timestamp in the future is unavailable.

Process search examines up to 8192 `/proc` entries and compares the executable
path, including its resolved path. It does not inspect arguments or other users'
processes. Inaccessible own processes with no match prevent confirming absence:
the result is unavailable. One accessible match confirms presence.

Temperature accepts `/sys/class/thermal/thermal_zoneN/temp` and
`/sys/class/hwmon/hwmonN/tempM_input`, reads at most 64 bytes and converts
millidegrees to Celsius. Numbering can change after reboot; verify the sensor
still corresponds to the intended component.

## HTTPS checks

Sends **HEAD** to the exact URL with destination headers/authentication and a
three-second deadline. The server must support HEAD. No redirects are followed
and the response body does not become event data. Transport failures and non-2xx
status yield zero; missing credentials or network-policy blocks are unavailable.

The same TLS/DNS/IPv4/IPv6 validation and exact private exceptions as outbound
requests apply. The capability includes the whole destination: changing URL,
authentication, headers or exceptions requires review.

## Journald

Select a user `.service` unit and maximum priority 0–7 (0 emergency, 3 error,
4 warning, 6 info, 7 debug). It includes that priority and more urgent ones,
using existing read permissions only.

The first sample starts now without importing old history. Later queries read up
to 64 records in order, bounded to three seconds and 1 MiB output. Events have type
`journal` with `unit`, `priority`, `timestamp`; `MESSAGE` is never requested/sent.
Only exact unit matches are emitted.

Cursor and events share a transaction; failure does not consume the batch.
Restart retains the cursor and record identity prevents duplicates during retention.
Changing the monitor starts a new period. Revocation prevents committing a batch
already read but not persisted.

If journald has removed the cursor, an error is shown rather than silently skipping.
Reset cursor requires confirmation and abandons previous continuity: the next
authorized sample starts then. It grants no permissions and is audited.
CLI: `quatrroctl monitors.reset '{"id":"my-monitor"}'`.
