#!/usr/bin/env python3
"""Prepare reviewable administrative installation artifacts without privileges.

This command does not install, enable or start anything. It uses the production
broker policy parser and rule generator, then records exact destination hashes.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

PROJECT = Path(__file__).resolve().parents[1]


def bounded_read(path, limit):
    with path.open('rb') as stream:
        raw = stream.read(limit + 1)
    if len(raw) > limit:
        raise ValueError(f'Artifact exceeds size limit: {path.name}')
    return raw


def prepare(policy_path, output):
    binary = PROJECT / 'build/quatrro-broker'
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise ValueError('Build quatrro-broker first')
    policy = bounded_read(policy_path, 65536)
    prepared = subprocess.run([str(binary), '--prepare-policy'], input=policy,
                              capture_output=True, timeout=10)
    if prepared.returncode:
        raise ValueError('Invalid policy or unresolved local account')
    data = json.loads(prepared.stdout)
    package = PROJECT / 'packaging/broker'
    entries = [
        ('quatrro-broker', '/usr/local/libexec/quatrro-broker', 0o755, bounded_read(binary, 64*1024*1024)),
        ('quatrro-broker.service', '/etc/systemd/system/quatrro-broker.service', 0o644, bounded_read(package / 'quatrro-broker.service', 65536)),
        ('quatrro-broker.socket', '/etc/systemd/system/quatrro-broker.socket', 0o644, bounded_read(package / 'quatrro-broker.socket', 65536)),
        ('org.quatrro.automations.service.policy', '/usr/share/polkit-1/actions/org.quatrro.automations.service.policy', 0o644,
         bounded_read(package / 'org.quatrro.automations.service.policy', 65536)),
        ('broker.json', '/etc/quatrro/broker.json', 0o600, (json.dumps(data['policy'], indent=2)+'\n').encode()),
        ('00-quatrro-automations.rules', '/etc/polkit-1/rules.d/00-quatrro-automations.rules', 0o644, data['polkit_rules'].encode()),
    ]
    manifest = {'version': 1, 'protocol': 1, 'activation': 'disabled', 'rules': data['policy']['rules'], 'files': []}
    # Exclusive mkdir refuses an existing bundle; no files are overwritten.
    output.mkdir(mode=0o700)
    try:
        for name, target, mode, raw in entries:
            with (output / name).open('xb') as stream:
                os.fchmod(stream.fileno(), mode)
                stream.write(raw)
                stream.flush()
                os.fsync(stream.fileno())
            manifest['files'].append({'name': name, 'target': target, 'mode': format(mode, '04o'),
                                      'size': len(raw), 'sha256': hashlib.sha256(raw).hexdigest()})
        with (output / 'manifest.json').open('x') as stream:
            json.dump(manifest, stream, indent=2)
            stream.write('\n')
            stream.flush()
            os.fsync(stream.fileno())
    except BaseException:
        # Only remove files this invocation could have created in its new directory.
        for name, *_ in entries:
            (output / name).unlink(missing_ok=True)
        (output / 'manifest.json').unlink(missing_ok=True)
        output.rmdir()
        raise
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--policy', type=Path, default=PROJECT / 'packaging/broker/broker.json')
    parser.add_argument('--output', type=Path, required=True, help='New local bundle directory')
    args = parser.parse_args()
    try:
        manifest = prepare(args.policy, args.output)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        parser.exit(1, f'Preparation failed: {error}\n')
    print(json.dumps({'bundle': str(args.output), 'files': len(manifest['files']),
                      'rules': len(manifest['rules']), 'installed': False, 'enabled': False}))


if __name__ == '__main__':
    main()
