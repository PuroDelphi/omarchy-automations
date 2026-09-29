#!/usr/bin/env python3
"""Run the example adapter through the compiled helper in a temporary unit."""
import json
from pathlib import Path
import subprocess
import uuid

root = Path(__file__).resolve().parents[1]
manifest = json.loads((root/'examples/adapters/status-normalizer.json').read_text())
event = {'version': 1, 'id': 'run-example', 'operation': 'transform',
         'event': {'source': 'local:test', 'type': 'build',
                   'data': {'status': 'failed', 'message': 'Build failed'}}}
request = {'version': 1, 'executable': '/usr/bin/python3',
           'args': ['-I', '-S', '--', '/quatrro/script'],
           'code': (root/'examples/adapters/status-normalizer.py').read_text(),
           'input': json.dumps(event)+'\n', 'output_limit': manifest['output_limit']}
unit = 'quatrro-adapter-test-'+uuid.uuid4().hex
try:
    result = subprocess.run([
        'systemd-run', '--user', '--wait', '--pipe', '--collect', '--quiet',
        '--expand-environment=no', '--unit='+unit,
        '--property=NoNewPrivileges=yes', '--property=MemoryMax=268435456',
        '--property=MemorySwapMax=0', '--property=CPUQuota=50%',
        '--property=TasksMax=32', '--property=RuntimeMaxSec=5s',
        '--property=KillMode=control-group', '--property=TimeoutStopSec=1s',
        '--', str(root/'build/quatrrod'), '--sandbox-runner'],
        input=json.dumps(request), text=True, capture_output=True, check=True, timeout=12)
    response = json.loads(result.stdout)
    assert response == {'version': 1, 'id': event['id'],
                        'data': {'severity': 'error', 'message': 'Build failed'}}
    assert len(result.stdout.encode()) <= manifest['output_limit']
    print(json.dumps({'example_adapter': True, 'isolated_process': True, 'bounded_output': True}))
finally:
    subprocess.run(['systemctl', '--user', 'stop', unit+'.service'],
                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=5)
