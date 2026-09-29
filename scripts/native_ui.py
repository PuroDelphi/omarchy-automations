"""Expose installed Omarchy QML modules to isolated development harnesses."""
from pathlib import Path
import os


def attach_native_ui(env, directory):
    imports = Path(directory) / 'native-imports'
    namespace = imports / 'qs'
    namespace.mkdir(parents=True, exist_ok=True)
    for name in ('Ui', 'Commons'):
        source = Path('/usr/share/omarchy/shell') / name
        if not source.is_dir():
            raise RuntimeError('Installed Omarchy QML module unavailable: ' + name)
        destination = namespace / name
        if not destination.exists():
            destination.symlink_to(source, target_is_directory=True)
    previous = env.get('QML_IMPORT_PATH', '')
    env['QML_IMPORT_PATH'] = str(imports) + (os.pathsep + previous if previous else '')
