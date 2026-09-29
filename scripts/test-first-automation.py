#!/usr/bin/env python3
"""Reproduce saved-draft vs active-flow onboarding with the real native panel."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT=Path(__file__).resolve().parents[1]
OUTPUT=ROOT/'.dev/first-automation'
OUTPUT.mkdir(parents=True,exist_ok=True)

def wait(check,label):
    deadline=time.monotonic()+20
    while time.monotonic()<deadline:
        value=check()
        if value:return value
        time.sleep(.1)
    raise AssertionError(label)

with tempfile.TemporaryDirectory(prefix='automations-first-flow-') as directory:
    home=Path(directory);ui=home/'ui';ui.mkdir()
    shutil.copytree(ROOT/'qml',ui/'qml')
    shutil.copy2(ROOT/'Panel.qml',ui/'Panel.qml')
    harness=(ROOT/'Development.qml').read_text().replace('loaded: panel.loaded,','loaded: panel.loaded, activeFlows: panel.activeFlowCount, testFlows: panel.realTestFlowCount, guidance: panel.realTestGuidance, section: panel.section,')
    harness=harness.replace('target: "quatrro-development"','''target: "quatrro-development"
        function rejectApproval(): void { panel.approvalDialog.reject(); }
        function reloadActive(): void { panel.refresh(); }
        function approvalCapture(path: string): void { panel.approvalDialog.background.parent.grabToImage(function(r) {r.saveToFile(path);}); }
''',1)
    (ui/'Development.qml').write_text(harness)
    current=home/'.local/state/omarchy/current';current.mkdir(parents=True)
    (current/'theme').symlink_to('/usr/share/omarchy/themes/tokyo-night',target_is_directory=True)
    env=dict(os.environ,HOME=str(home),QUATRRO_PROFILE=str(home/'profile'),PATH=str(ROOT/'build')+os.pathsep+os.environ['PATH'],QT_QPA_PLATFORM='offscreen',QT_QUICK_BACKEND='software')
    attach_native_ui(env,home)
    def control(op,data=None):
        args=[str(ROOT/'build/quatrroctl'),op]
        if data is not None:args+=['--stdin']
        return json.loads(subprocess.check_output(args,input=json.dumps(data) if data is not None else None,text=True,env=env))
    def ipc(method,*args):
        return subprocess.check_output(['quickshell','ipc','-p',str(ui/'Development.qml'),'call','quatrro-development',method,*args],env=env,text=True,stderr=subprocess.DEVNULL,timeout=10)
    def state():
        try:return json.loads(ipc('state'))
        except (subprocess.SubprocessError,ValueError):return {}
    def settled():
        return wait(lambda:s if (s:=state()).get('loaded') and not s.get('busy') and not s.get('queued') else None,'panel settled')
    def capture(method,name):
        image=OUTPUT/name;image.unlink(missing_ok=True);ipc(method,str(image));wait(image.exists,name)
    children=[]
    try:
        with (OUTPUT/'run.log').open('w') as log:
            children.append(subprocess.Popen([ROOT/'build/quatrrod','--listen='],env=env,stdout=log,stderr=log))
            wait(lambda:(home/'profile/runtime/control.sock').exists(),'engine')
            fixture={'version':1,'entries':[],'destinations':[],'monitors':[], 'actions':[{'id':'notice','kind':'notify','title':'Omarchy Automations — tutorial check','body':'{{data.message}}'}], 'flows':[{'id':'local-demo','enabled':True,'source':'local:demo','conditions':[{'field':'data.ready','op':'eq','value':True}],'steps':['notice']}]}
            control('config.save',fixture)
            children.append(subprocess.Popen(['quickshell','-p',str(ui/'Development.qml')],env=env,stdout=log,stderr=log))
            assert settled()['activeFlows']==0
            for language in ('en','es'):
                ipc('language',language);settled()
                ipc('simulate','local:demo',json.dumps({'message':'Tutorial activation check','ready':True}));s=settled()
                assert len(s['simulationReport']['matches'])==1 and not control('history')
                ipc('dismissSimulation');ipc('realTest');ipc('confirmRealTest','true');s=settled()
                assert s['testFlows']==0 and not control('history'),s
                assert ('No active flows' if language=='en' else 'No hay flujos activos') in s['error'],s
                capture('capture',f'before-activation-{language}.png')
            for language in ('en','es'):
                ipc('language',language);settled();ipc('review');settled()
                capture('approvalCapture',f'review-activation-{language}.png')
                # The repeated review is safe; only one explicit activation follows below.
                if language=='en':
                    # close without accepting via a private harness method added below
                    ipc('rejectApproval')
            ipc('activate');s=settled()
            assert s['activeFlows']==1,s
            for language in ('en','es'):
                ipc('language',language);settled()
                ipc('simulate','local:demo',json.dumps({'message':'Tutorial activation check','ready':True}));settled();ipc('dismissSimulation')
                before=len(control('history'))
                ipc('realTest');ipc('confirmRealTest','true');s=settled()
                assert s['section']==4 and ('Executions created: 1' if language=='en' else 'Ejecuciones creadas: 1') in s['notice'],s
                wait(lambda:len(control('history'))==before+1 and control('history')[0]['state']=='completed','notification completion')
                ipc('reloadActive');settled();capture('capture',f'completed-notification-{language}.png')
                ipc('simulate','local:demo',json.dumps({'message':'Must not run','ready':False}));settled();ipc('dismissSimulation')
                ipc('realTest');ipc('confirmRealTest','true');s=settled()
                assert ('No active flow matched' if language=='en' else 'Ningún flujo activo coincidió') in s['notice'],s
                assert len(control('history'))==before+1
    finally:
        for child in reversed(children):
            child.terminate()
            try:child.wait(timeout=5)
            except subprocess.TimeoutExpired:child.kill();child.wait()
    output=(OUTPUT/'run.log').read_text()
    assert not any(x in output for x in ('ERROR:','ReferenceError:','TypeError:')),output
print(json.dumps({'draft_does_not_execute':True,'explicit_activation_required':True,'real_notifications':2,'zero_matches_explained':True,'languages':['en','es']}))
