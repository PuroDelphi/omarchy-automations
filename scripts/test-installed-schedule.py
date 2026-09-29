#!/usr/bin/env python3
"""A5: installed timer -> pinned backup script -> HTTPS, then schema upgrade."""
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

ROOT = Path(__file__).resolve().parents[1]
SECRET = 'public-scheduled-backup-fixture-secret'
reports = []


class Receiver(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers['Content-Length']))
        authorized = self.headers.get('Authorization') == 'Bearer ' + SECRET
        reports.append((body, authorized, self.headers.get('Idempotency-Key')))
        self.send_response(204 if authorized else 401)
        self.end_headers()

    def log_message(self, *_):
        pass


def wait(predicate, label, seconds=25):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        if predicate():
            return
        time.sleep(.05)
    raise AssertionError(label)


with tempfile.TemporaryDirectory(prefix='omarchy-installed-schedule-') as directory:
    home = Path(directory)
    source, output = home / 'input', home / 'output'
    source.mkdir(); output.mkdir()
    (home / 'not-mounted').write_text('private host fixture')
    code = home / 'backup.py'
    code.write_text('''import os, sys, time
from pathlib import Path
assert "QUATRRO_ACCEPTANCE_SECRET" not in os.environ
assert not Path(''' + repr(str(home / 'not-mounted')) + ''').exists()
try:
    Path('/work/input/forbidden').write_text('must not write')
except OSError:
    pass
else:
    raise AssertionError('read-only input became writable')
if sys.argv[1] == 'slow':
    time.sleep(3)
    Path('/work/output/late-effect').write_text('timeout failed')
else:
    data = Path('/work/input/document.txt').read_bytes()
    Path('/work/output/backup.tmp').write_bytes(data)
    Path('/work/output/backup.tmp').replace('/work/output/backup.txt')
''')
    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(home/'key.pem'),
                    '-out',str(home/'cert.pem'),'-days','1','-subj','/CN=localhost',
                    '-addext','subjectAltName=IP:127.0.0.1'],check=True,timeout=20,
                   stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    (home/'key.pem').chmod(0o600)
    receiver = http.server.ThreadingHTTPServer(('127.0.0.1',0),Receiver)
    tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    tls.load_cert_chain(home/'cert.pem',home/'key.pem')
    receiver.socket = tls.wrap_socket(receiver.socket,server_side=True)
    thread = threading.Thread(target=receiver.serve_forever,daemon=True); thread.start()
    profile = home/'profile'
    env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(profile),
               SSL_CERT_FILE=str(home/'cert.pem'), QUATRRO_ACCEPTANCE_SECRET=SECRET)
    # Preserve the real user-manager runtime; the supported profile override
    # isolates the installed engine's state/config/socket from the desktop.
    installer = ['python3',str(ROOT/'scripts/install.py'),'--staging-root',str(home)]
    database = profile/'state/quatrro.db'
    engine = None
    results = []

    def ctl(op, data=None, check=True):
        result = subprocess.run([home/'.local/bin/quatrroctl',op,'--stdin'],env=env,
                                input=json.dumps(data or {}),text=True,capture_output=True,check=check,timeout=20)
        return json.loads(result.stdout) if check else result

    def count(flow, state):
        with sqlite3.connect(database) as db:
            return db.execute('SELECT count(*) FROM executions WHERE flow=? AND state=?',(flow,state)).fetchone()[0]

    def stop():
        global engine
        if engine:
            engine.terminate()
            try: engine.wait(timeout=10)
            except subprocess.TimeoutExpired:
                engine.kill(); engine.wait(); raise
            finally: engine = None

    try:
        for number, phase in enumerate(('clean','updated'),1):
            subprocess.run(installer,check=True,capture_output=True,timeout=30)
            payload = ('Backup content for ' + phase + '\n').encode()
            (source/'document.txt').write_bytes(payload)
            with (home/(phase+'.log')).open('w') as log:
                engine = subprocess.Popen([home/'.local/bin/quatrrod','--listen='],env=env,stdout=log,stderr=log)
                wait(lambda: ctl('status',check=False).returncode == 0,'engine startup')
                if phase == 'clean':
                    mounts = [ctl('directories.prepare',dict(source=str(path),target=target,access=access))
                              for path,target,access in [(source,'/work/input','ro'),(output,'/work/output','rw')]]
                    script = ctl('scripts.prepare',dict(id='backup',path=str(code),interpreter='python3',
                                 parameters=[dict(name='mode',type='string',max_length=6,choices=['backup','slow'])]))
                    ctl('secrets.put',dict(id='receiver',backend='file',value=SECRET))
                    config = ctl('config.get')
                    authority = f'127.0.0.1:{receiver.server_port}'
                    config['destinations'] = [dict(id='receiver',url=f'https://{authority}/backup',method='POST',
                                                   format='json',auth='bearer',secret='receiver',private_hosts=[authority])]
                    config['scripts'] = [script]
                    config['actions'] = [dict(id=action,kind='script',script='backup',script_revision=script['revision'],
                                              script_values={'mode':mode},timeout_seconds=timeout,directories=mounts,
                                              working_directory='/work/output')
                                         for action,mode,timeout in [('backup','backup',5),('slow','slow',1)]]
                    config['actions'].append(dict(id='report',kind='http',destination='receiver',
                                                 body='{"result":"backup-completed","scheduled_at":"{{data.scheduled_at}}"}'))
                    config['timers'] = [dict(id='backup',kind='interval',interval_seconds=10,missed='coalesce',enabled=True)]
                    config['flows'] = [dict(id='backup',source='timer:backup',enabled=True,steps=['backup','report']),
                                       dict(id='timeout',source='local:timeout',enabled=True,steps=['slow'])]
                    ctl('config.save',config)
                    preview = ctl('config.preview')
                    ctl('config.activate',dict(hash=preview['hash'],grants=preview['capabilities']))
                    # A file edit must not change an already reviewed revision.
                    code.write_text("raise AssertionError('unapproved source executed')\n")
                wait(lambda: count('backup','completed') == number,'scheduled backup completion')
                assert (output/'backup.txt').read_bytes() == payload
                assert not (source/'forbidden').exists()
                assert len(reports) == number
                body, authorized, key = reports[-1]
                report = json.loads(body)
                assert report['result'] == 'backup-completed' and report['scheduled_at']
                assert authorized and key and SECRET.encode() not in body
                started = time.monotonic()
                ctl('emit',dict(source='local:timeout',data={}))
                wait(lambda: count('timeout','failed') == number,'script timeout',seconds=8)
                failure_seconds = time.monotonic()-started
                time.sleep(3.2)
                assert not (output/'late-effect').exists()
                assert len(reports) == number
                with sqlite3.connect(database) as db:
                    assert db.execute('SELECT version FROM schema_version').fetchone()[0] == 5
                    assert db.execute("SELECT count(*) FROM events WHERE source='timer:backup' AND json_extract(payload,'$.type')='scheduled'").fetchone()[0] == number
                    for table, column in [('events','payload'),('steps','message'),('outbox','body')]:
                        assert not any(SECRET in str(row[0]) for row in db.execute(f'SELECT {column} FROM {table}'))
                results.append(dict(phase=phase,scenario='A5',timer_event=True,backup_bytes_match=True,
                                    pinned_revision=True,read_only_source=True,unmounted_host_file_hidden=True,
                                    secret_environment_absent=True,https_report=True,
                                    timeout_seconds=round(failure_seconds,3),late_effect_prevented=True))
                stop()
            if phase == 'clean':
                with sqlite3.connect(database) as db:
                    db.execute('ALTER TABLE executions DROP COLUMN context')
                    db.execute('DELETE FROM schema_version'); db.execute('INSERT INTO schema_version VALUES(4)')
        assert len({key for _,_,key in reports}) == 2
        subprocess.run(installer+['--uninstall'],check=True,capture_output=True,timeout=30)
        print(json.dumps(dict(scenarios=results,private_profile=True,schema_4_upgrade=True),indent=2))
    finally:
        stop()
        receiver.shutdown();receiver.server_close();thread.join(timeout=5)
