#!/usr/bin/env python3
"""Create a deterministic Linux amd64 runtime bundle from verified local builds."""
import argparse
import gzip
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile

import install

ROOT = Path(__file__).resolve().parents[1]


def package(output):
    # Explicit roots: no git metadata, profiles, caches or arbitrary workspace files.
    names = {'LICENSE', 'manifest.json', 'Panel.qml', 'Widget.qml', 'README.md', 'ROADMAP.md',
             'scripts/install.py', 'scripts/install-broker.py',
             'scripts/prepare-broker-install.py'}
    for directory, suffixes in [('qml', {'.qml', '.js', ''}), ('docs', {'.md', '.png'}),
                                ('examples', {'.json', '.py', '.sh'}),
                                ('schemas', {'.json'}), ('packaging', None)]:
        for path in (ROOT / directory).rglob('*'):
            if path.is_symlink():
                raise ValueError('Symlink in release inputs: ' + str(path.relative_to(ROOT)))
            if path.is_file() and (suffixes is None or path.suffix in suffixes):
                names.add(str(path.relative_to(ROOT)))
    names.update('build/' + name for name in ('quatrrod', 'quatrroctl', 'quatrro-broker'))
    files = {}
    for name in sorted(names):
        path = ROOT / name
        if path.is_symlink() or any(p.is_symlink() for p in path.parents if p != ROOT.parent):
            raise ValueError('Symlink input refused: ' + name)
        files[name] = path.read_bytes()
    binaries = {name: files['build/' + name] for name in ('quatrrod', 'quatrroctl')}
    compatibility = install.validate_components(binaries, files['qml/Compatibility.js'])
    version = compatibility['version']
    if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?', version):
        raise ValueError('Invalid artifact version')
    manifest = json.loads(files['manifest.json'])
    if manifest['name'] != 'Omarchy Automations' or manifest['version'] != version.split('-')[0]:
        raise ValueError('Manifest name/version does not match release')
    for name in ('quatrrod', 'quatrroctl', 'quatrro-broker'):
        raw = files['build/' + name]
        if raw[:6] != b'\x7fELF\x02\x01' or int.from_bytes(raw[18:20], 'little') != 62:
            raise ValueError('Only verified Linux amd64 artifacts are supported')
    with tempfile.TemporaryDirectory(prefix='broker-version-') as directory:
        binary = Path(directory) / 'broker'
        binary.write_bytes(files['build/quatrro-broker'])
        binary.chmod(0o700)
        result = subprocess.run([str(binary), '--version'], check=True, timeout=5,
                                capture_output=True, text=True)
        if result.stdout.strip() != 'quatrro-broker protocol 1':
            raise ValueError('Unexpected broker protocol')
    metadata = dict(product='Omarchy Automations', platform='linux-amd64',
                    compatibility=compatibility, broker_protocol=1,
                    status='development' if '-' in version else 'release',
                    files={name: hashlib.sha256(raw).hexdigest() for name, raw in files.items()})
    files['release.json'] = (json.dumps(metadata, sort_keys=True, indent=2) + '\n').encode()
    prefix = 'omarchy-automations-' + version
    output.mkdir(parents=True, exist_ok=True)
    destination = output / (prefix + '-linux-amd64.tar.gz')
    # Stable ordering, ownership, permissions and timestamps, including gzip header.
    with tempfile.TemporaryFile() as temporary:
        with gzip.GzipFile(fileobj=temporary, mode='wb', filename='', mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode='w', format=tarfile.USTAR_FORMAT) as archive:
                for name, raw in sorted(files.items()):
                    info = tarfile.TarInfo(prefix + '/' + name)
                    info.size = len(raw)
                    info.mode = 0o755 if name.startswith('build/') or name.startswith('packaging/hooks/') else 0o644
                    archive.addfile(info, io.BytesIO(raw))
        temporary.seek(0)
        raw = temporary.read()
    destination.write_bytes(raw)
    checksum = hashlib.sha256(raw).hexdigest()
    destination.with_name(destination.name + '.sha256').write_text(checksum + '  ' + destination.name + '\n')
    return destination


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=ROOT / 'dist')
    args = parser.parse_args()
    print(package(args.output.resolve()))
