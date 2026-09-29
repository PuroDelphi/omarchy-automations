# Hooks de Omarchy y administración opcional

[English](../en/system-integration.md) · [Guía de usuario](user-guide.md) · [Opciones](options.md)

## Hooks del escritorio

Los hooks introducen eventos locales en el mismo motor de flujos que los webhooks.
No necesitan receptor HTTP. Instala solo los adaptadores deseados, desde el proyecto
o la distribución extraída, con `quatrroctl` instalado en `~/.local/bin`:

```bash
omarchy hook install theme-set packaging/hooks/quatrro-theme-set
omarchy hook install battery-low packaging/hooks/quatrro-battery-low
```

Omarchy copia cada adaptador a su directorio `<hook>.d`. Reinstalar reemplaza
nuestro adaptador del mismo nombre. Conserva hooks planos y otros adaptadores.

| Hook | Fuente exacta del flujo | Tipo de evento | Datos |
|---|---|---|---|
| `theme-set` | `hook:theme-set` | `omarchy.theme-set` | `theme`: identificador del tema |
| `font-set` | `hook:font-set` | `omarchy.font-set` | `font`: nombre de fuente |
| `battery-low` | `hook:battery-low` | `omarchy.battery-low` | `percentage`: entero 0–100 |
| `post-boot` | `hook:post-boot` | `omarchy.post-boot` | Objeto vacío |
| `post-update` | `hook:post-update` | `omarchy.post-update` | Objeto vacío |
| `pre-refresh-pacman` | `hook:pre-refresh-pacman` | `omarchy.pre-refresh-pacman` | Objeto vacío |

Para las demás filas, instala `packaging/hooks/quatrro-HOOK` con el nombre de ese
hook. Los argumentos se convierten en JSON y no se interpretan como comandos.

### Ejemplo: informar de un cambio de tema

1. Crea una acción de notificación con cuerpo `Theme: {{data.theme}}`.
2. Crea un flujo habilitado de fuente `hook:theme-set` con esa acción como paso.
3. Simula tipo `omarchy.theme-set` con `{"theme":"tokyo-night"}`; espera
   `Theme: tokyo-night`. Repite con `{"theme":"catppuccin-latte"}`.
4. Guarda, revisa el permiso de notificación y activa. Un futuro cambio de tema
   de Omarchy invoca el adaptador instalado. Comprueba notificación e Historial.

Para emitir un evento real explícito sin cambiar el tema del escritorio:

```bash
quatrroctl hook theme-set tokyo-night
```

Esto puede ejecutar flujos activos. No es simulación ni evidencia de un cambio
real de tema. El socket local verifica el usuario, no procedencia exclusiva del
ejecutable Omarchy.

### Ejemplo: batería baja hacia un receptor HTTPS

Registra tu destino HTTPS real y una acción HTTP con cuerpo JSON
`{"battery":"{{data.percentage}}"}`. Conserva las comillas para que la plantilla
sea JSON válido; un marcador de campo exacto conserva el valor numérico original.
Crea un flujo de `hook:battery-low` con condición
`data.percentage lt 15` de tipo numérico. Simula tipo `omarchy.battery-low`:
`{"percentage":10}` coincide y resuelve `{"battery":10}`; `{"percentage":20}` no
coincide. Guarda, revisa y activa cuando quieras enviar. Un hook real de batería
alimenta ese flujo. Difiere de un monitor periódico: depende de que Omarchy
invoque su hook de batería baja.

Los adaptadores esperan al motor como máximo un segundo, con límite externo de
1,5 segundos y terminación forzada tras otros 0,2 segundos. Devuelven éxito a
Omarchy aunque falle el motor. No reintentan ni dejan procesos desacoplados; un
evento puede perderse durante una caída o persistirse aunque se pierda su
confirmación. El hook no espera a que terminen las acciones del flujo.

Para retirar solo estos dos adaptadores, inspecciona sus rutas y elimina sus archivos:

```bash
ls -l ~/.config/omarchy/hooks/theme-set.d/quatrro-theme-set ~/.config/omarchy/hooks/battery-low.d/quatrro-battery-low
rm -- ~/.config/omarchy/hooks/theme-set.d/quatrro-theme-set ~/.config/omarchy/hooks/battery-low.d/quatrro-battery-low
```

