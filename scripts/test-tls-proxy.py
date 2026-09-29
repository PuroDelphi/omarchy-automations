#!/usr/bin/env python3
"""Validate the shipped Caddy configuration against the real local engine.
Requires CADDY=/path/to/caddy. Never opens a non-loopback listener.
"""
import hashlib
import hmac
import http.client
import json
import os
from pathlib import Path
import socket
import ssl
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
CADDY = os.environ.get('CADDY', 'caddy')


def free_port():
    with socket.socket() as listener:
        listener.bind(('127.0.0.1', 0))
        return listener.getsockname()[1]


def wait_for(predicate, description):
    until = time.monotonic() + 10
    while time.monotonic() < until:
        if predicate():
            return
        time.sleep(.05)
    raise AssertionError(description)


def main():
    with tempfile.TemporaryDirectory(prefix='quatrro-proxy-') as temp:
        p = Path(temp)
        ingress_port, tls_port = free_port(), free_port()
        while tls_port == ingress_port:
            tls_port = free_port()
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
                        '-keyout', str(p/'key.pem'), '-out', str(p/'cert.pem'), '-days', '1',
                        '-subj', '/CN=localhost', '-addext', 'subjectAltName=DNS:localhost,IP:127.0.0.1'],
                       check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        (p/'key.pem').chmod(0o600)
        env = dict(os.environ, QUATRRO_PROFILE=str(p/'profile'),
                   QUATRRO_TLS_CERT=str(p/'cert.pem'), QUATRRO_TLS_KEY=str(p/'key.pem'),
                   QUATRRO_TLS_HOST='localhost', QUATRRO_TLS_PORT=str(tls_port),
                   QUATRRO_TLS_BIND='127.0.0.1', QUATRRO_HTTP_PORT=str(ingress_port),
                   XDG_DATA_HOME=str(p/'data'), XDG_CONFIG_HOME=str(p/'config'))
        config = str(ROOT/'packaging/proxy/Caddyfile')
        adapted = json.loads(subprocess.check_output([CADDY, 'adapt', '--config', config,
                                                      '--adapter', 'caddyfile'], env=env,
                                                     stderr=subprocess.PIPE))
        assert adapted['admin']['disabled'] is True
        assert adapted['admin']['config']['persist'] is False
        servers = adapted['apps']['http']['servers']
        assert all(server['listen'] == [f'127.0.0.1:{tls_port}'] for server in servers.values())
        subprocess.run([CADDY, 'validate', '--config', config, '--adapter', 'caddyfile'],
                       env=env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        children = []
        try:
            with (p/'engine.log').open('w') as engine_log, (p/'proxy.log').open('w') as proxy_log:
                engine = subprocess.Popen([ROOT/'build/quatrrod', f'--listen=127.0.0.1:{ingress_port}'],
                                          env=env, stdout=engine_log, stderr=engine_log)
                children.append(engine)

                def ctl(op, data=None):
                    args = [ROOT/'build/quatrroctl', op]
                    if data is not None:
                        args.append('--stdin')
                    return json.loads(subprocess.check_output(args, env=env, text=True,
                                      input=json.dumps(data) if data is not None else None,
                                      stderr=subprocess.PIPE))

                wait_for(lambda: (p/'profile/runtime/control.sock').exists(), 'engine start')
                secret = 'public-proxy-fixture-signing-secret'
                ctl('secrets.put', {'id': 'fixture', 'backend': 'file', 'value': secret})
                draft = ctl('config.get')
                draft['entries'] = [{'id': 'input', 'name': 'TLS fixture', 'auth': 'hmac',
                                     'secret': 'fixture', 'format': 'json', 'enabled': True}]
                ctl('config.save', draft)
                review = ctl('config.preview')
                ctl('config.activate', {'hash': review['hash'], 'grants': review['capabilities']})
                proxy = subprocess.Popen([CADDY, 'run', '--config', config, '--adapter', 'caddyfile'],
                                         env=env, stdout=proxy_log, stderr=proxy_log)
                children.append(proxy)
                tls = ssl.create_default_context(cafile=str(p/'cert.pem'))
                tls.minimum_version = ssl.TLSVersion.TLSv1_2
                tls.maximum_version = ssl.TLSVersion.TLSv1_2

                def request(path, body=b'', headers=None, method='POST', context=tls):
                    connection = http.client.HTTPSConnection('localhost', tls_port, context=context, timeout=3)
                    try:
                        connection.request(method, path, body, headers or {})
                        response = connection.getresponse()
                        return response.status, response.read()
                    finally:
                        connection.close()

                def ready():
                    if proxy.poll() is not None:
                        raise AssertionError((p/'proxy.log').read_text())
                    try:
                        return request('/', method='GET')[0] == 404
                    except (OSError, http.client.HTTPException):
                        return False

                wait_for(ready, 'TLS proxy start')
                body = '{ "message": "bytes originales: á", "value": 3 }\n'.encode()
                stamp = str(int(time.time()))
                signature = hmac.new(secret.encode(), f'{stamp}.fixture-1.'.encode()+body,
                                     hashlib.sha256).hexdigest()
                headers = {'Content-Type': 'application/json', 'X-Quatrro-Timestamp': stamp,
                           'X-Quatrro-Delivery': 'fixture-1', 'X-Quatrro-Signature': 'sha256='+signature}
                assert request('/hooks/input', body, headers)[0] == 202
                assert request('/hooks/input', body, headers)[0] == 202
                assert ctl('storage.status')['events'] == 1
                assert request('/hooks/input', body+b' ', headers)[0] == 401
                assert request('/hooks/input', body, {'Content-Type': 'application/json',
                                                     'X-Forwarded-For': '127.0.0.1'})[0] == 401
                for path in ['/status', '/config', '/config.activate', '/diagnostics',
                             '/control.sock', '/load', '/hooks/../config', '/hooks/input/extra']:
                    assert request(path, body, headers)[0] == 404, path
                assert request('/hooks/input', method='GET')[0] == 404
                assert request('/hooks/input', b'x'*262145, headers)[0] == 413
                # A trusted test CA is necessary; no insecure TLS flag is used.
                try:
                    request('/', method='GET', context=ssl.create_default_context())
                except ssl.SSLCertVerificationError:
                    pass
                else:
                    raise AssertionError('test certificate trusted unexpectedly')
                engine.terminate(); engine.wait(timeout=5)
                assert request('/hooks/input', body, headers)[0] == 502
                assert secret not in (p/'proxy.log').read_text()
                print(json.dumps({'tls_verified': True, 'signed_bytes_preserved': True,
                                  'events_persisted': 1, 'administration_routes_blocked': True,
                                  'oversize_status': 413, 'offline_status': 502,
                                  'caddy': subprocess.check_output([CADDY, 'version'], text=True).strip()}))
        finally:
            for child in reversed(children):
                if child.poll() is None:
                    child.terminate()
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    child.kill(); child.wait()


if __name__ == '__main__':
    main()
