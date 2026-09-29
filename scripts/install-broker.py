#!/usr/bin/env python3
"""Apply/remove an explicitly reviewed administrative bundle. Never activates it."""
import argparse
import base64
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import uuid

FILES = {
    'quatrro-broker': ('/usr/local/libexec/quatrro-broker', 0o755),
    'quatrro-broker.service': ('/etc/systemd/system/quatrro-broker.service', 0o644),
    'quatrro-broker.socket': ('/etc/systemd/system/quatrro-broker.socket', 0o644),
    'org.quatrro.automations.service.policy': ('/usr/share/polkit-1/actions/org.quatrro.automations.service.policy', 0o644),
    'broker.json': ('/etc/quatrro/broker.json', 0o600),
    '00-quatrro-automations.rules': ('/etc/polkit-1/rules.d/00-quatrro-automations.rules', 0o644),
}
STATE = '/var/lib/quatrro-broker-install'
RECEIPT = STATE + '/receipt.json'
JOURNAL = STATE + '/pending.json'
LIMIT = 64 * 1024 * 1024


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def strict_json(raw):
    def pairs(items):
        result = {}
        for key, value in items:
            if key in result:
                raise ValueError('Duplicate JSON key')
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=pairs)


class SafeFS:
    """Descriptor-relative operations; production anchors at / and UID 0."""
    def __init__(self, root='/', owner=0):
        self.root = os.open(root, os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
        self.owner = owner
        self.check_dir(self.root)

    def close(self):
        os.close(self.root)

    def check_dir(self, fd):
        info = os.fstat(fd)
        if not stat.S_ISDIR(info.st_mode) or info.st_uid != self.owner or info.st_mode & 0o022:
            raise ValueError('Unsafe administrative directory')

    @contextmanager
    def parent(self, path, create=False):
        parts = path.split('/')
        if not path.startswith('/') or any(p in ('', '.', '..') for p in parts[1:]):
            raise ValueError('Noncanonical administrative path')
        fd = os.dup(self.root)
        try:
            for part in parts[1:-1]:
                if create:
                    try:
                        os.mkdir(part, 0o755, dir_fd=fd)
                        os.fsync(fd)
                    except FileExistsError:
                        pass
                following = os.open(part, os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=fd)
                os.close(fd)
                fd = following
                self.check_dir(fd)
            yield fd, parts[-1]
        finally:
            os.close(fd)

    def read(self, path, limit=LIMIT):
        try:
            with self.parent(path) as (parent, name):
                fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=parent)
                with os.fdopen(fd, 'rb') as stream:
                    info = os.fstat(stream.fileno())
                    if not stat.S_ISREG(info.st_mode) or info.st_uid != self.owner or info.st_mode & 0o022:
                        raise ValueError('Unsafe administrative file')
                    raw = stream.read(limit + 1)
                    if len(raw) > limit:
                        raise ValueError('Administrative file too large')
                    return raw, stat.S_IMODE(info.st_mode)
        except FileNotFoundError:
            return None

    def write(self, path, raw, mode):
        with self.parent(path, create=True) as (parent, name):
            temporary = '.quatrro-' + uuid.uuid4().hex
            fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, mode, dir_fd=parent)
            try:
                with os.fdopen(fd, 'wb') as stream:
                    os.fchmod(stream.fileno(), mode)
                    stream.write(raw)
                    stream.flush()
                    os.fsync(stream.fileno())
                os.rename(temporary, name, src_dir_fd=parent, dst_dir_fd=parent)
                os.fsync(parent)
            finally:
                try:
                    os.unlink(temporary, dir_fd=parent)
                except FileNotFoundError:
                    pass

    def remove(self, path):
        try:
            with self.parent(path) as (parent, name):
                os.unlink(name, dir_fd=parent)
                os.fsync(parent)
        except FileNotFoundError:
            pass


def read_bundle(directory):
    # Hold one directory FD; reject symlinks and snapshot each bounded file once.
    fd = os.open(directory, os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC)
    def read(name, limit):
        file = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=fd)
        with os.fdopen(file, 'rb') as stream:
            if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
                raise ValueError('Bundle member is not regular')
            raw = stream.read(limit + 1)
            if len(raw) > limit:
                raise ValueError('Bundle member too large')
            return raw
    try:
        manifest = strict_json(read('manifest.json', 1024*1024))
        if manifest.get('version') != 1 or manifest.get('protocol') != 1 or manifest.get('activation') != 'disabled':
            raise ValueError('Incompatible administrative bundle')
        members = manifest.get('files')
        if not isinstance(members, list) or len(members) != len(FILES):
            raise ValueError('Incomplete administrative bundle')
        result = {}
        for member in members:
            name = member['name']
            if name not in FILES or name in result:
                raise ValueError('Unexpected bundle member')
            target, mode = FILES[name]
            if member['target'] != target or member['mode'] != format(mode, '04o'):
                raise ValueError('Unexpected destination or mode')
            raw = read(name, LIMIT if name == 'quatrro-broker' else 65536)
            if len(raw) != member['size'] or digest(raw) != member['sha256']:
                raise ValueError('Bundle integrity check failed')
            result[name] = raw
        return result
    finally:
        os.close(fd)


def systemctl(*args, check=True):
    return subprocess.run(['/usr/bin/systemctl', '--system', '--no-ask-password', *args],
                          env={'PATH': '/usr/bin:/bin', 'LANG': 'C', 'LC_ALL': 'C'},
                          stdin=subprocess.DEVNULL, capture_output=True, timeout=30, check=check)


