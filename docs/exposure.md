# Exponer webhooks por TLS

El motor solo acepta un listener de entrada en una IP loopback. La API de
administración usa un socket Unix del mismo usuario y no tiene rutas HTTP.
La exposición es opcional: instalar el plugin no publica puertos, configura
DNS, cambia el firewall ni crea túneles.

En **Connections → Inbound / Conexiones → Entrada**, el panel muestra la
dirección loopback que el motor abrió realmente, o indica que HTTP está
deshabilitado. Si pierde conexión con el motor, el estado del receptor pasa a
desconocido. La escucha local no demuestra que un proxy, DNS o firewall permita
acceso externo: la exposición pública se muestra siempre como no verificada.

`quatrroctl status` incluye `ingress.state` (`disabled`, `starting`, `listening`,
`stopped` o `failed`), `ingress.address` solo mientras escucha y
`ingress.public_exposure: "unverified"`. El estado no se conserva entre reinicios;
refleja la instancia actual. `--listen=` deshabilita HTTP sin impedir eventos
locales. La dirección mostrada no incluye secretos ni enlaces autenticados.

## Proxy incluido

`packaging/proxy/Caddyfile` ofrece un proxy Caddy con certificados explícitos.
Por defecto escucha en `127.0.0.1:9443`; solo permite POST a `/hooks/<id>` y
envía al receptor `127.0.0.1:8791`. Las demás rutas devuelven 404. La API
administrativa de Caddy, el guardado automático de configuración y HTTPS
automático están desactivados. No se abren listeners HTTP de redirección ni
HTTP/3. El cuerpo tiene un máximo de 256 KiB y hay límites de tiempos/cabeceras.

Se usan las directivas oficiales de [proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy),
[TLS](https://caddyserver.com/docs/caddyfile/directives/tls),
[bind](https://caddyserver.com/docs/caddyfile/directives/bind) y
[límite de cuerpo](https://caddyserver.com/docs/caddyfile/directives/request_body).
El proxy conserva los bytes que necesita la firma del proveedor.

## Probar sin exposición externa

Con Caddy disponible y el proyecto compilado:

```bash
CADDY=/ruta/al/binario/caddy python3 scripts/test-tls-proxy.py
```

La prueba crea certificados, un perfil del motor y dos listeners loopback
temporales. Verifica TLS sin `--insecure`, firma sobre JSON con Unicode y
espacios originales, deduplicación, cuerpo excesivo, firma inválida, rutas
administrativas y motor apagado. Limpia los procesos y archivos al terminar.
La configuración se probó con Caddy 2.11.4; no se ha publicado un endpoint real.

## Preparar LAN/VPN

En Omarchy puedes instalar Caddy con `omarchy pkg add caddy`. Los siguientes
pasos son explícitos y no forman parte de la instalación automática del plugin.

1. Crea una entrada autenticada y actívala en el panel. Registra el secreto
   por referencia. Decide qué dispositivo/dominio usará el receptor.
2. Obtén un certificado y su cadena, válidos para ese nombre o IP. En una LAN
   puedes utilizar tu CA privada, confiada explícitamente por los clientes.
   Para proveedores públicos necesitas una cadena que ellos reconozcan.
3. Conserva la clave privada en una ruta estable de tu usuario con permisos
   0600. No pongas las claves en el repositorio ni uses `/tmp` para un servicio
   permanente. Esta plantilla no solicita ni renueva certificados.
4. Desde una terminal, configura las variables con tus rutas y dirección:

```bash
export QUATRRO_TLS_HOST=webhooks.ejemplo.net
export QUATRRO_TLS_BIND=192.168.1.20
export QUATRRO_TLS_PORT=9443
export QUATRRO_HTTP_PORT=8791
export QUATRRO_TLS_CERT="$HOME/.config/quatrro/tls/fullchain.pem"
export QUATRRO_TLS_KEY="$HOME/.config/quatrro/tls/private-key.pem"
caddy validate --config packaging/proxy/Caddyfile --adapter caddyfile
caddy run --config packaging/proxy/Caddyfile --adapter caddyfile
```

La dirección LAN es un ejemplo: debe existir en tu máquina. Usa una dirección
concreta, en vez de un comodín, para limitar las interfaces. Para IPv6 puedes
usar una dirección explícita en `QUATRRO_TLS_BIND`; revisa también las reglas
IPv6 del firewall. No cambies el listener del motor a una IP pública.

5. En otro dispositivo que confíe en la CA, verifica que `/status` devuelve
   404 y una entrada sin firma devuelve 401. Envía después un fixture firmado
   y confirma su ejecución en Historial. Si el certificado falla, corrige la
   cadena, nombre o reloj; no desactives la verificación TLS.

No se incluye una regla universal de firewall: limita el puerto elegido a las
fuentes necesarias. El secreto/HMAC sigue siendo obligatorio aunque el cliente
esté en la LAN. Los clientes locales del mismo usuario pertenecen a la frontera
de confianza del sistema; loopback por sí solo no demuestra identidad remota.

## Internet y túnel opcional

Puedes situar el mismo proxy en un servidor que controles, con certificado
público y una interfaz explícita. Solo el puerto TLS debe publicarse. Configura
DNS y reglas de red de ese servidor; no publiques el puerto interno 8791.

Si Omarchy no tiene una dirección alcanzable, un túnel SSH inverso puede
transportar únicamente el receptor HTTP al loopback del servidor:

```bash
ssh -N -T \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -R 127.0.0.1:8791:127.0.0.1:8791 usuario@servidor
```

En ese caso ejecuta Caddy en el servidor, apuntando a su `127.0.0.1:8791`.
El cliente externo llega por TLS al proxy, y el tramo servidor–Omarchy va por
SSH. Verifica la clave del servidor SSH por un canal confiable y no reenvíes el
agente ni el socket administrativo de Omarchy Automations. En el servidor, comprueba que
el puerto reenviado solo escucha en loopback: `GatewayPorts yes` puede cambiar
esa exposición. Usa una política `GatewayPorts no` o `clientspecified` apropiada
y comprueba el listener efectivo. Estas opciones están descritas por
[OpenSSH](https://man.openbsd.org/ssh#R) y
[sshd_config](https://man.openbsd.org/sshd_config#GatewayPorts).

La sintaxis del túnel se verificó con `ssh -G`; no se ha probado una conexión
a un servidor externo ni configurado cuentas. El túnel no conserva eventos
cuando está caído: el proveedor debe reintentar según su contrato. La caída
del motor produce 502 en el proxy, no una confirmación falsa.

## Servicio opcional y mantenimiento

`packaging/proxy/quatrro-proxy.service` es una unidad opcional para Caddy
instalado en `/usr/bin/caddy`. Requiere que prepares previamente:

- `~/.config/quatrro/Caddyfile`, copia de la plantilla revisada.
- `~/.config/quatrro/proxy.env`, basado en `proxy.env.example`, con valores reales.
- `~/.local/state/quatrro-proxy/`, directorio propio 0700 para estado de Caddy.
- Certificados y clave en rutas legibles por el servicio.

El empaquetado R1 verificará instalación/actualización de esta unidad; no se
ha habilitado durante las pruebas actuales. Para renovaciones de certificados
o cambios de configuración, valida primero y reinicia el proxy. La API de
recarga está deshabilitada de forma intencional. Mantén Caddy actualizado y
vuelve a ejecutar la prueba al cambiar su versión.

La UI informa sobre el motor local. No afirma conocer el estado del DNS,
firewall, certificado o túnel externo; esas comprobaciones corresponden al
despliegue concreto.
