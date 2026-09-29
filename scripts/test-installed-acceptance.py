#!/usr/bin/env python3
"""Installed acceptance scenarios with real hooks, worker and local HTTPS.

Covers A4 before/after a v4 schema upgrade; opt-in A1 notifications and A3 user services. Does not change the desktop theme or
claim the other roadmap scenarios. All state and hook files live in a private HOME.
"""
import argparse
import hashlib
import hmac
import http.client
import socket
import http.server
import json
import os
from pathlib import Path
import sqlite3
import ssl
import subprocess
import tempfile
import threading
import time
import uuid

ROOT = Path(__file__).resolve().parents[1]
SECRET = 'public-acceptance-receiver-fixture'
received = []
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--native-notifications', action='store_true', help='Also run A1 using the current desktop notification daemon')
parser.add_argument('--user-services', action='store_true', help='Also run A3 using two temporary user services')
args = parser.parse_args()
units = []
marker = 'Omarchy acceptance ' + str(time.time_ns())
notification_directory = Path.home() / '.local/state/omarchy/notifications'
if args.native_notifications:
    for operation, expected in [('lock isLocked', 'false'), ('notifications dndState', 'off')]:
        actual = subprocess.check_output(['omarchy-shell', *operation.split()], text=True, timeout=5).strip()
        assert actual == expected, operation



