# Expose webhooks over TLS

[Español](../es/exposure.md) · [Security](security.md)

The engine accepts ingress only on a loopback IP. Administration uses a same-user
Unix socket without HTTP routes. Exposure is optional: installing never publishes
ports, configures DNS/firewall or creates tunnels.

Connections → Inbound reports the actual loopback listener or HTTP disabled.
Disconnecting the engine makes listener status unknown. Local listening does not
prove proxy/DNS/firewall reachability; public exposure is always unverified.
`status` contains ingress.state (disabled/starting/listening/stopped/failed),
ingress.address only while listening, and public_exposure `unverified`. It reflects
the current instance, not persisted state. `--listen=` disables HTTP while allowing
local events. Displayed addresses contain no secrets/authenticated links.

## Included proxy

`packaging/proxy/Caddyfile` uses explicit certificates, defaults to
`127.0.0.1:9443`, permits POST `/hooks/<id>` only, and forwards to `127.0.0.1:8791`.
Other paths return 404. Caddy admin API, configuration autosave and automatic HTTPS
are disabled; there is no HTTP redirect listener or HTTP/3. Body limit is 256 KiB,
with time/header bounds. Original signed body bytes are preserved.

Official directives: [reverse proxy](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy),
[TLS](https://caddyserver.com/docs/caddyfile/directives/tls),
[bind](https://caddyserver.com/docs/caddyfile/directives/bind),
[body limit](https://caddyserver.com/docs/caddyfile/directives/request_body).

With Caddy available and binaries built:

```bash
CADDY=/path/to/caddy python3 scripts/test-tls-proxy.py
```

This creates temporary certificates/profile and two loopback listeners, verifies
TLS without insecure bypass, signatures over Unicode/original whitespace,
deduplication, oversized bodies, invalid signatures, admin paths and stopped engine,
then cleans up. Tested with Caddy 2.11.4; no real public endpoint was published.

## Prepare LAN/VPN

Install Caddy using `omarchy pkg add caddy` if needed. These explicit steps are
separate from automatic plugin installation:

1. Create/activate an authenticated entry with a secret reference. Choose the
   receiver device/domain.
2. Obtain a certificate/chain valid for its name/IP. A LAN can use your private CA
   explicitly trusted by clients; public providers need a chain they recognize.
3. Keep the private key in a stable user path with mode 0600, not the repository
   or `/tmp` for a permanent service. This template does not obtain/renew certificates.
4. Set real paths/address and validate before running:

```bash
export QUATRRO_TLS_HOST=webhooks.example.net
export QUATRRO_TLS_BIND=192.168.1.20
export QUATRRO_TLS_PORT=9443
export QUATRRO_HTTP_PORT=8791
export QUATRRO_TLS_CERT="$HOME/.config/quatrro/tls/fullchain.pem"
export QUATRRO_TLS_KEY="$HOME/.config/quatrro/tls/private-key.pem"
caddy validate --config packaging/proxy/Caddyfile --adapter caddyfile
caddy run --config packaging/proxy/Caddyfile --adapter caddyfile
```

The sample LAN address must be replaced by one on your machine. Prefer an explicit
interface address over a wildcard. IPv6 can use an explicit bind address; review
IPv6 firewall rules too. Never change the engine to a public listener.

5. From another CA-trusting device, verify `/status` returns 404 and an unsigned
   request returns 401. Send a signed fixture and inspect History. Fix certificate
   chain/name/clock failures rather than disabling TLS verification.

There is no universal firewall rule: restrict the chosen port to needed sources.
LAN clients still need secret/HMAC authentication. Same-user local clients belong
to the system trust boundary; loopback alone does not establish remote identity.

## Internet and optional tunnel

You can run the proxy on your own server with a public certificate and explicit
interface, publishing only its TLS port. Configure DNS/network rules there; do not
publish internal port 8791. If Omarchy is unreachable, an SSH reverse tunnel can
carry only the webhook receiver to server loopback:

```bash
ssh -N -T \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=3 \
  -R 127.0.0.1:8791:127.0.0.1:8791 user@server
```

Run Caddy on that server pointing at its loopback. Clients reach TLS; server-to-Omarchy
uses SSH. Verify the SSH host key through a trusted channel; forward neither agent
nor administrative socket. Check actual forwarding listens only on loopback:
`GatewayPorts yes` can widen exposure. Choose suitable `no`/`clientspecified` policy
and verify the listener. See [OpenSSH](https://man.openbsd.org/ssh#R) and
[GatewayPorts](https://man.openbsd.org/sshd_config#GatewayPorts).

Tunnel syntax was checked with `ssh -G`, not an actual external server/account.
The tunnel stores no events while down; the provider must retry under its contract.
A stopped engine produces proxy HTTP 502, not false acknowledgement.

## Optional service and maintenance

`packaging/proxy/quatrro-proxy.service` is an optional unit for `/usr/bin/caddy`.
Prepare:

- `~/.config/quatrro/Caddyfile`, a reviewed template copy.
- `~/.config/quatrro/proxy.env`, based on `proxy.env.example` with actual values.
- Own mode-0700 `~/.local/state/quatrro-proxy/` for Caddy state.
- Certificate/key paths readable by the service.

This optional unit was not enabled during the current tests; its installation/update
needs deployment-specific verification. After certificate/configuration changes,
validate then restart: the reload API is intentionally disabled. Keep Caddy current
and rerun its test when changing versions.

The panel reports only the local engine, not DNS/firewall/certificate/tunnel health.
Verify those in the actual deployment.
