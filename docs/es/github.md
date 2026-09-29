# Primera publicación desde esta máquina

[English](../en/github.md) · [Guía de usuario](user-guide.md)

El proyecto local está en `/home/macondo/Work/quatrro-automations`. Su remoto
`origin` apunta a `https://github.com/PuroDelphi/omarchy-automations.git` y la rama
inicial es `main`. En esta auditoría no hay commits ni identidad local de autor
configurada. No se ha publicado ningún archivo. Completa los controles de entrega
del roadmap antes de publicar una versión estable; el paquete actual es de desarrollo.

## Iniciar sesión

GitHub CLI está preparado fuera del proyecto. Ejecuta en tu terminal:

```bash
/home/macondo/Work/.quatrro-tools/github-cli/gh auth login --hostname github.com --git-protocol https --web
/home/macondo/Work/.quatrro-tools/github-cli/gh auth setup-git --hostname github.com
/home/macondo/Work/.quatrro-tools/github-cli/gh auth status --hostname github.com
```

Sigue las instrucciones del navegador/código de dispositivo con la cuenta que
pueda escribir en el repositorio. Introduce credenciales solo en el flujo de acceso
del proveedor, no en el chat. Iniciar sesión no sube el proyecto. Si después
instalas `gh` en PATH, puedes sustituir la ruta completa por su nombre corto.

## Revisar destino y archivos

```bash
cd /home/macondo/Work/quatrro-automations
/home/macondo/Work/.quatrro-tools/github-cli/gh repo view PuroDelphi/omarchy-automations
git remote -v
git ls-remote origin
git status --short
git diff --cached --stat
git diff --stat
```

Un resultado vacío y satisfactorio de `git ls-remote` indica que no anuncia refs;
un error no significa repositorio vacío. Revisa archivos preparados, cambios sin
preparar y archivos nuevos. `.gitignore` excluye compilaciones, distribuciones,
perfiles temporales, bases de datos y directorios de secretos, pero no detecta
secretos pegados en código o ejemplos normales. La documentación pública incluye
esta ruta del workspace; sustitúyela si prefieres instrucciones independientes
de la máquina.

El propietario eligió [MIT](../../LICENSE). Incluye LICENSE y los
[avisos de dependencias](../third-party-notices.md) al publicar.

## Crear el primer commit local

Sustituye los valores de ejemplo por la identidad de autor que elijas. Puedes usar
la dirección no-reply de tus ajustes de correo de GitHub. `--local` afecta solo a
este repositorio y no cambia otros proyectos.

```bash
git config --local user.name "Tu nombre de autor elegido"
git config --local user.email "TU_CORREO_ELEGIDO"
git add --all
git diff --cached --check
git diff --cached --stat
git commit -m "Add Omarchy Automations"
```

Revisa el contenido preparado antes del commit. `git add --all` incluye todos los
archivos admitidos del proyecto; no lo ejecutes con archivos privados o ajenos dentro.

## Subir y verificar

Si el remoto está vacío, publica el commit revisado:

```bash
git push -u origin main
git rev-parse HEAD
git ls-remote origin refs/heads/main
```

Los identificadores de commit local y remoto deben coincidir. Esto publica código,
ejemplos y documentación; los paquetes ignorados de `dist/` no se suben con el código.

Si el remoto ya tiene historial, consulta primero su rama predeterminada y tráelo:

```bash
/home/macondo/Work/.quatrro-tools/github-cli/gh repo view PuroDelphi/omarchy-automations --json defaultBranchRef
git fetch origin
git log --oneline --graph --all -20
```

Integra la rama existente antes de subir. Para un remoto `main` creado con README
o licencia independientes, `git merge --allow-unrelated-histories origin/main`
combina ambos historiales. Revisa y resuelve conflictos, repite las comprobaciones
relevantes y confirma la fusión antes de subir. Sustituye la rama si el remoto usa
otra. Usa `git merge --abort` para reconsiderar una fusión en curso. No fuerces
un push sobre historial existente.

Un fallo de autenticación exige corregir acceso/sesión; un push rechazado también
puede deberse a protección de rama o commits remotos nuevos. Revisa la causa en
vez de eludirla. Subir binarios de release y etiquetar una versión estable son
pasos de publicación separados, posteriores a los controles de entrega.

## Auditoría local preparada

El conjunto revisado contiene fuentes, pruebas, ejemplos, empaquetado y guías
emparejadas. Compilaciones/distribuciones generadas y perfiles locales quedan
ignorados. También se excluyen variantes de entorno (`.env.*`, salvo un
`.env.example` deliberado) y perfiles SQLite. Un archivo ya seguido continúa
seguido aunque se añada una regla: revisa
`git ls-files --cached --others --exclude-standard` antes del commit.

El análisis local no encontró bloques de claves privadas ni patrones de tokens
GitHub/Google/AWS/Slack en 314 candidatos al auditar. Esto no demuestra ausencia
de cualquier secreto: revisa ediciones posteriores y texto de configuración
ordinario. Los vectores de autenticación usan intencionalmente un valor público.
Aún no hay primer commit y falta tu identidad de autor. La licencia elegida es MIT.
El área de staging conserva versiones antiguas de algunos archivos: tras revisar,
`git add --all` es necesario para preparar el trabajo actual antes del commit.