class Receiver(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        authorized = self.headers.get('Authorization') == 'Bearer ' + SECRET
        received.append(dict(body=body, authorized=authorized,
                             key=self.headers.get('Idempotency-Key'), path=self.path))
        self.send_response(204 if authorized else 401)
        self.end_headers()

    def log_message(self, *_):
        pass


def wait(check, label):
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        if check():
            return
        time.sleep(.05)
    raise AssertionError(label)


with tempfile.TemporaryDirectory(prefix='omarchy-acceptance-') as temporary:
    home = Path(temporary)
    runtime = home / 'runtime'
    runtime.mkdir(mode=0o700)
    subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
                    '-keyout', str(home / 'key.pem'), '-out', str(home / 'cert.pem'),
                    '-days', '1', '-subj', '/CN=localhost', '-addext',
                    'subjectAltName=IP:127.0.0.1'], check=True,
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=20)
    (home / 'key.pem').chmod(0o600)
    receiver = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Receiver)
    tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    tls.load_cert_chain(home / 'cert.pem', home / 'key.pem')
    receiver.socket = tls.wrap_socket(receiver.socket, server_side=True)
    thread = threading.Thread(target=receiver.serve_forever, daemon=True)
    thread.start()
    env = dict(os.environ, HOME=str(home), XDG_CONFIG_HOME=str(home / '.config'),
               XDG_STATE_HOME=str(home / '.local/state'), XDG_RUNTIME_DIR=str(runtime),
               SSL_CERT_FILE=str(home / 'cert.pem'))
    env.pop('QUATRRO_PROFILE', None)
    database = home / '.local/state/quatrro/quatrro.db'
    if args.user_services:
        # systemctl --user needs the real manager's runtime. Keep the engine's
        # socket/config/state separate through its supported profile override.
        env['XDG_RUNTIME_DIR'] = os.environ['XDG_RUNTIME_DIR']
        env['QUATRRO_PROFILE'] = str(home / 'profile')
        database = home / 'profile/state/quatrro.db'

    installer = ['python3', str(ROOT / 'scripts/install.py'), '--staging-root', str(home)]
    engine = None
    results = []

    def ctl(op, data=None, check=True):
        result = subprocess.run([home / '.local/bin/quatrroctl', op, '--stdin'],
                                input=json.dumps(data or {}), text=True, capture_output=True,
                                env=env, timeout=15, check=check)
        return json.loads(result.stdout) if check else result

    def stop():
        global engine
        if engine is not None:
            engine.terminate()
            try:
                engine.wait(timeout=10)
            except subprocess.TimeoutExpired:
                engine.kill()
                engine.wait()
                raise
            finally:
                engine = None

    def run_hook(theme):
        subprocess.run(['bash', '/usr/share/omarchy/bin/omarchy-hook', 'theme-set', theme],
                       env=env, check=True, capture_output=True, timeout=3)

    def completed(count):
        with sqlite3.connect(f'file:{database}?mode=ro', uri=True) as db:
            return db.execute("SELECT count(*) FROM executions WHERE state='completed'").fetchone()[0] == count

    try:
        if args.user_services:
            for role in ('allowed', 'denied'):
                unit = 'omarchy-acceptance-' + role + '-' + uuid.uuid4().hex[:12] + '.service'
                subprocess.run(['systemd-run','--user','--unit='+unit,'--collect','--property=Type=exec',
                                '--','/usr/bin/sleep','300'], check=True, capture_output=True, timeout=10)
                units.append(unit)
        def service_pid(unit):
            value = int(subprocess.check_output(['systemctl','--user','show',unit,'--property=MainPID','--value'], text=True, timeout=5).strip())
            assert value > 0
            return value
        for phase, theme in [('clean', 'tokyo-night'), ('updated', 'catppuccin-latte')]:
            subprocess.run(installer, check=True, capture_output=True, text=True, timeout=30)
            if phase == 'clean':
                subprocess.run(['bash', '/usr/share/omarchy/bin/omarchy-hook-install', 'theme-set',
                                ROOT / 'packaging/hooks/quatrro-theme-set'], env=env,
                               check=True, capture_output=True, timeout=5)
                foreign = home / '.config/omarchy/hooks/theme-set.d/another-tool'
                original = '#!/bin/bash\nprintf "%s" "$1" > "$HOME/foreign-hook-result"\n'
                foreign.write_text(original)
            with (home / (phase + '.log')).open('w') as log:
                with socket.socket() as probe:
                    probe.bind(('127.0.0.1', 0))
                    ingress_port = probe.getsockname()[1]
                engine = subprocess.Popen([home / '.local/bin/quatrrod', f'--listen=127.0.0.1:{ingress_port}'],
                                          env=env, stdout=log, stderr=log)
                wait(lambda: ctl('status', check=False).returncode == 0, 'installed engine startup')
                with sqlite3.connect(database) as db:
                    assert db.execute('SELECT version FROM schema_version').fetchone()[0] == 5
                if phase == 'clean':
                    ctl('secrets.put', dict(id='receiver', backend='file', value=SECRET))
                    config = ctl('config.get')
                    authority = f'127.0.0.1:{receiver.server_port}'
                    config['destinations'] = [dict(id='receiver', url=f'https://{authority}/theme',
                                                   method='POST', format='json', auth='bearer',
                                                   secret='receiver', private_hosts=[authority])]
                    config['actions'] = [dict(id='report', kind='http', destination='receiver',
                                              body='{"theme":"{{data.theme}}","type":"{{type}}"}')]
                    config['flows'] = [dict(id='theme', source='hook:theme-set', enabled=True,
                                            steps=['report'], conditions=[])]
                    if args.native_notifications:
                        ctl('secrets.put', dict(id='github', backend='file', value=SECRET))
                        config['entries'] = [dict(id='deploy', name='Deployment fixture', auth='github', secret='github', format='json', enabled=True)]
                        config['actions'] += [
                            dict(id='notice', kind='notify', title=marker + ' {{data.phase}}', body='Deployment failed: {{data.repository.full_name}}'),
                            dict(id='deploy-report', kind='http', destination='receiver', body='{"repository":"{{data.repository.full_name}}","environment":"{{data.deployment.environment}}","phase":"{{data.phase}}"}')]
                        config['flows'].append(dict(id='deployment', source='entry:deploy', enabled=True,
                            conditions=[dict(field='data.repository.full_name', op='eq', value='example/project'),
                                        dict(field='data.deployment.environment', op='eq', value='staging'),
                                        dict(field='data.deployment_status.state', op='eq', value='failure')],
                            steps=['notice','deploy-report']))
                    if args.user_services:
                        ctl('secrets.put', dict(id='maintenance', backend='file', value=SECRET))
                        config.setdefault('entries', []).append(dict(id='maintenance',name='Maintenance fixture',auth='bearer',secret='maintenance',format='json',enabled=True))
                        config['actions'] += [dict(id='restart',kind='service',unit=units[0],operation='restart'),
                                              dict(id='status',kind='service',unit=units[0],operation='status'),
                                              dict(id='other-restart',kind='service',unit=units[1],operation='restart'),
                                              dict(id='service-report',kind='http',destination='receiver',body='{"unit":"{{data.unit}}","phase":"{{data.phase}}","result":"active"}')]
                        config['flows'] += [dict(id='maintenance',source='entry:maintenance',enabled=True,
                                                conditions=[dict(field='data.unit',op='eq',value=units[0])],steps=['restart','status','service-report']),
                                            dict(id='maintenance-other',source='entry:maintenance',enabled=True,
                                                conditions=[dict(field='data.unit',op='eq',value=units[1])],steps=['other-restart'])]
                    ctl('config.save', config)
                    preview = ctl('config.preview')
                    ctl('config.activate', dict(hash=preview['hash'], grants=preview['capabilities']))
                    if args.user_services:
                        ctl('permissions.revoke', {'scope':'flow:maintenance-other:action:other-restart'})
                    ctl('preferences.set', {'language': 'es'})
                assert ctl('status')['language'] == 'es'
                before = len(received)
                run_hook(theme)
                wait(lambda: len(received) == before + 1, 'hook HTTPS delivery')
                wait(lambda: completed(before + 1), 'persisted completion')
                delivery = received[-1]
                assert delivery['body'] == dict(theme=theme, type='omarchy.theme-set'), delivery
                assert delivery['authorized'] and delivery['key'] and delivery['path'] == '/theme'
                assert (home / 'foreign-hook-result').read_text() == theme
                assert foreign.read_text() == original
                if args.native_notifications:
                    payload = dict(repository={'full_name':'example/project'}, deployment={'environment':'staging'},
                                   deployment_status={'state':'failure'}, phase=phase)
                    raw = json.dumps(payload).encode()
                    def github(body, signing_body, identity):
                        connection = http.client.HTTPConnection('127.0.0.1', ingress_port, timeout=5)
                        try:
                            connection.request('POST', '/hooks/deploy', body, headers={
                                'Content-Type':'application/json', 'X-GitHub-Delivery':identity,
                                'X-Hub-Signature-256':'sha256=' + hmac.new(SECRET.encode(), signing_body, hashlib.sha256).hexdigest()})
                            response = connection.getresponse(); response.read()
                            return response.status
                        finally:
                            connection.close()
                    assert github(raw + b' ', raw, 'invalid-' + phase) == 401
                    assert github(raw, raw, 'valid-' + phase) == 202
                    assert github(raw, raw, 'duplicate-' + phase) == 202
                    other = dict(payload, repository={'full_name':'example/other'})
                    other_raw = json.dumps(other).encode()
                    assert github(other_raw, other_raw, 'other-' + phase) == 202
                    environment_raw = json.dumps(dict(payload, deployment={'environment':'production'})).encode()
                    assert github(environment_raw, environment_raw, 'environment-' + phase) == 202
                    wait(lambda: len(received) == before + 2, 'signed deployment delivery')
                    wait(lambda: completed(before + 2), 'deployment completion')
                    assert received[-1]['body'] == dict(repository='example/project', environment='staging', phase=phase)
                    assert received[-1]['authorized']
                    def notifications():
                        matches = []
                        for path in notification_directory.glob('*.json'):
                            try:
                                value = json.loads(path.read_text())
                            except (OSError, json.JSONDecodeError):
                                continue
                            if value.get('summary') == marker + ' ' + phase:
                                matches.append(value)
                        return matches
                    wait(lambda: len(notifications()) == 1, 'one native deployment notification')
                    assert notifications()[0]['body'] == 'Deployment failed: example/project'
                    with sqlite3.connect(database) as db:
                        expected_phases = 1 if phase == 'clean' else 2
                        assert db.execute("SELECT count(*) FROM events WHERE source='entry:deploy'").fetchone()[0] == 3 * expected_phases
                        assert db.execute("SELECT count(*) FROM executions WHERE flow='deployment'").fetchone()[0] == expected_phases
                    results.append(dict(phase=phase, scenario='A1', signed_body=True, invalid_signature_rejected=True,
                                        duplicate_deduplicated=True, repository_environment_filtered=True,
                                        native_notification=True, https_delivery=True))
                if args.user_services:
                    initial_allowed, initial_other = map(service_pid, units)
                    service_before = len(received)
                    def maintenance(unit, token=SECRET):
                        connection = http.client.HTTPConnection('127.0.0.1', ingress_port, timeout=5)
                        try:
                            connection.request('POST','/hooks/maintenance',json.dumps(dict(unit=unit,phase=phase)),
                                               headers={'Content-Type':'application/json','Authorization':'Bearer '+token,
                                                        'X-Quatrro-Delivery':phase+'-'+unit+'-'+str(token == SECRET)})
                            response=connection.getresponse();response.read()
                            return response.status
                        finally:
                            connection.close()
                    assert maintenance(units[0], 'invalid-fixture-token') == 401
                    assert maintenance(units[0]) == 202
                    wait(lambda: len(received) == service_before + 1, 'service status report')
                    wait(lambda: completed(service_before + 1), 'service workflow completed')
                    restarted = service_pid(units[0])
                    assert restarted != initial_allowed
                    assert service_pid(units[1]) == initial_other
                    assert received[-1]['body'] == dict(unit=units[0],phase=phase,result='active')
                    assert received[-1]['authorized']
                    assert maintenance(units[1]) == 202
                    def denied():
                        with sqlite3.connect(database) as db:
                            return db.execute("SELECT count(*) FROM executions WHERE flow='maintenance-other' AND state='denied'").fetchone()[0] == (1 if phase == 'clean' else 2)
                    wait(denied, 'ungranted service denied')
                    assert service_pid(units[0]) == restarted and service_pid(units[1]) == initial_other
                    assert len(received) == service_before + 1
                    with sqlite3.connect(database) as db:
                        steps = db.execute("SELECT s.state FROM steps s JOIN executions e ON e.id=s.execution WHERE e.flow='maintenance'").fetchall()
                        assert len(steps) == 3 * (1 if phase == 'clean' else 2) and all(state == 'completed' for state, in steps), steps
                    results.append(dict(phase=phase,scenario='A3',authorized_restart_changed_pid=True,
                                        status_checked=True,https_result=True,other_unit_denied=True,
                                        invalid_bearer_rejected=True))
                expected_deliveries = len(received)
                stop()
            started = time.monotonic()
            run_hook('offline-fixture')
            elapsed = time.monotonic() - started
            assert elapsed < 3 and len(received) == expected_deliveries
            assert (home / 'foreign-hook-result').read_text() == 'offline-fixture'
            results.append(dict(phase=phase, scenario='A4', https_delivery=True,
                                persisted_completed=True, foreign_hook_preserved=True,
                                offline_hook_seconds=round(elapsed, 3)))
            if phase == 'clean':
                # Previous schema fixture, retaining completed history, grants,
                # configuration and secrets. No historical binary is executed.
                with sqlite3.connect(database) as db:
                    db.execute('ALTER TABLE executions DROP COLUMN context')
                    db.execute('DELETE FROM schema_version')
                    db.execute('INSERT INTO schema_version VALUES(4)')
        assert len({delivery['key'] for delivery in received}) == 2 * (1 + int(args.native_notifications) + int(args.user_services))
        subprocess.run(installer + ['--uninstall'], check=True, capture_output=True, timeout=30)
        assert foreign.read_text() == original
        print(json.dumps(dict(scenarios=results, host_theme_changed=False,
                              host_profile_changed=False, schema_4_upgrade=True, private_profile_override=args.user_services), indent=2))
    finally:
        stop()
        for unit in units:
            subprocess.run(['systemctl','--user','stop',unit], check=True, timeout=10, capture_output=True)
        try:
            if args.native_notifications:
                subprocess.run(['omarchy-shell','notifications','dismiss',marker], check=True, timeout=5, capture_output=True)
        finally:
            receiver.shutdown()
            receiver.server_close()
            thread.join(timeout=5)
