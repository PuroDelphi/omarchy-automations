#!/usr/bin/env python3
"""Widget contract and live engine state in an offscreen, temporary shell harness."""
import json
from native_ui import attach_native_ui
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
ARTIFACTS = ROOT / '.dev'
HARNESS = '''import QtQuick
import Quickshell
import Quickshell.Io
import "qml"
ShellRoot {
    QtObject {
        id: host
        property string target: ""
        property string payload: ""
        property bool panelOpen: false
        function toggle(name, data) { target = name; payload = data; panelOpen = !panelOpen; }
        function summon(name, data) { target = name; payload = data; panelOpen = true; }
        function hide(name) { target = name; panelOpen = false; }
        function isPluginOpen(name) { return name === "quatrro.automations" && panelOpen; }
    }
    QtObject { id: bar; property bool vertical: true; property int barSize: 26; property var shell: host
        property string fontFamily: "monospace"
        property color barForeground: "#eeeeee"
        property color urgent: "#ff7777"
        property bool foregroundAnimationEnabled: false
        function hideTooltip(item) {}
        function showTooltip(item, text) {}
        function registerClickTarget(item) {}
        function unregisterClickTarget(item) {}
    }
    FloatingWindow {
        implicitWidth: 600; implicitHeight: 100
        color: "#111821"
        Row {
            id: canvas
            anchors.centerIn: parent
            spacing: 20
            Widget { id: horizontal; shell: host }
            Widget { id: verticalWidget; bar: bar }
        }
    }
    IpcHandler {
        target: "widget-test"
        function state(): string {
            return JSON.stringify({connected: horizontal.backendClient.connected,
                pending: horizontal.pending, failed: horizontal.failed, opened: horizontal.opened,
                summary: horizontal.summary, vertical: verticalWidget.vertical,
                width: horizontal.width, verticalWidth: verticalWidget.width,
                language: I18n.language, target: host.target, payload: host.payload});
        }
        function refresh(): void {
            horizontal.backendClient.call("status", {});
            verticalWidget.backendClient.call("status", {});
        }
        function activate(vertical: bool): void {
            host.target = "";
            if (vertical) verticalWidget.activate(); else horizontal.activate();
        }
        function panel(open: bool): void {
            if (open) horizontal.open(); else horizontal.close();
        }
        function externalClose(): void { host.panelOpen = false; }
        function press(button: int): void {
            host.target = "";
            verticalWidget.triggerPress(button);
        }
        function capture(path: string): void {
            canvas.grabToImage(function(result) { result.saveToFile(path); });
        }
    }
}
'''


def wait_for(predicate, label):
    until = time.monotonic()+15
    while time.monotonic()<until:
        result = predicate()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError(label)


