# Scheduling

[Español](../es/scheduling.md) · [Guided examples](use-cases.md)

Schedules supports intervals and calendars. Save the draft, review permissions
and activate. Each enabled schedule requires `timer:<id>`, bound to its exact
configuration. A flow selects source `timer:<id>`; each action retains a separate grant.

## Intervals

Intervals range from five seconds to 365 days. The first due time is calculated
from the first evaluation of a new timer, approximately one second after activation.
Intervals use UTC elapsed time, not local clock time. Delays retain the original phase.

## Calendars

Configure one `HH:MM` time, an explicit IANA timezone such as `America/Bogota` or
`UTC`, and optional weekdays: `mon`, `tue`, `wed`, `thu`, `fri`, `sat`, `sun`.
No weekdays means daily. Implicit `Local` is not supported. The build embeds tzdata
as fallback when the system has no timezone database.

Scheduling begins at the next occurrence after initial evaluation. Nonexistent
times during a clock advance are skipped. Only the first occurrence of a repeated
hour is eligible, even when the engine starts during the second occurrence.
This is not cron syntax: create separate timers for multiple times per day.

## Missed times, pause and restart

Select an explicit policy:

- `coalesce`: generate one event for accumulated due times and calculate the next
  future occurrence. The event identifies the first pending time, not every missed one.
- `skip`: discard intervals more than five seconds late or calendars more than
  sixty seconds late. Within tolerance, generate the event.

Next due time and event are stored in one SQLite transaction. Restart does not
reset the interval. Failed persistence does not consume the occurrence; it is
reevaluated under the missed policy. A backwards clock change waits until the
stored next due time is reached again without repeating earlier occurrences.

Pausing with admission `retain` stores events for later effects. With `reject`,
no event is saved and its occurrence stays pending; resumption applies the missed
policy. Revoking a timer stops new timer events, but queued events retain separate
action permissions. Revoke actions or pause/cancel to stop those effects.
Reactivating an identical timer preserves its pending due time; changing its
configuration starts a new schedule.

## Event and diagnostics

Type is `scheduled`, with data such as:

```json
{"timer":"backup","scheduled_at":"2026-09-28T14:00:00Z","fired_at":"2026-09-28T14:00:01Z","late_seconds":1,"policy":"coalesce"}
```

Templates can use `{{data.scheduled_at}}`. Event timestamps are UTC.
`quatrroctl timers.status` reports authorization, next due time and last event for
active timers. The panel displays the next run in the desktop's local timezone.

Up to 64 timers are supported per configuration. Evaluation is approximately once
per second: it is not real-time scheduling and does not wake a suspended machine.
Suspend/logout delays are handled like other delays. Import disables all timers.
