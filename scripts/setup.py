#!/usr/bin/env python3
"""Install the published Linux amd64 runtime without a Go compiler."""
import argparse
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import subprocess
import sys
import tarfile
import tempfile
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = 'PuroDelphi/omarchy-automations'
TAG = 'v0.1.0-preview.1'
ARCHIVE = 'omarchy-automations-0.1.0-dev-linux-amd64.tar.gz'
MAX_ARCHIVE = 80 * 1024 * 1024
MAX_EXPANDED = 256 * 1024 * 1024


class HTTPSRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        url = urllib.parse.urlsplit(newurl)
        if url.scheme != 'https' or url.hostname not in {'github.com', 'release-assets.githubusercontent.com', 'objects.githubusercontent.com'}:
            raise ValueError('Unexpected release download redirect')
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def download(name, destination, limit):
    url = f'https://github.com/{REPOSITORY}/releases/download/{TAG}/{name}'
    opener = urllib.request.build_opener(HTTPSRedirect())
    with opener.open(url, timeout=60) as response:
        raw = response.read(limit + 1)
    if len(raw) > limit:
        raise ValueError('Download exceeds size limit')
    destination.write_bytes(raw)


def extract_verified(archive, checksum, destination):
    if archive.stat().st_size > MAX_ARCHIVE:
        raise ValueError('Archive exceeds size limit')
    expected = checksum.read_text().strip().split()
    if len(expected) != 2 or not re.fullmatch('[0-9a-f]{64}', expected[0]) or expected[1] != ARCHIVE:
        raise ValueError('Invalid release checksum file')
    if hashlib.sha256(archive.read_bytes()).hexdigest() != expected[0]:
        raise ValueError('Release checksum mismatch')
    with tarfile.open(archive, 'r:gz') as bundle:
        members = bundle.getmembers()
        names = set()
        total = 0
        for member in members:
            path = PurePosixPath(member.name)
            if (not member.isfile() or str(path) != member.name or path.is_absolute() or '..' in path.parts or
                    len(path.parts) < 2 or path.parts[0] != 'omarchy-automations-0.1.0-dev' or
                    member.name in names or member.size > 64 * 1024 * 1024):
                raise ValueError('Unsafe release archive member')
            names.add(member.name)
            total += member.size
        if total > MAX_EXPANDED or len(names) > 4096:
            raise ValueError('Expanded release exceeds limits')
        # Explicit regular-file extraction also works on Python versions before 3.12.
        for member in members:
            target = destination / member.name
            target.parent.mkdir(parents=True, exist_ok=True)
            with bundle.extractfile(member) as source, target.open('wb') as output:
                while block := source.read(1024 * 1024):
                    output.write(block)
            target.chmod(0o755 if '/build/' in member.name else 0o644)
    package = destination / 'omarchy-automations-0.1.0-dev'
    metadata = json.loads((package / 'release.json').read_text())
    if metadata.get('product') != 'Omarchy Automations' or metadata.get('platform') != 'linux-amd64':
        raise ValueError('Unexpected runtime package')
    actual = {str(p.relative_to(package)) for p in package.rglob('*') if p.is_file()} - {'release.json'}
    if actual != set(metadata['files']):
        raise ValueError('Incomplete release manifest')
    for name, digest in metadata['files'].items():
        if hashlib.sha256((package / name).read_bytes()).hexdigest() != digest:
            raise ValueError('Release file checksum mismatch: ' + name)
    return package


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archive', type=Path, help='Use a downloaded archive and its adjacent .sha256 file')
    parser.add_argument('--staging-root', type=Path, help='Preview installation in a temporary HOME; no services')
    parser.add_argument('--uninstall', action='store_true', help='Remove installed runtime; keep data and any Omarchy git checkout')
    parser.add_argument('--update', action='store_true', help='Stop the installed engine before installing the verified package')
    args = parser.parse_args()
    if os.geteuid() == 0:
        parser.error('Run as your desktop user, not root')
    if args.uninstall and (args.update or args.archive):
        parser.error('--uninstall cannot be combined with --update or --archive')
    if args.uninstall:
        command = [sys.executable, str(ROOT / 'scripts/install.py'), '--uninstall']
        if args.staging_root: command += ['--staging-root', str(args.staging_root.resolve())]
        subprocess.run(command, check=True)
        return
    if platform.system() != 'Linux' or platform.machine() not in ('x86_64', 'AMD64'):
        parser.error('The published runtime supports Linux amd64. Build from source for other systems.')
    with tempfile.TemporaryDirectory(prefix='omarchy-automations-setup-') as directory:
        temporary = Path(directory)
        if args.archive:
            archive = args.archive.resolve()
            checksum = archive.with_name(archive.name + '.sha256')
        else:
            archive = temporary / ARCHIVE
            checksum = temporary / (ARCHIVE + '.sha256')
            print(f'Downloading Omarchy Automations {TAG} from {REPOSITORY}…', flush=True)
            download(ARCHIVE, archive, MAX_ARCHIVE)
            download(ARCHIVE + '.sha256', checksum, 4096)
        package = extract_verified(archive, checksum, temporary / 'extracted')
        # Use this reviewed installer, not executable Python from a downloaded archive.
        import install
        install.PROJECT = package
        home, config, state = install.layout(args.staging_root.resolve() if args.staging_root else None)
        plugin = config / 'omarchy/plugins/quatrro.automations'
        git_plugin = plugin if ROOT == plugin and (plugin / '.git').is_dir() else None
        options = argparse.Namespace(staging_root=args.staging_root.resolve() if args.staging_root else None,
                                     activate=not bool(args.staging_root), git_plugin=git_plugin)
        install.git_plugin_directory(options, home, config)
        # Preflight ownership and component compatibility before stopping the service.
        receipt = install.load_receipt(state / 'quatrro-install/receipt.json')
        install.validate_owned(receipt, install.owned_paths(home, config, receipt))
        files, _ = install.artifacts(home, config)
        mode = 'git-plugin' if git_plugin else 'managed'
        if receipt and receipt.get('mode', 'managed') != mode:
            raise ValueError('Uninstall the previous installation before changing its management mode')
        for name in files:
            if git_plugin and Path(name).is_relative_to(git_plugin):
                continue
            install.safe_path(Path(name))
            if Path(name).exists() and (not receipt or name not in receipt['files']):
                raise ValueError('Refusing to replace an unowned file: ' + name)
        if args.update and not args.staging_root:
            subprocess.run(['systemctl', '--user', 'stop', 'quatrrod.service'], check=True)
        install.install(options)
        print('Ready. Open Automations from the Omarchy bar.' if not args.staging_root else 'Staging installation complete; no services activated.')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, KeyError, tarfile.TarError, subprocess.SubprocessError) as error:
        sys.exit('Setup failed: ' + str(error))
