# Local protocol 2 — operation contract 1

[Español](../es/protocol.md) · [Architecture](architecture.md)

The socket is `$XDG_RUNTIME_DIR/quatrro/control.sock`, mode 0600. Client and server
check the peer UID with `SO_PEERCRED`. It is not exposed over HTTP. Each connection
first negotiates compatibility, then accepts one operation. Messages are newline
JSON up to 1 MiB. The connection deadline is at most 15 seconds (or the caller's
shorter deadline); the handler has 12 seconds.

Send a greeting without payload:

```json
{"version":2,"contract":1,"op":"hello"}
```

The server confirms `{"version":2,"contract":1,"ok":true,"data":{"handshake":"ready"}}`.
Only then send the operation on the same connection:

```json
{"version":2,"contract":1,"op":"status","data":{}}
```

Responses contain `version`, `contract`, `ok` and either `data` or `error`.
The server checks version/contract again for the operation and retains one reader
per connection so concatenated messages are handled correctly. The handshake
cannot be omitted. Version 1 clients are rejected before invoking the engine;
a new client sends only its greeting to an old engine, without event or secret.
There is no fallback or automatic operation replay. Both versions require exact
equality. Broker protocol 1, configuration schemas and adapter contracts are separate.

For mixed installations, update engine, CLI and panel together, restart the engine
and reload the panel. Incompatibility does not erase persisted state. External
socket clients also need the handshake; hooks using the current CLI negotiate it.

Use `quatrroctl OP --stdin` for JSON payloads. **Secrets must use stdin**, never
process arguments or commands saved in shell history.

## Operations

| Operation | Input | Result/effect |
|---|---|---|
| `status` | None | Version, language, readiness and actual loopback ingress address/state; public exposure remains unverified |
| `scripts.prepare` | id, path, interpreter, parameters | Copy/validate a local revision without saving or executing |
| `adapters.prepare` | path, manifest | Copy local code and bind its revision to the manifest; no grants/effects |
| `directories.prepare` | source, target, access | Directory device/inode identity; no save/grant |
| `preferences.get` / `preferences.set` | None / language `en` or `es` | Read/save profile language without changing resources |
| `config.get` / `config.active` | None | Draft / active configuration |
| `config.save` | Full version 1 configuration | Validated draft hash |
| `config.preview` | None | Hash, configuration and required capabilities |
| `config.activate` | Exact reviewed hash and grants | New active revision |
| `permissions.revoke` | scope `flow:<flow>:action:<action>`, `monitor:<id>` or `timer:<id>` | Revoke future dispatch under that capability |
| `permissions.list` | None | Active capabilities |
| `secrets.put` | id, value, backend `keyring` or `file` | Reference/backend, never the value |
| `secrets.list` / `secrets.delete` | None / id | List references or remove value/reference; affected connections fail closed |
| `emit` | Event with `local:` or `hook:` source | Event ID and count of persisted executions |
| `simulate` | Test event | Matching flows and resolved steps without effects |
| `control` | paused boolean, admission `retain` or `reject` | Pause/admission state |
| `cancel` | None | Request process cancellation; started effects can become uncertain |
| `history` / `history.detail` | None / id | Latest 100 executions / steps and delivery attempts, without bodies |
| `timers.status` | None | Permission, next due time and last firing |
| `monitors.status` | None | Monitor state/value and next sample |
| `monitors.reset` | Journald monitor id | Drop cursor and start at next authorized sample; grants unchanged |
| `delivery.retry` | id | Retry eligible failed/cancelled HTTP with its stored key |
| `config.import` / `config.import-file` | Configuration / absolute `.json` path | Replace draft with disabled resources |
| `config.export-file` | New absolute `.json` path | Export without overwriting |
| `diagnostics` / `diagnostics.export-file` | None / new absolute `.json` path | Versions/counters without configuration data; export mode 0600, no overwrite |
| `queue.inspect` | Optional state, offset, limit 1–100 | Paginated pending/running/uncertain jobs without bodies |
| `storage.status` / `storage.policy` | None / max_events, max_payload_bytes, retention_days, dedup_days | Usage and retention policy |

`config.activate` expresses approval of the exact capabilities shown in the panel.
Remote webhook senders cannot call it. Google OAuth operations (`oauth.google.begin`,
`current`, `session`, `cancel`, `put`, `status` under that prefix) have a separate
[OAuth technical contract](../oauth2.md) and [user walkthrough](oauth-example.md).

