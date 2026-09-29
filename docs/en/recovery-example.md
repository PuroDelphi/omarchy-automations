# Recover a failed HTTP delivery

[Español](../es/recovery-example.md) · [User guide](user-guide.md) · [Other examples](use-cases.md)

Use a receiver you control that can return selected HTTP statuses and record the
request body plus `Idempotency-Key`. Use a test endpoint, since retries may repeat
an external effect. This example does not provide or publish a receiver server.

## Prepare the flow

1. Export any draft you want to preserve, then import
   [http-recovery.json](../../examples/use-cases/http-recovery.json).
2. Replace `https://receiver.example.com/events` with your receiver's trusted
   HTTPS URL. Create `receiver-token` in Security with the bearer value expected
   by that receiver. The example hostname is a placeholder, not an operating API.
3. Enable flow `send-demo` and save. Simulate source `local:delivery`, type `test`
   and `{"message":"Build finished","request":"demo-1"}`. Expect the same two
   fields in the resolved outbound JSON, without sending anything.
4. Repeat with `{"message":"Backup verified","request":"demo-1"}` to check a
   second message. Review the destination, authentication reference and action
   capability, then activate.

For private receivers, configure the exact host:port exception and a trusted TLS
certificate; do not disable certificate verification. These settings are part of
the reviewed capability.

## Observe an automatic retry

Configure the receiver to return HTTP 503, then confirm a real test using the first
payload. The execution should remain pending with attempt information in History.
The engine backs off before retrying; it does not immediately send another event.
Transport errors, 408, 429 and 5xx are eligible, subject to the eight-attempt and
24-hour limits. A stored retry delay survives an engine restart.

For this exercise, change the receiver to return HTTP 400 before the next attempt.
After that attempt, expect a terminal failed execution. Ordinary HTTP 400 is not
an automatically retried status. Select Details and inspect the status and attempts;
do not infer success from the local event having been accepted.

## Request a manual retry

1. Correct the receiver so it accepts this request and responds with HTTP 204.
2. Pause the engine. In History, turn off the pending/uncertain-only filter to
   locate the failed execution, then select Retry HTTP.
3. Review the confirmation and accept. The same execution becomes pending; it
   remains paused until you Resume. Cancel in the dialog would leave it failed.
4. Resume. Expect a successful delivery and eventually a completed execution.
   Compare receiver logs: body and `Idempotency-Key` should match the original.

A manual retry resets the attempt counter and retry lifetime for that delivery;
the count is not the lifetime sum of all manual retries. It does not render a new
body from the latest draft. Editing the draft message cannot repair the body of
an already persisted request. If the payload itself is wrong, correct and review
the configuration, then send an intentionally new event instead of assuming Retry
HTTP will rewrite it. Review possible earlier receiver-side effects first.

The receiver must implement idempotency. Reusing a key alone cannot prevent a
receiver from processing the same operation twice. Rotating a credential can fix
authentication for a subsequent attempt; it cannot undo a request already sent.
Changing a destination/configuration and grants may block old queued work rather
than redirect it to a new destination.

## Uncertain and non-HTTP results

Retry HTTP applies only to eligible failed/cancelled HTTP steps. It does not retry
completed deliveries, commands, scripts, service actions or arbitrary uncertain
results. An uncertain result means the external effect may have occurred without
a stored confirmation. Inspect the receiver/service/files involved before deciding
whether to initiate a new event. Do not treat uncertainty as proof that nothing ran.

Pause, stop/cancel and revocation are distinct: pause retains work, cancellation
cannot reverse sent effects, and revocation prevents future dispatch under a grant.
Closing the panel does none of these and leaves the engine running.

## Verification boundary

`TestDocumentedHTTPRecovery` loads the distributed configuration and tests both
simulation messages. Its controlled sender returns 503, then 400, then 204 while
the real worker/outbox handle delay, terminal failure and manual retry. It verifies
pause, stable body/key despite draft edits, completion and rejection of retrying a
completed delivery. To avoid a real wait, the test advances its own persisted retry
deadline. It uses no external network and does not prove receiver-side idempotency
or the deployment's bearer/TLS configuration.
