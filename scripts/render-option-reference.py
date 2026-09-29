#!/usr/bin/env python3
"""Render equivalent EN/ES option references and audit scoped schema coverage."""
import argparse
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
CATALOG = ROOT / 'docs/reference/options.json'
SCOPED_KINDS = ('entries', 'destinations', 'flows', 'timers', 'monitors', 'actions', 'scripts', 'adapters')

PANEL_FIELDS = {
    'credentialType': 'credentials.type', 'secretID': 'credentials.id',
    'secretValue': 'credentials.value', 'secretBackend': 'credentials.backend',
    'googleClient': 'credentials.client_id', 'googleSecret': 'credentials.client_secret',
    'googleRefresh': 'credentials.refresh_token', 'googleScopes': 'credentials.scopes',
    'eventLimit': 'storage.max_events', 'byteLimit': 'storage.max_payload_bytes',
    'retention': 'storage.retention_days', 'dedup': 'storage.dedup_days',
    'sampleSource': 'simulation.source', 'sampleType': 'simulation.type',
    'sampleData': 'simulation.data', 'filePath': 'files.path',
}


def render():
    rows = json.loads(CATALOG.read_text())
    ids = [r['id'] for r in rows]
    assert len(ids) == len(set(ids)), 'duplicate option ID'
    for row in rows:
        assert len(row['examples']) >= 2 and len(set(row['examples'])) >= 2, row['id']
        for lang in ('en', 'es'):
            examples = row.get('examples_' + lang, row['examples'])
            assert len(examples) == len(row['examples']) and len(set(examples)) >= 2, (row['id'], lang)
        for key in ('label_en', 'label_es', 'description_en', 'description_es', 'default'):
            assert row[key], (row['id'], key)
    schema = (ROOT / 'qml/Schema.js').read_text()
    coverage = []
    for kind in SCOPED_KINDS:
        block = re.search(r'if \(kind === "' + kind + r'"\) return (.*?)(?=\n    if |\n    return common)', schema, re.S)
        assert block, kind
        fields = set(re.findall(r'key:"([^"]+)"', block.group(1))) | {'id'}
        if 'concat(named' in block.group(1):
            fields.add('name')
        if '],enabled)' in block.group(1):
            fields.add('enabled')
        for key in sorted(fields):
            option = 'common.' + key if key in ('id', 'name', 'enabled') else kind + '.' + key
            assert option in ids, f'undocumented scoped field: {kind}.{key}'
            coverage.append((kind + '.' + key, option))
    panel = (ROOT / 'Panel.qml').read_text()
    for control, option in PANEL_FIELDS.items():
        assert re.search(r'\bid:\s*' + re.escape(control) + r'\b', panel), control
        assert option in ids, option
    outputs = {}
    for lang, other, title in [('en', 'es', 'Option reference'), ('es', 'en', 'Referencia de opciones')]:
        if lang == 'en':
            intro = ('This installment covers shared resource fields, entries, destinations, flows, conditions, schedules, monitors, actions, scripts and adapters. '
                     'Includes credentials, retention, operating controls, simulation and preparation. Semantic review and verification scope are recorded in docs/documentation-audit.md. '
                     'Defaults describe a new UI form, not implicit defaults for every imported JSON field. '
                     'Examples are independent values, not complete configurations. Resource references must exist. '
                     'Technical tokens remain identical in both languages.')
            default, example = 'Initial value', 'Example'
        else:
            intro = ('Esta entrega cubre campos compartidos, entradas, destinos, flujos, condiciones, programaciones, monitores, acciones, scripts y adaptadores. '
                     'Incluye credenciales, retención, controles operativos, simulación y preparación. La revisión semántica y el alcance comprobado se registran en docs/documentation-audit.md. '
                     'Los valores iniciales describen un formulario nuevo, no valores implícitos para todo JSON importado. '
                     'Los ejemplos son valores independientes, no configuraciones completas. Las referencias deben existir. '
                     'Los tokens técnicos son idénticos en ambos idiomas.')
            default, example = 'Valor inicial', 'Ejemplo'
        lines = [f'# {title}', '', f'[EN / ES](../{other}/options.md) · [Guide / Guía](user-guide.md)', '', intro, '']
        groups = {}
        for row in rows:
            groups.setdefault(row['id'].split('.')[0], []).append(row)
        for group, options in groups.items():
            lines.append(f'- [{group}](#group-{group}) ({len(options)})')
        lines.append('')
        for group, options in groups.items():
            lines += [f'<a id="group-{group}"></a>', f'## {group}', '']
            for row in options:
                lines += [f'<a id="{row["id"]}"></a>', f'### {row["label_" + lang]} — `{row["id"]}`', '']
                if row['default'] != '—':
                    lines += [f'**{default}:** `{row["default"]}`.', '']
                lines += [row['description_' + lang], '']
                for index, value in enumerate(row.get('examples_' + lang, row['examples']), 1):
                    lines.append(f'- {example} {index}: `{value}`.')
                lines.append('')
        outputs[ROOT / f'docs/{lang}/options.md'] = '\n'.join(lines)
    lines = ['# Option coverage / Cobertura de opciones', '',
             'Generated from `reference/options.json` by `scripts/render-option-reference.py`.', '',
             'Scope / Alcance: all Schema.js resource kinds, nested conditions, script parameters, arguments and directories. '
             'Includes credential, retention and operating controls / Incluye credenciales, retención y controles operativos.', '',
             '| UI/config field / Campo | English | Español | Examples / Ejemplos |',
             '|---|---|---|---|']
    covered = {option for _, option in coverage}
    coverage += [(r['id'], r['id']) for r in rows if r['id'] not in covered]
    for field, option in coverage:
        lines.append(f'| `{field}` | [EN](en/options.md#{option}) | [ES](es/options.md#{option}) | 2 |')
    lines += ['', 'The check verifies field presence and bilingual structure; it does not execute each value or prove semantic correctness.',
              'La comprobación verifica campos y estructura bilingüe; no ejecuta cada valor ni acredita su corrección semántica.', '']
    outputs[ROOT / 'docs/option-coverage.md'] = '\n'.join(lines)
    return outputs


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    outputs = render()
    for path, content in outputs.items():
        if args.check:
            assert path.exists() and path.read_text() == content, f'stale generated reference: {path}'
        else:
            path.write_text(content)
    print(f'{len(outputs)} bilingual reference/coverage files checked' if args.check else f'{len(outputs)} reference/coverage files written')


if __name__ == '__main__':
    main()
