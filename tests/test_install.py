import importlib.util
import json
import sqlite3
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("installer", Path(__file__).resolve().parents[1] / "scripts/install.py")
installer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(installer)


class InstallationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.args = SimpleNamespace(staging_root=self.root, activate=False)

    def test_install_update_and_backup(self):
        shell = self.root / ".config/omarchy/shell.json"
        shell.parent.mkdir(parents=True)
        shell.write_text('{"custom":"keep"}')
        installer.install(self.args)
        receipt_path = self.root / ".local/state/quatrro-install/receipt.json"
        receipt = json.loads(receipt_path.read_text())
        self.assertEqual(receipt["compatibility"], {"protocol":2,"contract":1,"ui_contract":1,"version":"0.1.0-dev"})
        for name, digest in receipt["files"].items():
            self.assertEqual(installer.digest(Path(name).read_bytes()), digest)
        self.assertEqual(receipt_path.stat().st_mode & 0o777, 0o600)
        self.assertEqual((self.root / ".local/bin/quatrrod").stat().st_mode & 0o777, 0o755)
        plugin = shell.parent / "plugins/quatrro.automations"
        panel = plugin / json.loads((plugin / "manifest.json").read_text())["entryPoints"]["panel"]
        self.assertIn(str(self.root / ".local/bin/quatrroctl"), (panel.parent / "qml/Backend.qml").read_text())
        installer.install(self.args)
        backups = list(receipt_path.parent.glob("backup-*"))
        self.assertEqual(len(backups), 1)
        index = json.loads((backups[0] / "index.json").read_text())
        for key, name in index.items():
            self.assertEqual(installer.digest((backups[0] / key).read_bytes()), receipt["files"][name])
        self.assertEqual(shell.read_text(), '{"custom":"keep"}')

    def test_foreign_file_is_not_overwritten(self):
        target = self.root / ".local/bin/quatrroctl"
        target.parent.mkdir(parents=True)
        target.write_text("unrelated")
        with self.assertRaisesRegex(ValueError, "unowned"):
            installer.install(self.args)
        self.assertEqual(target.read_text(), "unrelated")
        self.assertFalse((target.parent / "quatrrod").exists())

    def test_modified_install_is_preserved(self):
        installer.install(self.args)
        plugin = self.root / ".config/omarchy/plugins/quatrro.automations"
        target = plugin / json.loads((plugin / "manifest.json").read_text())["entryPoints"]["barWidget"]
        target.write_text("local customization")
        with self.assertRaisesRegex(ValueError, "modified"):
            installer.install(self.args)
        self.assertEqual(target.read_text(), "local customization")

    def test_symlink_parent_is_rejected(self):
        (self.root / "elsewhere").mkdir()
        (self.root / ".config").symlink_to(self.root / "elsewhere", target_is_directory=True)
        with self.assertRaisesRegex(ValueError, "Symlink"):
            installer.install(self.args)
        self.assertEqual(list((self.root / "elsewhere").iterdir()), [])

    def test_uninstall_preserves_data_secrets_and_untracked_files(self):
        installer.install(self.args)
        preserved = [self.root / ".local/state/quatrro/quatrro.db", self.root / ".config/quatrro/secrets/private", self.root / ".config/omarchy/plugins/quatrro.automations/notes.txt"]
        for path in preserved:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("preserve")
        installer.uninstall(self.args)
        for path in preserved:
            self.assertEqual(path.read_text(), "preserve")
        self.assertFalse((self.root / ".local/bin/quatrrod").exists())
        self.assertFalse((self.root / ".config/systemd/user/quatrrod.service").exists())

    def test_uninstall_rejects_forged_receipt_path(self):
        installer.install(self.args)
        receipt_path = self.root / ".local/state/quatrro-install/receipt.json"
        receipt = json.loads(receipt_path.read_text())
        foreign = self.root / "other.txt"
        foreign.write_text("keep")
        receipt["files"][str(foreign)] = installer.digest(foreign.read_bytes())
        receipt_path.write_text(json.dumps(receipt))
        with self.assertRaisesRegex(ValueError, "outside"):
            installer.uninstall(self.args)
        self.assertTrue(foreign.exists())
        self.assertTrue((self.root / ".local/bin/quatrrod").exists())

    def test_uninstall_stops_service_and_disables_plugin_before_removal(self):
        installer.install(self.args)
        receipt_path = self.root / '.local/state/quatrro-install/receipt.json'
        receipt = json.loads(receipt_path.read_text())
        owned = list(map(Path, receipt['files']))
        preserved = [self.root / '.config/omarchy/shell.json',
                     self.root / '.config/omarchy/hooks/theme-set.d/another-tool',
                     self.root / '.config/omarchy/plugins/another.plugin/manifest.json']
        for path in preserved:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text('foreign fixture')
        calls = []
        def command(argv):
            if len(calls) < 2:
                self.assertTrue(receipt_path.exists())
                self.assertTrue(all(path.exists() for path in owned))
            else:
                self.assertFalse(receipt_path.exists())
                self.assertTrue(all(not path.exists() for path in owned))
            calls.append(argv)
        # Exercise the non-staging control path while keeping all paths private.
        # Service/shell commands are recorded, not sent to the real desktop.
        with patch.object(installer, 'layout', return_value=(self.root, self.root / '.config', self.root / '.local/state')), patch.object(installer, 'run', side_effect=command):
            installer.uninstall(SimpleNamespace(staging_root=None))
        self.assertEqual(calls, [['systemctl','--user','disable','--now','quatrrod.service'],
                                ['omarchy','plugin','disable','quatrro.automations'],
                                ['systemctl','--user','daemon-reload']])
        self.assertTrue(all(path.read_text() == 'foreign fixture' for path in preserved))

    def test_uninstall_control_failure_preserves_all_owned_files(self):
        installer.install(self.args)
        receipt_path = self.root / '.local/state/quatrro-install/receipt.json'
        receipt_before = receipt_path.read_bytes()
        receipt = json.loads(receipt_before)
        for fail_at in (0, 1):
            with self.subTest(fail_at=fail_at):
                calls = []
                def command(argv):
                    calls.append(argv)
                    if len(calls) - 1 == fail_at:
                        raise RuntimeError('control unavailable')
                with patch.object(installer, 'layout', return_value=(self.root, self.root / '.config', self.root / '.local/state')), patch.object(installer, 'run', side_effect=command):
                    with self.assertRaisesRegex(RuntimeError, 'control unavailable'):
                        installer.uninstall(SimpleNamespace(staging_root=None))
                self.assertEqual(receipt_path.read_bytes(), receipt_before)
                self.assertTrue(all(installer.digest(Path(name).read_bytes()) == digest for name, digest in receipt['files'].items()))
                self.assertEqual(len(calls), fail_at + 1)

    def test_update_snapshots_committed_wal_and_preserves_queue(self):
        installer.install(self.args)
        state = self.root / ".local/state"
        connection = sqlite3.connect(state / "quatrro/quatrro.db")
        self.addCleanup(connection.close)
        connection.executescript("PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE schema_version(version INTEGER); INSERT INTO schema_version VALUES(4); CREATE TABLE executions(id TEXT,state TEXT); INSERT INTO executions VALUES('queued','pending');")
        connection.commit()
        self.assertTrue((state / "quatrro/quatrro.db-wal").exists())
        installer.install(self.args)
        backup = next((state / "quatrro-install").glob("backup-*"))
        metadata = json.loads((backup / "state.json").read_text())
        self.assertEqual(metadata["schema"], 4)
        self.assertEqual(metadata["sha256"], installer.digest((backup / "state.sqlite").read_bytes()))
        self.assertEqual((backup / "state.sqlite").stat().st_mode & 0o777, 0o600)
        snapshot = sqlite3.connect(backup / "state.sqlite")
        self.addCleanup(snapshot.close)
        self.assertEqual(snapshot.execute("SELECT * FROM executions").fetchall(), [("queued", "pending")])
        self.assertFalse((backup / "state.sqlite-wal").exists())

    def test_running_engine_lock_refuses_update(self):
        installer.install(self.args)
        with installer.engine_lock(self.root / ".local/state"):
            with self.assertRaisesRegex(ValueError, "Stop the engine"):
                installer.install(self.args)

    def restore_fixture(self):
        installer.install(self.args)
        state = self.root / ".local/state"
        with sqlite3.connect(state / "quatrro/quatrro.db") as db:
            db.executescript("""
                CREATE TABLE schema_version(version INTEGER); INSERT INTO schema_version VALUES(4);
                CREATE TABLE settings(key TEXT PRIMARY KEY,value TEXT);
                INSERT INTO settings VALUES('draft','configuration-to-preserve');
                CREATE TABLE grants(scope TEXT PRIMARY KEY,hash TEXT); INSERT INTO grants VALUES('flow:a:action:b','old-grant');
                CREATE TABLE executions(id TEXT,state TEXT); INSERT INTO executions VALUES('queued','pending'),('done','completed');
                CREATE TABLE outbox(state TEXT); INSERT INTO outbox VALUES('pending'),('delivered');
                CREATE TABLE audit(at TEXT,operation TEXT,resource TEXT,result TEXT);
            """)
        db.close()
        installer.install(self.args)
        self.args.restore_data = next((state / "quatrro-install").glob("backup-*"))
        return state

    def test_restore_pauses_revokes_and_retains_previous_state(self):
        state = self.restore_fixture()
        with sqlite3.connect(state / "quatrro/quatrro.db") as db:
            db.execute("UPDATE settings SET value='newer' WHERE key='draft'")
        db.close()
        installer.restore_data(self.args)
        with sqlite3.connect(state / "quatrro/quatrro.db") as db:
            self.assertEqual(dict(db.execute("SELECT * FROM settings")), {"draft": "configuration-to-preserve", "paused": "true", "admission": "reject"})
            self.assertEqual(db.execute("SELECT count(*) FROM grants").fetchone()[0], 0)
            self.assertEqual(dict(db.execute("SELECT * FROM executions")), {"queued": "uncertain", "done": "completed"})
            self.assertEqual(db.execute("SELECT state FROM outbox").fetchall(), [("uncertain",), ("delivered",)])
            self.assertEqual(db.execute("SELECT operation FROM audit").fetchall(), [("storage.restore",)])
        db.close()
        previous = next((state / "quatrro-install").glob("backup-before-restore-*"))
        with sqlite3.connect(previous / "state.sqlite") as db:
            self.assertEqual(db.execute("SELECT value FROM settings WHERE key='draft'").fetchone()[0], "newer")
        db.close()

    def test_restore_rejects_modified_backup_without_touching_current(self):
        state = self.restore_fixture()
        source = self.args.restore_data / "state.sqlite"
        with source.open("ab") as stream:
            stream.write(b"tampered")
        current = state / "quatrro/quatrro.db"
        before = current.read_bytes()
        with self.assertRaisesRegex(ValueError, "checksum"):
            installer.restore_data(self.args)
        self.assertEqual(current.read_bytes(), before)

    def test_restore_refuses_running_engine(self):
        state = self.restore_fixture()
        with installer.engine_lock(state):
            with self.assertRaisesRegex(ValueError, "Stop the engine"):
                installer.restore_data(self.args)



