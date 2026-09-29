#!/usr/bin/env python3
"""Filesystem transactions and adversarial bundles, without root or host services."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('installer', ROOT / 'scripts/install-broker.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

with tempfile.TemporaryDirectory(prefix='quatrro-install-test-') as temporary:
    root = Path(temporary)
    bundle = root / 'bundle'
    subprocess.run(['python3', str(ROOT / 'scripts/prepare-broker-install.py'), '--output', str(bundle)], check=True, capture_output=True)
    artifacts = module.read_bundle(bundle)
    assert set(artifacts) == set(module.FILES)
    # An attacker cannot redirect a manifest member to a different destination.
    manifest_path = bundle / 'manifest.json'
    manifest = manifest_path.read_bytes()
    wrong = json.loads(manifest)
    wrong['files'][0]['target'] = '/etc/sudoers'
    manifest_path.write_text(json.dumps(wrong))
    try:
        module.read_bundle(bundle)
        raise AssertionError('arbitrary destination accepted')
    except ValueError:
        pass
    manifest_path.write_bytes(manifest)
    (bundle / 'quatrro-broker.socket').write_text('tampered')
    try:
        module.read_bundle(bundle)
        raise AssertionError('hash mismatch accepted')
    except ValueError:
        pass

    filesystem = root / 'root'
    filesystem.mkdir(mode=0o700)
    fs = module.SafeFS(str(filesystem), owner=os.getuid())
    calls = []
    def control(*args, check=True):
        calls.append(args)
        return subprocess.CompletedProcess(args, 0, b'LoadState=not-found\nFragmentPath=\n', b'')
    module.systemctl = control
    try:
        current, receipt = module.installed(fs)
        module.transact(fs, artifacts, current, receipt)
        current, receipt = module.installed(fs)
        assert current == artifacts and receipt is not None and fs.read(module.JOURNAL) is None
        assert not any('enable' in call or 'start' in call for call in calls)
        # Failure after all writes must restore exact previous contents/receipt.
        updates = {**artifacts, 'quatrro-broker': b'new binary fixture'}
        failures = [True]
        def fail_reload(*args, check=True):
            if args == ('daemon-reload',) and failures:
                failures.pop()
                raise subprocess.CalledProcessError(1, args)
            return control(*args, check=check)
        module.systemctl = fail_reload
        try:
            module.transact(fs, updates, current, receipt)
            raise AssertionError('injected failure ignored')
        except subprocess.CalledProcessError:
            pass
        assert module.installed(fs)[0] == artifacts and fs.read(module.JOURNAL) is None
        def persistent_failure(*args, check=True):
            if args == ('daemon-reload',):
                raise subprocess.CalledProcessError(1, args)
            return control(*args, check=check)
        module.systemctl = persistent_failure
        try:
            module.transact(fs, updates, current, receipt)
            raise AssertionError('persistent failure ignored')
        except subprocess.CalledProcessError:
            pass
        pending = fs.read(module.JOURNAL)
        assert pending is not None
        module.systemctl = control
        module.restore(fs, json.loads(pending[0]))
        assert fs.read(module.JOURNAL) is None and module.installed(fs)[0] == artifacts
        # Recovery cannot be redirected outside the fixed installation set.
        bad = json.loads(pending[0])
        bad['before']['/etc/unrelated'] = None
        try:
            module.restore(fs, bad)
            raise AssertionError('foreign recovery path accepted')
        except ValueError:
            pass
        module.transact(fs, None, current, receipt)
        assert module.installed(fs) == ({}, None)
        fs.write('/etc/quatrro/broker.json', b'unmanaged', 0o600)
        try:
            module.installed(fs)
            raise AssertionError('unmanaged file accepted')
        except ValueError:
            pass
        fs.remove('/etc/quatrro/broker.json')
        outside = root / 'outside'
        outside.write_text('untouched')
        (filesystem / 'etc/quatrro/broker.json').symlink_to(outside)
        try:
            fs.read('/etc/quatrro/broker.json')
            raise AssertionError('symlink read accepted')
        except OSError:
            pass
        assert outside.read_text() == 'untouched'
    finally:
        fs.close()
print(json.dumps({'fixed_destinations': True, 'hash_validation': True,
                  'apply_remove': True, 'rollback': True, 'pending_recovery': True, 'unmanaged_and_symlink_rejection': True,
                  'systemd_simulated': True}))
