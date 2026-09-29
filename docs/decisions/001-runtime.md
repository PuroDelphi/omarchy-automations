# 001 — Runtime y frontera local

Estado: aceptada, 2026-09-28.

Motor y CLI en Go 1.26; SQLite mediante `modernc.org/sqlite`, dependencias fijadas en `go.mod`/`go.sum`. Plugin QML sin dependencias en objetos privilegiados del shell. Protocolo JSON de una petición por conexión Unix, limitado a 1 MiB, con plazo máximo y validación de UID por SO_PEERCRED.

El socket vive en un directorio privado XDG. Un flock protege contra dos motores y contra retirar el socket de un motor activo. El control no comparte listener con los webhooks HTTP.

Las configuraciones activas tienen hash y copia inmutable. Los permisos están ligados a flujo, acción y destino completos. La activación exige el hash revisado y capacidades exactas; modificar los datos de una acción no reutiliza permisos anteriores.

El aislamiento de procesos es adicional a la separación del motor y la UI. El mismo UID sigue siendo una frontera de confianza compartida; no se promete protección frente a otros procesos maliciosos del usuario.
