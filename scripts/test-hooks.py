#!/usr/bin/env python3
"""Exercise the installed Omarchy hook runner in an isolated HOME/profile."""
import json
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]
RUNNER = Path('/usr/share/omarchy/bin/omarchy-hook')


def main():
    with tempfile.TemporaryDirectory(prefix='quatrro-hooks-') as directory:
        home = Path(directory)
        profile = home / 'profile'
        env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(profile))
        bindir = home / '.local/bin'
        bindir.mkdir(parents=True)
        shutil.copy2(ROOT / 'build/quatrroctl', bindir / 'quatrroctl')
        hooks = home / '.config/omarchy/hooks'
        cases = {'battery-low': ['17'], 'font-set': ['A "font" $(echo value)'],
                 'post-boot': [], 'post-update': [], 'pre-refresh-pacman': [],
                 'theme-set': ['quatrro-fixture']}
        for name in cases:
            target = hooks / (name + '.d')
            target.mkdir(parents=True)
            shutil.copy2(ROOT / 'packaging/hooks' / ('quatrro-' + name), target)
        marker = hooks / 'theme-set.d/another-tool'
        original = '#!/bin/bash\nprintf "%s" "$1" > "$HOME/other-hook-ran"\n'
        marker.write_text(original)
        engine = subprocess.Popen([ROOT / 'build/quatrrod', '--listen='], env=env,
                                  stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        try:
            for _ in range(100):
                ready = subprocess.run([bindir / 'quatrroctl', 'status'], env=env,
                                       capture_output=True)
                if ready.returncode == 0:
                    break
                if engine.poll() is not None:
                    raise AssertionError(engine.stderr.read().decode())
                time.sleep(0.05)
            else:
                raise AssertionError('engine did not start')
            for name, args in cases.items():
                subprocess.run(['bash', RUNNER, name, *args], env=env,
                               check=True, timeout=3, capture_output=True)
            database = next(profile.rglob('*.db'))
            with sqlite3.connect(f'file:{database}?mode=ro', uri=True) as db:
                rows = db.execute('SELECT source,payload FROM events').fetchall()
            assert len(rows) == 6, rows
            events = {source: json.loads(payload) for source, payload in rows}
            assert events['hook:battery-low']['data']['percentage'] == 17
            assert events['hook:font-set']['data']['font'] == cases['font-set'][0]
            assert (home / 'other-hook-ran').read_text() == cases['theme-set'][0]
            assert marker.read_text() == original
        finally:
            engine.terminate()
            engine.communicate(timeout=5)
        start = time.monotonic()
        for name, args in cases.items():
            subprocess.run(['bash', RUNNER, name, *args], env=env,
                           check=True, timeout=3, capture_output=True)
        elapsed = time.monotonic() - start
        assert elapsed < 3, elapsed
        print(json.dumps({'hooks_persisted': 6, 'existing_hook_preserved': True,
                          'offline_total_seconds': round(elapsed, 3)}))


if __name__ == '__main__':
    main()
