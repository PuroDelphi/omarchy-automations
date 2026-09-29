# Action recipes

[Español](../es/action-recipes.md) · [User guide](user-guide.md) · [Options](options.md#group-actions)

Create actions in the draft, reference them from a flow, simulate, then review and
activate their exact permissions. Saving an action alone never runs it. The table
shows alternative configurations; use your own registered resources.

| Kind | Example A | Example B | Expected result |
|---|---|---|---|
| `notify` | Title `Build`, body `{{data.message}}` | Title `Disk`, body `Free: {{data.value}}%` | A desktop notification; missing template fields fail. |
| `http` | JSON destination, body `{"state":"{{type}}"}` | Raw destination, body `Build: {{data.message}}` | An HTTPS delivery tracked in History/outbox. |
| `service` | `backup.service`, `start` | `worker.service`, `stop` | Starts or stops that exact user unit. |
| `command` | Fixed `/usr/bin/true`, no args, timeout 5 | Fixed `/usr/bin/test`, args `-d` and `/tmp`, timeout 5 | Successful isolated process with no retained stdout. |
| `omarchy` | `theme.current` | `nightlight.status` | Checks command success; output is not returned as event data. |
| `script` | `check-project`, project `demo-project`, count 3 | Same revision, project `release-2`, count 10 | Typed arguments validated and supplied to approved code. |
| `adapter` | Normalizer payload `{"status":"failed","message":"Build failed"}` | `{"status":"passed","message":"Build passed"}` | Private `data.adapter` result used by subsequent steps. |
| `system-service` | Exact allowed unit, `status` | Exact allowed unit, `restart` | Requires the separate broker, policy, Polkit and flow grant. |

Scripts and adapters require a prepared revision selected explicitly. Follow the
[complete code examples](code-examples.md) for contracts, activation and expected
results. For outbound form encoding, use a `form` destination with JSON body
`{"project":"{{data.project}}"}`; the engine encodes the resolved fields. Do not
paste a URL-encoded string into a form action. See [delivery recovery](recovery-example.md).

## User-service operations

`status` checks whether the service is active; inactive means action failure.
`start` and `restart` also require it to be active after the command. This matters
for short-lived oneshot services that complete successfully and become inactive.
`stop` checks command success without requiring the unit to remain active.
These actions do not enable/disable units at login. The [service exercise](service-example.md)
uses a disposable unit and verifies status, restart and revocation. Changing the
unit or operation requires renewed permission review.

## Named command profiles and directories

Prepare an existing, dedicated host directory through **Approved directories**.
For example, create `/home/USER/automation-demo` yourself, enter its real absolute
path as source, target `/work/data`, and choose **Read only**. Select **Prepare and
add directory**. Replace USER; no example inode/device identity is reusable.
Preparation identifies the directory but does not grant access or create it.

| Profile | Example A | Example B | Access and result |
|---|---|---|---|
| `file-exists` | `/work/data/report.json` | `/work/data/ready.txt` | `ro` suffices; succeeds only if the direct child is a regular file. |
| `make-directory` | `/work/data/new-backup` | `/work/data/export` | Requires `rw`; creates a direct child with mode 0700 and fails if it exists. |

Use each path as **Profile path**, set timeout 5, and retain the prepared mount.
Named profiles derive their executable/arguments; they cannot be overridden.
For a fixed command reading that directory, choose `/usr/bin/test`, with one
argument per line: `-f`, then `/work/data/report.json`. For another example use
`-d`, then `/work/data`. These are literal arguments, without shell expansion.

Working directory can be `/tmp` or an approved target such as `/work/data`.
`/tmp` is private and temporary. A writable host mount is live: changes survive
execution and cancellation does not undo them. Permission covers the mounted tree,
including special files and nested mounts. A Unix socket there can communicate
with a host service despite IP network isolation. Prefer a dedicated directory.
Changing its identity, access, target or working directory requires new review.
Simulation previews mounts and arguments without testing file existence or creating
directories. A real test performs the filesystem operation.

## Desktop effects

`nightlight.toggle` changes the current night-light state. `system.lock` locks the
session. Unlike the two status operations above, these have visible desktop effects.
Simulate first; activate and send a real test only when that effect is intended.
None of these operations makes stdout available to later steps. In particular,
`theme.current` is not a way to populate `data.theme`; use the theme hook event in
[system integration](system-integration.md) for that information.
