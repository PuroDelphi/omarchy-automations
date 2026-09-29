# Paquetes de Omarchy Automations

Referencia de empaquetado para mantenedores. Para instalar el plugin sin compilar, usa [English](en/installation.md) · [Español](es/installation.md). Referencia actual de paquetes: [English](en/packages.md) · [Español](es/packages.md).

El empaquetador actual prepara artefactos locales de desarrollo para Linux amd64.
No implica que las fases pendientes del roadmap estén completas ni publica en
GitHub. Solo se anuncia una arquitectura comprobada.

## Construcción desde el código fuente

```bash
make build
python3 scripts/package-release.py
python3 scripts/test-release-package.py
python3 scripts/test-reproducible-build.py
```

`make build` utiliza `-trimpath -buildvcs=false`: el binario no incorpora rutas
locales ni el estado cambiante del repositorio. Usa la versión de Go indicada en
`go.mod` y las dependencias fijadas en `go.sum`; reproducir una compilación exige
la misma herramienta, fuentes, plataforma y opciones. La comprobación del
archivo comprimido por sí sola no demuestra reproducibilidad del compilador.

El resultado en `dist/` es `omarchy-automations-<versión>-linux-amd64.tar.gz` y
su archivo `.sha256`. La versión se lee de los binarios, que deben coincidir con
el contrato QML y la versión base del manifiesto. `release.json` registra versión,
contratos, plataforma y SHA-256 de cada archivo incluido. El broker conserva su
protocolo independiente y sigue siendo opcional.

El tar.gz normaliza orden, permisos, propietarios y fechas, también en gzip.
Incluye binarios, UI, instaladores, plantillas, ejemplos, esquemas y documentación
mediante una lista de rutas de entrada. No copia perfiles, repositorio Git,
cachés ni credenciales. Los enlaces simbólicos en entradas se rechazan.

## Instalación del paquete extraído

Desde el directorio que contiene el archivo descargado:

```bash
sha256sum -c omarchy-automations-<versión>-linux-amd64.tar.gz.sha256
tar -xzf omarchy-automations-<versión>-linux-amd64.tar.gz
cd omarchy-automations-<versión>
python3 scripts/install.py
```

Sustituye `<versión>` por el nombre real del archivo. El archivo de checksums
permite detectar corrupción; su autenticidad depende de obtenerlo de una fuente
confiable. El paquete es de ejecución, no un checkout para ejecutar `make build`.
Sigue [instalación](installation.md) para activación, actualización y recuperación.
El broker requiere su procedimiento opcional separado; no se habilita al extraer
ni instalar el paquete de usuario.

La prueba `scripts/test-release-package.py` genera dos archivos iguales byte a
byte, valida los hashes y ejecuta instalación, segunda instalación y eliminación
sobre el contenido extraído en un HOME de staging. No activa el servicio del host.
La compilación independiente también se comprobó con Go 1.26.8 en Linux amd64:
dos árboles fuente y cachés de compilación inicialmente vacías producen los tres
binarios idénticos. Se comparten herramienta y caché de módulos; GOPROXY=off
evita descargar dependencias. No acredita otras versiones, plataformas o cadenas
de herramientas. El proyecto usa [MIT](../LICENSE); conserva también los
[avisos de terceros](third-party-notices.md). Queda pendiente el cierre R1 antes
de anunciar una versión estable.
