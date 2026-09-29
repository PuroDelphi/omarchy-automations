#!/usr/bin/env python3
"""Exercise directory preparation, grants and dispatch through compiled binaries."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

root = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='quatrro-directory-api-') as temporary:
    base = Path(temporary)
    profile = base / 'profile'
    source = base / 'data'
    source.mkdir(mode=0o700)
    code = base / 'write.py'
    code.write_text("import sys\nopen('result','w').write(sys.argv[1])\n")
    env = dict(os.environ, QUATRRO_PROFILE=str(profile))
    engine = subprocess.Popen([root / 'build/quatrrod', '--listen='], env=env,
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    def call(operation, payload=None):
        return json.loads(subprocess.check_output([root / 'build/quatrroctl', operation, '--stdin'],
                          input=json.dumps(payload or {}), text=True, env=env, timeout=20))
    try:
        deadline = time.monotonic() + 5
        while not (profile / 'runtime/control.sock').exists():
            if engine.poll() is not None or time.monotonic() > deadline:
                raise AssertionError('engine did not start')
            time.sleep(.05)
        directory = call('directories.prepare', {'source': str(source), 'target': '/work/data', 'access': 'rw'})
        assert isinstance(directory['inode'], str) and directory['device'] == str(source.stat().st_dev)
        script = call('scripts.prepare', {'id': 'writer', 'path': str(code), 'interpreter': 'python3',
                      'parameters': [{'name': 'message', 'type': 'string', 'max_length': 100}]})
        config = call('config.get')
        config['scripts'] = [script]
        config['actions'] = [{'id': 'write', 'kind': 'script', 'script': script['id'],
                             'script_revision': script['revision'], 'script_bindings': {'message': 'data.message'},
                             'timeout_seconds': 5, 'directories': [directory], 'working_directory': directory['target']}]
        config['flows'] = [{'id': 'mount-flow', 'source': 'local:mount', 'enabled': True, 'steps': ['write']}]
        call('config.save', config)
        preview = call('config.preview')
        call('config.activate', {'hash': preview['hash'], 'grants': preview['capabilities']})
        event = {'source': 'local:mount', 'data': {'message': 'approved directory output'}}
        simulation = call('simulate', event)
        step = simulation['matches'][0]['steps'][0]
        assert step['directories'] == [directory] and step['working_directory'] == '/work/data'
        assert not (source / 'result').exists()
        call('emit', event)
        deadline = time.monotonic() + 10
        while not (source / 'result').exists():
            if time.monotonic() > deadline:
                raise AssertionError('directory action produced no output')
            time.sleep(.05)
        assert (source / 'result').read_text() == event['data']['message']
        print(json.dumps({'api_preparation': True, 'simulation': True, 'approved_script_wrote_directory': True}))
    finally:
        engine.terminate()
        try:
            engine.wait(timeout=10)
        except subprocess.TimeoutExpired:
            engine.kill()
            engine.wait()
