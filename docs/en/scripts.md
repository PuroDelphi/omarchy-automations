# Approved local scripts

[Español](../es/scripts.md) · [Illustrated examples](code-examples.md)

F3.1 is implemented/tested in development: preparation, validation, exact-revision
actions and isolated Bash/Python execution. The panel prepares/reviews scripts,
links actions and assigns parameters. Engine and form support approved directories.
Preparing/saving executes no code and grants no permission. Stable delivery still
requires the roadmap's final checks.

Open Scripts → New, enter ID, absolute path and interpreter, then add parameters
in program order. Strings have maximum length, pattern and choices; integers have
minimum/maximum. Prepare revision copies the file and displays code/hash. Save
keeps only prepared content, not source path. Field changes invalidate preparation.
Editing an existing revision shows stored content; replacement requires entering
the local source path again.

Open Actions → New, choose `script` and its registered ID. The form shows code,
available and selected revisions. Use this revision and reset parameters explicitly
pins it and resets values. Choose Literal value or Event field for each parameter;
strings/integers use text fields and literal booleans use a checkbox. Contract
limits are shown. Event paths use `data.message`, `source` or `type`.

Save requires the selected revision still match the available one. Changing script
clears revision/values. Opening an editor never silently replaces a pinned revision.
Add the action to a flow, save and review capabilities: approval shows script ID,
hash and maximum time. Values/contracts are checked at save, event values at execution.

`scripts.prepare` accepts `id`, absolute `path`, `interpreter` (`bash`/`python3`)
and `parameters`. It reads a regular UTF-8 file up to 32 KiB without NUL and rejects
symlinks in every component. Result: id, interpreter, code, parameters, revision.
Original path is not part of the revision.

SHA256 binds ID, interpreter, code and parameter contract. Any change requires new
preparation/review. Code is configuration content and therefore exported: never
embed credentials. The global protocol limit is 1 MiB; up to eight scripts per config.

Up to sixteen required ordered parameters:

- `string`: explicit maximum 1–4096 bytes, optional RE2 pattern matching the entire
  value and optional list of up to 32 choices.
- `integer`: explicit minimum/maximum within JSON's exact integer range; no text
  or fractional conversion.
- `boolean`: JSON boolean, not strings such as `"true"`.

Extra/missing values and NUL are rejected. Conversion produces argv, not shell-code
interpolation. This does not replace script review: a script author may interpret
arguments as code themselves.

A `script` action references ID `script` and exact hash `script_revision`, with
required `timeout_seconds` from 1 to 300. Every parameter has exactly one literal
in `script_values` or event path in `script_bindings`. Allowed paths are source,
type and data fields; resolved values are revalidated. Example for string `message`:

```json
{"id":"run-local-script","kind":"script","script":"my-script","script_revision":"<hash returned by scripts.prepare>","script_bindings":{"message":"data.message"},"timeout_seconds":30}
```

Replace the placeholder with the actual hash. Add the script/action to their draft
arrays; the flow requires explicit activation grants. Code/interpreter/contract
changes require updating both action reference and permission. A new revision's
grant does not authorize pending work from the old one. Import grants nothing.

Simulation shows script, revision and resolved arguments in contract order. It uses
execution validation and rejects missing/incompatible data without starting jobs
or processes. These values can include sample event data: do not paste secrets.

Execution uses code from the configuration pinned at event admission without
reopening the original file. Code is transported privately to Bubblewrap and
mounted read-only at `/quatrro/script`; no host code file is created. Bash uses
`--noprofile --norc`, Python `-I -S`; parameters follow as separate arguments.
systemd uses `--expand-environment=no` to preserve `$` values too.

Transient units enforce the command limits: 256 MiB memory, no swap, 50% CPU,
32 tasks, timeout and group termination. The sandbox has read-only `/usr`, private
temporary directories, minimal environment and isolated networking, with no HOME
access by default. `directories` and `working_directory` add approved trees; see
[directory access](directory-access.md). Scripts cannot enable IP networking.
Activation probes isolation and code transport with every required interpreter;
unavailable guarantees cause failure.

Script failure is not automatically retried. Cancellation after dispatch or crash
before result persistence leaves uncertainty for inspection. Process output is
discarded; history stores state and bounded errors, not stdout/stderr.

Host tests execute both interpreters, modify original files after activation,
check metacharacters and deny writes to mounted code. Evidence and pending checks
remain in the [validation log](../validation.md).
