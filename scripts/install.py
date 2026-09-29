#!/usr/bin/env python3
"""User installation of locally built Omarchy Automations; no package downloads or sudo."""
import argparse
from contextlib import closing, contextmanager
import fcntl
import hashlib
import json
import os
import re
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tempfile
from datetime import datetime, timezone

PROJECT = Path(__file__).resolve().parents[1]
PLUGIN = "quatrro.automations"
DATABASE_VERSION = 5


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def safe_path(path):
    if not path.is_absolute() or ".." in path.parts or any(ord(c) < 32 for c in str(path)):
        raise ValueError(f"Absolute canonical path required: {path}")
    for component in (path, *path.parents):
        if component.is_symlink():
            raise ValueError(f"Symlink destination refused: {component}")


def atomic_write(path, raw, mode):
    safe_path(path)
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    fd, name = tempfile.mkstemp(prefix=".quatrro-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            os.fchmod(stream.fileno(), mode)
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)


def layout(staging):
    home = staging or Path.home()
    config = home / ".config" if staging else Path(os.environ.get("XDG_CONFIG_HOME", home / ".config"))
    state = home / ".local/state" if staging else Path(os.environ.get("XDG_STATE_HOME", home / ".local/state"))
    for path in (home, config, state):
        safe_path(path)
    return home, config, state


def plugin_sources():
    return [Path("manifest.json"), Path("Widget.qml"), Path("Panel.qml"), Path("qml/qmldir"), *sorted(Path("qml") / p.name for p in (PROJECT / "qml").iterdir() if p.suffix in (".qml", ".js"))]


def owned_paths(home, config, receipt=None):
    plugin = config / "omarchy/plugins" / PLUGIN
    allowed = {str(home / ".local/bin" / name) for name in ("quatrrod", "quatrroctl")} | {str(plugin / relative) for relative in plugin_sources()} | {str(config / "systemd/user/quatrrod.service")}
    ui_files = {str(relative) for relative in plugin_sources() if relative != Path("manifest.json")}
    for name in (receipt or {}).get("files", {}):
        try:
            relative = Path(name).relative_to(plugin)
        except ValueError:
            continue
        parts = relative.parts
        if len(parts) >= 3 and parts[0] == "ui-revisions" and re.fullmatch(r"[0-9a-f]{64}", parts[1]) and str(Path(*parts[2:])) in ui_files:
            allowed.add(name)
    return allowed


def validate_components(binaries, ui_contract_source):
    match = re.fullmatch(r"\s*\.pragma library\s+var requirements = (\{[^;]+\});\s*", ui_contract_source.decode())
    if not match:
        raise ValueError("Unrecognized UI compatibility contract; rebuild the complete package")
    expected = json.loads(match.group(1))
    if set(expected) != {"protocol", "contract", "ui_contract"} or any(type(value) is not int or value < 1 for value in expected.values()):
        raise ValueError("Invalid UI compatibility requirements")
    versions = {}
    # Execute the exact immutable byte snapshots that will be installed, not
    # paths that could point to a different build by the time copying starts.
    with tempfile.TemporaryDirectory(prefix="quatrro-component-check-") as temp:
        for component, raw in binaries.items():
            executable = Path(temp) / component
            executable.write_bytes(raw)
            executable.chmod(0o700)
            try:
                result = subprocess.run([str(executable), "--version"], capture_output=True, timeout=5, check=True)
                if len(result.stdout) > 4096:
                    raise ValueError("Oversized version descriptor")
                descriptor = json.loads(result.stdout)
            except (subprocess.SubprocessError, ValueError, OSError) as error:
                raise ValueError("Cannot verify component versions; rebuild engine, CLI and plugin together") from error
            if not isinstance(descriptor, dict):
                raise ValueError("Invalid component descriptor; rebuild the complete package")
            if descriptor.get("component") != component or any(type(descriptor.get(key)) is not int or descriptor[key] != value for key, value in expected.items()):
                raise ValueError("Incompatible Omarchy Automations artifacts; rebuild engine, CLI and plugin together")
            version = descriptor.get("version")
            if not isinstance(version, str) or not version or len(version) > 128:
                raise ValueError("Invalid component release version")
            versions[component] = version
    if len(set(versions.values())) != 1:
        raise ValueError("Mixed Omarchy Automations release versions; rebuild the complete package")
    return {**expected, "version": next(iter(versions.values()))}


