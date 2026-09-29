#!/usr/bin/env python3
"""Compare binaries built from separate source paths and empty build caches."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
GO = os.environ.get('GO', 'go')
BINARIES = ('quatrrod', 'quatrroctl', 'quatrro-broker')

with tempfile.TemporaryDirectory(prefix='omarchy-reproducible-') as temporary:
    base = Path(temporary)
    results = []
    for name in ('first', 'second'):
        checkout = base / name
        checkout.mkdir()
        for directory in ('cmd', 'internal'):
            shutil.copytree(ROOT / directory, checkout / directory)
        for filename in ('go.mod', 'go.sum'):
            shutil.copy2(ROOT / filename, checkout / filename)
        output = checkout / 'build'
        output.mkdir()
        env = dict(os.environ, GOCACHE=str(base / (name + '-cache')),
                   GOPROXY='off', GOSUMDB='off', GOTOOLCHAIN='local')
        subprocess.run([GO, 'build', '-trimpath', '-buildvcs=false', '-o', str(output) + '/',
                        './cmd/...'], cwd=checkout, env=env, check=True, timeout=300)
        results.append({binary: hashlib.sha256((output / binary).read_bytes()).hexdigest()
                        for binary in BINARIES})
    assert results[0] == results[1], results
    version = subprocess.check_output([GO, 'version'], text=True).strip()
    print(json.dumps(dict(toolchain=version, separate_source_paths=True,
                          separate_empty_build_caches=True, dependency_downloads_disabled=True,
                          binary_sha256=results[0]), sort_keys=True))
