# Paquetes de ejecución y reproducibilidad

[English](../en/packages.md) · [Instalación](installation.md)

El empaquetador genera artefactos de desarrollo Linux amd64. Publicar una versión
preliminar no completa las pruebas aplazadas ni la convierte en estable. Solo se
anuncian arquitecturas verificadas. La instalación sin compilador está en la guía
de instalación; este capítulo explica cómo construir paquetes.

## Construir desde fuentes

```bash
make build
python3 scripts/package-release.py
python3 scripts/test-release-package.py
python3 scripts/test-reproducible-build.py
```

`make build` usa `-trimpath -buildvcs=false`: los binarios no incorporan rutas
locales ni el estado cambiante del repositorio. Usa la versión Go de `go.mod` y
las dependencias fijadas en `go.sum`; reproducir exige iguales herramienta,
fuentes, plataforma y opciones. Determinismo del archivo no demuestra por sí solo
reproducibilidad del compilador.

El resultado es `dist/omarchy-automations-<version>-linux-amd64.tar.gz` y su
`.sha256`. La versión procede de los binarios y debe coincidir con el contrato QML
y versión base del manifiesto. `release.json` registra versión, contratos,
plataforma y SHA256 por archivo. El broker opcional conserva protocolo separado.

El tar.gz normaliza orden, permisos, propietarios y fechas, incluida la cabecera
gzip. Rutas de entrada explícitas incluyen binarios, UI, el instalador del motor, plantillas,
ejemplos, esquemas y documentación. Excluye perfiles, metadatos Git, cachés y
credenciales; rechaza enlaces simbólicos. `scripts/setup.py` permanece en la copia
Git, donde fija el hash del archivo, y no se incluye dentro de ese archivo.

## Instalar un paquete extraído

Desde el directorio descargado, sustituye `<version>` por la versión real:

```bash
sha256sum -c omarchy-automations-<version>-linux-amd64.tar.gz.sha256
tar -xzf omarchy-automations-<version>-linux-amd64.tar.gz
cd omarchy-automations-<version>
python3 scripts/install.py --activate
```

Los hashes detectan corrupción; la autenticidad depende de obtenerlos de un
publicador confiable. Es un paquete de ejecución, no un checkout para `make build`.
El broker tiene instalación separada y no se activa al extraer o instalar el
paquete de usuario. Con panel Omarchy gestionado por Git usa `setup.py` desde su
checkout, sin instalar una segunda UI gestionada encima.

## Qué se comprueba

`test-release-package.py` genera dos archivos idénticos, verifica hashes y ejecuta
instalación/actualización/retirada desde el contenido extraído en HOME temporal,
sin activar servicios del host. `test-setup.py` comprueba setup precompilado en
modo gestionado y panel Git, conserva archivos y rechaza paquetes corruptos o inseguros.

También se comprobó compilación independiente con Go 1.26.8 en Linux amd64: dos
árboles y cachés inicialmente vacías producen binarios idénticos. Comparten
herramienta y caché de módulos; `GOPROXY=off` evita descargas. No acredita otras
herramientas o plataformas.

El proyecto usa [MIT](../../LICENSE); conserva también los
[avisos de terceros](../third-party-notices.md). La aceptación de versión estable
continúa en el [roadmap](../../ROADMAP.md).
