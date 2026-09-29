#!/usr/bin/env python3
"""Exercise built CLI UI contracts against a controlled Unix endpoint."""
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import threading

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='quatrro-cli-contract-') as temporary:
    profile = Path(temporary)/'profile'
    runtime = profile/'runtime'
    runtime.mkdir(parents=True)
    env = dict(os.environ, QUATRRO_PROFILE=str(profile))
    cli = ROOT/'build/quatrroctl'
    descriptors = [json.loads(subprocess.check_output([ROOT/'build'/name, '--version'], env=env)) for name in ('quatrrod', 'quatrroctl')]
    assert [d['component'] for d in descriptors]==['quatrrod','quatrroctl']
    for field in ('version','protocol','contract','ui_contract'):
        assert descriptors[0][field]==descriptors[1][field], descriptors
    assert not (profile/'state').exists() and not (profile/'config').exists()
    server = socket.socket(socket.AF_UNIX)
    server.bind(str(runtime/'control.sock')); server.listen(1); server.settimeout(.1)
    for contract in (0, 2, None):
        payload = {'contract':contract,'operation':'emit','data':{'secret':'never-forward'}}
        result = subprocess.run([cli,'ui','--stdin'],input=json.dumps(payload),capture_output=True,text=True,env=env,timeout=5)
        assert result.returncode!=0 and 'incompatible Omarchy Automations components' in result.stderr
        try:
            conn,_=server.accept();conn.close();raise AssertionError('invalid UI contract contacted backend')
        except socket.timeout:
            pass
    received=[]
    def respond():
        conn,_=server.accept()
        with conn:
            conn.settimeout(3)
            stream=conn.makefile('rwb')
            hello=json.loads(stream.readline());received.append(hello)
            stream.write(b'{"version":2,"contract":1,"ok":true,"data":{"handshake":"ready"}}\n');stream.flush()
            received.append(json.loads(stream.readline()))
            stream.write(b'{"version":2,"contract":1,"ok":true,"data":{"accepted":true}}\n');stream.flush()
    server.settimeout(5)
    thread=threading.Thread(target=respond);thread.start()
    result=subprocess.run([cli,'ui','--stdin'],input=json.dumps({'contract':1,'operation':'emit','data':{'marker':'unchanged'}}),capture_output=True,text=True,env=env,timeout=5,check=True)
    thread.join(timeout=5);server.close()
    assert len(received)==2 and received[0]['op']=='hello' and 'data' not in received[0]
    assert received[1]['op']=='emit' and received[1]['data']=={'marker':'unchanged'}
    assert json.loads(result.stdout)=={'accepted':True}
print(json.dumps({'metadata_without_startup':True,'invalid_ui_no_connection':True,'valid_ui_negotiated':True}))
