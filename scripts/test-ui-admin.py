#!/usr/bin/env python3
"""Real QML → engine → privileged broker test, only in the private container."""
import json
from native_ui import attach_native_ui
import os
from pathlib import Path
import subprocess
import tempfile
import time

PROJECT = Path('/project')


def main():
    assert os.getuid() == 1000
    assert Path('/run/systemd/container').read_text().strip() == 'quatrro-test'
    with tempfile.TemporaryDirectory(prefix='quatrro-admin-ui-') as tmp:
        directory = Path(tmp)
        for name in ('runtime', 'home'):
            (directory/name).mkdir(mode=0o700)
        env = {'PATH': '/project/build:/usr/bin:/bin', 'LANG': 'C.UTF-8',
               'HOME': str(directory/'home'), 'XDG_RUNTIME_DIR': str(directory/'runtime'),
               'QUATRRO_PROFILE': str(directory/'profile'), 'QT_QPA_PLATFORM': 'offscreen',
               'QT_QUICK_BACKEND': 'software'}
        attach_native_ui(env, directory)
        children = []
        def run(*args):
            return subprocess.check_output(args, env=env, text=True, stderr=subprocess.DEVNULL, timeout=12).strip()
        def ipc(method, *args):
            return run('quickshell', 'ipc', '-p', str(PROJECT/'Development.qml'), 'call', 'quatrro-development', method, *args)
        def state():
            try:
                return json.loads(ipc('state'))
            except (subprocess.CalledProcessError, json.JSONDecodeError):
                return {}
        def wait(predicate, description):
            deadline = time.monotonic()+12
            while time.monotonic()<deadline:
                value = predicate()
                if value:
                    return value
                time.sleep(.1)
            raise AssertionError(description)
        def settle():
            return wait(lambda: (s if s.get('loaded') and not s.get('busy') and s.get('queued')==0 else None) if (s:=state()) else None, 'UI settled')
        def active():
            return run('systemctl', 'show', 'quatrro-fixture.service', '--property=ActiveState', '--value')
        with (directory/'engine.log').open('w+') as engine_log, (directory/'ui.log').open('w+') as ui_log:
            try:
                children.append(subprocess.Popen(['/project/build/quatrrod', '--listen='], env=env, stdout=engine_log, stderr=engine_log))
                wait(lambda: (directory/'profile/runtime/control.sock').exists(), 'engine socket')
                children.append(subprocess.Popen(['quickshell', '-p', str(PROJECT/'Development.qml')], env=env, stdout=ui_log, stderr=ui_log))
                wait(lambda: state().get('loaded'), 'QML loading')
                assert state()['language']=='en'
                ipc('adminAction')
                ipc('adminOperation', 'start')
                ipc('checkAdministration')
                wait(lambda: state().get('administrationState')=='authorized', 'real authorization visible')
                assert active()=='inactive', 'permission check executed start'
                for language in ('en', 'es'):
                    ipc('language', language); settle()
                    expected = 'Operation currently authorized' if language=='en' else 'Operación autorizada ahora'
                    assert expected in state()['administrationText'], state()
                    image = Path('/captures')/('broker-admin-'+language+'.png')
                    ipc('capture', str(image))
                    wait(image.exists, 'authorization screenshot')
                ipc('adminUnit', 'other.service')
                assert state()['administrationState']=='unchecked'
                ipc('checkAdministration')
                wait(lambda: state().get('administrationState')=='denied', 'real denial visible')
                ipc('adminUnit', 'quatrro-fixture.service')
                ipc('checkAdministration')
                wait(lambda: state().get('administrationState')=='authorized', 'restored authorization')
                ipc('language', 'en'); settle()
                ipc('acceptEditor')
                ipc('resource', 'flows', json.dumps({'id':'admin-ui','name':'Administrative UI test','source':'local:admin-ui','conditions':[], 'steps':['admin-status'], 'enabled':True}))
                ipc('review'); review=settle()
                assert len(review['review']['capabilities'])==1 and 'start quatrro-fixture.service' in review['permissionText'], review
                ipc('activate'); final=settle()
                assert 'activated' in final['notice'] and not final.get('error'), final
                assert active()=='inactive', 'activation executed start'
                run('quatrroctl', 'emit', json.dumps({'source':'local:admin-ui','data':{}}))
                wait(lambda: active()=='active', 'authorized event started fixture')
                wait(lambda: json.loads(run('quatrroctl','status'))['counts'].get('completed')==1, 'administrative execution completed')
                print(json.dumps({'ui_real_authorization':True,'ui_real_denial':True,'english_spanish':True,'activation_without_effect':True,'event_started_fixture':True}), flush=True)
            except BaseException:
                for log in (engine_log, ui_log):
                    log.flush(); log.seek(0); print(log.read(), flush=True)
                raise
            finally:
                for child in reversed(children):
                    child.terminate()
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        child.kill(); child.wait(timeout=5)


if __name__=='__main__':
    main()
