#!/usr/bin/env python3
"""Validate private stdin helper transport in a real transient user unit."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid

root = Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='quatrro-mount-') as temporary:
    path = Path(temporary)
    source = path / 'data'
    source.mkdir(mode=0o700)
    (source / 'input').write_text('approved input')
    st = source.stat()
    request = {
        'version': 1,
        'executable': '/usr/bin/python3',
        'args': ['-I', '-S', '--', '/quatrro/script', '$(touch /tmp/unwanted); --flag'],
        'code': '''import os,sys
assert sys.argv[1] == '$(touch /tmp/unwanted); --flag'
assert not os.path.exists('/tmp/unwanted')
assert open('input').read() == 'approved input'
assert os.getcwd() == '/work/data'
try:
 open('/quatrro/script','w')
 raise AssertionError('writable code')
except OSError: pass
open('result','w').write('completed')
''',
        'directories': [{'source': str(source), 'target': '/work/data', 'access': 'rw',
                         'device': str(st.st_dev), 'inode': str(st.st_ino)}],
        'working_directory': '/work/data',
    }
    unit = 'quatrro-mount-test-' + uuid.uuid4().hex
    args = ['systemd-run', '--user', '--wait', '--pipe', '--collect', '--quiet',
            '--expand-environment=no', '--unit=' + unit,
            '--property=NoNewPrivileges=yes', '--property=MemoryMax=268435456',
            '--property=MemorySwapMax=0', '--property=CPUQuota=50%',
            '--property=TasksMax=32', '--property=RuntimeMaxSec=10s',
            '--property=KillMode=control-group', '--property=TimeoutStopSec=1s',
            '--', str(root / 'build/quatrrod'), '--sandbox-runner']
    try:
        subprocess.run(args, input=json.dumps(request), text=True, check=True, timeout=15,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        assert (source / 'result').read_text() == 'completed'
        # The private helper branch must not create or acquire an engine profile.
        invalid = dict(request, version=2)
        result = subprocess.run([root / 'build/quatrrod', '--sandbox-runner'],
                                input=json.dumps(invalid), text=True, timeout=5,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        assert result.returncode != 0 and 'unsupported sandbox request version' in result.stderr
        print(json.dumps({'descriptor_mount': True, 'script_transport': True,
                          'literal_arguments': True, 'systemd_unit': True}))
    finally:
        subprocess.run(['systemctl', '--user', 'stop', unit + '.service'],
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=5)
