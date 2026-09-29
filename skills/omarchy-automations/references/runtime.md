# Installed runtime reference

The current CLI reports protocol 2, contract 1 and UI contract 1 through
`quatrroctl --version`. Use direct CLI operations; it negotiates the socket
handshake. Successful output is the operation result JSON, not a raw wire envelope.
Errors return a nonzero exit status. Prefer structured subprocess arguments and
stdin over interpolated shell command strings.

## Operations

| Operation | JSON input / result |
|---|---|
| `config.get`, `config.active` | No input; entire draft / active configuration |
| `config.save` | Full configuration; validates and saves draft, no activation |
| `config.preview` | No input; `hash`, `config`, `capabilities` |
| `config.activate` | `{"hash": REVIEWED_HASH, "grants": REVIEWED_CAPABILITIES}` |
| `simulate` | Event; matched flows and resolved steps, no effects |
| `emit` | Event with `local:` or `hook:` source; execution count and event ID |
| `history` | No input; latest execution list |
| `history.detail` | `{"id":"EXECUTION_ID"}`; execution, steps and delivery results |
| `permissions.list` | No input; granted scopes/hashes |
| `monitors.status`, `timers.status` | No input; actual source runtime status |
| `secrets.list` | No input; references, not secret values |
| `secrets.put` | `{"id":"reference","value":"SECRET","backend":"keyring"}` via stdin only |

The explicitly selected `file` secret backend is private but not encrypted;
there is no automatic fallback from a missing desktop keyring. Never print values.

## Notification example: merge, do not replace

Read the existing draft, append these objects to its `actions` and `flows` arrays
only if their IDs are unused, and preserve the remaining configuration:

```json
{
  "action": {"id":"ai-notice","kind":"notify","title":"Omarchy Automations","body":"{{data.message}}"},
  "flow": {"id":"ai-demo","name":"Local notification","source":"local:ai-demo","enabled":true,"steps":["ai-notice"]}
}
```

The wrapper above is an illustration of the two objects, NOT a config.save payload.
A configuration has `version:1` and arrays such as `entries`, `destinations`,
`actions`, `flows`, `monitors`, `timers`, `scripts` and `adapters`; preserve every
existing field, including fields not used by this example.

Event for `simulate`, and for an authorized `emit` after activation:

```json
{"source":"local:ai-demo","type":"demo","data":{"message":"Automation test"}}
```

Safe CLI invocation pattern (adapt the executable path if necessary):

```python
import json, subprocess

def ctl(operation, payload=None):
    argv = ["quatrroctl", operation]
    if payload is not None:
        argv.append("--stdin")
    result = subprocess.run(argv, input=json.dumps(payload) if payload is not None else None,
                            text=True, capture_output=True, check=True, timeout=20)
    return json.loads(result.stdout)
```

Do not blindly retry a mutation after a timeout: it may have committed. Inspect
active configuration/history first. `config.preview` returns a map of capability
scopes to hashes. After review and within authorization, activation input is
`{"hash": preview["hash"], "grants": preview["capabilities"]}`. Do not pipe an
unreviewed preview straight into activation.

## Low disk space notification

Append a disk monitor with unused ID `ai-disk`:

```json
{"id":"ai-disk","metric":"disk","path":"/","threshold":10,"recovery":15,"duration_seconds":60,"recovery_duration_seconds":30,"interval_seconds":10,"cooldown_seconds":300,"enabled":true}
```

Disk values are free-space percentages: alert below 10%, recover above 15%.
Create a `notify` action (for example `ai-disk-notice`) with body
`Free disk space: {{data.value}}%`. Link it to an enabled flow whose source is
`monitor:ai-disk`, with steps `["ai-disk-notice"]`. To notify only on alert, use
conditions `[{"field":"type","op":"eq","value":"alert"}]`.
Confirm the event type against installed-version documentation before selecting
other monitor behavior. Simulate:

```json
{"source":"monitor:ai-disk","type":"alert","data":{"metric":"disk","value":8}}
```

After authorized activation, inspect `monitors.status`. A healthy disk need not
create history or a notification: the threshold must persist for the configured
duration. Do not fill the disk to test. Real monitors cannot be fired via `emit`.

## Advanced configurations

Use the installed plugin's `docs/en/` if present in the Omarchy Git checkout, or
the maintained [documentation](https://github.com/PuroDelphi/omarchy-automations/tree/main/docs/en)
and [Spanish guides](https://github.com/PuroDelphi/omarchy-automations/tree/main/docs/es).
Prefer the release tag matching the installed package when fields differ.
Read only the relevant topic: `options.md`, `monitor-recipes.md`,
`authentication-recipes.md`, `code-examples.md`, `scheduling.md`, `oauth2.md`,
`command-isolation.md` or `service-example.md`. These are documentation filenames,
not CLI operations. Do not require a source checkout or compiler to use the CLI.
