#!/usr/bin/env python3
"""Exercise the preview pin helper without changing the repository setup."""
import hashlib
import io
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
HELPER = ROOT / 'scripts/sync-release-pin.py'

with tempfile.TemporaryDirectory(prefix='release-pin-test-') as directory:
    base = Path(directory)
    archive = base / 'runtime.tar.gz'
    with tarfile.open(archive, 'w:gz') as package:
        data = b'example runtime'
        member = tarfile.TarInfo('package/build/quatrrod')
        member.size = len(data)
        package.addfile(member, io.BytesIO(data))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    sidecar = archive.with_name(archive.name + '.sha256')
    sidecar.write_text(f'{digest}  {archive.name}\n')
    setup = base / 'setup.py'
    setup.write_text("TAG = 'v0.1.0-preview.5'\nPINNED_ARCHIVE_SHA256 = '" + '0' * 64 + "'\n")
    def run(*args, success=True):
        result = subprocess.run([sys.executable, str(HELPER), *args,
                                 '--archive', str(archive), '--setup', str(setup)],
                                capture_output=True, text=True)
        assert (result.returncode == 0) == success, result.stderr + result.stdout
        return result
    run('--check', success=False)
    run('--write', '--tag', 'v0.1.0-preview.6')
    run('--check', '--tag', 'v0.1.0-preview.6')
    assert f"PINNED_ARCHIVE_SHA256 = '{digest}'" in setup.read_text()
    run('--write', '--tag', 'v0.1.0-preview.6')
    sidecar.write_text(f"{'0' * 64}  {archive.name}\n")
    run('--check', success=False)
    sidecar.write_text(f'{digest}  {archive.name}\n')
    with tarfile.open(archive, 'w:gz') as package:
        data = b'circular'
        member = tarfile.TarInfo('package/scripts/setup.py')
        member.size = len(data)
        package.addfile(member, io.BytesIO(data))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    sidecar.write_text(f'{digest}  {archive.name}\n')
    run('--write', '--tag', 'v0.1.0-preview.7', success=False)

print('release pin calculation, check, update and circular-package refusal: OK')
