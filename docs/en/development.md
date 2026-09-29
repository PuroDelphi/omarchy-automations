# Development and source builds

[Español](../es/development.md) · [Prebuilt installation](installation.md)

This chapter is for contributors and users testing unpublished changes.
Use the prebuilt installation for ordinary use on Linux amd64.

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

Simulation does not activate flows or execute actions. Activation requires the reviewed hash and capabilities returned by `config.preview`; see the [protocol](protocol.md). The panel lets you create connections, actions, flows and monitors, review capabilities and activate a revision. It includes simulation, history, credentials and storage limits.

`python3 scripts/smoke-native.py --offscreen` checks connectivity and produces a fresh render at `.dev/panel-render.png`, even while the session is locked. `python3 scripts/test-ui-flow.py` exercises QML form signals, activates a revision and checks notification, local HTTPS delivery and history. `QUATRRO_HOST_TEST=1 go test -race ./...` adds a notification, a dedicated temporary service and isolated command tests. These tests require access to the real session bus.

Omarchy plugins run as user code; this interface is not a sandbox. The plugin does not expose a remote shell. Registered commands use fixed arguments and run through systemd/Bubblewrap with networking disabled and read-only `/usr`.

## Install a source build

Keep the source checkout outside the installed plugin directory:

```bash
git clone https://github.com/PuroDelphi/omarchy-automations.git
cd omarchy-automations
make build
python3 scripts/install.py --activate
```

Do not install a source-managed panel over an existing Git-managed panel. For an
existing source-managed installation, rebuild, inspect pending work, stop the
user service and rerun the installer. The installer records a backup and preserves
application data. Source builds on other architectures are not verified releases.

[Installer internals and data restoration](../installation.md) ·
[Runtime packaging and reproducibility](packages.md).
