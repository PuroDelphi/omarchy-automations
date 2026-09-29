#!/usr/bin/env python3
"""Check uninstall against a real temporary user unit; shell command is recorded."""
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from types import SimpleNamespace
from unittest.mock import patch
import uuid

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('installer', ROOT/'scripts/install.py')
installer = importlib.util.module_from_spec(spec);spec.loader.exec_module(installer)
unit = 'omarchy-uninstall-' + uuid.uuid4().hex[:12] + '.service'
unit_path = Path(os.environ['XDG_RUNTIME_DIR'])/'systemd/user'/unit


def systemctl(*args, check=True):
    return subprocess.run(['systemctl','--user',*args],check=check,capture_output=True,text=True,timeout=20)


(ROOT/'.dev').mkdir(exist_ok=True)
# The packaged unit has PrivateTmp=yes, so executable/profile fixtures must be
# outside /tmp to remain visible without weakening that isolation setting.
with tempfile.TemporaryDirectory(prefix='omarchy-uninstall-', dir=ROOT/'.dev') as directory:
    home = Path(directory)
    env = dict(os.environ,QUATRRO_PROFILE=str(home/'profile'))
    installer.install(SimpleNamespace(staging_root=home,activate=False))
    def ctl(op,data=None):
        return json.loads(subprocess.check_output([home/'.local/bin/quatrroctl',op,'--stdin'],
                          input=json.dumps(data or {}),text=True,env=env,timeout=10))
    template=(ROOT/'packaging/systemd/quatrrod.service').read_text()
    template=template.replace('ExecStart=%h/.local/bin/quatrrod',
                              'ExecStart='+str(home/'.local/bin/quatrrod')+' --listen=\nEnvironment=QUATRRO_PROFILE='+str(home/'profile'))
    unit_path.parent.mkdir(parents=True,exist_ok=True)
    unit_path.write_text(template)
    calls=[]
    try:
        systemctl('daemon-reload')
        systemctl('enable','--runtime','--now',unit)
        deadline=time.monotonic()+10
        while not (home/'profile/runtime/control.sock').exists():
            assert time.monotonic()<deadline,'engine did not start'
            time.sleep(.05)
        assert ctl('status')['state']=='ready'
        assert systemctl('is-enabled',unit).stdout.strip()=='enabled-runtime'
        ctl('secrets.put',dict(id='preserved',backend='file',value='public-uninstall-test-fixture'))
        database=home/'profile/state/quatrro.db'
        secret=home/'profile/config/secrets/preserved'
        foreign=home/'.config/omarchy/hooks/theme-set.d/another-tool'
        foreign.parent.mkdir(parents=True);foreign.write_text('foreign fixture')
        def command(argv):
            calls.append(argv)
            if argv==['systemctl','--user','disable','--now','quatrrod.service']:
                systemctl('disable','--runtime','--now',unit)
                assert systemctl('is-active',unit,check=False).returncode!=0
                assert systemctl('is-enabled',unit,check=False).stdout.strip()=='disabled'
            elif argv==['omarchy','plugin','disable','quatrro.automations']:
                # No operation on the user's real bar. The separate native UI
                # validation is responsible for the shell-side behavior.
                assert not (home/'profile/runtime/control.sock').exists()
            elif argv==['systemctl','--user','daemon-reload']:
                systemctl('daemon-reload')
            else:
                raise AssertionError(argv)
        with patch.object(installer,'layout',return_value=(home,home/'.config',home/'.local/state')), patch.object(installer,'run',side_effect=command):
            installer.uninstall(SimpleNamespace(staging_root=None))
        assert len(calls)==3
        assert not (home/'.local/bin/quatrrod').exists()
        assert not (home/'.local/state/quatrro-install/receipt.json').exists()
        assert database.exists() and secret.read_text()=='public-uninstall-test-fixture'
        assert foreign.read_text()=='foreign fixture'
        assert systemctl('is-active',unit,check=False).returncode!=0
        print(json.dumps(dict(real_user_service_stopped=True,runtime_enablement_removed=True,
                              binaries_removed=True,data_and_secret_preserved=True,
                              foreign_hook_preserved=True,shell_disable='recorded-only',host_plugin_unchanged=True)))
    finally:
        systemctl('disable','--runtime','--now',unit,check=False)
        unit_path.unlink(missing_ok=True)
        systemctl('daemon-reload')
