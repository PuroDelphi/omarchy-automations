#!/usr/bin/env python3
"""Check bundled skill examples against an isolated installed runtime, without effects."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='automations-skill-') as directory:
    home = Path(directory)
    subprocess.run(['python3', str(ROOT / 'scripts/install.py'), '--staging-root', str(home)], check=True, stdout=subprocess.DEVNULL)
    skill = home / '.local/share/omarchy-automations/skills/omarchy-automations'
    samples = [json.loads(s) for s in re.findall(r'```json\n(.*?)\n```', (skill / 'references/runtime.md').read_text(), re.S)]
    objects, event, monitor, monitor_event = samples
    env = dict(os.environ, QUATRRO_PROFILE=str(home / 'profile'))
    def ctl(op, data=None):
        return json.loads(subprocess.check_output([str(home / '.local/bin/quatrroctl'), op] + (['--stdin'] if data is not None else []), input=json.dumps(data) if data is not None else None, text=True, env=env))
    with (home / 'engine.log').open('w') as log:
        engine = subprocess.Popen([str(home / '.local/bin/quatrrod'), '--listen='], env=env, stdout=log, stderr=log)
        try:
            for _ in range(100):
                if (home / 'profile/runtime/control.sock').exists(): break
                time.sleep(.05)
            original = {'version':1, 'actions':[{'id':'existing','kind':'notify','title':'Preserve me','body':'Existing'}], 'flows':[]}
            ctl('config.save', original)
            draft = ctl('config.get')
            draft['actions'].append(objects['action'])
            draft.setdefault('flows', []).append(objects['flow'])
            ctl('config.save', draft)
            assert len(ctl('simulate', event)['matches']) == 1
            preview = ctl('config.preview')
            ctl('config.activate', {'hash':preview['hash'], 'grants':preview['capabilities']})
            active = ctl('config.active')
            assert any(a['id']=='existing' and a['body']=='Existing' for a in active['actions'])
            assert any(f['id']=='ai-demo' for f in active['flows'])
            draft['monitors'] = (draft.get('monitors') or []) + [monitor]
            draft['actions'].append({'id':'ai-disk-notice','kind':'notify','title':'Disk','body':'Free disk space: {{data.value}}%'})
            draft['flows'].append({'id':'ai-disk-flow','source':'monitor:ai-disk','enabled':True,'steps':['ai-disk-notice'],'conditions':[{'field':'type','op':'eq','value':'alert'}]})
            ctl('config.save', draft)
            assert len(ctl('simulate', monitor_event)['matches']) == 1
            assert ctl('config.active') == active
            assert ctl('history') == []
            print(json.dumps({'installed_skill_examples_valid':True,'existing_action_preserved':True,'exact_preview_activation':True,'monitor_simulation_only':True,'effects_executed':False}))
        finally:
            engine.terminate()
            engine.wait(timeout=5)
