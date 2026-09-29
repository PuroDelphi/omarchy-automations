#!/usr/bin/env python3
"""Check paired guides, reciprocal language links, local links and option catalog."""
import json
from pathlib import Path
import re
import subprocess
import urllib.parse

ROOT = Path(__file__).resolve().parents[1]
chapters = {lang: {p.name for p in (ROOT / 'docs' / lang).glob('*.md')} for lang in ('en', 'es')}
assert chapters['en'] == chapters['es'], 'EN/ES chapter mismatch'
links = 0
for lang, other in (('en', 'es'), ('es', 'en')):
    for name in sorted(chapters[lang]):
        page = ROOT / 'docs' / lang / name
        source = page.read_text()
        assert f'../{other}/{name}' in source, (name, 'missing language link')
        for target in re.findall(r'!?\[[^\]]*\]\(([^\s)]+)\)', source):
            url = urllib.parse.urlsplit(target)
            if url.scheme:
                continue
            destination = (page.parent / urllib.parse.unquote(url.path)).resolve() if url.path else page
            assert destination.is_relative_to(ROOT), (name, target, 'outside project')
            assert destination.is_file(), (name, target, 'missing file')
            if url.fragment and destination.suffix == '.md':
                content = destination.read_text()
                anchors = set(re.findall(r'<a id="([^"]+)"', content))
                anchors.update(re.sub(r'[^\w\- ]', '', title.lower()).replace(' ', '-')
                               for title in re.findall(r'^#+\s+(.+)$', content, re.M))
                assert urllib.parse.unquote(url.fragment) in anchors, (name, target, 'missing anchor')
            links += 1
subprocess.run(['python3', str(ROOT / 'scripts/render-option-reference.py'), '--check'], check=True)
print(json.dumps({'paired_documents': len(chapters['en']), 'local_links': links,
                  'scope': 'structural checks; semantic review recorded in docs/documentation-audit.md'}))
