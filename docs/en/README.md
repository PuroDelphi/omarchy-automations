# Omarchy Automations — documentation

[Español](../es/README.md) · [Project](../../README.md)

English is the default interface language; Spanish can be selected and is saved
per profile. These paired guides describe the development version. The remaining
release checks are tracked in the [roadmap](../../ROADMAP.md).

| Guide | Content |
|---|---|
| [01](user-guide.md) | Start here: installation, daily operation and troubleshooting |
| [02](interface.md) | Interface, language and screenshots |
| [03](options.md) | Every option, defaults, constraints and examples |
| [04](use-cases.md) | Webhooks, disk alerts and scheduled reminders |
| [05](code-examples.md) | Typed scripts and event adapters |
| [06](provider-example.md) | Signed GitHub events |
| [07](service-example.md) | Authorized user-service control |
| [08](recovery-example.md) | HTTP retries and uncertain outcomes |
| [09](oauth-example.md) | Google OAuth setup and limitations |
| [10](system-integration.md) | Omarchy hooks and optional administrative broker |
| [11](security.md) | Security boundaries and tested guarantees |
| [12](performance.md) | Measured performance and acceptance scenarios |
| [13](github.md) | Login and publish from this machine |

Start with the user guide, then import one of the guided examples into a draft.
Simulation does not perform actions. Review credentials, enabled sources and
permissions before activating a real flow. Examples contain placeholders and
ship with sources and flows disabled.

The [coverage matrix](../option-coverage.md) maps fields to both references.
Each option has at least two example values; complete workflows and expected
results are described in the guides above. Example values are alternatives, not
fragments to combine automatically.

[Monitor recipes: two examples for each metric](monitor-recipes.md).

[Action recipes: all kinds and named command profiles](action-recipes.md).

[Payload formats and conditions](payload-recipes.md).

[Webhook authentication: HMAC, Slack, GitHub and bearer](authentication-recipes.md).

[Development release notes](release-notes.md).

- [Acceptance tests with the user](external-acceptance.md)

[Install without Go](installation.md).

## Technical references

| Reference | Languages |
|---|---|
| Product name and compatibility identifiers | [[en]](../en/product-name.md) · [[es]](../es/product-name.md) |
| Development packages | [[en]](../en/packages.md) · [[es]](../es/packages.md) |
| Architecture | [[en]](../en/architecture.md) · [[es]](../es/architecture.md) |
| Control protocol | [[en]](../en/protocol.md) · [[es]](../es/protocol.md) |
| Event formats | [[en]](../en/formats.md) · [[es]](../es/formats.md) |
| LAN access | [[en]](../en/lan.md) · [[es]](../es/lan.md) |
| Omarchy hooks | [[en]](../en/hooks.md) · [[es]](../es/hooks.md) |
| Scheduling | [[en]](../en/scheduling.md) · [[es]](../es/scheduling.md) |
| Webhook providers | [[en]](../en/providers.md) · [[es]](../es/providers.md) |
| System monitoring | [[en]](../en/monitoring.md) · [[es]](../es/monitoring.md) |
| Delivery recovery | [[en]](../en/recovery.md) · [[es]](../es/recovery.md) |
| Command isolation | [[en]](../en/command-isolation.md) · [[es]](../es/command-isolation.md) |
| Directory access | [[en]](../en/directory-access.md) · [[es]](../es/directory-access.md) |
| Typed scripts | [[en]](../en/scripts.md) · [[es]](../es/scripts.md) |
| OAuth 2 | [[en]](../en/oauth2.md) · [[es]](../es/oauth2.md) |
| External adapters | [[en]](../en/external-adapters.md) · [[es]](../es/external-adapters.md) |
| Operations | [[en]](../en/operations.md) · [[es]](../es/operations.md) |
| Network exposure | [[en]](../en/exposure.md) · [[es]](../es/exposure.md) |

[Development and optional source builds](development.md).

AI assistants can use the bundled [Omarchy Automations skill](ai-assistants.md), without Go or a source checkout.
