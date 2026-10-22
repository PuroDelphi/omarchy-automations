---
version: alpha
name: Omarchy Automations
description: Native desktop automation editor using the host Omarchy theme.
omitted:
  - section: colors
    reason: Owned by qs.Commons.Color from the installed Omarchy shell.
  - section: typography
    reason: Owned by qs.Commons.Style.font from the installed Omarchy shell.
  - section: spacing
    reason: Owned by qs.Commons.Style.spacing and native control geometry.
  - section: rounded
    reason: Owned by qs.Commons.Style.cornerRadius.
  - section: components
    reason: Native qs.Ui components are adapted by the shared QML primitives.
---

# Omarchy Automations interface

Product: native desktop automation tool for Omarchy users. English is the default;
Spanish is supported throughout. Preserve the host desktop identity rather than
introducing a separate palette or typography.

## Runtime ownership

`qs.Commons.Color` and `qs.Commons.Style` from the installed Omarchy shell own
colors, fonts, spacing and control geometry. `ActionButton`, `InputField`,
`ChoiceField`, `ThemedLabel` and `LocalizedDialog` adapt `qs.Ui` primitives.
Do not duplicate these tokens in screen-specific constants. The historical system
architecture proposal remains in `docs/DESIGN.md`.

## Visual direction

Use a readable sequence of event, conditions and ordered actions. Numbers indicate
execution order only. Hook events have human-readable names; technical source
paths remain available in advanced mode. Templates use a searchable, categorized
list with explicit add-to-draft actions. Reuse native dropdowns, focus and dialogs.
Long content wraps; each bounded dialog owns one scrolling content region.

## State and language

The primary save operation prepares a draft, never silently activates effects.
Errors stay adjacent to the operation and leave the input available for correction.
Loading disables repeated mutations. All product text and accessible names must
support both languages; stable resource IDs and event paths are never translated.

## Process ownership

Native visual tokens and controls are retained. Since Preview 8 the full editor
lives in a separate bounded process; the shared shell contains only the widget
and an input-free launch/retry surface. This changes process ownership, not the
visual language. See `UX-CONTRACT.md` and `docs/en/security.md`.
