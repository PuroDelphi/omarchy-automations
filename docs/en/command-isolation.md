# Command isolation

[Español](../es/command-isolation.md) · [Action recipes](action-recipes.md)

`command` actions execute a registered `/usr/bin` binary with at most 32 fixed
arguments. Approval covers executable and every argument; no event substitution
or shell expansion occurs. Paths must be absolute/canonical. NUL, templates and
arguments over 4096 bytes are rejected. A registered interpreter may interpret
its own arguments within isolation: review them as local code before approval.

## Command profiles

Missing `command_profile` or `fixed` selects the advanced fixed command contract;
it does not claim semantic understanding of arbitrary installed programs.
Named profiles generate executable/argv and reject configured executables or extra arguments:

| Profile | Parameter | Generated execution | Required access |
|---|---|---|---|
| `file-exists` | `command_path` | `/usr/bin/test -f PATH` | Parent mounted ro/rw |
| `make-directory` | `command_path` | `/usr/bin/mkdir --mode=0700 -- PATH` | Parent mounted rw |

The path must be absolute/canonical and a direct child of an approved mount, with
a name up to 255 bytes. `..`, nested paths without their own parent mount, NUL and
event templates are rejected. Spaces/metacharacters remain literal in one argv
item. Events cannot add options or change executables. Changing profile, path or
mounts changes the capability that must be reviewed.

`file-exists` checks a regular file, following symlinks according to `test -f`
within the sandbox's visible filesystem. It does not pin file identity or guarantee
existence for a later step. `make-directory` creates one directory with mode 0700,
without parents; it fails without changing an existing directory. Contracts were
checked against [GNU Coreutils](https://www.gnu.org/software/coreutils/manual/coreutils.html).

Validation, simulation, preflight and execution share profile resolution.
Simulation shows generated executable/argv without running them. In the UI select
Command profile, enter Profile path and prepare the parent under Approved directories.
Review shows profile, path and mount scope. Named profiles do not make `fixed`
commands semantically verified.

## Enforced limits

Each execution uses a transient user systemd unit and Bubblewrap:

| Resource | Limit/access |
|---|---|
| CPU | 50% of one core via cgroup |
| Memory | 256 MiB; no extra swap; OOM terminates group |
| Processes/threads | At most 32 tasks, including isolation processes |
| Duration | 1–300 seconds per action |
| Files | Read-only `/usr`; private temporary `/tmp` and `/home` |
| Session | No real HOME, session sockets or D-Bus inside command |
| Network | Private namespace without host network access |
| Privileges | `NoNewPrivileges=yes` |
| Output | stdout/stderr discarded, not saved in history |

Specific trees may be added through `directories`, with `working_directory`,
subject to identity, access and action capability. The directory form and installed
UI were verified in R1.UI. Desktop effects use explicit `notify`, `service` and
`omarchy` action types. See [approved directories](../directory-access.md).

## Activation and termination

Before activating a revision containing commands, the engine runs a harmless
probe under the actual profile. It reads `memory.max`, `memory.swap.max`,
`pids.max`, `cpu.max`, requires expected values and successful completion, and
rejects activation if verification fails. It never falls back to unisolated commands.

Units use `KillMode=control-group`, one-second stop timeout and SIGKILL enabled.
On cancellation/failure, the engine additionally requests group SIGKILL before
stopping the unit, covering children ignoring SIGTERM or creating their own session.
Cancellation cannot undo effects; interrupted actions retain uncertain-outcome
semantics and are not blindly replayed.

## Evidence

`QUATRRO_HOST_TEST=1 go test -race ./...` exercises:

- Kernel controls and an increasing CPU-throttling counter.
- Bounded allocation above the limit and `oom-kill` in the temporary unit journal.
- Process creation rejection at the task limit.
- Read-only `/usr`, inaccessible private host files and absent sensitive environment.
- Blocked host-loopback connection and absent network routes.
- Timeout/cancellation of a child ignoring SIGTERM and calling `setsid`, with all PIDs gone.
- The same termination for scripts with rw directories: writing stops and PIDs disappear.
- Engine-level script cancellation: persisted uncertainty, no later writes or replay.

Tests need user systemd, Bubblewrap, Python 3 and own-journal access. They create
uniquely named temporary resources. They do not prove protection against kernel
bugs or an attacker already controlling the user account.
