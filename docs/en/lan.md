# Private destinations (LAN)

[Español](../es/lan.md) · [Documentation](README.md)

Outbound requests block nonpublic addresses by default. For a private service,
configure `private_hosts` with **one** exact HTTPS destination authority:

- URL `https://nas.local/hooks`: exception `nas.local:443`.
- URL `https://nas.local:8443/hooks`: exception `nas.local:8443`.
- URL `https://[fd00::10]:8443/hooks`: exception `[fd00::10]:8443`.

Wildcards, other hosts/ports, duplicates and ambiguous ports are rejected. The
form explains the syntax and capability review displays the exception. Its hash
includes the entire destination and exception; changing either requires reviewing
a new revision. Google OAuth2 does not allow LAN exceptions.

An exception permits private unicast, loopback, link-local and CGNAT addresses
resolved for that authority. It does not open networks to other destinations or
disable TLS, DNS validation, IP pinning, environment-proxy blocking or redirect
blocking. Multicast, unspecified and blocked documentation/transition ranges
remain forbidden. Resolution containing unauthorized addresses fails before dialing.

The certificate must be valid for the hostname and issued by a trusted CA. There
is no TLS-verification bypass. IPv6 literals require brackets in both URL and
authority. Tests cover actual IPv4/IPv6 loopback TLS, exact authorization and
isolation from other destinations. They do not prove connectivity to a particular
LAN device or link-local IPv6 with an interface identifier; those need separate checks.

Old configurations with exceptions unrelated to the destination must be corrected
in the draft before activation. Permissions are not silently broadened or normalized.

Revoking the step capability prevents both its first pending send and a retry after
HTTP 503. Tests exercise the worker with an actual TLS receiver. QML integration
checks the authority appears in review before activation and delivery completes
under that grant.
