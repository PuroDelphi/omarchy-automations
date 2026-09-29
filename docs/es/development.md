# Desarrollo y compilación desde fuentes

[English](../en/development.md) · [Instalación precompilada](installation.md)

Esta guía es para contribuir al proyecto o probar cambios todavía no publicados.
Para usar el plugin en Linux amd64, instala los binarios precompilados: no necesitas Go.

## Preparar y comprobar el código

Necesitas Linux, Go 1.26 o posterior, Make, Omarchy/Quickshell y systemd de usuario.

```bash
git clone https://github.com/PuroDelphi/omarchy-automations.git
cd omarchy-automations
make build
make test
make check
```

## Perfil aislado

Usa estas variables en ambas terminales:

```bash
export QUATRRO_PROFILE="$PWD/.dev/profile"
export PATH="$PWD/build:$PATH"
```

Ejecuta `quatrrod` en la primera terminal. En la segunda:

```bash
quatrroctl status
quatrroctl config.save --stdin < examples/notification.json
quatrroctl simulate '{"source":"local:demo","type":"demo","data":{"message":"Hola"}}'
quatrroctl config.preview
python3 scripts/run-ui.py
```

Simular no activa flujos ni ejecuta acciones. Activa solo después de revisar
las capacidades y el hash; consulta el [protocolo](protocol.md). El perfil separa
los datos, pero cualquier acción que actives puede tener efectos reales.
Detén el motor manual con Ctrl+C al terminar.

`python3 scripts/smoke-native.py --offscreen` verifica la conexión y genera
`.dev/panel-render.png`. `python3 scripts/test-ui-flow.py` prueba formularios,
activación, notificaciones, entregas HTTPS locales e historial.
`QUATRRO_HOST_TEST=1 go test -race ./...` añade pruebas con notificaciones,
un servicio temporal dedicado y comandos aislados; necesitan el bus de la sesión.

## Instalar una compilación propia

Mantén el checkout fuera del directorio del plugin instalado. Después de compilar:

```bash
python3 scripts/install.py --activate
```

No instales el panel gestionado por fuentes sobre otro gestionado por Git.
Para actualizar una instalación propia, recompila, revisa el trabajo pendiente,
detén `quatrrod.service` y repite el instalador. Registra un respaldo y conserva
los datos de la aplicación. Compilar en otras arquitecturas no implica que se
haya verificado su compatibilidad con el escritorio.

Los plugins Omarchy ejecutan código de usuario; la interfaz no es un sandbox.
Los comandos registrados usan argumentos fijos y aislamiento systemd/Bubblewrap,
sin red y con `/usr` de solo lectura.

[Detalles del instalador y restauración](../installation.md) ·
[Paquetes y reproducibilidad](packages.md).
