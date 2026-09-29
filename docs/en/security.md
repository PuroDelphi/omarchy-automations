# Security boundaries and verification

[Español](../es/security.md) · [User guide](user-guide.md) · [Permissions and options](options.md)

Remote event data does not grant permissions. Activation binds grants to reviewed
resources and revisions; dispatch checks them again. Keep inbound credentials
private and review what each enabled flow can do. An authenticated sender can
trigger the effects you authorized for its matching flows.

The local control socket is restricted to the same Linux user. This is not a
defense against malware already running as that user, an administrator, a
compromised kernel or a malicious code revision you explicitly approved.
Scripts and commands run with isolation checked on the host; directory grants
expose the whole selected tree, including any sockets inside it.

| Boundary | Verified evidence |
|---|---|
| Signatures and replay | HMAC body/timestamp/delivery binding, GitHub and Slack signature vectors, tampering, duplicate headers, expired timestamps where supported, persisted deduplication. GitHub identity derives from signed bytes rather than its unsigned delivery header. |
| Outbound addresses | IPv4/IPv6 restrictions, rejection of mixed public/private DNS answers, exact host:port exceptions, validated addresses dialed directly, changed answers rechecked on a new connection, redirects not followed and environment proxies ignored by the transport. |
| Injection | Templates do not evaluate code; JSON/form escaping, typed literal script arguments, notification option separation and escaped markup; unsupported/ambiguous configuration rejected. |
| Revocation | Changed revisions require review. Revoked action, monitor, timer and private-destination permissions block corresponding future work, including queued/retry cases covered by tests. |
| Files and socket | Private directories, mode-0600 socket and credential files, peer UID check; secret symlinks, non-regular files, unsafe modes and oversized values rejected. FIFO opening is nonblocking before type validation. |
| Effective isolation | Real systemd/Bubblewrap tests cover cgroup CPU, memory and task limits; filesystem/network/environment restrictions; timeout and descendant cancellation. |
| Disclosure | Credential values excluded from configuration exports, diagnostics and operational queue views; controlled OAuth failures redact provider/transport details; script environment does not inherit fixture secrets. |
| Optional administrator broker | Exact unit/operation/UID policy, kernel peer identity, Polkit and repeated authorization checks; no arbitrary root command interface. Separate isolated systemd/Polkit integration tests are documented in the validation log. |

Representative test families are `ingress_test.go`, `providers_test.go`,
`egress_test.go`, `lan_test.go`, `config_test.go`, `events_test.go`,
`secrets_test.go`, `diagnostics_test.go`, `scripts_test.go`,
`isolation_host_test.go`, `internal/local` and `internal/broker`. The
[validation log](../validation.md) records commands, outcomes and integration scope.

## Practical limits

- A GitHub signature does not provide a trusted event timestamp. Deduplication
  has a retention window; it does not guarantee freshness forever or exactly-once
  external effects. Receivers must implement idempotency where needed.
- HTTPS private-host exceptions authorize one host and port, with current address
  validation. Do not treat a permitted hostname as a guarantee that the remote
  application or its DNS operator is trustworthy.
- Explicit file credential storage is plaintext protected by file permissions.
  Desktop keyring access can fail when unavailable or locked; it must not silently
  fall back to plaintext. Real-account Google OAuth/keyring integration remains a
  separate validation item. Do not put secrets in script code or ordinary payloads.
- Exported configuration includes your ordinary text and approved code, even
  though credential-store values are excluded. Review exports before sharing.
- Revocation and cancellation cannot undo an effect already accepted elsewhere.
  Investigate uncertain outcomes before repeating work.

## Reproduce the checks from a source checkout

```bash
go test -race ./...
QUATRRO_HOST_TEST=1 go test ./internal/core -run '^TestHostIsolation' -count=1 -v
```

The second command creates temporary user units and exercises real resource
limits; it needs the supported user systemd/Bubblewrap environment. Ordinary
tests skip integrations that require explicit host opt-in. DNS-changing tests
use a controlled resolver and local TLS receiver, not a public DNS service.
These are project tests and a source review, not an independent penetration test
or a claim that every possible vulnerability has been eliminated.
