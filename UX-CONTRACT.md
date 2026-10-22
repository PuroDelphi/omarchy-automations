# Automation editing contract

Sources: `docs/en/architecture.md`, `docs/en/hooks.md`,
`docs/en/protocol.md`, and `docs/en/security.md` define lifecycle and permission
rules. This file records the interface consequences.

| Capability | Canonical owner | Behavior | Verification |
|---|---|---|---|
| Select/Listbox | `qml/ChoiceField.qml` / native `qs.Ui.Dropdown` | Labels can be localized independently of stable values; advanced source entry is explicit | native QML workflow tests |
| Form | `qml/Field.qml`, `qml/Schema.js`, `qml/ResourceEditor.qml` | Reuse schema and validation for standalone and inline actions | `scripts/test-visual-flows.py` |
| CRUD | `Panel.qml` draft configuration | Inline actions and flow are staged together; cancel discards both; saving does not activate | visual flow test |
| Dialog | `qml/LocalizedDialog.qml` | Bounded to parent, modal, scroll long content; preserve the host keyboard behavior | native render and flow tests |
| Feedback | panel notice/errors and hook dialog error | Show failures without discarding input; disable pending hook mutations | hook ownership test |
| Templates | `qml/Recipes.js` | Clone with fresh IDs; no credentials or permissions; all new flows and producers disabled | recipe logic and engine validation |
| Hooks | `internal/hooks/manage.go` | Only exact unmodified adapters can be removed; conflicting paths require manual review | Go lifecycle and rejection tests |

Templates require adjustment and enabling before review/activation. Calendar
recipes initially use UTC, visibly disclosed. Hook recipes require a connected
adapter. A disconnected engine cannot deliver hook events; delivery is not durable.
Editing a shared action changes its behavior in every referencing flow after the
new revision is reviewed and activated; this must be explained in the editor.
