# Measured resources and admission limits

[Español](../es/performance.md) · [User guide](user-guide.md)

The development build was measured on this Linux amd64 machine with
`python3 scripts/test-installed-load.py`. The test installs the actual binaries
in a temporary HOME, runs them, stops them, reinstalls and repeats using retained
data. This is a same-version reinstall, not a previous-schema migration test.

Each phase samples five seconds of idle time and sends 400 authenticated JSON
requests over HTTP loopback, using four concurrent clients and four entries.
Each request includes a 1024-character message plus sequence/phase fields. There
are no flows, monitors, scripts, notifications or outbound deliveries. Only the
engine process is measured; CLI, installer and desktop UI are excluded.

| Measurement | Clean install | After reinstall |
|---|---:|---:|
| Idle CPU time during 5 seconds | 0.01 s | 0.02 s |
| Idle resident memory | 19856 KiB | 21940 KiB |
| Resident memory after burst | 23784 KiB | 24460 KiB |
| Peak resident memory | 23784 KiB | 24852 KiB |
| Time for 400 admitted requests | 0.406 s | 0.581 s |
| Median request latency | 3.789 ms | 5.571 ms |
| 95th-percentile request latency | 4.672 ms | 6.767 ms |
| Engine CPU time during burst | 0.58 s | 0.76 s |

These are single-run observations, not guaranteed capacity or a sustained request
rate. The updated phase starts with existing data; CPU time can exceed wall time
when threads run on multiple cores. Hardware, storage, payloads and active work
change results. Five idle seconds do not establish long-term memory stability.

After each burst, the test sends 20 more requests to one entry, reaching its
120-request window limit; the next request must return 429. It verifies 420 then
840 retained events, zero executions and SQLite integrity. English is the initial
language; the chosen Spanish preference survives reinstall. Uninstall retains
the database. No host service or desktop preference is changed.

## Operational limits

Inbound currently permits at most 16 simultaneous handlers, 120 attempts per
entry per minute and 600 attempts globally per minute. Excess concurrency returns
503; rate rejection returns 429 with `Retry-After: 60`. Attempts count before
authentication succeeds, so failures can consume a window. These are fixed
engine limits, not configurable per-entry form options. Request bodies are
limited to 262144 bytes. Storage quotas can reject otherwise valid new events.

For example, a burst of 100 requests to an otherwise unused entry fits its rate
window, while 121 requests within that window exceed it even if the engine has
spare CPU. Spreading requests across entries does not remove the global limit.
Respect retry responses and use stable delivery identifiers where the provider
contract supports deduplication; rate-limited events have not been accepted.

Queue dispatch, slow receivers, command concurrency and recovery need their own
functional tests. The roadmap's five complete acceptance scenarios remain a
separate gate; this measurement alone does not close R1.5.

Separately, `scripts/test-installed-acceptance.py --native-notifications` passed
A1 (signed deployment → native notification and HTTPS) and A4 (Omarchy theme hook
→ HTTPS), both on clean installation and after a schema-4-to-5 upgrade fixture.
It verifies signatures, deduplication, repository/environment filters, preserved
permissions, and a nonblocking offline hook. It shows and dismisses two uniquely
identified desktop notifications. The complete installed upgrade matrix is now verified as described below. See the [validation log](../validation.md) for scope and evidence.

The `--user-services` mode additionally passed A3 before/after migration: an
authenticated webhook restarts one temporary user service, checks active state
and reports over HTTPS. A second service with a revoked grant is denied without
changing either PID; invalid bearer authentication is rejected. This mode needs
the real user-manager runtime and uses an isolated `QUATRRO_PROFILE` for the
installed engine. It does not test automatic installer backups of custom profiles.
Both temporary units are stopped on exit; existing services are untouched.

`scripts/test-installed-schedule.py` also passed A5 on clean installation and
after the schema upgrade. A real 10-second timer runs a pinned script that copies
a temporary document from a read-only mount into an approved writable mount,
then reports completion over authenticated HTTPS. The backup bytes match. Editing
the original script does not change its approved revision. A separate slow run
fails under its one-second timeout and cannot perform its later write. The test
also checks an unmounted host file is hidden and a fixture secret is absent from
the script environment and event/step/outbox bodies. It uses an isolated profile
and temporary data; it does not back up your real documents.

Finally, `scripts/test-installed-monitor.py` passed A2 in both stages using actual
disk readings on an isolated 32 MiB tmpfs. Filling 30 MiB lowers free space to
6.25%. It verifies a five-second confirmation before alerting, two additional
low samples without duplicates, and confirmed recovery after freeing space.
Each phase delivers alert and recovered events over authenticated HTTPS. The host
filesystem is mounted read-only; the test cannot fill the host disk.

Together these installed tests and the separate idle/load measurement close R1.5.
They do not close real-account OAuth, desktop/session recovery or final visual
review. Upgrade tests reconstruct the prior schema rather than run an archived
old binary. The [validation log](../validation.md) maps each requirement to its test.
