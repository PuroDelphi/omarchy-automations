#!/usr/bin/env python3
"""Validate optional broker artifacts without installing or starting services."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import xml.etree.ElementTree as ET

root = Path(__file__).resolve().parents[1]
package = root / 'packaging' / 'broker'
policy = package / 'org.quatrro.automations.service.policy'
with tempfile.TemporaryDirectory(prefix='quatrro-polkit-xml-') as temp:
    # Resolve the declared DTD locally; never fetch it from the network.
    catalog = Path(temp) / 'catalog.xml'
    catalog.write_text('<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">'
                       '<system systemId="http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd" '
                       'uri="file:///usr/share/polkit-1/policyconfig-1.dtd"/></catalog>')
    subprocess.run(['xmllint', '--nonet', '--noout', '--dtdvalid',
                    '/usr/share/polkit-1/policyconfig-1.dtd', str(policy)],
                   env={**os.environ, 'XML_CATALOG_FILES': str(catalog)}, check=True)
actions = ET.parse(policy).getroot().findall('action')
assert {a.attrib['id'] for a in actions} == {
    f'org.quatrro.automations.service.{op}' for op in ('status', 'start', 'stop', 'restart')}
for action in actions:
    for scope in ('allow_any', 'allow_inactive', 'allow_active'):
        assert action.findtext(f'defaults/{scope}') == 'no'
    assert not action.findall('annotate')
assert json.loads((package / 'broker.json').read_text()) == {'version': 1, 'rules': []}
binary = root / 'build' / 'quatrro-broker'
assert binary.is_file(), 'Build quatrro-broker first'
with tempfile.TemporaryDirectory(prefix='quatrro-broker-units-') as temp:
    dest = Path(temp)
    for name in ('quatrro-broker.service', 'quatrro-broker.socket'):
        source = (package / name).read_text()
        if name.endswith('.service'):
            # Syntax/dependency verification requires an existing executable.
            # Only the uninstalled path is replaced; no unit is started.
            source = source.replace('ExecStart=/usr/local/libexec/quatrro-broker',
                                    f'ExecStart={binary}')
        (dest / name).write_text(source)
    subprocess.run(['systemd-analyze', 'verify', '--man=no',
                    str(dest / 'quatrro-broker.service'),
                    str(dest / 'quatrro-broker.socket')], check=True)
print(json.dumps({'polkit_dtd': True, 'defaults_deny': True,
                  'systemd_verify': True, 'installed': False, 'started': False}))