def artifacts(home, config):
    files = {}
    binaries = {}
    for binary in ("quatrrod", "quatrroctl"):
        source = PROJECT / "build" / binary
        if not source.is_file() or not os.access(source, os.X_OK):
            raise ValueError("Build binaries first with make build")
        with source.open("rb") as stream:
            raw = stream.read(64*1024*1024+1)
        if len(raw) > 64*1024*1024:
            raise ValueError("Component binary exceeds size limit")
        binaries[binary] = raw
        files[str(home / ".local/bin" / binary)] = (raw, 0o755)
    plugin = config / "omarchy/plugins" / PLUGIN
    ui = {}
    for relative in plugin_sources():
        if relative == Path("manifest.json"):
            continue
        raw = (PROJECT / relative).read_bytes()
        if relative == Path("qml/Backend.qml"):
            raw = raw.replace(b'property string cli: "quatrroctl"', ("property string cli: " + json.dumps(str(home / ".local/bin/quatrroctl"))).encode())
        ui[str(relative)] = raw
    compatibility = validate_components(binaries, ui["qml/Compatibility.js"])
    # Each UI revision has distinct QML URLs. A live shell can retain component
    # caches despite rescanning; immutable URLs avoid reusing the previous UI.
    revision = digest(b"".join(name.encode()+b"\0"+raw+b"\0" for name, raw in sorted(ui.items())))
    prefix = Path("ui-revisions") / revision
    for name, raw in ui.items():
        files[str(plugin / prefix / name)] = (raw, 0o644)
    manifest = json.loads((PROJECT / "manifest.json").read_text())
    manifest["entryPoints"] = {key: str(prefix / value) for key, value in manifest["entryPoints"].items()}
    files[str(plugin / "manifest.json")] = (json.dumps(manifest, indent=2).encode()+b"\n", 0o644)
    unit = (PROJECT / "packaging/systemd/quatrrod.service").read_text()
    executable = str(home / ".local/bin/quatrrod").replace("%", "%%").replace("\\", "\\\\").replace('"', '\\"')
    unit = unit.replace("ExecStart=%h/.local/bin/quatrrod", f'ExecStart="{executable}"')
    files[str(config / "systemd/user/quatrrod.service")] = (unit.encode(), 0o644)
    return files, compatibility


def load_receipt(path):
    safe_path(path)
    if not path.exists():
        return None
    receipt = json.loads(path.read_text())
    if receipt.get("format") != 1 or not isinstance(receipt.get("files"), dict):
        raise ValueError("Unrecognized installation receipt")
    return receipt


def validate_owned(receipt, allowed):
    if not receipt:
        return
    for name, expected in receipt["files"].items():
        if name not in allowed:
            raise ValueError("Receipt contains a path outside the package")
        path = Path(name)
        safe_path(path)
        if path.exists() and (not path.is_file() or digest(path.read_bytes()) != expected):
            raise ValueError(f"Installed file was modified; preserve/reconcile it first: {path}")


def run(args):
    subprocess.run(args, check=True, timeout=30)


