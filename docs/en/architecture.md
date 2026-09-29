# Architecture and trust boundaries

[Español](../es/architecture.md) · [Documentation](README.md) · [Protocol](protocol.md)

This describes the implemented development version. The original
[design proposal](../DESIGN.md) is a historical planning document; its future-tense
statements are not the current implementation status. Follow the roadmap for
acceptance evidence and deferred tests.

## Components

```text
Authenticated webhooks ─┐
Monitors / schedules ───┼─> validated events ─> SQLite ─> reviewed flows
Omarchy hooks / local ─┘                                   │
                                             typed actions / HTTPS
                                                          │
                                                  execution history

QML panel <─> quatrroctl <─> private Unix socket <─> quatrrod
                                                       │
                                   optional privileged service broker
```

The native QML panel edits drafts, simulates inputs and displays capabilities,
credentials, history and status. It does not host the HTTP receiver. `quatrrod`
is a separate user service, so hiding or reloading the panel does not stop flows.
`quatrroctl` negotiates the local protocol and sends one operation per connection.
The engine uses Go and SQLite; dependency versions are pinned in `go.mod`/`go.sum`.

Omarchy plugins run as user code in the shell process. The panel is not a sandbox,
and components running under the same UID do not create a strong boundary against
malicious local software. Command isolation constrains dispatched jobs; it does
not make the entire desktop trustworthy.

## Configuration and execution

Entries describe inbound authentication and payload format. Destinations define
approved HTTPS targets and credential references. Flows select a source, require
all configured conditions and execute ordered action steps. Monitors and timers
produce events. Scripts and adapters are local, reviewed code with pinned revisions.

Saving validates a draft. Simulation evaluates its conditions and templates with
no effects. Activation requires the exact reviewed hash and capabilities. An
execution retains its configuration revision; permissions are checked again
before dispatch. Revocation blocks future effects, including eligible queued work.
Remote event data cannot select an arbitrary executable, service, credential or URL.

Typed template mapping preserves whole-value scalar types and encodes structured
output, preventing data from creating extra JSON/form fields by breaking quoting.
Conditions are bounded declarative comparisons, not arbitrary JavaScript or shell.

## Ingress and egress

The HTTP listener binds loopback. A remote receiver needs an explicit trusted TLS
proxy/tunnel; neither setup nor a tunnel replaces webhook authentication. The
administrative Unix socket must not be exposed with the webhook listener.
Authentication checks original body bytes before normalization and persistence.
HTTP acceptance confirms a committed event, not completed actions.

Outbound targets are configured HTTPS destinations with redirects disabled.
Addresses are resolved, validated and pinned for connection while TLS verifies the
hostname. Exact private-host exceptions are explicit capabilities. Payloads are
persisted for retries; the receiver must implement idempotency. The engine cannot
promise exactly-once effects across an independent remote service.

## State and recovery

SQLite stores inbox, executions, step state, outbox, monitor/timer state, drafts,
active revisions and grants. Schema migrations are transactional; future schemas
are rejected. Quotas and retention bound accepted work. Pending or uncertain work
is not silently deleted to make room for new events.

Crash recovery distinguishes completed, failed, cancelled and uncertain effects.
An uncertain command may already have run and is not blindly replayed. HTTP retry
uses the stored body and idempotency key; it does not re-render a later draft.
There is no universal rollback for notifications, commands or external requests.
Pause and admission policy are distinct from cancellation and permission revocation.

## Secrets and system actions

Credentials use the desktop Secret Service or an explicitly selected private-file
backend. Private files are permission-protected, not encrypted; there is no silent
fallback. Configuration stores references. Secrets manually pasted into ordinary
headers or payload fields remain ordinary exportable configuration and must be avoided.

Commands/scripts run through user systemd and Bubblewrap with resource, time,
filesystem and network constraints. Unsupported isolation prevents activation.
Approved host directories have checked identity and access mode. Script revisions
are executed from their approved content rather than a mutable source path.

User-service actions target exact units and operations. System-service actions
require the optional separate broker, its administrative policy and Polkit checks.
The user engine never becomes a generic root shell. The broker is not installed
by the default setup.

## Installation ownership

In the recommended mode, Omarchy owns the Git checkout containing QML and setup
scripts. The runtime installer owns only two binaries and the user service unit,
recording file hashes in a private receipt. It compares checked-out UI files with
the selected runtime package before installing. Uninstall runtime first, then use
Omarchy to remove its checkout.

The source/archive alternative owns the installed UI too, using immutable UI
revision directories to avoid stale QML caches. Both modes preserve data and
credentials on uninstall, refuse modified owned files, and cannot be mixed over
an existing receipt. Separate hooks and broker policies have their own lifecycle.

Default locations (XDG overrides apply to backend paths):

| Purpose | Location |
|---|---|
| Omarchy panel | `~/.config/omarchy/plugins/quatrro.automations/` |
| Engine configuration | `$XDG_CONFIG_HOME/quatrro/` |
| SQLite state/history | `$XDG_STATE_HOME/quatrro/` |
| Local control socket | `$XDG_RUNTIME_DIR/quatrro/control.sock` |
| User service | `$XDG_CONFIG_HOME/systemd/user/quatrrod.service` |
| Runtime files | `~/.local/bin/quatrrod`, `~/.local/bin/quatrroctl` |
| Installation receipt/backups | `$XDG_STATE_HOME/quatrro-install/` |

A development `QUATRRO_PROFILE` isolates application data. It does not turn real
actions into simulations. See [security](security.md), [installation](installation.md)
and [external acceptance](external-acceptance.md) for operational boundaries.
