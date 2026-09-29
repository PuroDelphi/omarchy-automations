# Hooks de Omarchy

[English](../en/hooks.md) · [Documentación](README.md)

Los adaptadores de `packaging/hooks/` añaden eventos locales al motor. Cada
invocación tiene un identificador aleatorio; los argumentos se codifican como
JSON y nunca se interpretan como comandos.

| Hook | Fuente del flujo | Datos |
|---|---|---|
| battery-low | `hook:battery-low` | `percentage`, entero entre 0 y 100 |
| font-set | `hook:font-set` | `font`, nombre de fuente |
| post-boot | `hook:post-boot` | Objeto vacío |
| post-update | `hook:post-update` | Objeto vacío |
| pre-refresh-pacman | `hook:pre-refresh-pacman` | Objeto vacío |
| theme-set | `hook:theme-set` | `theme`, identificador del tema |

El tipo es `omarchy.<hook>`. Los campos del flujo se pueden consultar como
`data.theme`, `data.font` o `data.percentage`. El socket restringe el acceso al
mismo usuario; estos eventos no prueban que su origen sea exclusivamente el
ejecutable Omarchy.

## Instalar un adaptador

Con `quatrroctl` instalado en `~/.local/bin/`, desde la raíz del proyecto:

```bash
omarchy hook install theme-set packaging/hooks/quatrro-theme-set
```

Repetir para los hooks deseados. Omarchy los copia a directorios `<hook>.d`;
los nombres `quatrro-<hook>` evitan reemplazar los hooks de otras herramientas.
Una nueva instalación reemplaza únicamente nuestro archivo con el mismo nombre.
No modificar el hook plano existente ni `/usr/share/omarchy`.

## Comportamiento ante fallos

La llamada al motor espera como máximo un segundo. El adaptador añade un límite
externo de 1,5 segundos, con terminación forzada tras 0,2 segundos adicionales,
y siempre devuelve éxito a Omarchy. No hay procesos desacoplados ni reintentos.
Con el motor caído el evento se descarta; si la confirmación se pierde, el motor
puede haberlo persistido. No se garantiza entrega durante una interrupción.
El hook no espera a que se ejecuten las acciones del flujo.

Diagnóstico explícito, sin silenciar errores:

```bash
quatrroctl hook theme-set ejemplo
```

Esto emite un evento real y puede ejecutar un flujo activo. Para comprobar un
flujo sin efectos, utilizar primero la simulación del panel.

## Compatibilidad de permisos

Las capacidades de acciones usan `flow:<flujo>:action:<acción>` y las de
monitores `monitor:<id>`. Las revisiones de desarrollo anteriores que usaban
`<flujo>:<acción>` requieren revisión y activación nueva; sus permisos antiguos
no se convierten automáticamente. Un trabajo pendiente sin el permiso nuevo
se deniega antes de producir efectos.
