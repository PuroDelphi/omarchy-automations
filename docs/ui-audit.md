# Native UI audit

Acceptance evidence for R1.UI and F1D.7, closed after the installed-shell and
development checks below. System files were read-only references.

| Plugin component | Installed Omarchy component/pattern | Evidence |
|---|---|---|
| ActionButton | `qs.Ui.Button`; Network header buttons and selection pills | Native hover, pressed, selected and focus colors/borders from Style. Text actions retain labels; keyboard focus is enabled. |
| Destructive ActionButton | Urgent foreground, consistent with the urgent mode of native PanelActionButton | Revocation and destructive controls use Color.urgent; ordinary draft edits retain standard foreground. |
| InputField | Native TextField used by Network and Tailscale | Uses native dimensions, colors and input behavior. |
| ChoiceField | Native Dropdown | Adapter preserves the panel's index/text contract; real selection and editable-input tests. |
| ToggleField | Native Toggle | Real mouse/Space tests confirm state and signal behavior. |
| NumericField | Native NumberField | Editing, arrows, clamping and disabled state tested. |
| LocalizedDialog | Native BorderSurface and ActionButton footer | EN/ES Save/Cancel labels; theme font, spacing and border tokens. It remains a Qt Dialog with native contents. |
| Widget | Native BarIconButton | Connection icon and state variants; separate widget keyboard tests and actual bar inspection. |

References inspected: `/usr/share/omarchy/shell/Ui/Button.qml`,
`PanelActionButton.qml`, Network and Tailscale panel sources. Button reserves its
maximum border space, preventing size changes on hover/focus/press. The plugin
uses this behavior directly rather than reproducing it with hard-coded colors.

## Input/state checks

`scripts/test-native-controls.py` runs real Qt mouse/keyboard events using the
installed native components with an offscreen platform. ActionButton checks:

- Return, keypad Enter and Space emit clicks while focused.
- Focus ring, pointer hover and pressed fill use the native states.
- Press/release does not resize the control.
- Disabled buttons dim and ignore mouse and keyboard input.
- Destructive foreground uses the urgent theme token.
- Highlighted/checked states select the native button and expose accessibility state.

Nested editor labels are checked in both languages, including condition removal,
parameters, headers and step-order arrows. The arrows now announce Move step up /
Move step down (Subir paso / Bajar paso) with the step number.

These checks prove input/state contracts, not visual contrast in every theme or
operation through every installed panel dialog. Native screenshots and the layout
matrix are described in [interface documentation](en/interface.md). The step-label correction was installed with verified receipt hashes and preserved
configuration, grants and shell preferences. No runtime behavior or permissions changed.


## Current layout and desktop evidence

The earlier History filter elision and narrow-window toolbar overflow are fixed
and installed. History uses the native toggle description; the header adapts to
two rows and the action toolbar wraps. The sidebar scrolls within its viewport
and reveals the first/last control when focused. These corrections preserve
configuration, grants, language and shell preferences; all 176 installed receipt
hashes were checked.

The matrix covers 64 resource forms and 120 dialog layouts (15 variants across
two themes and four language/size cases). It checks dialog/footer bounds, seven
top-action buttons, filter text and sidebar focus visibility. OAuth import/browser
and file import/export/diagnostic variants are included. Dialogs are rejected,
never accepted; no OAuth consent session or file operation starts in this matrix.
The full functional flow passes with six HTTPS deliveries and revocation of
queued work after the layout changes.

Actual-shell inspection confirms Connections and History at 664×718. Tab,
Shift-Tab and Return opened History; both languages display the full toolbar and
filter. Captures: docs/images/native-history-{en,es}.png. English was restored,
the panel closed and engine state remained ready with no work. The earlier entry
form capture separately confirms native Save/Cancel and keyboard opening.

Scope: installed-shell inspection covers representative panels/forms/dialogs and
keyboard operation. The automated matrix covers all eight resource editors and
15 dialog variants in both tested themes, with 1100×800, 664×718 and 540×600
cases. This is not a claim about every third-party theme, hardware scale or a
screen-reader session.


Full PopupItem captures exposed a generic dark dialog header with insufficient
light-theme title contrast. LocalizedDialog now supplies a native-color heading
with wrapping text. Inspected light ES OAuth import, simulation, cancellation
and diagnostic dialogs, and dark EN/ES permissions/OAuth/simulation captures:
headings, fields and footers are legible. All 120 dialog/64 form layouts and
control checks pass. The correction was subsequently installed and inspected in the real shell.


The corrected heading is now installed (200 verified receipt files). Native
confirmation and simulation dialogs were opened with keyboard, inspected in
both languages and dismissed with Escape. No operation was submitted; English
and idle ready state restored. Captures native-confirm/native-simulation EN/ES
are included in the user guides.


Source-wide button inventory: 50 ActionButton uses; the only Button declaration
is Native.Button in the adapter. Form/dialog action rows share this control.
Native widget and input tests pass live states, translated status, unloading/
reloading and mouse/keyboard contracts. Actual bar cell is 27×26, matching its
neighbors. Historical/current panel and bar captures are linked in both guides.
Current monitor, schedule and queue screenshots were also visually inspected.


## Acceptance mapping

| Requirement | Evidence |
|---|---|
| Native appearance and icon | Installed Network/Tailscale source comparison; 50 native button call sites; real bar crop and before/after panel images. |
| States and keyboard | test-native-controls.py and test-widget-input.py; focus/hover/press/disabled/selected/urgent; actual Tab/Shift-Tab/Return/Escape navigation. |
| Languages, sizes, themes | 64 editor layouts, 120 dialog layouts, 60 full-dialog captures; light/dark and EN/ES inspection, complete title/filter/footer text after fixes. |
| Installed integration | Verified installer receipts, preserved preferences/grants; actual entry, Connections, History, confirmation and simulation captures. |
| Functionality after visual changes | Full QML configuration/activation flow with six HTTPS deliveries, credentials/retry/revocation; dialog-header matrix and control tests. |
| Errors and reload | test-ui-compatibility.py passes incompatible rejection, draft preservation, compatible recovery and disconnect; EN/ES error captures inspected. Widget unload/reload preserves engine. |

Additional inspected dialogs include HTTP retry, credential deletion, journal
reset, generic credential, import/export and real-event confirmation. Their text
and controls are complete in the captured view. The empty journal selection in
the test harness is fixture data, not a real pending reset.

R1.UI and F1D.7 are closed for the implemented scope and tested environments.
External Google consent, real logout/suspend acceptance, documentation/release
review and publication preparation remain independent roadmap gates.
