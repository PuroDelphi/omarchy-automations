import importlib.util
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

SCRIPTS = Path(__file__).resolve().parents[1] / 'scripts'
sys.path.insert(0, str(SCRIPTS))
spec = importlib.util.spec_from_file_location('editor_launcher', SCRIPTS / 'editor.py')
editor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(editor)


class EditorBoundaryTests(unittest.TestCase):
    def test_launch_has_limits_and_no_shell_or_secret_environment(self):
        with patch.dict(os.environ, {'HOME': '/tmp/a path/$literal', 'QUATRRO_PROFILE': '/tmp/editor-profile',
                                     'SECRET_TOKEN': 'never-forward', 'QML_IMPORT_PATH': '/tmp/untrusted'}, clear=True):
            command = editor.command()
        self.assertIn('--expand-environment=no', command)
        self.assertIn('--property=MemoryMax=512M', command)
        self.assertIn('--property=MemorySwapMax=0', command)
        self.assertIn('--property=CPUQuota=100%', command)
        self.assertIn('--property=TasksMax=128', command)
        self.assertIn('--setenv=HOME=/tmp/a path/$literal', command)
        self.assertNotIn('never-forward', ' '.join(command))
        self.assertNotIn('/tmp/untrusted', ' '.join(command))
        self.assertEqual(command[-3:], ['/usr/bin/python3', str(SCRIPTS / 'editor.py'), '--worker'])

    def test_worker_rejects_missing_or_disabled_limits_before_qt(self):
        with patch.object(editor, 'verify_limits', side_effect=RuntimeError('no limits')), \
             patch.object(editor.subprocess, 'call') as child:
            with self.assertRaises(RuntimeError):
                editor.worker()
            child.assert_not_called()
        with patch.object(editor.Path, 'read_text', return_value='0::/user.slice/unbounded.service\n'):
            with self.assertRaisesRegex(RuntimeError, 'dedicated'):
                editor.verify_limits()

    def test_effective_cgroup_limits(self):
        values = {'/proc/self/cgroup': '0::/user.slice/' + editor.unit_name(),
                  'memory.max': str(editor.MEMORY), 'memory.swap.max': '0',
                  'pids.max': str(editor.TASKS), 'cpu.max': '100000 100000'}
        def read(path):
            return values.get(str(path), values.get(path.name))
        with patch.object(editor.Path, 'read_text', read):
            editor.verify_limits()
            for field, invalid in [('memory.max', 'max'), ('memory.max', str(editor.MEMORY + 1)),
                                   ('memory.swap.max', '1'), ('pids.max', 'max'),
                                   ('cpu.max', 'max 100000'), ('cpu.max', '200000 100000')]:
                with self.subTest(field=field, invalid=invalid), patch.dict(values, {field: invalid}):
                    with self.assertRaises((ValueError, RuntimeError)):
                        editor.verify_limits()

    def test_failed_unit_never_falls_back_to_unbounded_qt(self):
        import subprocess
        from types import SimpleNamespace
        with patch.object(editor.subprocess, 'run', side_effect=[SimpleNamespace(returncode=3),
                          subprocess.CalledProcessError(1, ['systemd-run'])]) as run:
            with self.assertRaises(subprocess.CalledProcessError):
                editor.launch()
            self.assertEqual(run.call_count, 2)
            self.assertEqual(run.call_args.args[0][0], '/usr/bin/systemd-run')

    def test_shared_shell_entry_points_do_not_construct_editor(self):
        import json
        root = SCRIPTS.parent
        manifest = json.loads((root / 'manifest.json').read_text())
        for filename in manifest['entryPoints'].values():
            source = (root / filename).read_text()
            self.assertNotRegex(source, r'\b(?:Editor|InputField|MultiLineField|TextField|TextInput|TextArea|TextEdit|Loader)\s*\{')
        sources = (SCRIPTS / 'install.py').read_text()
        self.assertIn('Path("EditorWindow.qml")', sources)
        self.assertIn('Path("scripts/editor.py")', sources)
