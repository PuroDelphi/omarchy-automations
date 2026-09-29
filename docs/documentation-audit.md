# Bilingual documentation acceptance

R1.DOCS / R1.6 covers documentation for the current implemented development
version. It does not assert a stable release or completion of external acceptance.
Both language indexes link the same chapters; technical IDs and JSON keys remain
unchanged. The guides explicitly distinguish simulation from real effects.

| User requirement | English / Spanish evidence | Implementation or executable evidence |
|---|---|---|
| Install, update, uninstall, backup and recovery | user-guide.md; system-integration.md | install.py and installer/runtime/uninstall tests; owned-file receipts and real temporary-unit removal |
| First use, language, navigation, widget | user-guide.md; interface.md | persistent preferences, native input tests, actual installed captures |
| Every option, defaults and constraints, multiple examples | options.md and option-coverage.md | 119 catalog rows, at least two alternatives per row; Schema.js and panel-field mapping |
| Inbound/outbound formats and authentication | payload-recipes.md; authentication-recipes.md; provider-example.md | parsePayload/providers/rules/egress; ten payload fixtures, eight signature vectors and signed provider test |
| Conditions, sources and ordered actions | user-guide.md; payload-recipes.md; action-recipes.md | typed condition matching, simulation and action validation; documented examples tests |
| All monitor variants and schedules | monitor-recipes.md; use-cases.md | 24 validated monitor configurations; strict-boundary tests; actual monitor/scheduler acceptance |
| Commands, typed scripts, directories and adapters | action-recipes.md; code-examples.md; options.md | command profiles, prepared identities/revisions, argument contracts and isolated execution tests |
| Credentials, OAuth, permissions and revocation | user-guide.md; oauth-example.md; security.md | credential bounds, local OAuth fixtures, permission checks and queued-work revocation test |
| Pause, cancel, history, retry and uncertainty | user-guide.md; recovery-example.md; options.md | operations/storage/worker reviewed; persisted retry-body/key and uncertain-result tests |
| Import/export and diagnostics | user-guide.md; options.md | bounded file operations, new-file requirement, secret references versus ordinary configuration content |
| Optional hooks and administration | system-integration.md | local hook lifecycle, broker policy/install/recovery tests and isolated systemd/Polkit checks |
| Complete use cases with prerequisites/results | use-cases.md and linked code/provider/service/recovery/OAuth chapters | distributed JSON fixtures, TestDocumented suite, host acceptance records |
| Screenshots, glossary, troubleshooting, publication steps | interface.md; user-guide.md; github.md | native and development captures with provenance; local Git destination and prepared CLI |

## Semantic review

Reviewed source contracts include resource defaults/visibility in Schema.js,
model validation, parsePayload and XML/multipart normalization, provider signing
and deduplication, rules and template typing, command profiles, monitor transitions,
timer policies, storage bounds and retry limits. Recipes extend short option values
with variant-specific examples, prerequisites, effects and expected failures.

The two languages cover the same workflows and option IDs. Example resource names
are user content, not translated API tokens. Public authentication fixtures are
explicitly unsuitable as real credentials. Raw fixture files are distinguished
from importable configurations. Examples requiring user paths, service units,
HTTPS receivers or Google credentials identify those substitutions.

The walkthroughs state the actual integration limits: Google consent with a real
account remains pending; fixture OAuth is not live-provider evidence. Hooks do
not guarantee delivery during logout/suspend. Plain-file secrets are unencrypted,
receiver idempotency is external, and cancellation cannot undo completed effects.
No stable-release claim is inferred from a documentation or packaging check.

## Repeatable checks

- `python3 scripts/check-docs.py`: paired chapters, reciprocal language links,
  local files/anchors and generated option/schema coverage. This is structural.
- `go test ./internal/core -run '^TestDocumented' -count=1`: distributed configurations,
  payloads/signatures, code revisions, simulation and recovery behavior.
- `python3 scripts/test-release-package.py`: guides and use-case configurations
  match the source; local guide links work after extraction; install/update/remove.

The human/source review above supplies the semantic evidence that link and schema
checks alone cannot establish. Future option or behavior changes require updating
both languages and rerunning the corresponding examples and structural checks.
