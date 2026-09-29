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
