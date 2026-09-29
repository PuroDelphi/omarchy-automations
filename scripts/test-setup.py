#!/usr/bin/env python3
"""Exercise downloaded-runtime setup and Git-managed panel ownership in staging."""
import hashlib
import io
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import setup
import install

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='omarchy-setup-test-') as directory:
    base = Path(directory)
    archive = Path(subprocess.check_output([sys.executable, str(ROOT / 'scripts/package-release.py'),
                                          '--output', str(base / 'artifacts')], text=True).strip())
    checksum = archive.with_name(archive.name + '.sha256')
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    setup.verify_pinned_archive(archive, digest)
    try:
        setup.verify_pinned_archive(archive, '0' * 64)
        raise AssertionError('Unpinned release accepted')
    except ValueError as error:
        assert 'does not match' in str(error)
    def run(script, home, *args):
        subprocess.run([sys.executable, str(script), '--staging-root', str(home), *args], check=True,
                       stdout=subprocess.DEVNULL)
    # Complete managed lifecycle using only the prebuilt archive.
    home = base / 'managed'
    run(ROOT / 'scripts/setup.py', home, '--archive', str(archive))
    run(ROOT / 'scripts/setup.py', home, '--archive', str(archive), '--update')
    run(ROOT / 'scripts/setup.py', home, '--uninstall')
    assert not (home / '.local/bin/quatrrod').exists()
    # Omarchy clone and runtime have separate ownership.
    home = base / 'git'
    plugin = home / '.config/omarchy/plugins/quatrro.automations'
    (plugin / 'scripts').mkdir(parents=True)
    subprocess.run(['git', 'init', '-q', str(plugin)], check=True)
    for relative in install.plugin_sources():
        target = plugin / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(ROOT / relative, target)
    for name in ('setup.py', 'install.py'):
        shutil.copy2(ROOT / 'scripts' / name, plugin / 'scripts' / name)
    before = {p: p.read_bytes() for p in plugin.rglob('*') if p.is_file()}
    run(plugin / 'scripts/setup.py', home, '--archive', str(archive))
    receipt = json.loads((home / '.local/state/quatrro-install/receipt.json').read_text())
    assert receipt['mode'] == 'git-plugin' and len(receipt['files']) == 3 + len(install.SKILL_FILES)
    run(plugin / 'scripts/setup.py', home, '--archive', str(archive), '--update')
    run(plugin / 'scripts/setup.py', home, '--uninstall')
    assert all(p.read_bytes() == raw for p, raw in before.items())
    assert not (home / '.local/bin/quatrrod').exists()
    # Reject corruption before extracting or running package content.
    corrupt = base / 'corrupt.tar.gz'
    corrupt.write_bytes(b'corrupted')
    try:
        setup.extract_verified(corrupt, checksum, base / 'rejected')
        raise AssertionError('corruption accepted')
    except ValueError as error:
        assert 'checksum mismatch' in str(error)
    for name, kind in [('omarchy-automations-0.1.0-dev/../../escape', tarfile.REGTYPE),
                       ('omarchy-automations-0.1.0-dev/link', tarfile.SYMTYPE)]:
        with tarfile.open(corrupt, 'w:gz') as bundle:
            member = tarfile.TarInfo(name)
            member.type = kind
            member.linkname = '/tmp/escape'
            bundle.addfile(member, io.BytesIO())
        sidecar = base / 'bad.sha256'
        sidecar.write_text(hashlib.sha256(corrupt.read_bytes()).hexdigest() + '  ' + setup.ARCHIVE + '\n')
        try:
            setup.extract_verified(corrupt, sidecar, base / 'rejected')
            raise AssertionError('unsafe member accepted')
        except ValueError as error:
            assert 'Unsafe' in str(error)
    assert not (base / 'rejected').exists()
print(json.dumps({'managed_lifecycle': True, 'git_runtime_lifecycle': True,
                  'repository_preserved': True, 'corrupt_and_unsafe_archives_rejected': True}))
