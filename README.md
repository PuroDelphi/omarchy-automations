# Omarchy Automations

An automation plugin for Omarchy: connect inbound and outbound webhooks to notifications, services, commands and system monitoring.

Status: **in development, not production-ready**. The engine, CLI and native panel are implemented. Native visual review and bilingual documentation are complete for the current development version. Real Google OAuth and suspend/logout acceptance tests are deferred; final release checks remain tracked in the [roadmap](ROADMAP.md).

The interface defaults to **English**. Users can switch to **Spanish**, and the preference is saved per profile.

## What you can automate

| When this happens… | Omarchy Automations can… |
|---|---|
| A signed GitHub webhook arrives | Show a desktop notification and forward an HTTPS request. |
| Disk space or another monitored metric crosses a threshold | Alert you, then send a recovery event when it returns to normal. |
| An interval or calendar schedule fires | Run an approved script or command and report its outcome. |
| An Omarchy hook or local event arrives | Apply conditions and run a reviewed sequence of actions. |

Build a draft → simulate an event → review permissions → activate the flow.
History shows execution and delivery status, including failures and uncertain outcomes.

![Connections panel with an example inbound webhook](docs/images/tokyo-night-en-connections.png)
*Connections panel in the Tokyo Night theme, rendered with disabled example data.*

| Configure a webhook | Simulate before activating |
|---|---|
| ![Native inbound webhook editor](docs/images/native-entry-en.png) | ![Native event simulation dialog](docs/images/native-simulation-en.png) |

*The editor and simulation dialog are captures from the installed Omarchy shell.
More light/dark and English/Spanish screenshots: [[en]](docs/en/interface.md) · [[es]](docs/es/interface.md).*

## Install

Requires Omarchy Quattro with the plugin CLI, Git, Make, Python 3 and Go 1.26+.
The verified environment is Linux amd64 with Omarchy 4.0.4-1. A user systemd
session runs the engine; commands/scripts need Bubblewrap. Desktop credentials
use Secret Service when selected. See [verified compatibility](docs/compatibility.md).

Run as your normal desktop user, from a directory where you keep source projects:

```bash
git clone https://github.com/PuroDelphi/omarchy-automations.git
cd omarchy-automations
make build
python3 scripts/install.py --activate
```

The installer installs the engine and CLI in `~/.local/bin`, registers
`quatrrod.service` and enables the panel through `omarchy plugin enable`.
Keep the checkout for updates and uninstallation. No administrator broker is
installed by this command; system-service control is an optional separate setup.

Verify and open the plugin:

```bash
systemctl --user status quatrrod.service
~/.local/bin/quatrroctl status
omarchy plugin enable quatrro.automations
```

Click the connections icon in the bar. `quatrro.automations` is the compatible
technical plugin ID; the displayed product name is **Omarchy Automations**.

**About `omarchy plugin add`:** Omarchy's standard command clones a plugin;
it does not build our Go binaries or install the background service. This
version therefore requires the installation steps above. Do not combine a
clone inside Omarchy's plugin directory with this installer: its ownership
checks reject untracked existing files. A complete one-command installation is
still a packaging improvement, not a supported shortcut.

## Try your first automation

1. Open **Actions** and create a `notify` action named `notice`, with title
   `Omarchy Automations` and message `{{data.message}}`.
2. Open **Flows** and create an enabled flow with source `local:demo` and step `notice`.
3. Save the draft. Open **Simulate / test**, use source `local:demo`, type `demo`
   and sample data `{"message":"Hello from Omarchy Automations"}`.
4. Choose **Simulate without effects**, then review permissions and activate.
5. Choose **Run real test** and confirm. Look for the notification and its result
   in **History**.

Follow the illustrated user guide for credentials, inbound webhooks and more:
[[en]](docs/en/user-guide.md) · [[es]](docs/es/user-guide.md).

## Update

From your source checkout, review the incoming changes and current work in History.
Then build and install matching engine, CLI and panel versions together:

```bash
git pull --ff-only
make build
systemctl --user stop quatrrod.service
python3 scripts/install.py --activate
```

The installer records backups and refuses unexpected modifications to owned files.
`omarchy plugin update` alone does not update this installer-managed engine and UI.

## Disable or uninstall

To hide the panel while keeping automations running:

```bash
omarchy plugin disable quatrro.automations
```

To show it again, use `omarchy plugin enable quatrro.automations`. To stop the
engine as well, use `systemctl --user stop quatrrod.service`.

For a complete uninstall, run from the source checkout:

```bash
python3 scripts/install.py --uninstall
```

This stops/disables the service and uses `omarchy plugin disable` before removing
verified installed files. Configuration, history, credentials and backups are
preserved. Separately installed hooks and the optional administrator broker have
[their own removal steps](docs/en/system-integration.md).

Omarchy also provides `omarchy plugin remove quatrro.automations`, but it only
removes the shell plugin directory; it does not uninstall the engine, service or
installation receipt. Use the complete uninstall above for this plugin.

## Troubleshooting and current limits

- **Panel disconnected:** check `systemctl --user status quatrrod.service` and
  `~/.local/bin/quatrroctl status`. Engine, CLI and UI must come from the same build.
- **Simulation works but nothing runs:** review enabled resources, active revision,
  permissions, pause state and History. Saving a draft alone does not activate it.
- **No notification:** check the desktop notification service and do-not-disturb mode.
- **Remote webhooks:** the listener defaults to loopback; exposing it requires an
  explicit network/TLS setup. See the security guide [[en]](docs/en/security.md) · [[es]](docs/es/security.md).
- **Google and session recovery:** real OAuth consent/renewal and physical
  suspend/logout acceptance tests remain deferred. This is `0.1.0-dev`, not a stable release.

Report reproducible problems in [GitHub Issues](https://github.com/PuroDelphi/omarchy-automations/issues)
with your version, Omarchy version, steps and redacted errors. Do not attach tokens,
credential files or unreviewed database/configuration exports.

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
