# Install, update and remove

[Español](../es/installation.md) · [User guide](user-guide.md)

## Recommended: Omarchy plugin plus prebuilt runtime

Use Linux amd64, Omarchy Quattro's plugin CLI and Python 3. Commands/scripts need
Bubblewrap; notifications need the desktop notification service. No Go compiler
is required for this installation. Run as your normal desktop user, not root.

```bash
omarchy plugin add https://github.com/PuroDelphi/omarchy-automations.git --yes
python3 ~/.config/omarchy/plugins/quatrro.automations/scripts/setup.py
```

The first command clones trusted code without enabling it. Setup downloads the
specific preview release selected in `scripts/setup.py`, validates the archive
SHA256 and every file's hash, checks UI compatibility, installs `quatrrod` and
`quatrroctl` under `~/.local/bin`, enables the user service and enables the widget.
It does not install the optional privileged broker. Checksums detect corruption;
they are not an independent publisher signature. Download assets only from this
repository's [releases](https://github.com/PuroDelphi/omarchy-automations/releases).

Click the connections icon in the bar. English is the default; select Spanish in
the sidebar to persist that preference.

![Connections panel, example configuration](../images/tokyo-night-en-connections.png)

*This screenshot contains a disabled tutorial entry. A fresh install has no
configured automations; importing an example populates its draft.*

Check `systemctl --user status quatrrod.service` and `~/.local/bin/quatrroctl status`.
Keep `~/.local/bin` in your desktop PATH so the Git-managed panel can run the CLI.

## Update

Review pending work in History first. From the desktop session:

```bash
omarchy plugin update quatrro.automations
python3 ~/.config/omarchy/plugins/quatrro.automations/scripts/setup.py --update
```

The checkout selects a matching release. Setup rejects mismatched panel files and
modified installer-owned files. Once the package passes preflight, it stops the
engine and installs the runtime, then starts/enables it. If setup fails after the
stop, inspect the error before restarting with `systemctl --user start quatrrod.service`.
Backups are stored privately beside the installation receipt. Sources and secrets
are not uploaded. Updating the Omarchy checkout alone does not update the engine.

## Remove

```bash
python3 ~/.config/omarchy/plugins/quatrro.automations/scripts/setup.py --uninstall
omarchy plugin remove quatrro.automations
```

Run the commands in this order. The runtime uninstaller stops/disables the engine,
disables the widget and removes only verified runtime files. Omarchy then removes
its checkout; inspect local edits before confirming. Your data, credentials and
backups are retained. Separately installed hooks and the administrative broker
have [separate removal steps](system-integration.md).

## Offline installation and preview

Download the `.tar.gz` and adjacent `.tar.gz.sha256` from the same release on a
connected machine. Keep both filenames unchanged and copy both to the target.
From your checkout, use an absolute archive path:

```bash
python3 scripts/setup.py --archive /absolute/path/omarchy-automations-0.1.0-dev-linux-amd64.tar.gz
```

For a disposable HOME that never activates services:

```bash
python3 scripts/setup.py --archive /absolute/path/omarchy-automations-0.1.0-dev-linux-amd64.tar.gz --staging-root /tmp/automations-preview
python3 scripts/setup.py --staging-root /tmp/automations-preview --uninstall
```

A source checkout outside the Omarchy plugin directory uses installer-managed UI
files. A checkout inside the standard Omarchy plugin directory keeps the UI under
Git ownership. Switching modes requires uninstalling the old mode first; setup
never adopts or deletes an unrelated existing checkout.

## Existing source installations and building yourself

If you previously used `scripts/install.py`, update from the original source
checkout with `python3 scripts/setup.py --update`, and uninstall from there with
`python3 scripts/setup.py --uninstall`. Do not run `omarchy plugin add` over it.

For unpublished changes or development, see [Development and source builds](development.md).
Only Linux amd64 prebuilt artifacts are currently verified.

## Common installation problems

| Message or symptom | Next step |
|---|---|
| Plugin ID already installed | Use its existing management mode; update instead of adding a duplicate. |
| Git plugin differs from runtime package | Use the release-matching checkout or build your changes from source. Do not overwrite local edits. |
| Checksum mismatch | Download both files again from the same official release. Do not bypass verification. |
| Missing dependency | Install the named dependency through your normal Omarchy package tools, then retry. |
| Runtime still active | Review History and use `--update`, which stops it before replacement. |
| Missing receipt or modified owned file | Inspect the local installation; ownership checks deliberately stop replacement/removal. |
| Download unavailable | Check connectivity and release availability; use the offline archive or source workflow. |

The CI workflow checks the portable engine, installer, documentation and archive.
It does not replace native Omarchy UI tests or the deferred Google/session tests.
