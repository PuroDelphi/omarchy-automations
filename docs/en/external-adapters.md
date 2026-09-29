# External adapters — transformation protocol

[Español](../es/external-adapters.md) · [Illustrated examples](code-examples.md)

The contract, isolated transport and local revision preparation/registration are
implemented. Execution integrates exact references, capabilities, preflight, worker
and private result persistence. The panel prepares/reviews adapters and explicitly
selects action revisions. Final release gates still apply.

The initial contract transforms event data in a separate Python process without
network or user-directory access. The engine retains responsibility for authorized
flow effects. Adapters cannot request commands, install code, change configuration
or obtain secrets. Registration uses an explicitly selected local file, never received events.

## Preparation and registration

`adapters.prepare` accepts absolute `path` and `manifest` below. It reads a regular
UTF-8 file up to 32 KiB without NUL or symlink components. It returns manifest fields,
code and revision without retaining the path, saving configuration, granting permission
or launching processes.

SHA256 binds code and entire manifest, including protocol, capabilities, quotas
and timeout. Up to eight adapters can be stored in the draft, within the global
500-resource limit. Validation recomputes the hash. Editing source does not alter
copied code; code/manifest changes need a new revision. Export includes code, so
never embed credentials. Unsupported protocols/capabilities fail before source reads.

Register an action using the returned hash:

```json
{"id":"normalize","kind":"adapter","adapter":"status-normalizer","adapter_revision":"<hash returned by adapters.prepare>"}
```

The action cannot choose another executable, arguments, timeout or directories:
the approved manifest/code and mount-free isolation apply. Revision changes alter
the capability; activation requires explicit approval. Revocation/replacement blocks
pending work using the old reference.

Preflight checks sandbox limits and a harmless JSON exchange using engine-owned
code with the manifest runtime/quotas, not registered adapter code. The worker
executes the revision pinned at event admission, validates output and commits
result with step advance. Failure, timeout, quota or invalid response stops the
execution without passing results onward.

Simulation does not execute adapters. It shows the reference with
`result_available:false` and subsequent steps with `requires_adapter_result:true`.
It never invents output or trusts sample `data.adapter` as the adapter's result.

## Manifest version 1

See `examples/adapters/status-normalizer.json`. Required fields:

- `id`: local identifier, up to 64 characters.
- `protocol`: exactly `1`.
- `runtime`: `python3`, executed with `-I -S`.
- `capabilities`: exactly `event.read` and `data.write`, without duplicates.
- `input_limit`: 1–262144 bytes, including the message newline.
- `output_limit`: 1–65536 bytes.
- `timeout_seconds`: 1–30 seconds.

`event.read` permits receiving the selected event; `data.write` permits returning
a data object. Neither authorizes host effects. Integration binds manifest/code to
the approved hash, enforces revocation and systemd limits before accepting results.

## Process exchange

One invocation reads one JSON document from stdin and exits after writing one to
stdout. Response must repeat the exact invocation ID. Request:

```json
{"version":1,"id":"run-1","operation":"transform","event":{"source":"local:test","type":"build","data":{"status":"failed","message":"Build failed"}}}
```

Response:

```json
{"version":1,"id":"run-1","data":{"severity":"error","message":"Build failed"}}
```

Result must be a non-null object. Extra envelope fields, concatenated documents,
wrong version/ID, over 2048 nodes, depth above sixteen or keys above 256 bytes are
rejected. Envelope fields cannot request execution or change permissions; data
fields carry no authority.

The helper receives approved code in a sealed memfd and input through a separate
channel. Exceeding output quota fails the process; partial output must never apply,
even if it looks like valid JSON. stderr is discarded. A zero quota retains existing
command/script output-discard behavior. The invoking unit owns timeout/group termination.

`status-normalizer.py` maps common statuses to severity and limits message to
512 characters. It uses no network, user files or engine API. It is a protocol
reference, not an automatically installed adapter.

## Execution context

SQLite schema 5 adds a private per-execution event view. Result is stored in
`data.adapter`, retaining other data/metadata. Another transform replaces it rather
than accumulating unbounded output. The shared original inbox event never changes.

Context, step record and advance commit atomically. Later steps read only committed
context, even after reopen. The full view is limited to 256 KiB and counts toward
global payload quota, replacing its previous consumption. Write/quota failure
rolls back context and progress together. Execution retention removes its context.
Schema 4 migration preserves pending work with empty context reading original events.

Host tests cover execution, reuse after reopen, revocation/revision change,
invalid output, timeout and quotas. Anticipated context overflow fails the execution;
persistence errors do not confirm progress and retain general crash uncertainty.
`test-ui-flow.py` covers preparation/revision selection, permissions and HTTP use
of the result. `test-adapter-flow.py` exercises real binaries, normalization and a
later script writing severity into an approved directory.
