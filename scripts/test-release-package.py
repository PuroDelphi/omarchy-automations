#!/usr/bin/env python3
"""Verify deterministic archive contents and extracted staging installation."""
import hashlib
import json
import re
import urllib.parse
from pathlib import Path
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='omarchy-package-') as temporary:
    base = Path(temporary)
    def bundle(directory):
        result = subprocess.check_output(['python3', str(ROOT / 'scripts/package-release.py'),
                                          '--output', str(directory)], text=True).strip()
        return Path(result)
    first, second = bundle(base / 'first'), bundle(base / 'second')
    assert first.read_bytes() == second.read_bytes(), 'nondeterministic tar.gz'
    expected = first.with_name(first.name + '.sha256').read_text().split()[0]
    assert hashlib.sha256(first.read_bytes()).hexdigest() == expected
    with tarfile.open(first) as archive:
        members = archive.getmembers()
        assert all(m.isfile() and m.uid == 0 and m.gid == 0 and m.mtime == 0 for m in members)
        assert not any(part in {'.git', '.dev', 'secrets', '__pycache__'}
                       for m in members for part in Path(m.name).parts)
        assert not any(m.name.endswith(('.db', '.db-wal', '.db-shm', '.sock', '.pyc')) for m in members)
        archive.extractall(base / 'extracted', filter='data')
    extracted = next((base / 'extracted').iterdir())
    assert (extracted / 'LICENSE').read_bytes() == (ROOT / 'LICENSE').read_bytes()
    metadata = json.loads((extracted / 'release.json').read_text())
    assert metadata['product'] == 'Omarchy Automations'
    assert metadata['platform'] == 'linux-amd64'
    for language in ('en', 'es'):
        # New chapters and examples must ship with the runtime bundle too.
        for source in (ROOT / 'docs' / language).glob('*.md'):
            relative = source.relative_to(ROOT)
            assert (extracted / relative).read_bytes() == source.read_bytes(), str(relative)
        assert (extracted / 'docs' / language / 'interface.md').is_file()
        assert f'docs/images/native-entry-{language}.png' in metadata['files']
        for theme in ('tokyo-night', 'catppuccin-latte'):
            for section in ('connections', 'security'):
                name = f'docs/images/{theme}-{language}-{section}.png'
                assert name in metadata['files'], name
                assert (extracted / name).read_bytes().startswith(b'\x89PNG\r\n\x1a\n'), name
    for source in (ROOT / 'examples/use-cases').glob('*.json'):
        relative = source.relative_to(ROOT)
        assert (extracted / relative).read_bytes() == source.read_bytes(), str(relative)
    # Documentation is usable after extraction, not just in the source tree.
    for language in ('en', 'es'):
        for page in (extracted / 'docs' / language).glob('*.md'):
            for target in re.findall(r'!?\[[^\]]*\]\(([^\s)]+)\)', page.read_text()):
                link = urllib.parse.urlsplit(target)
                if link.scheme or not link.path:
                    continue
                destination = (page.parent / urllib.parse.unquote(link.path)).resolve()
                assert destination.is_relative_to(extracted), (page.name, target)
                assert destination.is_file(), (page.name, target)
    for name, checksum in metadata['files'].items():
        assert hashlib.sha256((extracted / name).read_bytes()).hexdigest() == checksum, name
    home = base / 'home'
    def install(*args):
        subprocess.run(['python3', str(extracted / 'scripts/install.py'),
                        '--staging-root', str(home), *args], check=True,
                       capture_output=True, text=True, timeout=20)
    install()
    manifest = json.loads((home / '.config/omarchy/plugins/quatrro.automations/manifest.json').read_text())
    assert manifest['name'] == 'Omarchy Automations'
    assert (home / '.local/bin/quatrrod').is_file()
    install()  # Existing receipt / second installation remains usable.
    install('--uninstall')
    assert not (home / '.local/bin/quatrrod').exists()
    assert not (home / '.config/omarchy/plugins/quatrro.automations/manifest.json').exists()
    print(json.dumps(dict(deterministic_archive=True, hashes_verified=True,
                          extracted_install_update_uninstall=True, no_host_activation=True)))
