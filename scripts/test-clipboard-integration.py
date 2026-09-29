#!/usr/bin/env python3
"""End-to-end isolated bridge -> real engine -> completed notification history."""
import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time

ROOT=Path(__file__).resolve().parents[1]
SCRIPT=ROOT/'examples/clipboard-bridge/clipboard_bridge.py'
with tempfile.TemporaryDirectory(prefix='clipboard-engine-') as directory:
    home=Path(directory);bin_dir=home/'bin';bin_dir.mkdir()
    (bin_dir/'wl-copy').write_text('#!/usr/bin/env python3\nimport sys\nsys.stdin.buffer.read()\n')
    (bin_dir/'notify-send').write_text('#!/bin/sh\nexit 0\n')
    for path in bin_dir.iterdir():path.chmod(0o755)
    env=dict(os.environ,HOME=str(home),QUATRRO_PROFILE=str(home/'profile'),PATH=str(bin_dir)+os.pathsep+str(ROOT/'build')+os.pathsep+os.environ['PATH'])
    def ctl(op,data=None):
        return json.loads(subprocess.check_output([str(ROOT/'build/quatrroctl'),op]+(['--stdin'] if data is not None else []),input=json.dumps(data) if data is not None else None,text=True,env=env))
    def wait(check,label):
        for _ in range(150):
            result=check()
            if result:return result
            time.sleep(.05)
        raise AssertionError(label)
    fixture=json.loads((ROOT/'examples/clipboard-bridge/automation.json').read_text())
    fixture['flows'][1]['enabled']=True
    with (home/'engine.log').open('w') as engine_log, (home/'bridge.log').open('w') as bridge_log:
        engine=subprocess.Popen([ROOT/'build/quatrrod','--listen='],env=env,stdout=engine_log,stderr=engine_log)
        bridge=None
        try:
            wait(lambda:(home/'profile/runtime/control.sock').exists(),'engine socket')
            ctl('config.save',fixture)
            preview=ctl('config.preview')
            ctl('config.activate',{'hash':preview['hash'],'grants':preview['capabilities']})
            subprocess.run([sys.executable,str(SCRIPT),'init'],check=True,env=env,capture_output=True)
            with socket.socket() as probe:probe.bind(('127.0.0.1',0));port=probe.getsockname()[1]
            bridge=subprocess.Popen([sys.executable,str(SCRIPT),'serve','--port',str(port)],env=env,stdout=bridge_log,stderr=bridge_log)
            token=home/'.config/omarchy-automations/clipboard-bridge.token'
            file=home/'sample.bin';file.write_bytes(b'\x00\xffABC')
            for _ in range(100):
                if bridge.poll() is not None:raise AssertionError('bridge exited')
                try:
                    result=subprocess.run([sys.executable,str(SCRIPT),'send-text','--port',str(port),'--token-file',str(token)],input='Hola',text=True,env=env,capture_output=True,timeout=5)
                    if result.returncode==0:break
                except subprocess.TimeoutExpired:pass
                time.sleep(.05)
            else:raise AssertionError('bridge did not listen')
            assert json.loads(result.stdout)['automation_event_sent']
            result=subprocess.run([sys.executable,str(SCRIPT),'send-file',str(file),'--port',str(port),'--token-file',str(token)],env=env,capture_output=True,text=True,check=True)
            assert json.loads(result.stdout)['automation_event_sent']
            history=wait(lambda: h if len(h:=ctl('history'))==2 and all(row['state']=='completed' for row in h) else None,'completed receipt actions')
            assert all(row['flow']=='clipboard-received' for row in history),history
            assert len(list((home/'Downloads/ClipboardInbox').iterdir()))==1
            print(json.dumps({'real_engine_notifications_completed':len(history),'binary_saved':True,'user_clipboard_changed':False,'host_config_changed':False}))
        finally:
            if bridge:bridge.terminate();bridge.wait(timeout=5)
            engine.terminate();engine.wait(timeout=5)
