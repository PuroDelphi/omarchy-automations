#!/usr/bin/env python3
"""Exercise UI contract rejection/recovery with real CLI/engine in a private profile.
Only the temporary Development harness can change the UI contract under test.
"""
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


def wait_for(predicate, description):
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        value = predicate()
        if value:
            return value
        time.sleep(.1)
    raise AssertionError(description)


def main():
    ARTIFACTS.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='quatrro-compat-ui-') as temp:
        directory = Path(temp)
        ui = directory / 'ui'
        ui.mkdir()
        shutil.copytree(ROOT / 'qml', ui / 'qml')
        shutil.copy2(ROOT / 'Panel.qml', ui / 'Panel.qml')
        harness = (ROOT / 'Development.qml').read_text()
        harness = 'import "qml/Compatibility.js" as Compatibility\n' + harness
        harness = harness.replace('target: "quatrro-development"', '''target: "quatrro-development"
        function contract(value: int): void {
            panel.backendClient.polling = false;
            Compatibility.requirements.ui_contract = value;
        }''', 1)
        (ui / 'Development.qml').write_text(harness)
        env = dict(os.environ, QUATRRO_PROFILE=str(directory / 'profile'),
                   PATH=str(ROOT / 'build') + os.pathsep + os.environ['PATH'],
                   QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
        attach_native_ui(env, directory)
        children = []

        def run(*args):
            return subprocess.check_output(args, env=env, text=True,
                                           stderr=subprocess.DEVNULL, timeout=10).strip()

        def ipc(method, *args):
            return run('quickshell', 'ipc', '-p', str(ui / 'Development.qml'),
                       'call', 'quatrro-development', method, *args)

        def state():
            try:
                return json.loads(ipc('state'))
            except (subprocess.CalledProcessError, json.JSONDecodeError):
                return {}

        def settled():
            return wait_for(lambda: s if (s := state()).get('loaded') and
                            not s.get('busy') and s.get('queued') == 0 else None,
                            'UI settled')

        def config():
            return json.loads(run('quatrroctl', 'config.get'))

        with (ARTIFACTS / 'ui-compatibility.log').open('w+') as log:
            try:
                children.append(subprocess.Popen([ROOT / 'build/quatrrod', '--listen='],
                                                 env=env, stdout=log, stderr=log))
                wait_for(lambda: (directory / 'profile/runtime/control.sock').exists(), 'engine socket')
                children.append(subprocess.Popen(['quickshell', '-p', str(ui / 'Development.qml')],
                                                 env=env, stdout=log, stderr=log))
                assert settled()['language'] == 'en'
                baseline = config()
                for language, message in [('en', 'Incompatible Omarchy Automations components.'),
                                          ('es', 'Componentes de Omarchy Automations incompatibles.')]:
                    ipc('language', language)
                    localized = settled()
                    assert localized['language'] == language
                    assert ('HTTP receiver disabled' if language=='en' else 'Receptor HTTP deshabilitado') in localized['ingressText'], localized
                    # Dirty draft must survive rejection, without reaching persistent config.
                    identifier = 'compatibility-' + language
                    ipc('resource', 'actions', json.dumps({'id': identifier, 'name': identifier,
                                                          'type': 'notification', 'title': 'Compatibility',
                                                          'body': 'Preserved draft'}))
                    draft = state()['config']
                    ipc('contract', '999')
                    ipc('save')
                    rejected = settled()
                    assert message in rejected['error'], rejected
                    assert rejected['config'] == draft and rejected['dirty'], rejected
                    assert config() == baseline, 'incompatible UI mutated engine config'
                    capture = ARTIFACTS / ('ui-compatibility-' + language + '.png')
                    capture.unlink(missing_ok=True)
                    ipc('capture', str(capture))
                    wait_for(capture.exists, 'incompatibility screenshot')
                    assert capture.read_bytes().startswith(b'\x89PNG\r\n\x1a\n')
                    ipc('contract', '1')
                    ipc('save')
                    recovered = settled()
                    assert not recovered['error'] and not recovered['dirty'], recovered
                    assert recovered['connected'] and recovered['language'] == language, recovered
                    baseline = config()
                    assert any(action['id'] == identifier for action in baseline['actions']), baseline
                children[0].terminate()
                children[0].wait(timeout=5)
                ipc('refreshStatus')
                disconnected = settled()
                assert not disconnected['connected'] and 'motor desconectado' in disconnected['ingressText'], disconnected
                print(json.dumps({'incompatible_ui_rejected': True, 'english_spanish': True,
                                  'draft_preserved': True, 'compatible_save_recovered': True}))
            except BaseException:
                log.flush(); log.seek(0); print(log.read())
                raise
            finally:
                for child in reversed(children):
                    child.terminate()
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        child.kill(); child.wait(timeout=5)


if __name__ == '__main__':
    main()
