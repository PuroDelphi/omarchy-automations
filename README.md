# Omarchy Automations

Plugin de automatizaciones para Omarchy: webhooks de entrada y salida conectados con notificaciones, servicios, comandos y monitoreo.

Estado: **en desarrollo, no listo para producción**. Motor, CLI y panel nativo funcionales e instalados; la revisión visual nativa y la documentación bilingüe están completadas para la versión actual; siguen abiertas las validaciones externas y los controles finales de entrega del roadmap.

- [Documentation index — English](docs/en/README.md)
- [Índice de documentación — Español](docs/es/README.md)
- [User guide — English](docs/en/user-guide.md)
- [Guía de usuario — Español](docs/es/user-guide.md)
- [Roadmap y seguimiento](ROADMAP.md)
- [Nombre e identificadores compatibles](docs/product-name.md)
- [Paquetes de desarrollo](docs/releases.md)
- [Preparación de GitHub](docs/github-publication.md)
- [Instalación en desarrollo](docs/installation.md)
- [English / Español](docs/languages.md)
- [Diseño de arquitectura](docs/DESIGN.md)
- [Protocolo y operaciones](docs/PROTOCOL.md)
- [Hooks de Omarchy](docs/hooks.md)
- [Formatos y límites](docs/formats.md)
- [Proveedores y contratos](docs/providers.md)
- [Intervalos y calendarios](docs/scheduling.md)
- [Monitores del sistema](docs/monitoring.md)
- [Aislamiento de comandos](docs/command-isolation.md)
- [Operación, cola y diagnóstico](docs/operations.md)
- [Recuperación y resultados inciertos](docs/recovery.md)
- [Exposición TLS opcional](docs/exposure.md)
- [Evidencia de validación](docs/validation.md)
- [Compatibilidad comprobada](docs/compatibility.md)

## Desarrollo

Requiere Linux, Go 1.26+, Omarchy/Quickshell para UI y systemd de usuario para acciones del sistema. El motor usa SQLite integrado mediante Go.

```bash
make build
make test
make check
```

Para probar sin alterar tu configuración, en una terminal:

```bash
export QUATRRO_PROFILE="$PWD/.dev/profile"
export PATH="$PWD/build:$PATH"
quatrrod
```

En otra terminal, con las mismas variables:

```bash
quatrroctl status
quatrroctl config.save --stdin < examples/notification.json
quatrroctl simulate '{"source":"local:demo","type":"demo","data":{"message":"Hola"}}'
quatrroctl config.preview
python3 scripts/run-ui.py
```

La simulación no activa el flujo ni ejecuta acciones. La activación requiere el hash y las capacidades revisadas que devuelve `config.preview`; consultar el protocolo. El panel permite crear conexiones, acciones, flujos y monitores, revisar capacidades y activar una revisión. Incluye simulación, historial, credenciales y límites de almacenamiento.

`python scripts/smoke-native.py --offscreen` comprueba la conexión y produce un render nuevo en `.dev/panel-render.png`, incluso con la sesión bloqueada. `python scripts/test-ui-flow.py` ejercita las señales de los formularios QML, activa una revisión y verifica notificación, entrega HTTPS local e historial. `QUATRRO_HOST_TEST=1 go test -race ./...` añade una notificación, un servicio temporal propio y pruebas de comandos aislados. Estas pruebas requieren acceso al bus real de la sesión.

Los plugins Omarchy se ejecutan como código del usuario; esta interfaz no es un sandbox. No se publica un shell remoto. Los comandos registrados usan argumentos fijos y se ejecutan mediante systemd/Bubblewrap con red deshabilitada y `/usr` de solo lectura.

## Licencia

[MIT](LICENSE). Los avisos de las dependencias se conservan en
[avisos de terceros](docs/third-party-notices.md).
