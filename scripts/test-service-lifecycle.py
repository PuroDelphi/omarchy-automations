#!/usr/bin/env python3
"""Exercise a private session target and cgroup freeze; never stop the desktop."""
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import time
import uuid

project = Path(__file__).resolve().parents[1]
name = 'quatrro-lifecycle-' + uuid.uuid4().hex[:12]
unit, target = name + '.service', name + '.target'
unit_dir = Path(os.environ['XDG_RUNTIME_DIR']) / 'systemd/user'
unit_dir.mkdir(parents=True, exist_ok=True)


def ctl(*args, check=True):
    return subprocess.run(['systemctl', '--user', *args], check=check,
                          capture_output=True, text=True, timeout=20)


def quote(value):
    return '"' + str(value).replace('\\', '\\\\').replace('"', '\\"').replace('%', '%%') + '"'


def wait(predicate, label):
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        if predicate():
            return
        time.sleep(.05)
    raise AssertionError(label)


(project / '.dev').mkdir(exist_ok=True)
with tempfile.TemporaryDirectory(prefix='lifecycle-', dir=project / '.dev') as directory:
    profile = Path(directory)
    env = {**os.environ, 'QUATRRO_PROFILE': directory}

    def call(op, data=None, check=True):
        p = subprocess.run([str(project / 'build/quatrroctl'), op, '--stdin'],
                           input=json.dumps(data or {}), env=env, capture_output=True,
                           text=True, timeout=5, check=check)
        return json.loads(p.stdout) if check else p.returncode

    def query(sql):
        with sqlite3.connect(f'file:{profile}/state/quatrro.db?mode=ro', uri=True) as db:
            return db.execute(sql).fetchall()

    def prop(key):
        return ctl('show', unit, '--property=' + key, '--value').stdout.strip()

    template = (project / 'packaging/systemd/quatrrod.service').read_text()
    assert 'After=graphical-session.target' in template
    template = template.replace('After=graphical-session.target', 'After=' + target)
    template = template.replace('ExecStart=%h/.local/bin/quatrrod',
                                'ExecStart=' + quote(project / 'build/quatrrod') + ' -listen=\nEnvironment=' + quote('QUATRRO_PROFILE=' + directory))
    (unit_dir / unit).write_text(template)
    (unit_dir / target).write_text('[Unit]\nDescription=Private Quatrro session lifecycle fixture\n')
    try:
        ctl('daemon-reload')
        ctl('start', target, unit)
        wait(lambda: call('status', check=False) == 0, 'engine startup')
        first_pid = prop('MainPID')
        assert first_pid != '0'
        ctl('stop', target)
        assert ctl('is-active', target, check=False).returncode != 0
        assert prop('MainPID') == first_pid and prop('ActiveState') == 'active'
        call('emit', {'id': 'without-session', 'source': 'local:lifecycle'})
        ctl('start', target)
        assert prop('MainPID') == first_pid
        call('control', {'paused': True, 'admission': 'retain'})
        config = json.loads((project / 'examples/notification.json').read_text())
        config['actions'], config['flows'] = [], []
        config['timers'] = [dict(id=policy, kind='interval', interval_seconds=5,
                                 missed=policy, enabled=True) for policy in ('coalesce', 'skip')]
        call('config.save', config)
        preview = call('config.preview')
        call('config.activate', {'hash': preview['hash'], 'grants': preview['capabilities']})
        wait(lambda: len(query('SELECT next_at FROM timer_state')) == 2, 'timer initialization')
        ctl('freeze', unit)
        assert prop('FreezerState') == 'frozen'
        before = query('SELECT next_at FROM timer_state ORDER BY next_at')
        # Exceed a full interval plus the skip tolerance, with no wall-clock edit.
        time.sleep(12)
        assert query('SELECT next_at FROM timer_state ORDER BY next_at') == before
        ctl('thaw', unit)
        wait(lambda: all(row[0] > before[0][0] for row in query('SELECT next_at FROM timer_state')), 'timer catchup')
        counts = dict(query("SELECT source,count(*) FROM events WHERE source LIKE 'timer:%' GROUP BY source"))
        assert counts == {'timer:coalesce': 1}, counts
        assert prop('MainPID') == first_pid
        call('control', {'paused': True, 'admission': 'reject'})
        ctl('stop', unit)
        assert call('status', check=False) != 0
        ctl('start', unit)
        wait(lambda: call('status', check=False) == 0, 'engine restart')
        assert call('status')['paused']
        assert dict(query("SELECT source,count(*) FROM events WHERE source LIKE 'timer:%' GROUP BY source")) == counts
        assert query('PRAGMA integrity_check') == [('ok',)]
        assert query("SELECT count(*) FROM events WHERE id='without-session'") == [(1,)]
        print(json.dumps(dict(private_session_target=True, engine_survives_target_stop=True,
                              cgroup_freeze_thaw=True, coalesce_once=True, skip_overdue=True,
                              stop_start_durable=True, actual_logout_or_suspend=False)))
    finally:
        ctl('thaw', unit, check=False)
        ctl('stop', unit, target, check=False)
        (unit_dir / unit).unlink(missing_ok=True)
        (unit_dir / target).unlink(missing_ok=True)
        ctl('daemon-reload', check=False)
        ctl('reset-failed', unit, target, check=False)
