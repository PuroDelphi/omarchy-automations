# Omarchy hooks and optional administration

[Español](../es/system-integration.md) · [User guide](user-guide.md) · [Options](options.md)

## Desktop hooks

Hooks feed local events into the same flow engine as webhooks. They do not need
an HTTP listener. Install only the hook adapters you want, from the project or
extracted distribution directory, with `quatrroctl` installed in `~/.local/bin`:

```bash
omarchy hook install theme-set packaging/hooks/quatrro-theme-set
omarchy hook install battery-low packaging/hooks/quatrro-battery-low
```

Omarchy copies each adapter into its own `<hook>.d` directory. Reinstalling replaces
our same-named adapter. Existing flat hooks and other adapters are preserved.

| Hook | Exact flow source | Event type | Data |
|---|---|---|---|
| `theme-set` | `hook:theme-set` | `omarchy.theme-set` | `theme`: theme identifier |
| `font-set` | `hook:font-set` | `omarchy.font-set` | `font`: font name |
| `battery-low` | `hook:battery-low` | `omarchy.battery-low` | `percentage`: integer 0–100 |
| `post-boot` | `hook:post-boot` | `omarchy.post-boot` | Empty object |
| `post-update` | `hook:post-update` | `omarchy.post-update` | Empty object |
| `pre-refresh-pacman` | `hook:pre-refresh-pacman` | `omarchy.pre-refresh-pacman` | Empty object |

For each other row, install `packaging/hooks/quatrro-HOOK` with that hook's name.
Arguments become JSON values and are not interpreted as commands.

### Example: report a theme change

1. Create a notification action with body `Theme: {{data.theme}}`.
2. Create an enabled flow with source `hook:theme-set` and that action as its step.
3. Simulate type `omarchy.theme-set` with `{"theme":"tokyo-night"}`; expect
   `Theme: tokyo-night`. Repeat with `{"theme":"catppuccin-latte"}`.
4. Save, review the notification permission and activate. A future Omarchy theme
   change invokes the installed adapter. Check the notification and History.

For an explicit real event without changing the desktop theme:

```bash
quatrroctl hook theme-set tokyo-night
```

This can execute active flows. It is not simulation or evidence that the theme
actually changed. The local socket verifies the user, not exclusive origin from
the Omarchy executable.

### Example: low battery to an HTTPS receiver

Register your actual HTTPS destination and an HTTP action whose JSON body is
`{"battery":"{{data.percentage}}"}`. Keep the quotes so the template is valid
JSON; an exact field placeholder resolves to the original numeric value. Create a flow from `hook:battery-low` with
condition `data.percentage lt 15` using numeric type. Simulate type
`omarchy.battery-low`: `{"percentage":10}` matches and resolves `{"battery":10}`;
`{"percentage":20}` does not match. Save, review and activate when ready to send.
A real battery hook then feeds this flow. This differs from a periodic battery
monitor: it depends on Omarchy invoking its low-battery hook.

Hook adapters wait at most one second for the engine, with an outer 1.5-second
timeout and forced termination after another 0.2 seconds. They return success to
Omarchy even if the engine fails. There are no retries or detached jobs; an event
can be lost during downtime, or persisted despite a lost acknowledgement. The
hook does not wait for flow actions to finish.

To remove only these two adapters, inspect their paths, then remove their files:

```bash
ls -l ~/.config/omarchy/hooks/theme-set.d/quatrro-theme-set ~/.config/omarchy/hooks/battery-low.d/quatrro-battery-low
rm -- ~/.config/omarchy/hooks/theme-set.d/quatrro-theme-set ~/.config/omarchy/hooks/battery-low.d/quatrro-battery-low
```

This stops future invocations of those adapters; it does not cancel queued events.
The normal plugin uninstaller does not manage adapters installed separately.

## Optional system-service administration

User-service actions do not need this component. `system-service` actions require
the separate root broker, an exact local allowlist and Polkit authorization.
The broker is not installed or enabled by normal plugin installation. It offers
only `status`, `start`, `stop` and `restart` for exact `.service` names; no shell,
arbitrary root commands, wildcards, package installation or enable/disable action.

Two policy examples are shown below. Replace `1000` with the intended non-root
account's numeric UID (`id -u`). The example unit must already exist and belong
to your intended workload; preparation does not create it.

Read-only example, saved as `broker-policy.json`:

```json
{"version":1,"rules":[{"uid":1000,"unit":"example-worker.service","operations":["status"]}]}
```

Status plus restart example (restart can interrupt that service):

```json
{"version":1,"rules":[{"uid":1000,"unit":"example-worker.service","operations":["status","restart"]}]}
```

An empty `rules` array denies every operation. Policy size is limited to 64 KiB
and 128 rules. Root UID, duplicate rules/operations, unknown keys and uninstantiated
template units are rejected. The account must resolve locally when preparing.

### Prepare, review and install

Use a new output directory. This first command needs no administrator privileges
and installs or authorizes nothing:

```bash
python3 scripts/prepare-broker-install.py --policy broker-policy.json --output broker-preview
```

Review `broker-preview/manifest.json`, `broker.json`, the generated Polkit rules
and the six files' fixed destinations. Hashes detect changes but do not establish
who produced the package. Use a reviewed build. From an administrative terminal,
apply the reviewed bundle and explicitly enable its socket:

```bash
sudo python3 scripts/install-broker.py apply --bundle broker-preview
sudo systemctl enable --now quatrro-broker.socket
```

Installation leaves the broker stopped and disabled until the second command.
In the panel create a `system-service` action for the exact unit and operation,
then use **Check administrative permissions**. Checking has no service effect and grants nothing.
For example, the first policy authorizes status but denies restart; the second
authorizes both. Saving a draft remains possible without the broker, but activating
an enabled administrative flow requires successful checks. Execution checks again.
Review the flow's permission before activation. Real start/stop/restart affects
the system service; inspect its state and History to verify the result.

### Update, revoke and remove

Prepare a fresh bundle for an update. Applying normally preserves installed
policy and Polkit rules; use `--replace-policy` only when intentionally replacing
those permissions. Every apply stops/disables the broker until explicitly enabled.
To revoke all broker permissions, prepare the default empty policy in a new
directory, review it, then apply with `--replace-policy`:

```bash
python3 scripts/prepare-broker-install.py --output broker-deny-all
sudo python3 scripts/install-broker.py apply --bundle broker-deny-all --replace-policy
```

Plugin grant revocation is another independent control and does not undo completed
effects. To remove the broker's verified owned files:

```bash
sudo python3 scripts/install-broker.py remove
```

If installation was interrupted and a pending transaction is reported, use
`sudo python3 scripts/install-broker.py recover` before applying/removing again.
Recovery restores the previous installation state and leaves the socket disabled;
it refuses when no transaction is pending. Foreign or modified files cause a
refusal rather than silent overwrite. Do not delete the receipt to bypass this.

The broker has been tested with real systemd/Polkit in an isolated environment;
it has not been installed on this host. Hook tests exercise argument handling and
engine failure behavior. These checks do not establish your service's operational
safety or delivery during a real logout/suspend. See the [validation log](../validation.md).
