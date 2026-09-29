# Omarchy Automations

An automation plugin for Omarchy: connect inbound and outbound webhooks to notifications, services, commands and system monitoring.

Status: **in development, not production-ready**. The engine, CLI and native panel are implemented. Native visual review and bilingual documentation are complete for the current development version. Real Google OAuth and suspend/logout acceptance tests are deferred; final release checks remain tracked in the [roadmap](ROADMAP.md).

The interface defaults to **English**. Users can switch to **Spanish**, and the preference is saved per profile.

## Documentation

Choose a language for each guide. English is listed first throughout.

| Guide | Languages |
|---|---|
| Documentation index | [[en]](docs/en/README.md) · [[es]](docs/es/README.md) |
| User guide: installation and troubleshooting | [[en]](docs/en/user-guide.md) · [[es]](docs/es/user-guide.md) |
| Interface, language and screenshots | [[en]](docs/en/interface.md) · [[es]](docs/es/interface.md) |
| Options reference | [[en]](docs/en/options.md) · [[es]](docs/es/options.md) |
| Use cases | [[en]](docs/en/use-cases.md) · [[es]](docs/es/use-cases.md) |
| Scripts and event adapters | [[en]](docs/en/code-examples.md) · [[es]](docs/es/code-examples.md) |
| GitHub webhook example | [[en]](docs/en/provider-example.md) · [[es]](docs/es/provider-example.md) |
| Service control example | [[en]](docs/en/service-example.md) · [[es]](docs/es/service-example.md) |
| HTTP retries and recovery | [[en]](docs/en/recovery-example.md) · [[es]](docs/es/recovery-example.md) |
| Google OAuth example | [[en]](docs/en/oauth-example.md) · [[es]](docs/es/oauth-example.md) |
| System integration and Omarchy hooks | [[en]](docs/en/system-integration.md) · [[es]](docs/es/system-integration.md) |
| Security | [[en]](docs/en/security.md) · [[es]](docs/es/security.md) |
| Performance | [[en]](docs/en/performance.md) · [[es]](docs/es/performance.md) |
| GitHub setup and publishing | [[en]](docs/en/github.md) · [[es]](docs/es/github.md) |
| Monitor recipes | [[en]](docs/en/monitor-recipes.md) · [[es]](docs/es/monitor-recipes.md) |
| Action recipes | [[en]](docs/en/action-recipes.md) · [[es]](docs/es/action-recipes.md) |
| Payload formats and conditions | [[en]](docs/en/payload-recipes.md) · [[es]](docs/es/payload-recipes.md) |
| Webhook authentication | [[en]](docs/en/authentication-recipes.md) · [[es]](docs/es/authentication-recipes.md) |
| Development release notes | [[en]](docs/en/release-notes.md) · [[es]](docs/es/release-notes.md) |
| Acceptance tests with the user | [[en]](docs/en/external-acceptance.md) · [[es]](docs/es/external-acceptance.md) |

## Technical references

The original engineering notes below are maintained in Spanish. The documentation above provides the English and Spanish user guides.

- [Roadmap and progress](ROADMAP.md)
- [Product name and compatibility identifiers](docs/product-name.md)
- [Development packages](docs/releases.md)
- [Architecture](docs/DESIGN.md)
- [Protocol and operations](docs/PROTOCOL.md)
- [Validation evidence](docs/validation.md)
- [Verified compatibility](docs/compatibility.md)

## Development

Requires Linux, Go 1.26+, Omarchy/Quickshell for the UI, and user systemd for system actions. The engine uses SQLite through Go.

```bash
make build
make test
make check
```

To try an isolated profile without changing your existing configuration, run in one terminal:

```bash
export QUATRRO_PROFILE="$PWD/.dev/profile"
export PATH="$PWD/build:$PATH"
quatrrod
```

In another terminal, with the same environment variables:

```bash
quatrroctl status
quatrroctl config.save --stdin < examples/notification.json
quatrroctl simulate '{"source":"local:demo","type":"demo","data":{"message":"Hello"}}'
quatrroctl config.preview
python3 scripts/run-ui.py
```

Simulation does not activate flows or execute actions. Activation requires the reviewed hash and capabilities returned by `config.preview`; see the [protocol](docs/PROTOCOL.md). The panel lets you create connections, actions, flows and monitors, review capabilities and activate a revision. It includes simulation, history, credentials and storage limits.

`python3 scripts/smoke-native.py --offscreen` checks connectivity and produces a fresh render at `.dev/panel-render.png`, even while the session is locked. `python3 scripts/test-ui-flow.py` exercises QML form signals, activates a revision and checks notification, local HTTPS delivery and history. `QUATRRO_HOST_TEST=1 go test -race ./...` adds a notification, a dedicated temporary service and isolated command tests. These tests require access to the real session bus.

Omarchy plugins run as user code; this interface is not a sandbox. The plugin does not expose a remote shell. Registered commands use fixed arguments and run through systemd/Bubblewrap with networking disabled and read-only `/usr`.

## License

[MIT](LICENSE). Dependency notices are preserved in
[third-party notices](docs/third-party-notices.md).
