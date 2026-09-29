# Webhook authentication recipes

[Español](../es/authentication-recipes.md) · [User guide](user-guide.md) · [Payloads](payload-recipes.md)

Create a generic credential in Security and use its ID in the entry's Secret
reference. Every entry requires authentication. Use a private value for a real
integration; the [eight test vectors](../../examples/authentication-vectors.json)
use a deliberately public fixture value and a fixed clock. They are not live
credentials or importable plugin configuration. Each provider has two signed or
authenticated bodies, one failed build and one passed build.

## Generic HMAC

Example A: entry `deploy`, credential `deploy-signing`, JSON body
`{"type":"build","state":"failed"}`. Example B: entry `backup`, credential
`backup-signing`, JSON body `{"type":"backup","state":"passed"}`.
For either entry the sender computes HMAC-SHA256 over these exact bytes:

```text
ASCII(timestamp) + "." + delivery_id + "." + original_HTTP_body
```

Send `X-Quatrro-Timestamp` as canonical Unix seconds, `X-Quatrro-Delivery` as the
nonempty delivery ID, and `X-Quatrro-Signature` as `sha256=` plus hexadecimal HMAC.
The timestamp must be within five minutes of the receiver's clock. Retrying the
same event should retain its delivery ID; use a new ID for a genuinely new event.
The ID is signed. Changing the body or ID without recomputing the signature fails.
The retained deduplication window is finite; authentication is not a promise of
unlimited exactly-once delivery.

## Slack

Use the app's Signing Secret, not a bot token. Example A: entry `mentions`, JSON,
condition `data.event.type eq app_mention`. Example B: entry `commands`, form,
condition `data.command eq /backup` (Text). Use separate entries for the formats.
The verifier accepts only JSON/form for Slack and checks:

```text
X-Slack-Request-Timestamp: canonical Unix seconds
X-Slack-Signature: v0= + hex(HMAC-SHA256(secret, "v0:" + timestamp + ":" + original_body))
```

The same five-minute clock tolerance applies. A signed JSON `url_verification`
request returns its challenge (maximum 2048 bytes) without running a flow.
Normal events receive HTTP 200 with `{}` after persistence. Internal JSON event
fields remain under `data.event`; an interaction inside form field `payload`
remains text, not automatically decoded JSON. No implicit response is sent to
`response_url`. An outbound destination must be registered and authorized.

Identical signed Slack bodies deduplicate even with a new timestamp or changed
retry headers. This also coalesces legitimate identical bodies during retention.
The test vectors exercise this implementation locally; no real Slack workspace
integration has been verified.

## GitHub and bearer

For GitHub, example A filters a release's repository and action; example B filters
a deployment status and environment. See the [complete signed release example](provider-example.md).
Send `X-Hub-Signature-256: sha256=<hex HMAC of original body>` and a nonempty
`X-GitHub-Delivery`. GitHub event/delivery headers are not body-signed; this plugin
uses body identity and authenticated body conditions. There is no signed timestamp
in this contract, so retained deduplication does not prove indefinite freshness.

For bearer, example A is a CI sender to entry `builds`; example B is a local job
to entry `backups`. Each sends `Authorization: Bearer <its private value>`.
An optional `X-Quatrro-Delivery` provides the delivery ID. Bearer has no signed
clock freshness or payload binding; use TLS for remote traffic and protect the
credential. It is not OAuth merely because the header says Bearer.

## Verify the boundary

Configure a flow for the exact entry source and enable/review it only when real
effects are intended. For a positive test send the unchanged signed body; for a
negative test alter one body byte without updating its signature. HMAC, Slack and
GitHub must reject the latter. For bearer test an incorrect token instead: bearer
does not sign the body. Repeated authentication headers are rejected, rather than
choosing one ambiguously. Normal non-Slack acceptance is HTTP 202 after durable
persistence; History reports subsequent action outcomes.

The vector test uses a fixed verification clock and never sends HTTP. Real senders
must generate current timestamps and private signatures locally. Simulation tests
flow conditions and templates, not signatures or network transport.
