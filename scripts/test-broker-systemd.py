#!/usr/bin/env python3
"""Boot a private systemd manager under a temporary delegated user scope.

No host system units or policies are installed. The only host mutation is a
transient user scope, stopped in finally. Requires subordinate UID/GID ranges.
"""
import json
import pty
import select
import os
from pathlib import Path
import subprocess
import tempfile
import uuid
import time
import argparse
import shutil

PROJECT = Path(__file__).resolve().parents[1]


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--ui', action='store_true', help='Also exercise real QML administrative UI')
    options=parser.parse_args()
    unit = 'quatrro-broker-test-' + uuid.uuid4().hex[:12] + '.scope'
    with tempfile.TemporaryDirectory(prefix='quatrro-systemd-') as tmp:
        root = Path(tmp)
        for name, rules in (('empty', []), ('allowed', [{'uid': 1000, 'unit': 'quatrro-fixture.service', 'operations': ['status', 'start', 'stop', 'restart']}])):
            policy = root / (name + '.json')
            policy.write_text(json.dumps({'version': 1, 'rules': rules}))
            subprocess.run(['python3', str(PROJECT / 'scripts/prepare-broker-install.py'), '--policy', str(policy), '--output', str(root / name)], check=True, capture_output=True)
        (root/'captures').mkdir(mode=0o777)
        (root/'captures').chmod(0o777)
        master, slave = pty.openpty()
        launcher = root / 'launch.sh'
        launcher.write_text("""#!/bin/sh
set -eu
CGROUP_PATH=/sys/fs/cgroup$(awk -F: '$1 == 0 {print $3}' /proc/self/cgroup)
export CGROUP_PATH
exec unshare --user --map-auto --map-root-user --mount --net --ipc --uts --cgroup /bin/sh -c '
    mkdir "$CGROUP_PATH/supervisor"
    echo 0 > "$CGROUP_PATH/supervisor/cgroup.procs"
    exec "$@"
' isolated "$@"
""")
        boot = root / 'boot.sh'
        boot.write_text('''#!/bin/sh
set -eu
mount -t cgroup2 none /sys/fs/cgroup
echo 0 > /sys/fs/cgroup/cgroup.procs
mount -o remount,ro /sys
mount --make-rshared /
mkdir -p /etc/systemd/system /var /run/systemd
cat /proc/sys/kernel/random/uuid | tr -d - > /etc/machine-id
printf 'quatrro-test\\n' > /run/systemd/container
printf '%s\\n' '[Unit]' 'Description=Quatrro isolated boot probe' 'DefaultDependencies=no' 'Requires=quatrro-probe.service' > /etc/systemd/system/quatrro-test.target
printf '%s\\n' '[Unit]' 'DefaultDependencies=no' 'Requires=dbus.service' 'After=dbus.service' '[Service]' 'Type=oneshot' 'ExecStart=/bin/sh /probe.sh' 'StandardOutput=tty' 'StandardError=tty' 'TTYPath=/dev/console' > /etc/systemd/system/quatrro-probe.service
mkdir -p /run/dbus /etc/polkit-1/rules.d
cp /broker-build /usr/local/libexec/quatrro-broker
chmod 755 /usr/local/libexec/quatrro-broker
cp /packaging/org.quatrro.automations.service.policy /usr/share/polkit-1/actions/

cp /packaging/quatrro-broker.service /packaging/quatrro-broker.socket /etc/systemd/system/
for target in basic sysinit sockets paths timers; do
    printf '%s\\n' '[Unit]' 'DefaultDependencies=no' > /etc/systemd/system/$target.target
done
QUATRRO_BROKER_FIXTURE=prepare /fixture-test -test.run=^TestSystemdBrokerFixture$
cat > /etc/systemd/system/polkit.service <<'UNIT'
[Unit]
DefaultDependencies=no
Requires=dbus.service
After=dbus.service
[Service]
Type=dbus
BusName=org.freedesktop.PolicyKit1
ExecStart=/usr/lib/polkit-1/polkitd --no-debug
UNIT
cat > /etc/dbus.conf <<'BUS'
<busconfig><type>system</type><listen>systemd:</listen><auth>EXTERNAL</auth><policy context="default"><allow user="*"/><allow own="*"/><allow send_destination="*"/><allow receive_sender="*"/></policy></busconfig>
BUS
cat > /etc/systemd/system/dbus.socket <<'UNIT'
[Unit]
DefaultDependencies=no
[Socket]
ListenStream=/run/dbus/system_bus_socket
SocketMode=0666
UNIT
cat > /etc/systemd/system/dbus.service <<'UNIT'
[Unit]
DefaultDependencies=no
Requires=dbus.socket
After=dbus.socket
[Service]
Type=simple
ExecStart=/usr/bin/dbus-daemon --nofork --config-file=/etc/dbus.conf
ExecStartPost=/bin/sh -c "while ! test -S /run/dbus/system_bus_socket; do sleep 0.05; done"
TimeoutStartSec=5
UNIT
cat > /etc/systemd/system/quatrro-fixture.service <<'UNIT'
[Unit]
DefaultDependencies=no
[Service]
Type=exec
ExecStart=/usr/bin/sleep infinity
UNIT
cat > /probe.sh <<'PROBE'
#!/bin/sh
set -eu
trap 'systemctl --no-block exit 1' EXIT
attempt=0
until busctl --system introspect org.freedesktop.systemd1 /org/freedesktop/systemd1 >/dev/null; do
    attempt=$((attempt+1))
    test "$attempt" -lt 40
    sleep 0.1
done
systemctl start quatrro-fixture.service
systemctl is-active --quiet quatrro-fixture.service
test "$(systemctl show quatrro-fixture.service --property=SubState --value)" = running
systemctl stop quatrro-fixture.service
test "$(systemctl show quatrro-fixture.service --property=ActiveState --value)" = inactive
systemctl start polkit.service quatrro-broker.socket
call() {
    QUATRRO_BROKER_FIXTURE=call QUATRRO_TEST_OPERATION="$1" QUATRRO_TEST_STATE="${2:-}" setpriv --reuid=1000 --regid=1000 --clear-groups /fixture-test -test.run=^TestSystemdBrokerFixture$
}
QUATRRO_TEST_CHECK=1 call start
test "$(systemctl show quatrro-fixture.service --property=ActiveState --value)" = inactive
QUATRRO_TEST_DENIED=1 QUATRRO_TEST_CHECK=1 QUATRRO_TEST_UNIT=other.service call start
call status inactive
broker_pid=$(systemctl show quatrro-broker.service --property=MainPID --value)
test "$(awk '/^CapEff:/ {print $2}' /proc/$broker_pid/status)" = 0000000000000000
test "$(awk '/^NoNewPrivs:/ {print $2}' /proc/$broker_pid/status)" = 1
broker_group=$(systemctl show quatrro-broker.service --property=ControlGroup --value)
test "$(cat /sys/fs/cgroup$broker_group/memory.max)" = 134217728
test "$(cat /sys/fs/cgroup$broker_group/cpu.max)" = '25000 100000'
call start
call status active
fixture_pid=$(systemctl show quatrro-fixture.service --property=MainPID --value)
QUATRRO_TEST_CHECK=1 call restart
QUATRRO_TEST_CHECK=1 call stop
test "$(systemctl show quatrro-fixture.service --property=MainPID --value)" = "$fixture_pid"
systemctl is-active --quiet quatrro-fixture.service
call restart
call stop
call status inactive
QUATRRO_TEST_DENIED=1 QUATRRO_TEST_UNIT=other.service call start
QUATRRO_BROKER_FIXTURE=call QUATRRO_TEST_DENIED=1 QUATRRO_TEST_OPERATION=start setpriv --reuid=102 --regid=102 --clear-groups /fixture-test -test.run=^TestSystemdBrokerFixture$
cp /etc/polkit-1/rules.d/00-quatrro.rules /etc/polkit-1/original
printf '%s\\n' 'polkit.addRule(function(action, subject) { if (action.id.indexOf("org.quatrro.automations.service.") === 0) return polkit.Result.NO; });' > /etc/polkit-1/rules.d/replacement
mv /etc/polkit-1/rules.d/replacement /etc/polkit-1/rules.d/00-quatrro.rules
systemctl restart polkit.service
QUATRRO_TEST_DENIED=1 call status
QUATRRO_TEST_DENIED=1 QUATRRO_TEST_CHECK=1 call start
mv /etc/polkit-1/original /etc/polkit-1/rules.d/00-quatrro.rules
systemctl restart polkit.service
call status inactive
printf '%s\\n' '{"version":1,"rules":[]}' > /etc/quatrro/replacement
chmod 600 /etc/quatrro/replacement
mv /etc/quatrro/replacement /etc/quatrro/broker.json
QUATRRO_TEST_DENIED=1 call start
test "$(systemctl show quatrro-fixture.service --property=ActiveState --value)" = inactive
systemctl stop quatrro-broker.service quatrro-broker.socket
systemctl stop polkit.service
rm /etc/systemd/system/quatrro-broker.service /etc/systemd/system/quatrro-broker.socket /usr/local/libexec/quatrro-broker /etc/quatrro/broker.json /etc/polkit-1/rules.d/00-quatrro.rules /usr/share/polkit-1/actions/org.quatrro.automations.service.policy
systemctl daemon-reload
python3 /installer.py apply --bundle /allowed
! systemctl is-enabled --quiet quatrro-broker.socket
! systemctl is-active --quiet quatrro-broker.socket
systemctl start polkit.service
systemctl enable --now quatrro-broker.socket
call status inactive
python3 /crash-test.py
python3 /installer.py apply --bundle /empty
! systemctl is-enabled --quiet quatrro-broker.socket
! systemctl is-active --quiet quatrro-broker.socket
systemctl start quatrro-broker.socket
call status inactive
python3 /installer.py apply --bundle /empty --replace-policy
systemctl start quatrro-broker.socket
QUATRRO_TEST_DENIED=1 call status
printf 'preserve' > /etc/quatrro/unrelated
python3 /installer.py remove
test ! -e /usr/local/libexec/quatrro-broker
test ! -e /etc/systemd/system/quatrro-broker.socket
test ! -e /etc/polkit-1/rules.d/00-quatrro-automations.rules
test ! -e /var/lib/quatrro-broker-install/receipt.json
test ! -e /var/lib/quatrro-broker-install/pending.json
test "$(cat /etc/quatrro/unrelated)" = preserve
python3 /crash-test.py --fresh
echo QUATRRO_SYSTEMD_READY
trap - EXIT
systemctl --no-block exit 0
PROBE
exec /usr/lib/systemd/systemd --system --unit=quatrro-test.target --log-target=console --log-level=info
''')
        if options.ui:
            source=boot.read_text()
            source=source.replace('call start\ncall status active', 'setpriv --reuid=1000 --regid=1000 --clear-groups python3 /project/scripts/test-ui-admin.py\nsystemctl stop quatrro-fixture.service\ncall start\ncall status active', 1)
            boot.write_text(source)
        args = ['systemd-run' , '--user', '--scope', '--quiet', '--collect', '--unit='+unit, '-p', 'Delegate=yes', '-p', 'MemoryAccounting=yes', '-p', 'CPUAccounting=yes',
                '/bin/sh', str(launcher),
                'bwrap', '--unshare-pid', '--as-pid-1', '--cap-add', 'ALL', '--tmpfs', '/', '--dir', '/usr', '--ro-bind', '/usr/bin', '/usr/bin', '--ro-bind', '/usr/lib', '/usr/lib', '--dir', '/usr/share',
                '--symlink', 'usr/bin', '/bin', '--symlink', 'usr/lib', '/lib', '--symlink', 'usr/lib', '/lib64',
                '--dir', '/etc', '--ro-bind', '/etc/passwd', '/etc/passwd', '--ro-bind', '/etc/group', '/etc/group',
                '--ro-bind', '/etc/nsswitch.conf', '/etc/nsswitch.conf', '--symlink', '../usr/lib/os-release', '/etc/os-release',
                '--proc', '/proc', '--dev', '/dev', '--dev-bind', os.ttyname(slave), '/dev/console', '--tmpfs', '/run', '--tmpfs', '/tmp', '--tmpfs', '/sys', '--dir', '/sys/fs/cgroup',
                '--tmpfs', '/usr/lib/systemd/system/sysinit.target.wants', '--tmpfs', '/usr/lib/systemd/system/sockets.target.wants',
                '--tmpfs', '/usr/local', '--dir', '/usr/local/libexec',
                '--tmpfs', '/usr/share/polkit-1/actions', '--tmpfs', '/usr/share/polkit-1/rules.d',
                '--ro-bind', str(PROJECT / 'packaging/broker'), '/packaging',
                '--ro-bind', str(PROJECT / 'scripts/install-broker.py'), '/installer.py',
                '--ro-bind', str(PROJECT / 'scripts/test-broker-install-crash.py'), '/crash-test.py',
                '--ro-bind', str(root / 'empty'), '/empty', '--ro-bind', str(root / 'allowed'), '/allowed',
                '--ro-bind', str(PROJECT / 'build/quatrro-broker'), '/broker-build',
                '--ro-bind', str(PROJECT / 'build/broker-integration.test'), '/fixture-test',
                '--ro-bind', str(boot), '/boot.sh', '--chdir', '/', '--setenv', 'container', 'quatrro-test',
                '/bin/sh', '/boot.sh']
        if options.ui:
            args[-2:-2]=['--ro-bind', str(PROJECT), '/project', '--bind', str(root/'captures'), '/captures',
                         '--ro-bind', '/usr/share/fonts', '/usr/share/fonts', '--ro-bind', '/usr/share/fontconfig', '/usr/share/fontconfig',
                         '--ro-bind', '/etc/fonts', '/etc/fonts']
        try:
            process = subprocess.Popen(args, stdout=slave, stderr=slave)
            console = bytearray()
            deadline = time.monotonic() + (110 if options.ui else 55)
            while True:
                ready = select.select([master], [], [], 0.1)[0]
                if ready:
                    console.extend(os.read(master, 65536))
                    if len(console) > 2*1024*1024:
                        raise RuntimeError('Private console output exceeded quota')
                elif process.poll() is not None:
                    break
                if time.monotonic() > deadline:
                    raise TimeoutError('Private systemd test timed out')
            result = process.wait(timeout=2)
            output = console.decode(errors='replace')
            print(output)
            if result != 0 or 'QUATRRO_SYSTEMD_READY' not in output:
                raise RuntimeError(f'private systemd manager failed its boot probe: exit {result}')
            if options.ui:
                if '"ui_real_authorization": true' not in output or '"event_started_fixture": true' not in output:
                    raise RuntimeError('Missing successful QML integration report')
                for language in ('en', 'es'):
                    image = root/'captures'/('broker-admin-'+language+'.png')
                    if not image.is_file() or not image.read_bytes().startswith(b'\x89PNG\r\n\x1a\n'):
                        raise RuntimeError('Missing administrative UI screenshot')
                    shutil.copyfile(image, PROJECT/'.dev'/image.name)
            print(json.dumps({'ui_admin_integration': options.ui, 'isolated_systemd_boot': True, 'private_dbus': True, 'fixture_start_status_stop': True, 'broker_polkit_roundtrip': True, 'permission_checks_without_effects': True, 'denial_and_revocation': True, 'polkit_revocation': True, 'effective_resource_limits': True, 'install_update_remove': True, 'installer_sigkill_recovery': True, 'host_system_units_modified': False}))
        finally:
            os.close(slave)
            os.close(master)
            subprocess.run(['systemctl', '--user', 'stop', unit], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
            if 'process' in locals():
                process.wait(timeout=5)


if __name__ == '__main__':
    main()
