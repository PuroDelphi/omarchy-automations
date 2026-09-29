#!/usr/bin/env python3
"""A2: real disk samples on a bounded private tmpfs, before/after upgrade."""
import http.server
import json
import os
from pathlib import Path
import sqlite3
import ssl
import subprocess
import sys
import tempfile
import threading
import time

ROOT = Path(__file__).resolve().parents[1]
if '--inside' not in sys.argv:
    # The host filesystem is read-only; both writable filesystems are private
    # and bounded. Keep network for the loopback-only HTTPS receiver.
    subprocess.run(['bwrap','--unshare-all','--share-net','--die-with-parent',
                    '--ro-bind','/','/','--size','268435456','--tmpfs','/tmp',
                    '--size','33554432','--tmpfs','/tmp/observed','--proc','/proc','--dev','/dev',
                    '--setenv','TMPDIR','/tmp','python3',str(Path(__file__).resolve()),'--inside'],
                   check=True,timeout=180)
    sys.exit(0)

disk = Path('/tmp/observed')
for path, limit in [(disk,32<<20),(Path('/tmp'),256<<20)]:
    filesystem = subprocess.check_output(['stat','-f','-c','%T',str(path)],text=True).strip()
    info = os.statvfs(path)
    assert filesystem == 'tmpfs' and info.f_blocks * info.f_frsize <= limit, 'refusing host disk'

received = []
SECRET = 'public-monitor-acceptance-fixture-secret'


