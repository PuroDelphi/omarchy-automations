# Notas de versión de desarrollo — 0.1.0-dev

[English](../en/release-notes.md) · [Documentación](README.md)

Versión preliminar publicada para Linux amd64; no es la versión estable 1.0.
Preview 5 añade una captura en la raíz para el marketplace y fija el SHA256 del
paquete descargado en la copia de código fuente. La etiqueta de la versión
apunta al commit de código revisado.

## Incluido

- Webhooks autenticados de entrada, entregas HTTPS, condiciones y permisos revisados por flujo.
- Notificaciones, servicios de usuario, comandos/scripts aislados, adaptadores locales, monitores y programación.
- Excepciones explícitas de red privada, implementación Google OAuth y broker administrativo instalado por separado.
- Controles/icono nativos, inglés por defecto, español persistente, diseño adaptable y etiquetas accesibles.
- Guías equivalentes, referencia de opciones, ejemplos completos, capturas, instalador y recuperación.

## Compatibilidad y actualización

Motor/CLI 0.1.0-dev, protocolo local 2, contrato de operaciones 1, contrato UI 1;
esquema SQLite 5. El broker opcional conserva protocolo 1. Los identificadores
quatrroctl, quatrrod.service y quatrro.automations conservan compatibilidad.
Instala juntos los componentes correspondientes y detén el motor antes de
actualizar. El instalador verifica archivos propios y registra respaldos; las
bases restauradas empiezan pausadas y sin permisos. Consulta la [guía](user-guide.md).

## Verificación y requisitos pendientes

Los binarios actuales Linux amd64 se reprodujeron idénticos con Go 1.26.8 en
árboles separados y cachés de compilación inicialmente vacías. Pasaron determinismo
del paquete, hashes e instalación/actualización/retirada del archivo extraído.
Las revisiones funcional, seguridad, UI nativa y documentación bilingüe constan
en el roadmap y registro de validación.

Quedan pendientes consentimiento/renovación/keyring con cuenta Google real y
pruebas de suspensión física/cierre completo de sesión. Fixtures de proveedor
y congelación de cgroup/reinicio de target no las sustituyen. El proyecto usa la [licencia MIT](../../LICENSE); los
[avisos de dependencias](../third-party-notices.md) conservan sus términos originales. Estos requisitos impiden declarar completada la versión 1.0.