Esto detiene futuras invocaciones de esos adaptadores; no cancela eventos en cola.
El desinstalador normal del plugin no gestiona adaptadores instalados por separado.

## Administración opcional de servicios del sistema

Las acciones de servicios de usuario no necesitan este componente. Las acciones
`system-service` requieren un broker root separado, allowlist local exacta y
autorización Polkit. La instalación normal no instala ni habilita el broker.
Solo ofrece `status`, `start`, `stop` y `restart` sobre nombres `.service` exactos;
no shell, comandos root arbitrarios, comodines, instalación de paquetes ni
acciones enable/disable.

A continuación hay dos políticas de ejemplo. Sustituye `1000` por el UID numérico
de la cuenta no root deseada (`id -u`). La unidad de ejemplo debe existir y ser
el servicio que quieres gestionar; preparar no la crea.

Ejemplo de solo consulta, guardado como `broker-policy.json`:

```json
{"version":1,"rules":[{"uid":1000,"unit":"example-worker.service","operations":["status"]}]}
```

Ejemplo de consulta y reinicio (reiniciar puede interrumpir ese servicio):

```json
{"version":1,"rules":[{"uid":1000,"unit":"example-worker.service","operations":["status","restart"]}]}
```

Un array `rules` vacío deniega todo. La política admite como máximo 64 KiB y
128 reglas. Se rechazan UID root, reglas/operaciones duplicadas, claves desconocidas
y unidades template sin instancia. La cuenta debe resolverse localmente al preparar.

### Preparar, revisar e instalar

Usa un directorio de salida nuevo. Este primer comando no requiere privilegios
administrativos y no instala ni autoriza nada:

```bash
python3 scripts/prepare-broker-install.py --policy broker-policy.json --output broker-preview
```

Revisa `broker-preview/manifest.json`, `broker.json`, las reglas Polkit generadas
y los destinos fijos de los seis archivos. Los hashes detectan cambios pero no
acreditan quién produjo el paquete. Usa una compilación revisada. Desde una terminal
administrativa, aplica el paquete revisado y habilita explícitamente su socket:

```bash
sudo python3 scripts/install-broker.py apply --bundle broker-preview
sudo systemctl enable --now quatrro-broker.socket
```

La instalación deja el broker detenido y deshabilitado hasta el segundo comando.
En el panel crea una acción `system-service` con unidad y operación exactas y usa
**Comprobar permisos administrativos**. Comprobar no actúa sobre el servicio ni concede permisos.
Por ejemplo, la primera política autoriza consulta pero deniega reinicio; la
segunda autoriza ambos. Puedes guardar borradores sin broker, pero activar un
flujo administrativo habilitado requiere comprobaciones satisfactorias. Ejecutar
comprueba otra vez. Revisa el permiso del flujo antes de activar. Start/stop/restart
reales afectan al servicio del sistema; comprueba su estado e Historial.

### Actualizar, revocar y retirar

Prepara un paquete nuevo para actualizar. Aplicar conserva normalmente la política
y reglas Polkit instaladas; usa `--replace-policy` solo para sustituir esos
permisos intencionalmente. Cada apply detiene/deshabilita el broker hasta habilitarlo
explícitamente. Para revocar todos sus permisos, prepara la política vacía
predeterminada en otro directorio, revísala y aplica con `--replace-policy`:

```bash
python3 scripts/prepare-broker-install.py --output broker-deny-all
sudo python3 scripts/install-broker.py apply --bundle broker-deny-all --replace-policy
```

Revocar permisos del plugin es otro control independiente y no deshace efectos
completados. Para retirar los archivos propios verificados del broker:

```bash
sudo python3 scripts/install-broker.py remove
```

Si se interrumpió la instalación y se informa de transacción pendiente, usa
`sudo python3 scripts/install-broker.py recover` antes de aplicar/retirar otra vez.
Recuperar restaura el estado de instalación anterior y deja el socket deshabilitado;
rechaza si no hay transacción pendiente. Archivos ajenos o modificados provocan
rechazo en vez de sobrescritura silenciosa. No borres el recibo para eludirlo.

El broker se probó con systemd/Polkit reales en un entorno aislado; no se ha
instalado en este host. Las pruebas de hooks ejercitan argumentos y fallos del
motor. Estas comprobaciones no acreditan la seguridad operativa de tu servicio
ni entrega durante un cierre de sesión/suspensión real. Consulta el [registro de validación](../validation.md).
