# Publicar Omarchy Automations en GitHub

Repositorio elegido: https://github.com/PuroDelphi/omarchy-automations .
El remoto `origin` local ya apunta a esa dirección por HTTPS. Todavía no se ha
verificado su contenido ni el permiso de escritura con una sesión autenticada.
El proyecto continúa en desarrollo; la revisión de publicación se hará al cerrar
las fases funcionales, el cambio de nombre y la estética nativa.

## Iniciar sesión en esta máquina

GitHub CLI 2.100.0 está disponible en una carpeta de herramientas del workspace,
fuera del proyecto. Su descarga se verificó contra el checksum oficial.
Ejecuta en tu terminal:

```bash
/home/macondo/Work/.quatrro-tools/github-cli/gh auth login --hostname github.com --git-protocol https --web
/home/macondo/Work/.quatrro-tools/github-cli/gh auth setup-git --hostname github.com
/home/macondo/Work/.quatrro-tools/github-cli/gh auth status --hostname github.com
```

Sigue el enlace/código que muestre la herramienta y autentícate con la cuenta
que tiene acceso a `PuroDelphi/omarchy-automations`. No necesitas enviar tokens,
contraseñas ni claves privadas al asistente. Iniciar sesión no sube archivos.
Si después instalas `gh` en el PATH, puedes sustituir la ruta completa por `gh`.

Procedimiento basado en la [documentación oficial de GitHub](https://docs.github.com/en/get-started/git-basics/caching-your-github-credentials-in-git).

## Comprobaciones y subida al terminar el proyecto

1. Comprobar acceso con `gh repo view PuroDelphi/omarchy-automations` y consultar
   las ramas remotas antes de decidir cómo integrar el primer commit local.
2. Revisar `.gitignore`, archivos preparados, secretos, licencias y artefactos;
   completar las validaciones R1 y las notas de la versión.
3. Configurar nombre y correo de autor elegidos por ti con `git config --local`.
   El nombre de la cuenta de GitHub no sustituye esa elección.
4. Crear el commit local final. Si el remoto tiene contenido, integrarlo sin
   sobrescribir su historial; no usar force push como paso de inicio.
5. Con acceso e instrucción de publicación, subir la rama acordada y comprobar
   que el commit remoto coincide con el local.

Las guías [English](en/github.md) y [Español](es/github.md) incluyen comandos
para el primer commit/push y distinguen remoto vacío de historial existente.
La rama definitiva requiere inspeccionar el remoto autenticado. No se ha publicado nada.