class ComponentCompatibilityTests(unittest.TestCase):
    @staticmethod
    def binary(name, **overrides):
        descriptor = dict(component=name, version="0.1.0-dev", protocol=2, contract=1, ui_contract=1)
        descriptor.update(overrides)
        # Tiny executable fixture; exact snapshot is executed by the verifier.
        return ("#!/usr/bin/python3\nprint(" + repr(json.dumps(descriptor)) + ")\n").encode()

    def test_all_contract_axes_and_release_must_match(self):
        requirements = b'.pragma library\nvar requirements = {"protocol":2,"contract":1,"ui_contract":1};\n'
        binaries = {name: self.binary(name) for name in ("quatrrod", "quatrroctl")}
        self.assertEqual(installer.validate_components(binaries, requirements)["protocol"], 2)
        for changed in ({"protocol":1}, {"contract":2}, {"ui_contract":2}, {"version":"other"}, {"component":"other"}, {"protocol":True}):
            with self.subTest(changed=changed), self.assertRaises(ValueError):
                installer.validate_components({**binaries, "quatrroctl": self.binary("quatrroctl", **changed)}, requirements)
        with self.assertRaises(ValueError):
            installer.validate_components(binaries, b'.pragma library\nvar requirements = {"ui_contract":1};')

    def test_incompatible_update_does_not_touch_installed_files(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            args = SimpleNamespace(staging_root=root, activate=False)
            installer.install(args)
            receipt_path = root / ".local/state/quatrro-install/receipt.json"
            receipt_before = receipt_path.read_bytes()
            receipt = json.loads(receipt_before)
            before = {name: Path(name).read_bytes() for name in receipt["files"]}
            with patch.object(installer, "validate_components", side_effect=ValueError("Incompatible Quatrro artifacts")):
                with self.assertRaisesRegex(ValueError, "Incompatible"):
                    installer.install(args)
            self.assertEqual(receipt_path.read_bytes(), receipt_before)
            self.assertTrue(all(Path(name).read_bytes() == raw for name, raw in before.items()))
            self.assertFalse(list(receipt_path.parent.glob("backup-*")))


if __name__ == "__main__":
    unittest.main()