class Receiver(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        raw = self.rfile.read(int(self.headers['Content-Length']))
        authorized = self.headers.get('Authorization') == 'Bearer ' + SECRET
        received.append(dict(body=json.loads(raw),authorized=authorized,
                             key=self.headers.get('Idempotency-Key')))
        self.send_response(204 if authorized else 401); self.end_headers()

    def log_message(self, *_):
        pass


def wait(check, label, seconds=25):
    deadline = time.monotonic()+seconds
    while time.monotonic()<deadline:
        if check(): return
        time.sleep(.05)
    raise AssertionError(label)


with tempfile.TemporaryDirectory(prefix='omarchy-monitor-') as directory:
    home = Path(directory)
    runtime = home/'runtime';runtime.mkdir(mode=0o700)
    env = dict(os.environ,HOME=str(home),XDG_CONFIG_HOME=str(home/'.config'),
               XDG_STATE_HOME=str(home/'.local/state'),XDG_RUNTIME_DIR=str(runtime),
               SSL_CERT_FILE=str(home/'cert.pem'))
    env.pop('QUATRRO_PROFILE',None)
    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(home/'key.pem'),
                    '-out',str(home/'cert.pem'),'-days','1','-subj','/CN=localhost',
                    '-addext','subjectAltName=IP:127.0.0.1'],check=True,timeout=20,
                   stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    (home/'key.pem').chmod(0o600)
    receiver = http.server.ThreadingHTTPServer(('127.0.0.1',0),Receiver)
    tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);tls.load_cert_chain(home/'cert.pem',home/'key.pem')
    receiver.socket=tls.wrap_socket(receiver.socket,server_side=True)
    thread=threading.Thread(target=receiver.serve_forever,daemon=True);thread.start()
    installer=['python3',str(ROOT/'scripts/install.py'),'--staging-root',str(home)]
    database=home/'.local/state/quatrro/quatrro.db'
    engine=None; results=[]

    def ctl(op,data=None,check=True):
        result=subprocess.run([home/'.local/bin/quatrroctl',op,'--stdin'],env=env,input=json.dumps(data or {}),
                              text=True,capture_output=True,timeout=15,check=check)
        return json.loads(result.stdout) if check else result

    def state():
        with sqlite3.connect(database) as db:
            return db.execute("SELECT phase,since,last,next,value,error FROM monitor_state WHERE id='disk'").fetchone()

    def completed(count):
        with sqlite3.connect(database) as db:
            return db.execute("SELECT count(*) FROM executions WHERE state='completed'").fetchone()[0]==count

    def stop():
        global engine
        if engine:
            engine.terminate()
            try:engine.wait(timeout=10)
            except subprocess.TimeoutExpired:engine.kill();engine.wait();raise
            finally:engine=None

    try:
        for number,phase in enumerate(('clean','updated')):
            subprocess.run(installer,check=True,capture_output=True,timeout=30)
            with (home/(phase+'.log')).open('w') as log:
                engine=subprocess.Popen([home/'.local/bin/quatrrod','--listen='],env=env,stdout=log,stderr=log)
                wait(lambda:ctl('status',check=False).returncode==0,'engine startup')
                if phase=='clean':
                    ctl('secrets.put',dict(id='receiver',backend='file',value=SECRET))
                    config=ctl('config.get');authority=f'127.0.0.1:{receiver.server_port}'
                    config['destinations']=[dict(id='receiver',url=f'https://{authority}/disk',method='POST',format='json',
                                                  auth='bearer',secret='receiver',private_hosts=[authority])]
                    config['actions']=[dict(id='report',kind='http',destination='receiver',
                                            body='{"type":"{{type}}","value":"{{data.value}}","metric":"{{data.metric}}"}')]
                    config['flows']=[dict(id='disk',source='monitor:disk',enabled=True,steps=['report'])]
                    config['monitors']=[dict(id='disk',metric='disk',path=str(disk),threshold=10,recovery=20,
                                             duration_seconds=5,recovery_duration_seconds=5,interval_seconds=5,
                                             cooldown_seconds=10,enabled=True)]
                    ctl('config.save',config);preview=ctl('config.preview')
                    ctl('config.activate',dict(hash=preview['hash'],grants=preview['capabilities']))
                wait(lambda:state() is not None and state()[0]=='normal' and state()[4]>20,'normal disk')
                assert len(received)==2*number
                # Allocate real pages only in the independently bounded 32 MiB tmpfs.
                with (disk/'fill').open('wb') as filler:
                    block=b'x'*(1<<20)
                    for _ in range(30):filler.write(block)
                    filler.flush();os.fsync(filler.fileno())
                wait(lambda:state()[0]=='pending','confirmation pending')
                pending=state();assert pending[4]<10 and not pending[5]
                assert len(received)==2*number, 'alert before confirmation duration'
                wait(lambda:len(received)==2*number+1,'disk alert')
                wait(lambda:completed(2*number+1),'alert completed')
                alert=state();assert alert[2]-pending[1]>=5
                assert received[-1]['body']['type']=='alert' and received[-1]['body']['value']<10
                # More samples at the same level must not generate a notification storm.
                next_sample=state()[3]
                wait(lambda:state()[3]>=next_sample+10,'two stable low samples')
                assert len(received)==2*number+1
                (disk/'fill').unlink()
                wait(lambda:state()[0]=='alert' and state()[1]>0 and state()[4]>20,'recovery confirmation')
                recovering=state();assert len(received)==2*number+1
                wait(lambda:len(received)==2*number+2,'disk recovery')
                wait(lambda:completed(2*number+2),'recovery completed')
                assert time.time()-recovering[1]>=5
                assert received[-1]['body']['type']=='recovered' and received[-1]['body']['value']>20
                assert all(item['authorized'] and item['key'] and item['body']['metric']=='disk' for item in received)
                with sqlite3.connect(database) as db:
                    assert db.execute('SELECT version FROM schema_version').fetchone()[0]==5
                    events=db.execute("SELECT payload FROM events WHERE source='monitor:disk'").fetchall()
                    assert len(events)==2*(number+1)
                    assert [json.loads(row[0])['type'] for row in events].count('alert')==number+1
                results.append(dict(phase=phase,scenario='A2',actual_statfs=True,private_tmpfs_mib=32,
                                    low_percent=pending[4],alert_confirmation=True,no_duplicate_alerts=True,
                                    recovery_confirmation=True,https_deliveries=2))
                print('A2 '+phase+' passed',flush=True)
                stop()
            if phase=='clean':
                with sqlite3.connect(database) as db:
                    db.execute('ALTER TABLE executions DROP COLUMN context');db.execute('DELETE FROM schema_version')
                    db.execute('INSERT INTO schema_version VALUES(4)')
        assert len({item['key'] for item in received})==4
        subprocess.run(installer+['--uninstall'],check=True,capture_output=True,timeout=30)
        print(json.dumps(dict(scenarios=results,host_filesystem_read_only=True,schema_4_upgrade=True),indent=2))
    finally:
        stop();receiver.shutdown();receiver.server_close();thread.join(timeout=5)