def stop_managed():
    # Never stop a same-named service loaded from an unrelated location.
    for name in ('quatrro-broker.socket', 'quatrro-broker.service'):
        probe = systemctl('show', name, '--property=LoadState,FragmentPath', check=False)
        properties = dict(line.split('=', 1) for line in probe.stdout.decode().splitlines() if '=' in line)
        if probe.returncode not in (0, 1) or properties.get('LoadState') not in ('loaded', 'not-found'):
            raise ValueError('Cannot inspect broker unit before changing it')
        fragment = properties.get('FragmentPath', '')
        if properties['LoadState'] == 'not-found':
            if fragment:
                raise ValueError('Inconsistent broker unit state')
            continue
        if fragment and fragment != FILES[name][0]:
            raise ValueError('Broker unit is loaded from an unmanaged location')
        if fragment:
            if name.endswith('.socket'):
                systemctl('disable', '--now', name)
            else:
                systemctl('stop', name)


def receipt_for(artifacts):
    return {'version': 1, 'files': {name: digest(raw) for name, raw in artifacts.items()}}


def installed(fs):
    receipt = fs.read(RECEIPT, 65536)
    if receipt is None:
        if any(fs.read(path) is not None for path, _ in FILES.values()):
            raise ValueError('Unmanaged destination exists; refusing overwrite')
        return {}, None
    record = strict_json(receipt[0])
    if record.get('version') != 1 or set(record.get('files', {})) != set(FILES):
        raise ValueError('Invalid installation receipt')
    current = {}
    for name, (path, mode) in FILES.items():
        file = fs.read(path)
        if file is None or file[1] != mode or digest(file[0]) != record['files'][name]:
            raise ValueError('Managed file changed; restore or review before continuing')
        current[name] = file[0]
    return current, receipt


def restore(fs, journal):
    allowed = {path for path, _ in FILES.values()} | {RECEIPT}
    if journal.get('version') != 1 or set(journal.get('before', {})) != allowed:
        raise ValueError('Invalid recovery journal')
    decoded = {}
    for path, entry in journal['before'].items():
        if entry is None:
            decoded[path] = None
            continue
        expected_mode = 0o600 if path == RECEIPT else next(mode for target, mode in FILES.values() if target == path)
        if not isinstance(entry, dict) or entry.get('mode') != expected_mode:
            raise ValueError('Invalid recovery mode')
        raw = base64.b64decode(entry['data'], validate=True)
        if len(raw) > LIMIT:
            raise ValueError('Recovery member too large')
        decoded[path] = (raw, expected_mode)
    stop_managed()
    for path, entry in decoded.items():
        if entry is None:
            fs.remove(path)
        else:
            fs.write(path, *entry)
    systemctl('daemon-reload')
    fs.remove(JOURNAL)


def transact(fs, artifacts, current, receipt):
    before = {path: None if name not in current else {'mode': mode, 'data': base64.b64encode(current[name]).decode()}
              for name, (path, mode) in FILES.items()}
    before[RECEIPT] = None if receipt is None else {'mode': 0o600, 'data': base64.b64encode(receipt[0]).decode()}
    journal = {'version': 1, 'before': before}
    fs.write(JOURNAL, json.dumps(journal).encode(), 0o600)
    try:
        stop_managed()
        for name, (path, mode) in FILES.items():
            if artifacts is None:
                fs.remove(path)
            else:
                fs.write(path, artifacts[name], mode)
        if artifacts is None:
            fs.remove(RECEIPT)
        else:
            fs.write(RECEIPT, json.dumps(receipt_for(artifacts)).encode(), 0o600)
        systemctl('daemon-reload')
        fs.remove(JOURNAL)
    except BaseException:
        # If rollback itself fails, leave the journal for explicit recovery.
        restore(fs, journal)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('operation', choices=('apply', 'remove', 'recover'))
    parser.add_argument('--bundle', type=Path)
    parser.add_argument('--replace-policy', action='store_true', help='Explicitly replace existing administrative grants')
    args = parser.parse_args()
    if os.getuid() != 0 or os.geteuid() != 0:
        parser.exit(1, 'Administrative installation requires root\n')
    if (args.operation == 'apply') != (args.bundle is not None) or (args.replace_policy and args.operation != 'apply'):
        parser.error('apply requires --bundle; --replace-policy applies only to apply')
    fs = SafeFS()
    try:
        artifacts = read_bundle(args.bundle) if args.bundle is not None else None
        with fs.parent(STATE + '/lock', create=True) as (parent, name):
            lock = os.open(name, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600, dir_fd=parent)
            with os.fdopen(lock, 'rb'):
                info = os.fstat(lock)
                if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or stat.S_IMODE(info.st_mode) != 0o600:
                    raise ValueError('Unsafe installation lock')
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                pending = fs.read(JOURNAL, 128*1024*1024)
                if args.operation == 'recover':
                    if pending is None:
                        raise ValueError('No pending installation to recover')
                    restore(fs, strict_json(pending[0]))
                else:
                    if pending is not None:
                        raise ValueError('Pending installation requires recover first')
                    current, receipt = installed(fs)
                    if args.operation == 'apply' and current and not args.replace_policy:
                        for name in ('broker.json', '00-quatrro-automations.rules'):
                            artifacts[name] = current[name]
                    if args.operation == 'remove' and not current:
                        raise ValueError('No managed installation')
                    transact(fs, artifacts, current, receipt)
        print(json.dumps({'operation': args.operation, 'activation': 'disabled'}))
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        parser.exit(1, f'Administrative operation failed: {error}\n')
    finally:
        fs.close()


if __name__ == '__main__':
    main()
