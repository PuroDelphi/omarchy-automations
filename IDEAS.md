# Future extension: seamless control between computers

Inspired by PowerToys Mouse Without Borders, explore sharing a keyboard, mouse,
clipboard and files between trusted desktops. This is a **proposal**, not a
feature of the current plugin or its marketplace submission.

Omarchy Automations would manage pairing approvals, the companion service,
device-specific permissions, health monitoring, notifications and history. A
separate low-latency agent on each computer would carry input and clipboard/file
data directly. High-frequency pointer events must not pass through webhook flows.
Start with two Omarchy desktops; evaluate Windows only after the Linux path is
usable. Wayland/compositor input integration, secure peer authentication,
reconnection, screen-edge transitions, clipboard loop prevention, file size
limits and explicit remote-control consent need separate design and testing.

## Español

Inspirado en Mouse Without Borders de PowerToys, estudiar el uso compartido de
teclado, ratón, portapapeles y archivos entre equipos de confianza. Es una
**propuesta futura**, no una función actual ni parte de la solicitud al catálogo.

Omarchy Automations gestionaría autorización de emparejamiento, servicio auxiliar,
permisos por dispositivo, estado, notificaciones e historial. Un agente separado
en cada equipo transportaría directamente entrada y datos con baja latencia; los
movimientos del ratón no pasarían por flujos webhook. Empezar con dos equipos
Omarchy y considerar Windows después de validar Linux. Diseñar y probar aparte
la integración Wayland, autenticación, reconexión, bordes de pantalla, prevención
de bucles de portapapeles, límites de archivos y consentimiento de control remoto.
