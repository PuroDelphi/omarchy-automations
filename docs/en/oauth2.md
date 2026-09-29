# Google OAuth2 technical contract

[Español](../es/oauth2.md) · [Illustrated example](oauth-example.md)

F3.3 includes Desktop consent, refresh-token import/renewal, HTTP destinations and
bilingual UI. Tests use temporary profiles, real loopback and a simulated provider.
A real Google account remains deferred. References:
[Desktop applications](https://developers.google.com/identity/protocols/oauth2/native-app),
[token renewal](https://developers.google.com/identity/protocols/oauth2/web-server).

## Connect from the panel

1. Security → Create or rotate credential → Google OAuth2 browser authorization.
   Use your own Google **Desktop** client ID/secret, local connection ID and
   current scopes required by the APIs.
2. Explicitly choose desktop store or unencrypted private file. Save starts a
   ten-minute local session, without granting flow permissions.
3. Open consent in the browser and review Google's permissions. Enter your account
   password only on the provider's website.
4. Return and inspect the result. Cancel stops waiting/exchange; success adds the
   connection under Credentials.
5. Select `oauth2-google` on an allowed destination, reference the connection ID,
   and review/grant flow permissions separately.

Token import stores existing authorization without contacting the provider or
verifying scopes. Sensitive inputs are masked and cleared on save/cancel. The
project distributes no Google client credentials of its own.

## Session and recovery

Uses PKCE S256, independent 256-bit state/verifier and one exchange. The listener
binds only `127.0.0.1` at an ephemeral port, path `/oauth/callback`. It validates
host, path, GET, state, deadline and unique critical parameters. Malformed callbacks
do not consume the session. Initial exchange requires refresh token and every
requested scope.

Headers are bounded to 16 KiB; read/write deadlines are three seconds, headers two,
with keep-alive disabled. Bodies are rejected and URLs/codes are not logged. The
fixed bilingual response has no-store, no-referrer and resource-free CSP. Completion,
cancellation and expiry close the port; remaining connections are force-closed
after one second of graceful shutdown.

Panel reload preserves the engine session: refresh Security to recover state and,
only while waiting, its link. Failed status queries stop polling and ask for refresh.
Engine restart discards sessions and needs new authorization. Engine shutdown
cancels/waits for its session before closing SQLite.

Cancellation and persistence are serialized. Accepted cancellation during waiting
or exchange prevents storing a late response. Before save, credentials are compared
with those seen initially; appearance, removal or change rejects the result.
Cancel does not revoke a connection already saved.

## Tokens, scopes and storage

Client ID/secret, refresh/access tokens, expiry and scopes are stored together in
the chosen backend within its 8192-byte limit. They are not exported or returned
by API. Status exposes metadata only, including expiry and requested/granted scopes.

Renewals are serialized. Every request rereads storage with no independent cache.
Tokens with over sixty seconds remaining are reused. Renewal persists tokens before
returning access. Network I/O does not lock out credential changes/deletion: later
save compares the original value and discards stale responses. No backend fallback.

Scopes compare literally, without historical-alias translation. Partial consent or
missing scope on initial exchange yields `scopes_missing` without saving. Extra
scopes remain metadata, not flow grants. Explicit scope reduction during renewal
blocks use until reconnect; omitted renewal scopes retain known ones. Parsing
allows 8192 bytes, 64 scopes and 512 bytes per scope, without duplicates/controls.

## Destinations and failures

Initial hosts are exactly `www.googleapis.com` and `calendar.googleapis.com`, HTTPS
port 443 or implicit, with no userinfo, fragments or LAN exceptions. Validation
repeats at dispatch. Public transport validates/pins DNS, ignores environment
proxies and rejects redirects. Token endpoint is fixed at
`https://oauth2.googleapis.com/token`, timeout fifteen seconds, response 32 KiB.

Flow grants bind destination, OAuth mode and connection ID. The sender attaches
only the access token. Valid OAuth credential sets cannot be reused as generic
Bearer/HMAC. Errors are static and expose no provider body or transport detail.
Temporary/concurrent-change errors permit queue retry; terminal errors need user
intervention. `invalid_grant` requires reconnect and alone cannot distinguish
expiry from remote revocation.

HTTP 401 terminates delivery without immediate repeat. It marks `token_rejected`
and clears access only if the private credential revision matches the one sent,
retaining refresh for a later delivery. Late rejection cannot change replacement
credentials. Persistence failure is reported terminally without exposing secrets.
Deleting local credentials **does not revoke** Google consent.

Connection states: unavailable, renewal_required, available, expired,
reconnect_required, client_rejected, temporary_error, invalid_response,
token_rejected, scopes_missing. Status does not contact Google or establish remote validity.

## Local API

| Operation | Input/result |
|---|---|
| `oauth.google.put` | ID, backend, client_id, client_secret, refresh_token; returns ID/provider/state |
| `oauth.google.status` | ID; metadata state/expiry/scopes without secrets |
| `oauth.google.begin` | ID, backend, client_id, client_secret, scopes; session, ID, state, URL |
| `oauth.google.current` | Empty object; current session or none; URL only while waiting |
| `oauth.google.session` | session; ID/state |
| `oauth.google.cancel` | session; cancel waiting/exchange and return state |
| `secrets.delete` | ID; remove local connection |

Only one session can be active. States: waiting, exchanging, completed, cancelled,
denied, failed, credential_changed, storage_failed, scopes_missing. A new session
can replace a completed reference. Authorization URLs contain client ID, scopes,
state and challenge, never client secret, verifier, authorization code or tokens.

## Verification limits

Tests cover PKCE, replay, callbacks, socket closure, exchange/renewal, concurrency,
persistence, replacement/deletion, scope reduction, limits, redaction, destinations
and worker delivery. QML covers import, begin, recovery and cancellation using a
fake client without opening a browser.

Real browser consent with the user's Google client/account remains deferred.
OAuth persistence paths were tested with temporary private files; the keyring
backend reuses the existing store. Native UI review is complete, while real OAuth,
session and stable-release gates remain open. See the [validation log](../validation.md).
