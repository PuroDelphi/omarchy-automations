#!/usr/bin/env python3
"""Send one fixture through the actual engine/session daemon; dismiss only it."""
import argparse
import html
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

root=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser()
parser.add_argument('--capture-region',help='Explicit compositor rectangle covering the fixture toast')
args=parser.parse_args()

def run(*argv,env=None):
    return subprocess.check_output(argv,text=True,env=env,timeout=10).strip()

def wait_for(predicate,label):
    deadline=time.monotonic()+5
    while time.monotonic()<deadline:
        value=predicate()
        if value:return value
        time.sleep(.05)
    raise AssertionError(label)

assert run('omarchy-shell','lock','isLocked')=='false','session locked'
assert run('omarchy-shell','notifications','dndState')=='off','do not disturb enabled'
popup_dir=Path.home()/'.local/state/omarchy/notifications'
previous=set(popup_dir.glob('*.json'))
marker='Quatrro QA '+str(time.time_ns())[-6:]
title=marker+' <b>literal</b> &'
body='<b>literal</b> & "quoted"'
artifacts=root/'.dev';artifacts.mkdir(exist_ok=True)
with tempfile.TemporaryDirectory(prefix='quatrro-notification-') as temp:
    env=dict(os.environ,QUATRRO_PROFILE=temp)
    with (artifacts/'notification-native.log').open('w') as log:
        engine=subprocess.Popen([root/'build/quatrrod','--listen='],env=env,stdout=log,stderr=log)
        try:
            wait_for(lambda:Path(temp,'runtime/control.sock').exists(),'engine start')
            def cli(op,data=None):
                return json.loads(subprocess.check_output([root/'build/quatrroctl',op,'--stdin'],input=json.dumps(data or {}),env=env,text=True,timeout=10))
            config=cli('config.get')
            config['actions']=[{'id':'notice','kind':'notify','title':title,'body':body}]
            config['flows']=[{'id':'notice','name':'Notification QA','source':'local:notification-qa','enabled':True,'steps':['notice']}]
            cli('config.save',config)
            preview=cli('config.preview')
            cli('config.activate',{'hash':preview['hash'],'grants':preview['capabilities']})
            cli('emit',{'source':'local:notification-qa','data':{}})
            wait_for(lambda:cli('status')['counts'].get('completed')==1,'notification execution completed')
            def snapshot():
                for path in set(popup_dir.glob('*.json'))-previous:
                    try: value=json.loads(path.read_text())
                    except (OSError,json.JSONDecodeError):continue
                    if value.get('summary')==title:return value
            notification=wait_for(snapshot,'real notification daemon fixture')
            assert notification['body']==html.escape(body,quote=True).replace('&#x27;','&#39;').replace('&quot;','&#34;'),notification
            if args.capture_region:
                subprocess.run(['grim','-g',args.capture_region,str(artifacts/'notification-native.png')],check=True,timeout=5)
            print(json.dumps({'session_daemon_received':True,'title_literal':True,'body_escaped':True,'execution_completed':True,'capture_requested':bool(args.capture_region)}))
        finally:
            run('omarchy-shell','notifications','dismiss',marker)
            engine.terminate()
            try:engine.wait(timeout=5)
            except subprocess.TimeoutExpired:engine.kill();engine.wait(timeout=5)
