# Preparing a preview release

Finish edits to files that enter the archive **before** packaging. The package
contains README, documentation, UI, binaries and runtime support files. It does
not contain `scripts/setup.py`; that lets the checkout pin the package SHA256
without changing the package while the pin is written.

```bash
make build
python3 scripts/package-release.py
python3 scripts/sync-release-pin.py --write --tag v0.1.0-preview.6
python3 scripts/package-release.py
python3 scripts/sync-release-pin.py --check --tag v0.1.0-preview.6
python3 scripts/test-release-package.py
python3 scripts/test-setup.py
```

The helper calculates SHA256 from the archive, checks the adjacent `.sha256`
file, refuses an archive containing `setup.py`, and updates the tag and pinned
digest atomically. After committing and passing CI, create the tag **at that
exact commit** and upload the archive plus its `.sha256` file to the matching
GitHub prerelease. Verify the remote tag target and published asset digest, then
run `scripts/setup.py --staging-root` against the HTTPS release before promoting
the commit to a marketplace review. Do not reuse an older mutable release tag.

## Español

Termina los cambios en archivos incluidos en el paquete **antes** de empaquetar.
El paquete contiene README, documentación, interfaz, binarios y archivos de
soporte, pero no `scripts/setup.py`; así se puede fijar su SHA256 sin cambiar el
propio paquete. Ejecuta los comandos anteriores con la nueva etiqueta. La
herramienta calcula el SHA256, comprueba el archivo `.sha256`, rechaza un paquete
que incluya `setup.py` y actualiza la etiqueta y el hash de forma atómica.
Después de pasar CI, crea la etiqueta en el commit exacto, sube ambos archivos
y verifica la descarga en un directorio temporal antes de solicitar revisión.
