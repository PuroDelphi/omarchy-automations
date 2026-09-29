#!/usr/bin/env python3
"""SIGKILL tests, callable only inside the private systemd test container.

Profiling pauses the unmodified installer immediately after a durable file
operation. The parent then kills it and runs the public recovery command.
"""
import importlib.util
import json
import os
from pathlib import Path
import runpy
import signal
import subprocess
import sys
import time


def isolated():
    mapping = Path('/proc/self/uid_map').read_text().split()
    if os.getuid() != 0 or len(mapping) < 3 or mapping[0] != '0' or mapping[1] == '0':
        raise RuntimeError('Requires subordinate namespace root')
    if Path('/run/systemd/container').read_text().strip() != 'quatrro-test':
        raise RuntimeError('Requires private test container')


def child():
    method, path, *arguments = sys.argv[2:]
    def profile(frame, event, arg):
        if event == 'return' and frame.f_code.co_filename == '/installer.py' and frame.f_code.co_name == method and frame.f_locals.get('path') == path:
            sys.setprofile(None)
            os.kill(os.getpid(), signal.SIGSTOP)
    sys.argv = ['/installer.py', *arguments]
    sys.setprofile(profile)
    runpy.run_path('/installer.py', run_name='__main__')
    raise RuntimeError('Installer never reached requested crash boundary')


def main():
    isolated()
    if len(sys.argv) > 1 and sys.argv[1] == '--child':
        child()
        return
    spec = importlib.util.spec_from_file_location('installer', '/installer.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    paths = [Path(path) for path, _ in module.FILES.values()] + [Path(module.RECEIPT)]
    def snapshot():
        return {str(path): (path.read_bytes(), path.stat().st_mode & 0o777, path.stat().st_uid) if path.exists() else None for path in paths}
    def command(*arguments, check=True):
        return subprocess.run(arguments, check=check, capture_output=True, text=True, timeout=20)
    baseline = snapshot()
    fresh = sys.argv[1:] == ['--fresh']
    cases = [
        ('write', module.JOURNAL, ['apply', '--bundle', '/empty', '--replace-policy']),
        ('write', module.FILES['quatrro-broker'][0], ['apply', '--bundle', '/empty', '--replace-policy']),
        ('write', module.FILES['00-quatrro-automations.rules'][0], ['apply', '--bundle', '/empty', '--replace-policy']),
        ('write', module.RECEIPT, ['apply', '--bundle', '/empty', '--replace-policy']),
        ('remove', module.FILES['quatrro-broker.socket'][0], ['remove']),
        ('remove', module.RECEIPT, ['remove']),
    ]
    if fresh:
        assert all(value is None for value in baseline.values())
        cases = [('write', path, ['apply', '--bundle', '/allowed']) for path in (module.FILES['quatrro-broker'][0], module.RECEIPT)]
    for method, path, arguments in cases:
        if not fresh:
            command('systemctl', 'enable', '--now', 'quatrro-broker.socket')
        process = subprocess.Popen([sys.executable, __file__, '--child', method, path, *arguments], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        try:
            deadline = time.monotonic() + 15
            while True:
                event = os.waitid(os.P_PID, process.pid, os.WSTOPPED | os.WEXITED | os.WNOHANG | os.WNOWAIT)
                if event is not None:
                    assert event.si_code == os.CLD_STOPPED and event.si_status == signal.SIGSTOP, ('missed crash boundary', method, path, event)
                    break
                assert time.monotonic() < deadline, ('crash boundary timeout', method, path)
                time.sleep(0.02)
            journal_before = Path(module.JOURNAL).read_bytes()
            concurrent = command(sys.executable, '/installer.py', 'apply', '--bundle', '/empty', check=False)
            assert concurrent.returncode != 0, 'concurrent installer accepted'
            assert Path(module.JOURNAL).read_bytes() == journal_before
            process.kill()
            assert process.wait(timeout=5) == -signal.SIGKILL
        finally:
            if process.poll() is None:
                process.kill()
                process.wait(timeout=5)
        assert Path(module.JOURNAL).is_file(), ('missing journal', path)
        blocked = command(sys.executable, '/installer.py', 'apply', '--bundle', '/empty', check=False)
        assert blocked.returncode != 0 and 'requires recover first' in blocked.stderr
        recovery = command(sys.executable, '/installer.py', 'recover', check=False)
        assert recovery.returncode == 0, (method, path, recovery.stderr)
        assert snapshot() == baseline, ('inexact restoration', method, path)
        assert not Path(module.JOURNAL).exists()
        assert command('systemctl', 'is-enabled', '--quiet', 'quatrro-broker.socket', check=False).returncode != 0
        assert command('systemctl', 'is-active', '--quiet', 'quatrro-broker.socket', check=False).returncode != 0
        print('CRASH_RECOVERY_OK', method, path, flush=True)
    if fresh:
        print(json.dumps({'fresh_install_sigkill_recovery': len(cases), 'no_managed_files_left': True}))
        return
    # Restored production files must still serve a real unprivileged request.
    command('systemctl', 'restart', 'polkit.service')
    command('systemctl', 'start', 'quatrro-broker.socket')
    environment = dict(os.environ, QUATRRO_BROKER_FIXTURE='call', QUATRRO_TEST_OPERATION='status', QUATRRO_TEST_STATE='inactive')
    subprocess.run(['setpriv', '--reuid=1000', '--regid=1000', '--clear-groups', '/fixture-test', '-test.run=^TestSystemdBrokerFixture$'], env=environment, check=True, timeout=20)
    print(json.dumps({'installer_sigkill_recovery': len(cases), 'restored_broker_call': True}))


if __name__ == '__main__':
    main()