## HTTP ingress

POST `/hooks/ID` defaults to `127.0.0.1:8791`; the engine binds only loopback.
Remote access requires explicitly configured TLS proxying. Bodies are limited to
256 KiB. Authentication precedes normalization and admission.

Generic HMAC uses `X-Quatrro-Timestamp` (Unix seconds, five-minute tolerance),
`X-Quatrro-Delivery` and `X-Quatrro-Signature`. The signature is `sha256=` followed
by hexadecimal HMAC-SHA256 over `timestamp.delivery.` plus original body bytes.
Type comes from the authenticated body's `type`, otherwise `webhook`.

GitHub signs original bytes in `X-Hub-Signature-256`. `X-GitHub-Delivery` is
required, but deduplication uses the authenticated body digest: changing a delivery
header cannot replay identical bytes during retention. Legitimate identical bodies
are consolidated too. `X-GitHub-Event` is not signed and is not trusted as type.
There is no signed timestamp, so retained deduplication is not indefinite freshness.

Bearer uses `Authorization: Bearer ...` and requires TLS on public transport.
Slack checks v0 signature/timestamp, answers authenticated challenges without
executions, and returns HTTP 200 for normal events after commit. Other accepted
webhooks return 202 after persistence, not after all actions finish. Deduplication
records are separate from payload retention (30 days by default).

JSON must be an object. URL-encoded forms reject duplicate names. Raw produces
`data.body`/`data.content_type`. XML and multipart use restricted parsing and size
limits; see [payload formats](payload-recipes.md) and [authentication](authentication-recipes.md).

## HTTP egress

Targets are registered HTTPS destinations. `private_hosts` permits exact
`host:port` exceptions without disabling TLS validation. No global exceptions.
Formats are `json`, `form` and `raw`. JSON/form bodies are parsed templates whose
mapped values are encoded for the destination; whole scalar references preserve
their type. Raw uses text. The resolved body is persisted in the outbox.

`Idempotency-Key` stays stable per execution/step. Transport errors, HTTP 408,
429 and 5xx can retry, bounded by eight attempts or 24 hours. The receiver must
implement idempotency to prevent duplicated effects after crashes.

## Storage

Transactional migrations reach SQLite schema 5 and reject future schemas. Defaults:
10,000 events, 64 MiB payloads (events plus private execution contexts), seven days
terminal history and 30 days deduplication. Quotas count UTF-8 bytes. Pending and
uncertain work is not evicted to admit more. The main database is limited to
65,536 pages (256 MiB with 4 KiB pages); periodic WAL consolidation does not
promise the instantaneous database-plus-WAL total equals that cap.

Admission fails without a successful commit when storage is full. Event cleanup
and monitor state changes use transactions. Credential-store values are not
exported; manually inserting secrets into ordinary headers/bodies defeats this.

## Administrative capability check

`administration.check` accepts `{"unit":"backup.service","operation":"restart"}`.
Allowed operations are status/start/stop/restart, with a fixed valid unit and no
identity, script or argv fields. It checks the engine user's administrative policy
without executing systemctl, granting capabilities or activating flows.

A success contains state `authorized`, unit, operation and `effects_executed:false`.
Other states are `denied`, `unavailable` (including invalid/disconnected replies)
and `unsupported` (older broker). The check is bounded to eight seconds without
retry. It does not prove the unit exists or guarantee future permission; dispatch
checks policy and Polkit again.

## UI/CLI envelope

The panel sends every request through `quatrroctl ui --stdin`:

```json
{"contract":1,"operation":"status","data":{}}
```

The CLI validates contract, fields and duplicate keys before resolving paths or
connecting. Concatenated envelopes and reserved hello/ui/hook operations are
rejected. `data` travels only through stdin and the negotiated socket, never argv.
Results retain the operation's normal QML-facing format. New UI does not fall
back to unversioned calls against old CLI/engine; it displays localized upgrade
instructions. Direct manual CLI operations remain available. This envelope is
not an authentication boundary against other processes owned by the same user.

`quatrroctl --version` and `quatrrod --version` return component, version, protocol,
contract and ui_contract JSON without opening profiles, databases or sockets.
The installer verifies these descriptors together before installing components.
