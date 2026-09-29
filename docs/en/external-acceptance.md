# Acceptance tests with the user

[Español](../es/external-acceptance.md) · [Guide](user-guide.md)

These tests complete F3.3 and F2.7. Preparing the procedure does not constitute a
pass. Record date, version, outcome and redacted errors; never tokens, client
secrets, full authorization URLs or private store contents.

## Real Google connection

1. Select your test project in Google Cloud and enable Calendar API. Configure
   Google Auth Platform for your account; for an external app in testing, add
   your account as a test user.
2. Under Google Auth Platform → Clients → Create client, choose **Desktop app**.
   Keep the client ID and client secret locally outside the repository.
   See the [official guide](https://developers.google.com/workspace/guides/create-credentials).
3. In the plugin, Security → Create or rotate credential, select Google OAuth2
   browser authorization, ID `google-calendar`, your own client and scope
   `https://www.googleapis.com/auth/calendar.freebusy`. Select the desktop store
   and unlock it if prompted. Silently switching to private file storage does
   not resolve a desktop store failure for this test.
4. Save and open consent using the plugin button. Sign in, review the permission
   and return to the panel. Expect `available` and verify granted scopes. Being
   signed into Google in the browser alone is insufficient.
5. Follow the [Calendar example](oauth-example.md) to preserve your draft,
   import, simulate, review and activate the flow. Run a test query for an
   accessible calendar and check completed delivery and HTTP status. The plugin
   discards the response body: this does not prove per-calendar results.
6. Record `expires_at` from OAuth status. After the actual access token expires,
   run another query. It must complete without fresh consent and show a later
   expiry. Do not change the system clock or edit the secret to simulate this.
7. After the session cycle below, verify the connection persists and another
   query works with the desktop store unlocked.
8. At the end, disable the flow and activate that revision. Keep the connection
   only if you want to use it; deleting it locally does not revoke Google consent.
   Manage remote revocation in your Google account.

Minimum evidence: version, chosen backend, state/scopes without secrets, first
HTTP result, expiry before/after, second delivery and persistence across login.
Local revocation, error and redaction tests supplement this procedure; they do
not replace real consent and renewal.

## Suspend and logout

Perform each transition separately and save your work first. Logging out may
interrupt this conversation. These transitions are not performed automatically.

1. Beforehand, record engine status, active configuration and draft, grants,
   queue and existing execution IDs. Export the draft for recovery and keep
   local evidence outside the repository.
2. Suspend using the usual Omarchy menu and resume. Open the plugin and verify
   it responds, retains language/configuration/grants, resumes listening and
   correctly displays previous executions. Inspect History before repeating
   any effect reported as uncertain.
3. Run a uniquely identifiable new test of a reviewed flow and verify one
   expected result. Compare the queue with the earlier evidence. Proving no
   duplicates also requires inspecting the receiver when using HTTP.
4. Save your work again, log out through Omarchy and log back in. Repeat the
   checks and run a new test with a different identifier.
5. Record startup errors, configuration loss, repetitions or uncertain states.
   An initially empty queue proves only idle recovery, not recovery of actions
   in progress. Automated crash tests cover other boundaries and must be cited
   separately.

Do not mark F2.7 passed merely because a process is active: compare before/after
state and test results. Do not mark F3.3 passed until real consent, use, renewal
and persistence have been observed.