def main():
    ARTIFACTS.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='quatrro-widget-') as temp:
        directory = Path(temp)
        shutil.copytree(ROOT/'qml', directory/'qml')
        shutil.copy2(ROOT/'Widget.qml', directory/'Widget.qml')
        harness = directory/'shell.qml'
        harness.write_text(HARNESS)
        env = dict(os.environ, QUATRRO_PROFILE=str(directory/'profile'),
                   PATH=str(ROOT/'build')+os.pathsep+os.environ['PATH'],
                   QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
        attach_native_ui(env, directory)
        children = []
        def cli(op, payload=None):
            return json.loads(subprocess.check_output(['quatrroctl',op,'--stdin'],
                input=json.dumps(payload or {}),env=env,text=True,timeout=10))
        def ipc(method, *args):
            return subprocess.check_output(['quickshell','ipc','-p',str(harness),'call','widget-test',method,*args],env=env,text=True,stderr=subprocess.DEVNULL,timeout=5).strip()
        def state():
            try: return json.loads(ipc('state'))
            except (subprocess.CalledProcessError,json.JSONDecodeError): return {}
        with (ARTIFACTS/'widget-test.log').open('w+') as log:
            try:
                engine = subprocess.Popen([ROOT/'build/quatrrod','--listen='],env=env,stdout=log,stderr=log)
                children.append(engine)
                wait_for(lambda:(directory/'profile/runtime/control.sock').exists(),'engine socket')
                ui = subprocess.Popen(['quickshell','-p',str(harness)],env=env,stdout=log,stderr=log)
                children.append(ui)
                wait_for(lambda:state().get('connected'),'widget connected')
                assert state()['language']=='en' and state()['vertical'] and state()['verticalWidth']==26,state()
                for mode in ('false','true'):
                    ipc('activate',mode)
                    assert state()['target']=='quatrro.automations' and state()['payload']=='{}',state()
                ipc('press','2')
                assert state()['target']=='','right click unexpectedly activated panel'
                ipc('press','1')
                assert state()['target']=='quatrro.automations','bar click contract did not open panel'
                ipc('panel','true');assert state()['opened'],state()
                ipc('externalClose');assert not state()['opened'],state()
                ipc('panel','true');ipc('panel','false');assert not state()['opened'],state()
                ready=ARTIFACTS/'widget-ready.png';ready.unlink(missing_ok=True)
                ipc('capture',str(ready));wait_for(ready.exists,'ready icon screenshot')
                config=cli('config.get')
                config['actions']=[{'id':'notice','kind':'notify','title':'Unused fixture','body':'No effect'}]
                config['flows']=[{'id':'widget','name':'Widget fixture','source':'local:widget','steps':['notice'],'enabled':True}]
                cli('config.save',config)
                review=cli('config.preview')
                cli('config.activate',{'hash':review['hash'],'grants':review['capabilities']})
                cli('control',{'paused':True,'admission':'retain'})
                cli('emit',{'source':'local:widget','data':{}})
                ipc('refresh')
                wait_for(lambda:state().get('pending')==1 and 'Paused' in state().get('summary',''),'paused queue displayed')
                for language, term in [('en','pending'),('es','pendientes')]:
                    cli('preferences.set',{'language':language});ipc('refresh')
                    wait_for(lambda:state().get('language')==language and term in state().get('summary',''),'localized widget')
                    assert state()['width']>=16 and state()['width']<=64 and state()['failed']==0,state()
                    assert ('does not stop' if language=='en' else 'no detiene') in state()['summary'],state()
                    capture=ARTIFACTS/('widget-'+language+'.png');capture.unlink(missing_ok=True)
                    ipc('capture',str(capture));wait_for(capture.exists,'widget screenshot')
                cli('permissions.revoke',{'scope':'flow:widget:action:notice'})
                cli('control',{'paused':False,'admission':'retain'})
                wait_for(lambda:cli('status')['counts'].get('denied')==1,'effect denied')
                ipc('refresh');wait_for(lambda:state().get('failed')==1 and state().get('pending')==0,'denied count displayed')
                ui.terminate();ui.wait(timeout=5)
                assert engine.poll() is None and cli('status')['counts']['denied']==1,'unloading widget stopped engine'
                ui=subprocess.Popen(['quickshell','-p',str(harness)],env=env,stdout=log,stderr=log)
                children.append(ui)
                wait_for(lambda:state().get('connected') and state().get('failed')==1,'widget reloaded')
                engine.terminate();engine.wait(timeout=5)
                ipc('refresh');wait_for(lambda:state().get('connected') is False and 'Sin conexión' in state().get('summary',''),'disconnection shown')
                assert 'pendientes' not in state()['summary'],'stale counters presented as current'
                print(json.dumps({'live_states':True,'english_spanish':True,'shell_contract':True,'unload_preserves_engine':True,'reload':True}))
            except BaseException:
                log.flush();log.seek(0);print(log.read());raise
            finally:
                for child in reversed(children):
                    child.terminate()
                    try: child.wait(timeout=5)
                    except subprocess.TimeoutExpired: child.kill();child.wait(timeout=5)


if __name__=='__main__':
    main()
