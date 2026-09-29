#!/usr/bin/env python3
"""Check the unprivileged artifact-preparation command against actual binaries."""
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

PROJECT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='quatrro-broker-preparation-') as tmp:
    temp = Path(tmp)
    output = temp / 'bundle'
    command = ['python3', str(PROJECT / 'scripts/prepare-broker-install.py'), '--output', str(output)]
    first = subprocess.run(command, check=True, capture_output=True, text=True)
    assert json.loads(first.stdout)['installed'] is False
    manifest = json.loads((output / 'manifest.json').read_text())
    assert manifest['activation'] == 'disabled' and manifest['rules'] == []
    for file in manifest['files']:
        path = output / file['name']
        assert hashlib.sha256(path.read_bytes()).hexdigest() == file['sha256']
        assert path.stat().st_mode & 0o777 == int(file['mode'], 8)
    original = (output / 'manifest.json').read_bytes()
    assert subprocess.run(command, capture_output=True).returncode != 0
    assert (output / 'manifest.json').read_bytes() == original
    invalid = temp / 'invalid.json'
    invalid.write_text('{"version":1,"rules":[],"rules":[]}')
    result = subprocess.run(['python3', str(PROJECT / 'scripts/prepare-broker-install.py'),
                             '--policy', str(invalid), '--output', str(temp / 'invalid-output')], capture_output=True)
    assert result.returncode != 0 and not (temp / 'invalid-output').exists()
print(json.dumps({'default_deny': True, 'artifact_hashes_and_modes': True,
                  'no_overwrite': True, 'invalid_policy_no_bundle': True, 'installed': False}))
