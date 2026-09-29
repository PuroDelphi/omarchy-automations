#!/usr/bin/env python3
"""Capture genuine native forms, review and completed test history in an isolated profile."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT=Path(__file__).resolve().parents[1]
OUTPUT=ROOT/'.dev/clipboard-guide'
OUTPUT.mkdir(parents=True,exist_ok=True)
FIXTURE=json.loads((ROOT/'examples/clipboard-bridge/automation.json').read_text())

def wait(check,label):
    deadline=time.monotonic()+20
    while time.monotonic()<deadline:
        value=check()
        if value:return value
        time.sleep(.1)
    raise AssertionError(label)

with tempfile.TemporaryDirectory(prefix='clipboard-capture-') as directory:
    home=Path(directory);ui=home/'ui';ui.mkdir()
    shutil.copy2(ROOT/'Panel.qml',ui/'Panel.qml')
    shutil.copytree(ROOT/'qml',ui/'qml')
    harness=(ROOT/'Development.qml').read_text().replace('target: "quatrro-development"','''target: "quatrro-development"
        function guideEdit(kind: string, index: int): void {
            panel.resourceEditor.kind = kind;
            panel.resourceEditor.edit(panel.config[kind][index]);
        }
        function guideTypeSource(value: string): string {
            function visit(item) {
                if (!item) return null;
                if (item.objectName === "field-source") return item;
                for (var child of item.children || []) {
                    var found = visit(child);
                    if (found) return found;
                }
                return null;
            }
            var choice = visit(panel.resourceEditor.contentItem);
            choice.editText = value;
            choice.textEdited(value);
            return panel.resourceEditor.draft.source;
        }
        function guideSourceState(): string {
            function visit(item) {
                if (!item) return null;
                if (item.objectName === "field-source") return { text: item.editText, selected: item.currentText };
                for (var child of item.children || []) {
                    var found = visit(child);
                    if (found) return found;
                }
                return null;
            }
            return JSON.stringify(visit(panel.resourceEditor.contentItem));
        }
        function guideCaptureDialog(path: string): void {
            panel.resourceEditor.background.parent.grabToImage(function(result) { result.saveToFile(path); });
        }
        function guideCaptureApproval(path: string): void {
            panel.approvalDialog.background.parent.grabToImage(function(result) { result.saveToFile(path); });
        }
        function guideReload(): void { panel.backendClient.call("config.get", {}); }
        function guideHistory(): void { panel.notice = ""; panel.section = 4; panel.refresh(); }
        function guideRejectApproval(): void { panel.approvalDialog.reject(); }
''',1)
    (ui/'Development.qml').write_text(harness)
    theme=home/'.local/state/omarchy/current';theme.mkdir(parents=True)
    (theme/'theme').symlink_to('/usr/share/omarchy/themes/tokyo-night',target_is_directory=True)
    fake=home/'bin';fake.mkdir()
    (fake/'notify-send').write_text('#!/bin/sh\nexit 0\n');(fake/'notify-send').chmod(0o755)
    env=dict(os.environ,HOME=str(home),QUATRRO_PROFILE=str(home/'profile'),PATH=str(fake)+os.pathsep+str(ROOT/'build')+os.pathsep+os.environ['PATH'],QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software')
    attach_native_ui(env,home)
    def cli(op,data=None):
        return json.loads(subprocess.check_output([ROOT/'build/quatrroctl',op]+(['--stdin'] if data is not None else []),input=json.dumps(data) if data is not None else None,text=True,env=env))
    def ipc(method,*args):return subprocess.check_output(['quickshell','ipc','-p',str(ui/'Development.qml'),'call','quatrro-development',method,*map(str,args)],env=env,text=True,stderr=subprocess.DEVNULL,timeout=10)
    def state():
        try:return json.loads(ipc('state'))
        except (subprocess.SubprocessError,ValueError):return {}
    def settled():return wait(lambda:s if (s:=state()).get('loaded') and not s.get('busy') and not s.get('queued') else None,'panel settled')
    def capture(method,name):
        path=OUTPUT/name;path.unlink(missing_ok=True);ipc(method,path);wait(path.exists,name)
    children=[]
    try:
        with (OUTPUT/'capture.log').open('w') as log:
            children.append(subprocess.Popen([ROOT/'build/quatrrod','--listen='],env=env,stdout=log,stderr=log))
            wait(lambda:(home/'profile/runtime/control.sock').exists(),'engine')
            shown=json.loads(json.dumps(FIXTURE))
            for flow in shown['flows']: flow['enabled']=True
            cli('config.save',shown)
            children.append(subprocess.Popen(['quickshell','-p',str(ui/'Development.qml')],env=env,stdout=log,stderr=log))
            settled()
            for lang in ('en','es'):
                ipc('language',lang);settled()
                for kind,index,label in [('actions',0,'service-action'),('flows',0,'start-flow'),('flows',1,'received-flow'),('actions',1,'notice-action')]:
                    ipc('guideEdit',kind,index);settled()
                    assert not state()['editorError'],state()['editorError']
                    if kind == 'flows':
                        source=json.loads(ipc('guideSourceState').strip())
                        assert source == {'text': FIXTURE['flows'][index]['source'], 'selected': FIXTURE['flows'][index]['source']}, source
                    capture('guideCaptureDialog',f'clipboard-{label}-{lang}.png')
                    if kind == 'flows':
                        assert ipc('guideTypeSource','local:typed-check').strip() == 'local:typed-check'
                    ipc('dismissEditor')
            # Review both enabled flows, but only activate notification in this private test.
            both=json.loads(json.dumps(FIXTURE))
            for flow in both['flows']:flow['enabled']=True
            cli('config.save',both);ipc('guideReload');settled()
            for lang in ('en','es'):
                ipc('language',lang);settled();ipc('review');settled()
                capture('guideCaptureApproval',f'clipboard-review-{lang}.png')
                ipc('guideRejectApproval')
            notify_only=json.loads(json.dumps(FIXTURE));notify_only['flows'][1]['enabled']=True
            cli('config.save',notify_only)
            preview=cli('config.preview')
            cli('config.activate',{'hash':preview['hash'],'grants':preview['capabilities']})
            ipc('guideReload');settled()
            cli('emit',{'source':'local:clipboard-received','type':'file','data':{'kind':'file','name':'report.pdf','size':4096}})
            wait(lambda:cli('history') and cli('history')[0]['state']=='completed','history completed')
            for lang in ('en','es'):
                ipc('language',lang);settled();ipc('guideHistory');settled()
                capture('capture',f'clipboard-history-{lang}.png')
    finally:
        for child in reversed(children):
            child.terminate()
            try:child.wait(timeout=5)
            except subprocess.TimeoutExpired:child.kill();child.wait()
    output=(OUTPUT/'capture.log').read_text()
    assert not any(error in output for error in ('ERROR:','ReferenceError','TypeError')),output
print(json.dumps({'screenshots':12,'private_profile':True,'real_clipboard_changed':False,'service_started':False}))
