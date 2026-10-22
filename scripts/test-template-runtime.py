#!/usr/bin/env python3
"""Validate every template and repeated insertion using the real engine, without effects."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

root = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='automation-recipes-') as temporary:
    profile = Path(temporary)
    env = dict(os.environ, QUATRRO_PROFILE=temporary)
    fixture = profile/'templates.json'
    subprocess.run(['node', str(root/'scripts/test-visual-recipes.js'), str(fixture)], check=True)
    with (profile/'engine.log').open('w') as log:
        engine = subprocess.Popen([root/'build/quatrrod', '--listen='], env=env, stdout=log, stderr=log)
        try:
            deadline = time.monotonic()+10
            while not (profile/'runtime/control.sock').exists():
                if engine.poll() is not None or time.monotonic()>deadline:
                    raise RuntimeError((profile/'engine.log').read_text())
                time.sleep(.05)
            def call(operation, data=None):
                args = [root/'build/quatrroctl',operation]
                if data is not None:
                    args += ['--stdin']
                return json.loads(subprocess.check_output(args, input=json.dumps(data) if data is not None else None, env=env, text=True))
            call('config.save',json.loads(fixture.read_text()))
            saved = call('config.get')
            assert len(saved['flows'])==39
            assert all(not f['enabled'] for f in saved['flows'])
            assert not call('config.active')['flows'], 'saving templates activated flows'
            print('PASS: all template drafts accepted by the engine; active configuration unchanged')
        finally:
            engine.terminate()
            try: engine.wait(timeout=5)
            except subprocess.TimeoutExpired: engine.kill(); engine.wait()
