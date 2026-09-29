#!/usr/bin/env python3
"""Example Wayland clipboard bridge; only reachable over loopback/SSH forwarding."""
import argparse
import getpass
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request

PORT = 8786
MAX_BYTES = 32 * 1024 * 1024
SETTINGS = Path.home() / '.config/omarchy-automations'
TOKEN_FILE = SETTINGS / 'clipboard-bridge.token'
INBOX = Path.home() / 'Downloads/ClipboardInbox'


def initialize():
    SETTINGS.mkdir(mode=0o700, parents=True, exist_ok=True)
    INBOX.mkdir(mode=0o700, parents=True, exist_ok=True)
    if TOKEN_FILE.exists():
        raise SystemExit(f'Token already exists: {TOKEN_FILE}')
    fd = os.open(TOKEN_FILE, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as stream:
        stream.write(secrets.token_urlsafe(32) + '\n')
    print(f'Created private token: {TOKEN_FILE}\nInbox: {INBOX}')


def token():
    path = Path(os.environ.get('CLIPBOARD_BRIDGE_TOKEN_FILE', TOKEN_FILE))
    if not path.is_file() or path.stat().st_mode & 0o077:
        raise ValueError('Token file missing or not private (mode 0600 required)')
    value = path.read_text().strip()
    if len(value) < 32:
        raise ValueError('Token is too short')
    return value


def copy_to_clipboard(data, mime):
    subprocess.run(['wl-copy', '--type', mime], input=data, check=True, timeout=10,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


def emit(kind, name, size):
    payload = {'source': 'local:clipboard-received', 'type': kind,
               'data': {'kind': kind, 'name': name, 'size': size}}
    installed_cli = Path.home() / '.local/bin/quatrroctl'
    cli = str(installed_cli) if installed_cli.is_file() else 'quatrroctl'
    try:
        result = subprocess.run([cli, 'emit', '--stdin'], input=json.dumps(payload),
                                text=True, stdout=subprocess.DEVNULL,
                                stderr=subprocess.DEVNULL, timeout=10)
        return result.returncode == 0
    except (OSError, subprocess.SubprocessError):
        return False


class LocalServer(HTTPServer):
    def get_request(self):
        connection, address = super().get_request()
        connection.settimeout(45)
        return connection, address


class Handler(BaseHTTPRequestHandler):
    server_version = 'OmarchyClipboardExample/1'
    def log_message(self, _format, *args):
        pass  # Never log clipboard contents, names or bearer token.

    def answer(self, code, result):
        raw = json.dumps(result).encode()
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_POST(self):
        expected = self.server.secret
        header = self.headers.get('Authorization', '')
        if not secrets.compare_digest(header, 'Bearer ' + expected):
            self.answer(401, {'error': 'unauthorized'})
            return
        url = urllib.parse.urlsplit(self.path)
        if url.path not in ('/text', '/file') or url.fragment:
            self.answer(404, {'error': 'unknown endpoint'})
            return
        length = self.headers.get('Content-Length', '')
        if not length.isdecimal() or len(length) > 9 or not 0 < int(length) <= MAX_BYTES:
            self.answer(413, {'error': 'missing, empty or oversized body'})
            return
        raw = self.rfile.read(int(length))
        if len(raw) != int(length):
            self.answer(400, {'error': 'incomplete body'})
            return
        kind = 'text' if url.path == '/text' else 'file'
        saved = None
        try:
            if kind == 'text':
                raw.decode('utf-8', 'strict')
                name = 'text'
                copy_to_clipboard(raw, 'text/plain;charset=utf-8')
            else:
                args = urllib.parse.parse_qs(url.query, strict_parsing=True)
                if set(args) != {'name'} or len(args['name']) != 1:
                    raise ValueError('one filename is required')
                name = args['name'][0]
                if (not 1 <= len(name.encode('utf-8')) <= 180 or name in ('.', '..')
                    or '/' in name or '\\' in name or any(ord(c) < 32 for c in name)):
                    raise ValueError('invalid filename')
                # Unique private file; no path traversal or overwrite of existing files.
                saved = INBOX / (secrets.token_hex(8) + '-' + name)
                fd = os.open(saved, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
                with os.fdopen(fd, 'wb') as stream:
                    stream.write(raw)
                copy_to_clipboard((saved.as_uri() + '\r\n').encode(), 'text/uri-list')
            audited = emit(kind, name, len(raw))
            self.answer(200, {'accepted': True, 'kind': kind, 'name': name,
                              'size': len(raw), 'automation_event_sent': audited})
        except (ValueError, UnicodeError):
            if saved: saved.unlink(missing_ok=True)
            self.answer(400, {'error': 'invalid text or filename'})
        except (OSError, subprocess.SubprocessError):
            if saved: saved.unlink(missing_ok=True)
            self.answer(503, {'error': 'clipboard unavailable; check Wayland session'})


def serve(port):
    if not 1024 <= port <= 65535:
        raise SystemExit('Choose an unprivileged port 1024–65535')
    server = LocalServer(('127.0.0.1', port), Handler)
    server.secret = token()
    print(f'Clipboard bridge listening on 127.0.0.1:{port}', flush=True)
    server.serve_forever()


def send(kind, path, port, token_file):
    secret = token_file.read_text().strip() if token_file else getpass.getpass('Receiver token: ')
    if kind == 'text':
        raw = sys.stdin.buffer.read(MAX_BYTES + 1)
        endpoint = '/text'
    else:
        file = Path(path).expanduser().resolve(strict=True)
        if not file.is_file(): raise SystemExit('Expected a regular file')
        if file.stat().st_size > MAX_BYTES: raise SystemExit('File exceeds 32 MiB')
        raw = file.read_bytes()
        endpoint = '/file?' + urllib.parse.urlencode({'name': file.name})
    if not 0 < len(raw) <= MAX_BYTES: raise SystemExit('Empty or oversized transfer')
    request = urllib.request.Request(f'http://127.0.0.1:{port}{endpoint}', raw,
                                     {'Authorization': 'Bearer ' + secret}, method='POST')
    try:
        with urllib.request.urlopen(request, timeout=45) as response:
            print(response.read().decode())
    except urllib.error.HTTPError as error:
        raise SystemExit(f'Receiver error {error.code}: {error.read().decode()}') from error


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='operation', required=True)
    commands.add_parser('init')
    server = commands.add_parser('serve')
    server.add_argument('--port', type=int, default=PORT)
    for kind in ('text', 'file'):
        client = commands.add_parser('send-' + kind)
        if kind == 'file': client.add_argument('path')
        client.add_argument('--port', type=int, default=PORT)
        client.add_argument('--token-file', type=Path)
    args = parser.parse_args()
    if args.operation == 'init': initialize()
    elif args.operation == 'serve': serve(args.port)
    else: send(args.operation[5:], getattr(args, 'path', None), args.port, args.token_file)


if __name__ == '__main__':
    main()
