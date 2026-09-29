# Approved directories

[Español](../es/directory-access.md) · [Action recipes](action-recipes.md)

The engine supports identity/descriptor-based mounts for `command` and `script`
actions, with capabilities, preflight and forms. This is available in the verified
development installation (R1.1); executor checks F1C.3 are complete in development.

In a command/script action, Approved directories accepts a host path, `/work/id`
target and access mode, defaulting to Read only. Prepare and add directory checks
and adds identity to the draft; a repeated target replaces that entry. Invalid
paths report errors without removing previous entries. Remove deletes the mount
and resets working directory to `/tmp` if it used that target. Working directory
offers `/tmp` and prepared targets.

Saving preserves mounts, including edits. Save is blocked while preparation is
pending. Capability review shows source, target, identity, access, working directory
and tree scope. Adding a draft mount grants nothing. Text is available in both languages.

`directories.prepare` accepts source, target, access and returns the object below.
It does not save configuration, grant permission or create directories. Add prepared
objects to the action's `directories` and optionally set `working_directory`.
Save/activation still require capability review. Simulation shows intended mounts
and working directory.

The capability hash includes every field; changing access, identity, source, target
or working directory requires a new grant. Pending jobs with another hash are denied.
Preflight checks identity and runs a harmless mount probe inside isolation; scripts
also probe their interpreter and memfd code transport. Execution reopens and verifies
identity in the helper before descriptor mounting.

Each directory permission contains:

- `source`: canonical absolute host path, without symlinks.
- `target`: `/work/identifier` mount point inside the sandbox.
- `access`: required `ro` or `rw`.
- `device`, `inode`: identity observed at preparation, represented as decimal
  strings to preserve precision through JSON/QML.

Up to eight distinct targets are allowed. Working directory is `/tmp` or an approved
target. Host root and interfaces under `/proc`, `/sys`, `/dev`, `/run` are forbidden.
Custom targets cannot replace `/usr`, `/quatrro/script` or other sandbox points.

Opening uses `openat2` with `O_PATH | O_DIRECTORY` and `RESOLVE_NO_SYMLINKS`.
`fstat` checks that same descriptor, passed to Bubblewrap using `--ro-bind-fd` or
`--bind-fd`. The path is not reopened between validation and mounting. See
[openat2](https://man7.org/linux/man-pages/man2/openat2.2.html) and
[Bubblewrap source](https://github.com/containers/bubblewrap/blob/main/bubblewrap.c).

A path now pointing to another inode/device is rejected until prepared/approved
again. If it changes after opening, the mount still refers to the opened object.
This does not freeze contents: files remain live, and `rw` permits changes/deletions
under normal user permissions. It is neither a backup nor cryptographic identity
against account/filesystem control. Restore/filesystem changes can require new preparation.

Permission covers the mounted tree, including submounts and special resources.
A Unix socket inside it can communicate with a host service: IP-network isolation
does not turn the tree into ordinary files only. UI/review disclose this scope
before granting access. Profiles without mounts retain their previous isolation.

## Internal transport

`quatrrod --sandbox-runner` is a private executable branch receiving one JSON
document up to 1 MiB on stdin and launching Bubblewrap. It opens no engine profile,
socket or database. It is neither an authorization API nor privileged service:
the engine must check capabilities before invoking it within a limited transient unit.

Internal protocol 1 contains executable, arguments, optional code, mounts and cwd.
It rejects unknown fields, incompatible versions, trailing data and excessive limits.
Script code is copied to a sealed memfd and mounted read-only; no host code file
is created. Descriptors close when the helper exits; Bubblewrap consumes them
before executing the program.

## Evidence

Host tests in `internal/sandbox` change the path after command preparation but
before Bubblewrap starts. They verify reading the original tree, ro write rejection
and rw writes only in that tree. A second execution rejects the substituted path's
old identity. Invalid paths/targets, symlinks and transport bounds are also tested.

`scripts/test-sandbox-runner.py` runs the compiled binary in a limited transient
user unit: Python receives literal arguments, reads/writes an approved temporary
directory and cannot modify its code. `scripts/test-ui-flow.py` prepares a mount
from the form, preserves it through editing, reviews its capability and executes
a script writing there before completing HTTPS delivery.