@contextmanager
def engine_lock(state):
    directory = state / "quatrro"
    safe_path(directory)
    directory.mkdir(parents=True, exist_ok=True, mode=0o700)
    if directory.stat().st_uid != os.getuid() or directory.stat().st_mode & 0o077:
        raise ValueError("Engine state must be owned by the user and private")
    lock = directory / "engine.lock"
    safe_path(lock)
    fd = os.open(lock, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise ValueError("Stop the engine before changing its installation") from exc
        yield
    finally:
        os.close(fd)


def snapshot_database(state, backup):
    source = state / "quatrro/quatrro.db"
    safe_path(source)
    if not source.exists():
        return
    if not source.is_file():
        raise ValueError("Engine database must be a regular file")
    target = backup / "state.sqlite"
    atomic_write(target, b"", 0o600)
    # SQLite's backup API includes committed WAL pages, unlike copying only db.
    with closing(sqlite3.connect(source.as_uri() + "?mode=ro", uri=True)) as incoming:
        with closing(sqlite3.connect(target)) as outgoing:
            incoming.backup(outgoing)
            outgoing.execute("PRAGMA journal_mode=DELETE")
            if outgoing.execute("PRAGMA integrity_check").fetchone() != ("ok",):
                raise ValueError("Database snapshot failed integrity check")
            version = outgoing.execute("SELECT MAX(version) FROM schema_version").fetchone()[0]
    atomic_write(backup / "state.json", json.dumps({"format": 1, "schema": version, "sha256": digest(target.read_bytes()), "secrets_included": False}).encode(), 0o600)


def install(args):
    _, config, state = layout(args.staging_root)
    with engine_lock(state):
        result = install_files(args)
    # The engine needs the same lock; activation must happen after releasing it.
    if args.activate:
        run(["systemctl", "--user", "daemon-reload"])
        run(["systemctl", "--user", "enable", "--now", "quatrrod.service"])
        shell_settings = config / "omarchy/shell.json"
        safe_path(shell_settings)
        backup = Path(tempfile.mkdtemp(prefix="shell-before-enable-", dir=state / "quatrro-install"))
        if shell_settings.exists():
            atomic_write(backup / "shell.json", shell_settings.read_bytes(), 0o600)
        atomic_write(backup / "metadata.json", json.dumps({"original_existed": shell_settings.exists()}).encode(), 0o600)
        run(["omarchy-shell", "shell", "rescanPlugins"])
        run(["omarchy", "plugin", "enable", PLUGIN])
    print(json.dumps(result))


def git_plugin_directory(args, home, config):
    """A git-managed panel stays owned by Omarchy; install only its runtime."""
    requested = getattr(args, "git_plugin", None)
    if requested is None:
        return None
    expected = config / "omarchy/plugins" / PLUGIN
    safe_path(requested)
    if requested != expected or not (requested / ".git").is_dir():
        raise ValueError("Git plugin must be the Omarchy-managed checkout at " + str(expected))
    # The runtime package and checked-out panel must be one matching revision.
    # Never overwrite or adopt repository files into the installer receipt.
    for relative in plugin_sources():
        target = requested / relative
        safe_path(target)
        if target.read_bytes() != (PROJECT / relative).read_bytes():
            raise ValueError("Git plugin differs from runtime package: " + str(relative))
    return requested


def install_files(args):
    home, config, state = layout(args.staging_root)
    git_plugin = git_plugin_directory(args, home, config)
    files, compatibility = artifacts(home, config)
    if git_plugin:
        files = {name: value for name, value in files.items()
                 if not Path(name).is_relative_to(git_plugin)}
    receipt_path = state / "quatrro-install/receipt.json"
    receipt = load_receipt(receipt_path)
    validate_owned(receipt, owned_paths(home, config, receipt))
    installation_mode = "git-plugin" if git_plugin else "managed"
    if receipt and receipt.get("mode", "managed") != installation_mode:
        raise ValueError("Uninstall the previous installation before changing its management mode")
    # Retain tracked prior UI revisions so loaded components remain usable
    # until the host has switched. Uninstall still owns/removes these files.
    for name in (receipt or {}).get("files", {}):
        if name not in files and Path(name).exists():
            files[name] = (Path(name).read_bytes(), Path(name).stat().st_mode & 0o777)
    if receipt and not args.staging_root:
        active = subprocess.run(["systemctl", "--user", "is-active", "--quiet", "quatrrod.service"], timeout=10)
        if active.returncode == 0:
            raise ValueError("Stop quatrrod.service before updating installed files")
    for name in files:
        path = Path(name)
        safe_path(path)
        if path.exists() and (not receipt or name not in receipt["files"]):
            raise ValueError(f"Refusing to replace an unowned file: {path}")
    if args.activate:
        for command in ("systemctl", "omarchy", "notify-send", "bwrap"):
            if not shutil.which(command):
                raise ValueError(f"Missing dependency: {command}")
        run(["omarchy", "plugin", "validate", str(PROJECT)])
    # Keep complete prior bytes before touching destinations. Updates refuse
    # locally edited files, rather than silently erasing their changes.
    before = {name: (Path(name).read_bytes(), Path(name).stat().st_mode & 0o777) if Path(name).exists() else None for name in files}
    backup = None
    if receipt or (state / "quatrro/quatrro.db").exists():
        receipt_path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        backup = Path(tempfile.mkdtemp(prefix="backup-", dir=receipt_path.parent))
        for index, (name, previous) in enumerate(before.items()):
            if previous:
                atomic_write(backup / str(index), previous[0], 0o600)
        atomic_write(backup / "index.json", json.dumps({str(i): name for i, name in enumerate(before)}).encode(), 0o600)
        if receipt:
            atomic_write(backup / "receipt.json", receipt_path.read_bytes(), 0o600)
        snapshot_database(state, backup)
    try:
        for name, (raw, mode) in files.items():
            atomic_write(Path(name), raw, mode)
        value = {"format": 1, "mode": installation_mode, "compatibility": compatibility, "version": json.loads((PROJECT / "manifest.json").read_text())["version"], "files": {name: digest(raw) for name, (raw, _) in files.items()}}
        atomic_write(receipt_path, json.dumps(value, indent=2).encode()+b"\n", 0o600)
    except Exception:
        for name, previous in before.items():
            if previous is None:
                Path(name).unlink(missing_ok=True)
            else:
                atomic_write(Path(name), *previous)
        raise
    return {"installed": len(files), "activated": args.activate, "compatibility": compatibility, "backup": str(backup) if backup else None}


def uninstall(args):
    home, config, state = layout(args.staging_root)
    receipt_path = state / "quatrro-install/receipt.json"
    receipt = load_receipt(receipt_path)
    if not receipt:
        raise ValueError("No installation receipt; refusing untracked deletion")
    validate_owned(receipt, owned_paths(home, config, receipt))
    if not args.staging_root:
        run(["systemctl", "--user", "disable", "--now", "quatrrod.service"])
        run(["omarchy", "plugin", "disable", PLUGIN])
    for name in receipt["files"]:
        Path(name).unlink(missing_ok=True)
    receipt_path.unlink()
    # Empty package directories only. Never recursively delete unexpected files.
    plugin = config / "omarchy/plugins" / PLUGIN
    directories = {Path(name).parent for name in receipt["files"] if Path(name).is_relative_to(plugin)}
    for directory in list(directories):
        directories.update(parent for parent in directory.parents if parent.is_relative_to(plugin))
    for directory in sorted(directories, key=lambda p: len(p.parts), reverse=True):
        if directory.exists() and not any(directory.iterdir()):
            directory.rmdir()
    if not args.staging_root:
        run(["systemctl", "--user", "daemon-reload"])
    print(json.dumps({"removed": len(receipt["files"]), "data_preserved": True}))


def restore_data(args):
    _, _, state = layout(args.staging_root)
    backup = args.restore_data
    safe_path(backup)
    if backup.parent != state / "quatrro-install" or not backup.name.startswith("backup-"):
        raise ValueError("Select a backup directory from this installation")
    source = backup / "state.sqlite"
    metadata_path = backup / "state.json"
    for path in (source, metadata_path):
        safe_path(path)
        if not path.is_file() or path.stat().st_uid != os.getuid() or path.stat().st_mode & 0o077:
            raise ValueError("Backup must be a private regular file owned by the user")
    if source.stat().st_size > 256 * 1024 * 1024:
        raise ValueError("Database backup exceeds size limit")
    metadata = json.loads(metadata_path.read_text())
    raw = source.read_bytes()
    if metadata.get("format") != 1 or metadata.get("sha256") != digest(raw):
        raise ValueError("Backup checksum or format mismatch")
    with engine_lock(state):
        # Prepare and sanitize a private independent image before replacing any
        # live data. Never reactivate old grants or replay pending restored work.
        with tempfile.TemporaryDirectory(prefix=".restore-", dir=backup.parent) as temporary:
            candidate = Path(temporary) / "candidate.sqlite"
            atomic_write(candidate, raw, 0o600)
            with closing(sqlite3.connect(candidate)) as restored:
                restored.execute("PRAGMA trusted_schema=OFF")
                if restored.execute("SELECT count(*) FROM sqlite_master WHERE type IN ('trigger','view')").fetchone()[0]:
                    raise ValueError("Unexpected executable database schema")
                version = restored.execute("SELECT MAX(version) FROM schema_version").fetchone()[0]
                if not isinstance(version, int) or not 1 <= version <= DATABASE_VERSION or version != metadata.get("schema"):
                    raise ValueError("Unsupported backup database version")
                if restored.execute("PRAGMA integrity_check").fetchone() != ("ok",):
                    raise ValueError("Backup integrity check failed")
                with restored:
                    restored.execute("DELETE FROM grants")
                    for key, value in (("paused", "true"), ("admission", "reject")):
                        restored.execute("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", (key, value))
                    restored.execute("UPDATE executions SET state='uncertain' WHERE state IN ('pending','running')")
                    restored.execute("UPDATE outbox SET state='uncertain' WHERE state='pending'")
                    restored.execute("INSERT INTO audit(at,operation,resource,result) VALUES(?,?,?,?)", (datetime.now(timezone.utc).isoformat(), "storage.restore", "database", "paused; grants revoked; pending work uncertain"))
                recovery = Path(tempfile.mkdtemp(prefix="backup-before-restore-", dir=backup.parent))
                snapshot_database(state, recovery)
                destination = state / "quatrro/quatrro.db"
                safe_path(destination)
                for suffix in ("-wal", "-shm", "-journal"):
                    safe_path(Path(str(destination) + suffix))
                if not destination.exists():
                    atomic_write(destination, b"", 0o600)
                # SQLite backup replaces the target transactionally and handles
                # existing WAL correctly; never unlink a live db's sidecars.
                with closing(sqlite3.connect(destination)) as current:
                    restored.backup(current)
                os.chmod(destination, 0o600)
    print(json.dumps({"restored": str(backup), "paused": True, "grants": 0, "previous_state": str(recovery)}))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--staging-root", type=Path, help="Install under a temporary home, never activate")
    parser.add_argument("--git-plugin", type=Path, help="Use an existing matching Omarchy git checkout; install runtime only")
    parser.add_argument("--activate", action="store_true", help="Enable user service and Omarchy widget after installation")
    parser.add_argument("--uninstall", action="store_true", help="Remove owned files; preserve state, secrets and backups")
    parser.add_argument("--restore-data", type=Path, help="Restore a private backup with effects paused and grants revoked")
    args = parser.parse_args()
    if args.staging_root and args.activate:
        parser.error("Staged installations cannot be activated")
    if args.uninstall and args.activate:
        parser.error("Uninstall cannot activate services")
    if args.restore_data and (args.uninstall or args.activate):
        parser.error("Restore cannot uninstall or activate services")
    if os.geteuid() == 0:
        parser.error("Run as the desktop user, never root")
    try:
        (restore_data if args.restore_data else uninstall if args.uninstall else install)(args)
    except (ValueError, OSError, sqlite3.Error, subprocess.SubprocessError) as exc:
        parser.exit(1, f"Installation failed: {exc}\n")


if __name__ == "__main__":
    main()
