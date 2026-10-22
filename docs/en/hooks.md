# Omarchy hooks

[Español](../es/hooks.md) · [System integration](system-integration.md)

Adapters in `packaging/hooks/` send local events to the engine. Each invocation
has a random identifier; arguments are encoded as JSON and never interpreted as commands.

| Hook | Flow source | Data |
|---|---|---|
| battery-low | `hook:battery-low` | `percentage`, integer 0–100 |
| font-set | `hook:font-set` | `font`, font name |
| post-boot | `hook:post-boot` | Empty object |
| post-update | `hook:post-update` | Empty object |
| pre-refresh-pacman | `hook:pre-refresh-pacman` | Empty object |
| theme-set | `hook:theme-set` | `theme`, theme identifier |

Type is `omarchy.<hook>`. Flow fields can read `data.theme`, `data.font` or
`data.percentage`. The socket restricts access to the same user; these events do
not prove that only the Omarchy executable could have produced them.


## Visual setup

These visual additions are included in Preview 7. Install matching UI and runtime components.

Open **Flows → Omarchy events** to connect or disconnect each of the six supported
hooks. This manages only the exact `quatrro-<event>` adapters; modified files,
symlinks and other tools' hooks are never replaced. A disconnected engine loses
hook events, as described below. Development profiles use an isolated hook folder.

The flow editor lists event names and suggested condition fields. Advanced source
and field paths remain available. Create or edit actions from the ordered steps;
new action changes are staged with the flow and discarded when cancelling it.
Shared action edits affect all referencing flows after activation.

**Flows → Templates** offers 38 bilingual recipes covering desktop hooks, theme
filters, recurring breaks, calendar reminders, resource alerts and local events.
Search or filter by category. Each insertion allocates fresh IDs and adds disabled
resources to the draft, without credentials or permissions. Adjust the values,
enable the flow and any monitor/timer, save, simulate and review before activating.
Connect desktop hooks separately. Calendar templates initially use UTC; set your
IANA timezone and desired time. Notifications in local-event recipes read
`data.message` supplied by the event producer. The startup hook means the desktop
started, not a delayed readiness check; the update hook does not verify success.


## Install an adapter

With `quatrroctl` installed under `~/.local/bin`, from the project root:

```bash
omarchy hook install theme-set packaging/hooks/quatrro-theme-set
```

Repeat for desired hooks. Omarchy copies them into `<hook>.d` directories;
`quatrro-<hook>` filenames avoid replacing other tools' hooks. Reinstallation
replaces only our same-named file. Do not edit an existing flat hook or
`/usr/share/omarchy`.

## Failure behavior

The engine call waits at most one second. The adapter adds a 1.5-second outer
limit and forced termination after another 0.2 seconds, and always returns success
to Omarchy. There are no detached processes or retries. If the engine is down, the
event is dropped; if acknowledgement is lost, it may already have been persisted.
Delivery during an outage is not guaranteed. The hook does not wait for flow actions.

For explicit diagnostics without suppressing errors:

```bash
quatrroctl hook theme-set example
```

This emits a real event and can execute an active flow. Use panel simulation first
to inspect a flow without effects.

## Permission compatibility

Action capabilities use `flow:<flow>:action:<action>` and monitors use
`monitor:<id>`. Earlier development revisions using `<flow>:<action>` require
new review and activation; old grants are not automatically converted. A queued
job without the new grant is denied before effects.
