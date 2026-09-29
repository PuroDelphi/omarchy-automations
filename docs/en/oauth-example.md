# Google OAuth: an authenticated Calendar request

[Español](../es/oauth-example.md) · [User guide](user-guide.md) · [Other examples](use-cases.md)

This example sends a Calendar availability query without creating calendar events.
It demonstrates an authenticated outbound POST. The plugin records delivery status,
**not the response body**: it does not display availability, feed returned calendar
data into later steps or inspect per-calendar errors inside a successful response.
A completed delivery is therefore not proof of calendar-level success.

## Prepare your Google client

Use your own Google Cloud project, enable Calendar API and configure OAuth consent
for your test account. Create OAuth credentials of type **Desktop app**, retaining
its client ID and client secret locally. Account/project restrictions and consent
requirements apply. Follow [Google's installed-app setup](https://developers.google.com/identity/protocols/oauth2/native-app).

Do not paste these credentials or tokens into chat, source code or the importable
configuration. The plugin has no bundled Google client. A real account integration
has not yet been verified for this release; local fixture tests do not close that gate.

## Authorize in the plugin

1. Open Security → Create or rotate credential.
2. Select Google OAuth2: authorize in browser. Set local ID `google-calendar`,
   your Desktop client ID/secret and scope
   `https://www.googleapis.com/auth/calendar.freebusy`.
3. Choose the desktop credential store if available. The alternative private file
   store is explicitly unencrypted and is not selected automatically on failure.
4. Save to start the ten-minute authorization session. Choose Open consent in
   browser, select your test account and review the requested permissions.
5. Return to Security and inspect the connection state. Refresh can recover the
   waiting session after panel reload; an engine restart requires a new session.
   Cancel authorization stops a pending session but does not revoke a saved connection.

If you already have a suitable refresh token issued to that client, Google OAuth2:
import token is a separate option. It saves client ID, client secret and refresh
token without contacting Google or verifying its scopes at import time. Do not
substitute an access token for a refresh token.

## Import, simulate and activate

Export the draft you want to keep, then import
[google-oauth.json](../../examples/use-cases/google-oauth.json). It has destination
`google-freebusy`, authentication `oauth2-google`, reference `google-calendar` and
a disabled flow `calendar-query` from `local:calendar`.

The destination uses POST `https://www.googleapis.com/calendar/v3/freeBusy`, an
endpoint accepting the scope above. It takes an interval and calendar identifiers.
[Official Freebusy request contract](https://developers.google.com/workspace/calendar/api/v3/reference/freebusy/query).

1. Enable the flow and save. Simulate source `local:calendar`, type `test` with:

   ```json
   {
     "start": "2030-01-07T09:00:00Z",
     "end": "2030-01-07T10:00:00Z",
     "calendar": "calendar-a@example.com"
   }
   ```

   Expect an outbound body with `timeMin`, `timeMax` and one `items[].id`. The
   example calendar identifier is fictitious. Simulation requires no Google token.
2. Replace only `calendar` with `calendar-b@example.com` for a second simulation;
   expect the new identifier in the request. These are template checks, not real
   calendar access checks.
3. For a real request, replace the identifier with one accessible to your test
   account, and choose the interval you intend to query. Review the fixed Google
   destination, credential reference and flow capability, then activate.
4. Confirm Run real test. Inspect History for HTTP status and completion. The body
   of the Google response is discarded; inspect API-specific results separately
   when you need to confirm calendar availability or per-calendar errors.

Do not add an Authorization header manually. The engine fetches the access token
from the selected private store and adds it for the supported Google destination.
Only `www.googleapis.com` and `calendar.googleapis.com` on HTTPS port 443 are
allowed in this mode; no private network exceptions or redirects are supported.
Authorization consent and permission to execute the flow are separate approvals.

## Troubleshooting and disconnecting

- A locally available token is not proof of remote validity. OAuth2 status reads
  metadata without contacting Google.
- `reconnect_required` or `scopes_missing` needs a suitable new authorization;
  check that the requested scopes were actually granted.
- `client_rejected` requires checking the client configuration. Temporary provider
  failures may be retried by pending delivery handling.
- HTTP 401 ends that delivery rather than immediately repeating it. Inspect the
  connection before requesting an eligible manual HTTP retry.
- Deleting `google-calendar` stops subsequent authentication using that reference;
  it does not revoke consent in Google. Remove the provider-side consent separately
  if that is your intended result.

## Verification boundary

`TestDocumentedGoogleOAuth` validates the actual distributed JSON and simulates both
calendar identifiers without any credential. With a fixture credential and controlled
HTTP transport, it checks method, exact URL, access-token header and body. Deleting
the credential prevents another transport call. It does not perform real browser
consent, refresh against Google, TLS/DNS communication or a real calendar query.
The full real-account test remains pending under F3.3.
