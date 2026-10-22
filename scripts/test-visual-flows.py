#!/usr/bin/env python3
"""Real QML editor and hook lifecycle tests in a temporary development profile."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

root = Path(__file__).resolve().parents[1]
artifacts = root / '.dev'
artifacts.mkdir(exist_ok=True)
for name in ('visual-hooks.png','visual-inline.png','visual-templates-en.png','visual-templates-es.png','visual-templates-narrow.png'):
    (artifacts/name).unlink(missing_ok=True)

def wait(fn, description):
    deadline = time.monotonic() + 20
    while time.monotonic() < deadline:
        result = fn()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError(description)

with tempfile.TemporaryDirectory(prefix='visual-automations-') as directory:
    env = dict(os.environ, QUATRRO_PROFILE=directory, QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software', PATH=str(root/'build')+os.pathsep+os.environ['PATH'])
    attach_native_ui(env, directory)
    with (artifacts/'visual-flows.log').open('w') as log:
        children = []
        try:
            children.append(subprocess.Popen([root/'build/quatrrod','--listen='], env=env, stdout=log, stderr=log))
            wait(lambda: Path(directory,'runtime/control.sock').exists(),'engine startup')
            children.append(subprocess.Popen(['quickshell','-p',str(root/'Development.qml')],env=env,stdout=log,stderr=log))
            def ipc(method,*args):
                return subprocess.check_output(['quickshell','ipc','-p',str(root/'Development.qml'),'call','quatrro-development',method,*args],env=env,text=True,stderr=subprocess.DEVNULL).strip()
            def state():
                try: return json.loads(ipc('state'))
                except (subprocess.CalledProcessError,json.JSONDecodeError): return {}
            wait(lambda: state().get('loaded'),'panel loading')
            assert len([v for v in state()['hookOptions'] if v.startswith('hook:')]) == 6
            ipc('visual','hooks','')
            wait(lambda: len(state().get('hooks',[]))==6,'hook status')
            ipc('capture',str(artifacts/'visual-hooks.png'))
            time.sleep(.3)
            ipc('visual','hook-install','theme-set')
            wait(lambda: state()['hooks'][0]['state']=='installed','hook installation')
            hook = Path(directory,'omarchy-hooks/theme-set.d/quatrro-theme-set')
            assert hook.exists()
            hook.write_text('personal modification')
            ipc('visual','hook-remove','theme-set')
            wait(lambda: state().get('hookError'),'modified hook rejection')
            assert hook.read_text()=='personal modification'
            hook.write_bytes((root/'packaging/hooks/quatrro-theme-set').read_bytes())
            ipc('visual','hook-remove','theme-set')
            wait(lambda: not hook.exists(),'hook removal')
            ipc('openEditor','flows')
            ipc('visual','flow-id','visual-flow')
            ipc('visual','inline','')
            ipc('capture',str(artifacts/'visual-inline.png'))
            time.sleep(.3)
            ipc('visual','inline-save',json.dumps(dict(id='inline-notice',kind='notify',title='Visual notice',body='Hello')))
            wait(lambda: len(state().get('pendingActions',[]))==1,'staged inline action')
            assert not state()['config']['actions'],'inline action saved before parent'
            ipc('visual','cancel-flow','')
            assert not state()['config']['actions'],'cancel leaked inline action'
            ipc('openEditor','flows')
            ipc('visual','flow-id','visual-flow')
            ipc('visual','inline','')
            ipc('visual','inline-save',json.dumps(dict(id='inline-notice',kind='notify',title='Visual notice',body='Hello')))
            ipc('visual','accept-flow','')
            wait(lambda: len(state()['config']['actions'])==1,'atomic inline save')
            assert state()['config']['flows'][0]['steps']==['inline-notice']
            for language in ('en','es'):
                ipc('language',language)
                wait(lambda:state().get('language')==language,'language switch')
                ipc('visual','templates','')
                ipc('capture',str(artifacts/('visual-templates-'+language+'.png')))
                time.sleep(.3)
                ipc('visual','template','0')
            assert len(state()['config']['flows'])==3
            assert all(not f['enabled'] for f in state()['config']['flows'][1:])
            ipc('visual','narrow','')
            time.sleep(.5)
            ipc('capture',str(artifacts/'visual-templates-narrow.png'))
            time.sleep(.3)
            assert (artifacts/'visual-inline.png').exists(), 'inline editor render missing'
            assert (artifacts/'visual-templates-narrow.png').exists(), 'narrow template render missing'
            import struct
            dimensions = struct.unpack('>II',(artifacts/'visual-templates-narrow.png').read_bytes()[16:24])
            assert dimensions[0] <= 712 and dimensions[1] <= 512, dimensions
            # Use the real engine validator on the entire generated library.
            generated = Path(directory,'recipes.json')
            subprocess.run(['node',str(root/'scripts/test-visual-recipes.js'),str(generated)],check=True)
            subprocess.run([root/'build/quatrroctl','config.save','--stdin'],input=generated.read_text(),text=True,env=env,check=True,stdout=subprocess.DEVNULL)
            print('PASS: visual events, adapter ownership, inline save/cancel, bilingual templates and engine validation')
        finally:
            for child in reversed(children):
                child.terminate()
                try: child.wait(timeout=5)
                except subprocess.TimeoutExpired: child.kill(); child.wait()
