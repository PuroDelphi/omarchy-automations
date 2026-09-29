# Operations and diagnostics

[Español](../es/operations.md) · [User guide](user-guide.md)

## Inspect a queue

In History, enable the pending/uncertain-only filter to see pending, running and
uncertain jobs in pages of 100. Normal history shows the latest 100 executions.
Equivalent CLI:

```bash
quatrroctl queue.inspect '{"state":"pending","offset":0,"limit":100}'
quatrroctl queue.inspect '{"state":"uncertain","offset":0,"limit":100}'
quatrroctl history.detail '{"id":"EXECUTION_ID"}'
```

`queue.inspect` returns items, total, offset, limit, has_more. Rows contain ID,
flow, state, step, creation, attempts and next HTTP retry, without events, outbound
bodies or credentials. This is a live queue: pages can change between queries.

A completed step has finished even when the execution awaits another step. Pending
HTTP awaits an attempt. Older successful steps stored internally as pending are
presented as completed without rewriting records. Permission-blocked steps show
denied with a reason. Revocation cannot undo earlier effects.

Pause lets you inspect work before continuation. Admission retain keeps new events;
reject refuses them. Cancel cannot reverse started effects. Never blindly repeat
uncertain work. Manual retry is limited to eligible failed/cancelled HTTP and keeps
original idempotency key/body.

Panel Pause retains new events and stops future dispatch; Resume continues processing.
Stop executions requests confirmation: cancel changes nothing; accept cancels
pending work and attempts to stop active processes. It does not replace pause or
prevent later admission. Closing the panel neither stops nor pauses the engine.

For failed HTTP recovery:

1. In History, disable the queue filter to find terminal failures. Open Details
   and inspect HTTP status/attempts.
2. Fix the receiver. Select Retry HTTP and review the idempotency warning; cancelling
   the dialog sends nothing.
3. Confirm. The same execution becomes pending and completes if the receiver accepts.
   Inspect its final status. Pause and permissions still apply.

Manual retry resets the attempt counter; it is not a lifetime total across manual
cycles. Reusing a key cannot prevent duplicates unless the receiver implements idempotency.

## Simulate first

Simulate / test accepts source, type and JSON sample data. Simulate without effects
uses the draft (saving changes first) without activating grants, reading credentials
or creating executions. Results explain disabled flows, different sources and
matching conditions, and show resolved parameters. Conditions of disabled/different
source flows are evaluated for explanation only.

Adapters are not executed; dependent steps are marked as requiring their result,
without fabricated output. Run real test has separate confirmation: accepting emits
against the active revision and permissions and can cause effects. Cancel emits
nothing. Real tests accept local/hook sources, not a substitute for inbound signature validation.

## Diagnostics for sharing

Security offers View diagnostics and Export diagnostics, creating a new mode-0600 JSON file:

```bash
quatrroctl diagnostics
quatrroctl diagnostics.export-file '{"path":"/absolute/path/diagnostics.json"}'
```

Report version 1 contains engine/protocol/database versions, platform, pause state,
a known-list operational error, resource/queue counts, quotas/retention and tool
availability on PATH. Executable presence does not prove a working service/session;
isolation is checked separately at command activation.

The report uses allowed fields only: no resource names, configuration, URLs, paths,
environment, event bodies, headers, secret values or arbitrary journal messages.
Nothing is automatically shared with any service.

## Portable configuration

Export saves the draft and credential references, never reads secret values.
Ordinary manually entered fields (headers/action bodies) remain exportable; use
credential references instead of embedding secrets.

Export requires a new absolute `.json` path, never overwrites or follows a final
symlink. Failed writing can leave an incomplete file; the error says so, and it
must be removed before reusing that name. Portable limit is 1 MiB; compact JSON
is used when indentation would exceed it.

Import reads a bounded regular file without following a final symlink; FIFO/devices
are rejected. It replaces only the draft and disables entries, flows, monitors and
timers, without affecting active revision or grants. Create referenced credentials,
review resources and explicitly activate afterward.

## Credentials and retention

Create or rotate credential replaces an existing reference in the same backend.
Subsequent receiver/sender reads use the new value; already authenticated/in-flight
requests cannot be undone. Deletion breaks connections still referring to it.
Changing between Secret Service and file requires deleting then recreating the
reference, avoiding an unmanaged old copy.

Missing outbound credentials never cause unauthenticated delivery: work stays
pending under retry policy. Restoring the reference lets a later attempt use its
current value. After retries exhaust, use eligible manual retry.

Private files are explicitly selected plaintext with restrictive permissions.
Absent/locked Secret Service returns an error without silent backend fallback.
Security controls quotas/retention. Cleanup retains pending/running/uncertain work;
deduplication has its own window. `quatrroctl storage.status` reports policy/usage.

## Notifications and session

Notifications use the user's session D-Bus service. Title is plain text up to
200 UTF-8 bytes; body up to 4096 bytes before markup escaping. Event-provided tags
appear literally, without formatting, images or active links.

Unavailable service causes a static failed action, not automatic replay on login.
There is no separate queue for notifications to show later. Completion means the
service accepted it; do-not-disturb, desktop policy or locking may hide it.

The user service belongs to `default.target`. `After=graphical-session.target`
orders startup; it does not stop the engine when that target stops. Neither panel
closure nor shell shutdown stops it. User-manager shutdown stops the service;
restart recovers persisted work. Manager lifetime after logout depends on logind
and linger. The installer neither enables linger nor promises unattended operation.
Notifications/keyring still depend on session services. Suspension stops execution;
timers apply their missed policy on resume.
