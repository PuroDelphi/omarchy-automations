#!/usr/bin/env python3
"""Exercise text, binary file and rejection paths with fake Wayland and CLI."""
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

SCRIPT=Path(__file__).resolve().parents[1]/'examples/clipboard-bridge/clipboard_bridge.py'
with tempfile.TemporaryDirectory(prefix='clipboard-guide-') as directory:
    home=Path(directory);bin_dir=home/'bin';bin_dir.mkdir()
    (bin_dir/'wl-copy').write_text('''#!/usr/bin/env python3
import os,sys
from pathlib import Path
name='uri' if sys.argv[-1]=='text/uri-list' else 'text'
(Path(os.environ['HOME'])/(name+'.clipboard')).write_bytes(sys.stdin.buffer.read())
''')
    (bin_dir/'quatrroctl').write_text('''#!/usr/bin/env python3
import os,sys
from pathlib import Path
with (Path(os.environ['HOME'])/'events.log').open('a') as out: out.write(sys.stdin.read()+'\\n')
''')
    for path in bin_dir.iterdir(): path.chmod(0o755)
    env=dict(os.environ,HOME=str(home),PATH=str(bin_dir)+os.pathsep+os.environ['PATH'])
    subprocess.run([sys.executable,str(SCRIPT),'init'],env=env,check=True,capture_output=True)
    with socket.socket() as sock:
        sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
    server=subprocess.Popen([sys.executable,str(SCRIPT),'serve','--port',str(port)],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    token=(home/'.config/omarchy-automations/clipboard-bridge.token').read_text().strip()
    def request(path,body,authorization='Bearer '+token):
        req=urllib.request.Request(f'http://127.0.0.1:{port}'+path,body,{'Authorization':authorization},method='POST')
        return urllib.request.urlopen(req,timeout=5)
    try:
        for _ in range(100):
            if server.poll() is not None: raise AssertionError(server.stderr.read())
            try:
                with request('/text',b'hello') as reply:
                    assert json.load(reply)['accepted'];break
            except (ConnectionError,urllib.error.URLError):time.sleep(.05)
        else:raise AssertionError('server did not listen')
        assert (home/'text.clipboard').read_bytes()==b'hello'
        with request('/text', 'Café'.encode()) as reply: assert json.load(reply)['automation_event_sent']
        assert (home/'text.clipboard').read_bytes()=='Café'.encode()
        raw=b'\x00\xff\nexample\x80'
        with request('/file?name=report.bin',raw) as reply:result=json.load(reply)
        assert result['kind']=='file' and result['size']==len(raw)
        saved=list((home/'Downloads/ClipboardInbox').iterdir())
        assert len(saved)==1 and saved[0].read_bytes()==raw
        assert (home/'uri.clipboard').read_bytes()==(saved[0].as_uri()+'\r\n').encode()
        client_file=home/'source.dat';client_file.write_bytes(b'from sender\x00')
        token_file=home/'.config/omarchy-automations/clipboard-bridge.token'
        subprocess.run([sys.executable,str(SCRIPT),'send-text','--port',str(port),'--token-file',str(token_file)],input='Client text',text=True,env=env,check=True,capture_output=True)
        subprocess.run([sys.executable,str(SCRIPT),'send-file',str(client_file),'--port',str(port),'--token-file',str(token_file)],env=env,check=True,capture_output=True)
        assert (home/'text.clipboard').read_bytes()==b'Client text'
        assert any(p.read_bytes()==b'from sender\x00' for p in (home/'Downloads/ClipboardInbox').iterdir())
        for path,body,auth,expected in [('/text',b'bad','Bearer wrong',401),('/file?name=..%2Fescape',b'bad','Bearer '+token,400),('/text',b'\xff','Bearer '+token,400)]:
            try:request(path,body,auth);raise AssertionError('request accepted')
            except urllib.error.HTTPError as error:assert error.code==expected
        assert len(list((home/'Downloads/ClipboardInbox').iterdir()))==2
        events=[json.loads(line) for line in (home/'events.log').read_text().splitlines()]
        assert len(events)==5 and events[-1]['data']=={'kind':'file','name':'source.dat','size':len(b'from sender\x00')}
        # Clipboard success must remain a success even if the automation engine is down.
        (bin_dir/'quatrroctl').write_text('#!/bin/sh\nexit 1\n')
        with request('/file?name=offline.bin', b'offline') as reply: offline=json.load(reply)
        assert offline['accepted'] and not offline['automation_event_sent']
        assert any(p.read_bytes()==b'offline' for p in (home/'Downloads/ClipboardInbox').iterdir())
        print(json.dumps({'text_and_binary_file_received':True,'wayland_mimes':['text/plain','text/uri-list'],'unsafe_requests_rejected':True,'automation_events':len(events),'real_clipboard_changed':False}))
    finally:
        server.terminate();server.wait(timeout=5)
