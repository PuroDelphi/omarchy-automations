#!/usr/bin/env python3
"""Measure installed ingress in an isolated HOME before/after reinstall.

No flows or effects are activated. Results measure admission/persistence, not
command execution or remote delivery capacity. Only loopback listeners are used.
"""
from concurrent.futures import ThreadPoolExecutor
import http.client
import json
import os
from pathlib import Path
import socket
import sqlite3
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
SECRET = 'public-installed-load-fixture-secret'


def usage(pid):
    fields = Path(f'/proc/{pid}/stat').read_text().rsplit(')', 1)[1].split()
    cpu = (int(fields[11]) + int(fields[12])) / os.sysconf('SC_CLK_TCK')
    status = dict(line.split(':', 1) for line in Path(f'/proc/{pid}/status').read_text().splitlines())
    return dict(cpu_seconds=cpu, rss_kib=int(status['VmRSS'].split()[0]),
                peak_rss_kib=int(status['VmHWM'].split()[0]))


with tempfile.TemporaryDirectory(prefix='omarchy-installed-load-') as temporary:
    home = Path(temporary)
    runtime = home / 'runtime'
    runtime.mkdir(mode=0o700)
    env = dict(os.environ, HOME=str(home), XDG_CONFIG_HOME=str(home / '.config'),
               XDG_STATE_HOME=str(home / '.local/state'), XDG_RUNTIME_DIR=str(runtime))
    env.pop('QUATRRO_PROFILE', None)
    installer = ['python3', str(ROOT / 'scripts/install.py'), '--staging-root', str(home)]
    engine = None
    results = []

    def ctl(operation, data=None, check=True):
        result = subprocess.run([str(home / '.local/bin/quatrroctl'), operation, '--stdin'],
                                input=json.dumps(data or {}), text=True, env=env,
                                capture_output=True, timeout=15, check=check)
        return json.loads(result.stdout) if check else result

    def stop():
        global engine
        if engine is not None:
            engine.terminate()
            try:
                engine.wait(timeout=10)
            except subprocess.TimeoutExpired:
                engine.kill()
                engine.wait()
                raise
            finally:
                engine = None

    try:
        for phase in ('clean', 'updated'):
            installed = json.loads(subprocess.check_output(installer, text=True))
            if phase == 'updated':
                assert Path(installed['backup']).is_dir()
            with socket.socket() as listener:
                listener.bind(('127.0.0.1', 0))
                port = listener.getsockname()[1]
            with (home / (phase + '.log')).open('w') as log:
                engine = subprocess.Popen([home / '.local/bin/quatrrod', f'--listen=127.0.0.1:{port}'],
                                          env=env, stdout=log, stderr=log)
                deadline = time.monotonic() + 15
                while ctl('status', check=False).returncode:
                    assert engine.poll() is None, 'engine exited'
                    assert time.monotonic() < deadline, 'engine startup timeout'
                    time.sleep(.05)
                status = ctl('status')
                assert status['language'] == ('en' if phase == 'clean' else 'es')
                if phase == 'clean':
                    ctl('secrets.put', dict(id='load-token', backend='file', value=SECRET))
                    config = ctl('config.get')
                    config['entries'] = [dict(id=f'load{i}', name=f'Load {i}', auth='bearer',
                                              secret='load-token', format='json', enabled=True)
                                         for i in range(4)]
                    ctl('config.save', config)
                    preview = ctl('config.preview')
                    ctl('config.activate', dict(hash=preview['hash'], grants=preview['capabilities']))
                else:
                    assert len(ctl('config.get')['entries']) == 4
                before = usage(engine.pid)
                idle_start = time.monotonic()
                time.sleep(5)
                idle_elapsed = time.monotonic() - idle_start
                idle = usage(engine.pid)

                def request(index):
                    body = json.dumps(dict(sequence=index, phase=phase, message='x' * 1024)).encode()
                    connection = http.client.HTTPConnection('127.0.0.1', port, timeout=10)
                    started = time.monotonic()
                    try:
                        connection.request('POST', f'/hooks/load{index % 4}', body=body,
                                           headers={'Authorization': 'Bearer ' + SECRET,
                                                    'Content-Type': 'application/json',
                                                    'X-Quatrro-Delivery': f'{phase}-{index}'})
                        response = connection.getresponse()
                        response.read()
                        return response.status, time.monotonic() - started
                    finally:
                        connection.close()

                load_start = time.monotonic()
                with ThreadPoolExecutor(max_workers=4) as pool:
                    deliveries = list(pool.map(request, range(400)))
                elapsed = time.monotonic() - load_start
                loaded = usage(engine.pid)
                assert all(code == 202 for code, _ in deliveries), deliveries
                # Each entry has received 100 requests. Reach its 120/min boundary.
                for index in range(400, 480, 4):
                    assert request(index)[0] == 202
                assert request(480)[0] == 429
                database = home / '.local/state/quatrro/quatrro.db'
                with sqlite3.connect(f'file:{database}?mode=ro', uri=True) as db:
                    count = db.execute('SELECT count(*) FROM events').fetchone()[0]
                    assert count == (420 if phase == 'clean' else 840), count
                    assert db.execute('SELECT count(*) FROM executions').fetchone()[0] == 0
                    assert db.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
                latencies = sorted(seconds * 1000 for _, seconds in deliveries)
                results.append(dict(phase=phase, requests=400, concurrency=4,
                                    elapsed_seconds=round(elapsed, 3),
                                    requests_per_second=round(400 / elapsed, 1),
                                    latency_p50_ms=round(latencies[199], 3),
                                    latency_p95_ms=round(latencies[379], 3),
                                    idle_seconds=round(idle_elapsed, 3),
                                    idle_cpu_seconds=round(idle['cpu_seconds'] - before['cpu_seconds'], 3),
                                    idle_rss_kib=idle['rss_kib'],
                                    load_cpu_seconds=round(loaded['cpu_seconds'] - idle['cpu_seconds'], 3),
                                    load_rss_kib=loaded['rss_kib'],
                                    peak_rss_kib=loaded['peak_rss_kib'],
                                    persisted_events=count, entry_rate_limit_verified=True))
                ctl('preferences.set', {'language': 'es'})
                stop()
        subprocess.run(installer + ['--uninstall'], check=True, capture_output=True, text=True)
        assert (home / '.local/state/quatrro/quatrro.db').exists()
        print(json.dumps(dict(installed_ingress=results, no_effects=True,
                              preference_preserved=True, uninstall_preserves_data=True), indent=2))
    finally:
        stop()
