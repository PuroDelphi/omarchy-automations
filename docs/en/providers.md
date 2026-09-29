# Inbound providers

[Español](../es/providers.md) · [Documentation](README.md)

Choose provider authentication on the entry and create its secret reference in
Security. An entry has one format; use separate entries for JSON and forms from
the same provider. Public exposure requires deployment TLS proxying; the engine
keeps its local listener.

## GitHub

The `github` adapter verifies HMAC-SHA256 over original bytes using
`X-Hub-Signature-256`. Tests include the published
[GitHub vector](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries).

Omarchy Automations requires `X-GitHub-Delivery` but derives identity from SHA256
of the authenticated body because that header is unsigned. Identical bodies are
consolidated during deduplication retention. `X-GitHub-Event` does not determine
permissions or event type: filter body fields such as `data.action` and
`data.repository.full_name`. There is no signed timestamp; retained deduplication
does not establish indefinite freshness.

## Slack

The `slack` adapter supports JSON and URL-encoded forms. Use the **Signing Secret**,
not a bot token or legacy verification token. `X-Slack-Signature` verifies
HMAC-SHA256 of `v0:<timestamp>:<original-body>` with five-minute timestamp tolerance,
following [Slack's signature contract](https://docs.slack.dev/authentication/verifying-requests-from-slack/).

An authenticated `url_verification` returns HTTP 200 and JSON `challenge`, as
specified by [URL verification](https://docs.slack.dev/reference/events/url_verification/).
The plugin limits the challenge to 2048 bytes and creates no executions for it.
Normal events receive HTTP 200 with `{}` after commit. Type is the body's `type`;
nested events are under `data.event.type`, for example:

```json
{"field":"data.event.type","op":"eq","value":"app_mention"}
```

Identity binds to the signed body. A retry with a new timestamp or retry headers
does not execute the same body again during retention. Legitimate identical bodies
are consolidated too. Invalid signatures, missing/expired timestamps and duplicate
authentication headers are rejected.

The adapter neither responds to `response_url` nor sends messages by itself.
Interactions wrapped in a form `payload` remain text; universal interaction
parsing is not claimed. Validation used fixtures and local HTTP, not an actual
Slack app or GitHub account integration.

## Adding a built-in provider

Version 1 `inboundAdapter` is in `internal/core/providers.go`, with a static
`inboundProvider` registry. Requests cannot name executable code to download/run.

1. Implement `Verify(request, raw, secret, now)` to validate original bytes,
   reject ambiguity, check freshness when supported and return stable authenticated
   identity. Perform no I/O and store no credentials. `now` enables deterministic tests.
2. Declare `Formats`, `SuccessStatus`, `EmptyAcknowledgement`. The handler
   authenticates before parsing and persists before normal success.
3. Optionally implement `Challenge(data)`, returning response, handled flag and
   error. It receives authenticated parsed data only; keep responses bounded and
   never execute flow effects during a challenge.
4. Add the name to JSON schema and QML selector; validate provider/format combinations.
5. Add an official known vector plus altered-body, replay, duplicate-header,
   error and failed-persistence tests. Document signed fields, deduplication and limits.

External process adapters have separate permissions and isolation; this contract
does not replace them. See the [code examples](code-examples.md).
