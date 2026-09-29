#!/usr/bin/env python3
"""Calculate and check/write the archive digest pinned by the reviewed setup."""
import argparse
import hashlib
import os
from pathlib import Path
import re
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
ARCHIVE = ROOT / 'dist/omarchy-automations-0.1.0-dev-linux-amd64.tar.gz'
SETUP = ROOT / 'scripts/setup.py'
TAG_RE = re.compile(r'v\d+\.\d+\.\d+-preview\.\d+\Z')
DIGEST_RE = re.compile(r'[0-9a-f]{64}\Z')


def archive_digest(archive):
    if not archive.is_file() or archive.stat().st_size > 80 * 1024 * 1024:
        raise ValueError('Missing or oversized runtime archive')
    digest = hashlib.sha256()
    with archive.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    actual = digest.hexdigest()
    sidecar = archive.with_name(archive.name + '.sha256')
    expected = sidecar.read_text().strip()
    if expected != f'{actual}  {archive.name}':
        raise ValueError('Archive and adjacent .sha256 file disagree')
    with tarfile.open(archive, 'r:gz') as package:
        if any(member.name.endswith('/scripts/setup.py') for member in package):
            raise ValueError('Archive contains setup.py; pinning would be circular')
    return actual


def assignment(source, key):
    pattern = re.compile(rf"^{key} = '([^'\n]+)'$", re.MULTILINE)
    matches = list(pattern.finditer(source))
    if len(matches) != 1:
        raise ValueError(f'Expected one simple {key} assignment in setup.py')
    return matches[0], pattern


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--check', action='store_true', help='Fail if the checkout pin differs from the archive')
    mode.add_argument('--write', action='store_true', help='Update the reviewed setup with the computed digest')
    parser.add_argument('--tag', help='New preview tag for --write; optional expected tag for --check')
    parser.add_argument('--archive', type=Path, default=ARCHIVE)
    parser.add_argument('--setup', type=Path, default=SETUP, help='Setup file; primarily for isolated tests')
    args = parser.parse_args()
    if args.write and not args.tag:
        parser.error('--write requires --tag')
    if args.tag and not TAG_RE.fullmatch(args.tag):
        parser.error('Tag must look like v0.1.0-preview.6')
    digest = archive_digest(args.archive.resolve())
    source = args.setup.read_text()
    tag_match, tag_pattern = assignment(source, 'TAG')
    pin_match, pin_pattern = assignment(source, 'PINNED_ARCHIVE_SHA256')
    current_tag, current_pin = tag_match.group(1), pin_match.group(1)
    if not TAG_RE.fullmatch(current_tag) or not DIGEST_RE.fullmatch(current_pin):
        raise ValueError('Current release pin has an unexpected format')
    if args.check:
        if current_pin != digest or (args.tag and current_tag != args.tag):
            raise ValueError(f'Pin mismatch: checkout {current_tag} {current_pin}; archive {digest}')
    else:
        source, tag_count = tag_pattern.subn(f"TAG = '{args.tag}'", source)
        source, pin_count = pin_pattern.subn(f"PINNED_ARCHIVE_SHA256 = '{digest}'", source)
        if tag_count != 1 or pin_count != 1:
            raise ValueError('Could not update release pin exactly once')
        mode_bits = args.setup.stat().st_mode & 0o777
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(mode='w', dir=args.setup.parent,
                                             prefix='.setup-pin-', delete=False) as output:
                temporary = Path(output.name)
                output.write(source)
            os.chmod(temporary, mode_bits)
            os.replace(temporary, args.setup)
        finally:
            if temporary is not None:
                temporary.unlink(missing_ok=True)
    print(f'{args.tag or current_tag}: {digest}')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, tarfile.TarError) as error:
        raise SystemExit('Release pin failed: ' + str(error))
