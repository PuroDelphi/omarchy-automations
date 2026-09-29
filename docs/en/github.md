# First publication from this machine

[Español](../es/github.md) · [User guide](user-guide.md)

The local project is `/home/macondo/Work/quatrro-automations`. Its `origin` points
to `https://github.com/PuroDelphi/omarchy-automations.git`, and its initial branch
is `main`. At this audit there are no commits or configured local author identity.
No files have been published. Complete the roadmap's release checks before
publishing a stable release; the current package is a development version.

## Sign in

GitHub CLI is prepared outside the project. Run these commands in your terminal:

```bash
/home/macondo/Work/.quatrro-tools/github-cli/gh auth login --hostname github.com --git-protocol https --web
/home/macondo/Work/.quatrro-tools/github-cli/gh auth setup-git --hostname github.com
/home/macondo/Work/.quatrro-tools/github-cli/gh auth status --hostname github.com
```

Follow the browser/device-code instructions with the account that can write to
the repository. Enter credentials only in the provider's login flow, not chat.
Signing in does not upload the project. If `gh` is later installed in PATH, its
short name can replace the full executable path.

## Review the destination and files

```bash
cd /home/macondo/Work/quatrro-automations
/home/macondo/Work/.quatrro-tools/github-cli/gh repo view PuroDelphi/omarchy-automations
git remote -v
git ls-remote origin
git status --short
git diff --cached --stat
git diff --stat
```

An empty successful `git ls-remote` result means no advertised refs; an error
does not mean an empty repository. Review both staged and unstaged files, plus
untracked files. `.gitignore` excludes builds, distributions, temporary profiles,
databases and secrets directories, but cannot detect secrets pasted into ordinary
source or examples. The public documentation includes this workspace path;
replace it if you prefer machine-neutral publication instructions.

The owner selected [MIT](../../LICENSE). Include LICENSE and the dependency
notices when publishing.

## Create the first local commit

Replace these example author values with your chosen identity. You can use the
no-reply address shown in your GitHub email settings. `--local` affects only this
repository; it does not change other projects.

```bash
git config --local user.name "Your chosen author name"
git config --local user.email "YOUR_CHOSEN_EMAIL"
git add --all
git diff --cached --check
git diff --cached --stat
git commit -m "Add Omarchy Automations"
```

Review the staged content before the commit. `git add --all` includes all eligible
files in the project, so do not run it while unrelated/private files are present.

## Upload and verify

If the remote is empty, publish the reviewed commit:

```bash
git push -u origin main
git rev-parse HEAD
git ls-remote origin refs/heads/main
```

The local and remote commit IDs must match. This publishes source, examples and
documentation; ignored `dist/` archives are not uploaded by a source push.

If the remote already has history, first inspect its default branch and fetch it:

```bash
/home/macondo/Work/.quatrro-tools/github-cli/gh repo view PuroDelphi/omarchy-automations --json defaultBranchRef
git fetch origin
git log --oneline --graph --all -20
```

Integrate the existing branch before pushing. For a remote `main` created with
an independent README/license, `git merge --allow-unrelated-histories origin/main`
combines the two histories. Review and resolve any conflicts, rerun the relevant
checks and commit the merge before pushing. Substitute the actual remote branch
if it differs. Use `git merge --abort` if you need to reconsider an in-progress
merge. Do not force-push over existing history.

Authentication failure requires login/access correction; a rejected push can
also mean branch protection or new remote commits. Inspect the reason rather
than bypassing it. Binary release uploads and a stable version tag are separate
publication steps after the release gates pass.

## Prepared local audit

The reviewed candidate set contains source, tests, examples, packaging and paired
guides. Generated builds/distributions and local profiles stay ignored. Environment
variants (`.env.*`, except a deliberate `.env.example`) and SQLite profile files
are also excluded. Tracked files remain tracked even if an ignore rule is added:
review `git ls-files --cached --others --exclude-standard` before committing.

A local high-signal scan found no private-key blocks or GitHub/Google/AWS/Slack
token patterns in 314 candidate files at the audit. This does not prove arbitrary
secrets are absent; inspect any later edits and ordinary configuration text.
Published authentication vectors intentionally use a public fixture value.
There is no first commit yet, and your author identity is still to be chosen.
The project license is MIT. The staged snapshot contains older versions of some files: after review,
`git add --all` is necessary to stage the current completed work before committing.
