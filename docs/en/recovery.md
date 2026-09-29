# Recovery and uncertain outcomes

[Español](../es/recovery.md) · [HTTP recovery walkthrough](recovery-example.md)

The engine acknowledges input only after saving the event and executions. Each
execution retains its configuration revision. Restart does not replace actions
with a newer draft; permissions are checked again before the next effect.

| Interruption boundary | Restart behavior |
|---|---|
| Input before commit | No acknowledgement; sender must retry |
| Persisted input, pending step | Continue pending step |
| Local action marked running, result not committed | `uncertain`; no automatic replay |
| Local action and step advance persisted | Continue next step |
| HTTP sent, response not persisted | Retry same body and `Idempotency-Key` |
| HTTP delivered, step advance not persisted | Advance without resending |
| HTTP failure or uncertain cancellation persisted, step not closed | Preserve failure/uncertainty without resending |
| Next HTTP retry persisted | Respect its due time after restart |

External effects and SQLite do not share a transaction, so exactly-once execution
is not promised. The HTTP receiver must implement idempotency; keeping a key alone
cannot prevent its duplicate effects. The gap between reserving a local action
and dispatching can produce uncertainty even if no effect occurred yet.

Each HTTP attempt is reserved before dispatch. A crash consumes it even before
sending; restart cannot bypass the eight-attempt limit. Confirmed cancellation
of in-flight HTTP remains uncertain if the engine dies before the network call ends.

## User intervention

Cancel first commits cancellation of pending work and uncertainty of in-flight
HTTP in a transaction, then requests termination of active jobs. Failed commit
returns an error without acknowledging or signalling cancellation. A dispatched
action may have occurred; inspect its service/receiver before creating new work.

Retry delivery accepts only eligible failed/cancelled HTTP awaiting dispatch,
not completed or uncertain executions. It keeps body/key, resets the attempt
budget, and still requires authorization.

If persistence fails after an effect, the engine keeps the job running and
reports a storage error. It does not execute it again in that process. Fix storage
and restart to apply the table above; do not edit SQLite states directly.

Without a session notification service, notification fails generically and is
not automatically retried on login. D-Bus success proves service acceptance,
not that the user saw the notification.

## Evidence and current limits

Tests SIGKILL an independent process before dispatch, after reservation, after
the effect, after step commit, after HTTP send and after cancellation confirmation.
They reopen WAL without clean shutdown and check integrity, state and retry
identity. A file marker models the external effect; this is not physical power loss.

Injected failures cover outbox creation, execution reservation, attempt reservation,
result persistence and step persistence after terminal HTTP results. Tests verify
no sending before reservations commit and no repetition of running work. Timer
and monitor tests cover discontinuous clocks and transactional cursor/state rollback.

Local-action tests cover reservation, step insertion and execution advance:
pre-effect failures allow continuation; post-effect failures retain running state
without a partially committed step and reopen as uncertain. Real SQLite page-limit
tests require SQLITE_FULL code 13, no partial event/execution/dedup rows, and retry
of the same event after raising the limit. This does not fill the user's filesystem.

`TestFilesystemFullRecovery` fills a private 16 MiB tmpfs until kernel ENOSPC, with
the host filesystem read-only. It truncates WAL first to force new writes, requires
SQLite storage failure without partial rows, frees space, retries the same ID and
reopens to check integrity and one execution. It runs with `QUATRRO_HOST_TEST=1`.
This covers ingress storage exhaustion, not every external-effect boundary or
physical device failure.

`scripts/test-service-lifecycle.py` uses a temporary unit with packaged settings
and a private graphical target. Stopping that target retains engine/admission;
freezing its cgroup for twelve seconds checks timer coalescing/skipping. Stop/start
retains pause/history. Freezing is not physical suspend; a private target does not
reproduce full user-manager logout.

Real session/suspend acceptance remains deferred under F2.7. The user's session
was not suspended or logged out for these tests.
