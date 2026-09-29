# Destinos privados (LAN)

[English](../en/lan.md) · [Documentación](README.md)

El emisor bloquea direcciones no públicas por defecto. Para un servicio privado,
configurar `private_hosts` con **una sola** autoridad exacta del destino HTTPS:

- URL `https://nas.local/hooks`: excepción `nas.local:443`.
- URL `https://nas.local:8443/hooks`: excepción `nas.local:8443`.
- URL `https://[fd00::10]:8443/hooks`: excepción `[fd00::10]:8443`.

No admite comodines, otro host/puerto, entradas duplicadas ni puertos ambiguos.
El formulario explica el formato y la revisión muestra la excepción. El hash del
permiso incluye destino completo y excepción; cambiarlo exige aprobación de la
nueva revisión. OAuth2 Google no admite excepciones LAN.

Una excepción permite direcciones unicast privadas, loopback, link-local y CGNAT
que resuelvan para esa autoridad. No abre redes a otros destinos ni desactiva
TLS, validación DNS, pinning de IP, bloqueo de proxies ambientales o redirecciones.
No permite multicast, unspecified ni rangos de documentación/transición bloqueados.
Una resolución con direcciones no autorizadas se rechaza antes de conectar.

El certificado debe ser válido para el host y pertenecer a una CA de confianza.
No existe opción para omitir verificación TLS. Para IPv6 literal se requieren
corchetes en la URL y en host:puerto. Las pruebas verifican conexiones reales TLS
sobre loopback IPv4 e IPv6, permiso exacto y aislamiento de otros destinos.
No acreditan conectividad a un dispositivo LAN concreto ni IPv6 link-local con
identificador de interfaz; esos entornos necesitan sus propias comprobaciones.

Configuraciones antiguas con excepciones ajenas al destino deben corregirse en el
borrador antes de activar. No se normalizan silenciosamente ni se amplían permisos.

Revocar el permiso del paso impide tanto el primer envío pendiente como el
reintento de una entrega que respondió 503. Las pruebas pasan por el worker y
un receptor TLS real. El recorrido QML verifica que la autoridad autorizada
aparece en la revisión antes de activar y que la entrega completa con ese permiso.
