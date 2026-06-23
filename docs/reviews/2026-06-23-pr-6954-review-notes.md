# PR 6954 Review Notes

Review target: `pingcap/tidb-operator#6954`

Branch: `br-lock-operation-observability`

Base: `public/release-1.x` at `436f05e21794164676ab47c5475451ee37abb9db`

Review date: 2026-06-23

## Purpose

This document records review findings for the BR lock operation observability
changes so we can decide, item by item, whether to fix them in this PR.

The review intentionally considers both implementation correctness and whether
the design choices are appropriate.

## Findings

### 1. `lockBlocker` can report a transient lock as the final failure cause

Status: fixed by removing `lockBlocker` from the first-phase API

Severity: high

Relevant code:

- `cmd/backup-manager/app/restore/restore.go`
- `cmd/backup-manager/app/backup/backup.go`
- `cmd/backup-manager/app/brlog/parser.go`

Current behavior:

- The observer stores the latest lock blocker candidate whenever it sees a BR
  lock-conflict-style log event.
- If the BR process later exits with a non-zero status, backup-manager writes
  the stored candidate to `status.lockBlocker`.

Concern:

The parser accepts retry-style logs such as `Encountered lock, will retry`.
Those logs can represent a transient conflict that BR later recovers from. If
BR subsequently fails for an unrelated reason, `status.lockBlocker` will still
point to the old candidate and can mislead operators into treating that remote
operation as the cause of the failure.

Design mismatch:

The design document says `lockBlocker` should be written only when the BR
execution finally fails due to lock conflict. The implementation currently
uses "observed a blocker at some point" plus "BR failed" as the condition.

Resolution:

This PR now keeps only the owning CR's observed `brOperations` in status. It no
longer parses lock-conflict logs into a status event, no longer stores a
candidate blocker during BR execution, and no longer exposes
`status.lockBlocker` on `Backup` or `Restore`.

This avoids freezing a log-derived causal diagnosis into the API before BR has a
stable terminal lock-conflict signal or machine-readable report file.

### 2. New BR command runners drop ordinary stderr from returned errors

Status: fixed

Severity: medium-high

Relevant code:

- `cmd/backup-manager/app/backup/backup.go`
- `cmd/backup-manager/app/restore/restore.go`
- `cmd/backup-manager/app/brlog/parser.go`

Current behavior:

- The new stdout/stderr streaming path includes only lines recognized by
  `brlog.IsErrorLine` in the returned `errMsg`.
- Ordinary stderr lines are logged, but they are omitted from the error
  returned to the caller.

Concern:

The previous restore path and backup helper behavior preserved stderr in the
returned error message. Some real BR or runtime failures may write useful
diagnostic text to stderr without using `[ERROR]`, `[FATAL]`, `[PANIC]`, or JSON
`level=error/fatal`. Those failures would now become harder to diagnose from
the CR condition or backup-manager error path.

Resolution:

The streaming runners now preserve the old stderr behavior: every stderr line is
included in the returned `errMsg`, while stdout remains filtered to BR
error/fatal/panic lines. The observer still receives both streams in real time,
so operation parsing is unaffected.

### 3. `BROperation.Command` needs a stronger safety contract

Status: undecided

Severity: medium

Relevant code:

- `pkg/apis/pingcap/v1alpha1/types.go`
- `cmd/backup-manager/app/brlog/parser.go`

Current behavior:

- The CR status exposes `BROperation.Command` directly from the BR log field
  named `command`.

Concern:

This is safe if BR guarantees the field is only a short subcommand such as
`log-restore` or `restore full`. It is risky if BR can emit a full argv or a
string derived from user-provided `spec.br.options`, because status fields are
more visible than pod logs and could expose storage query parameters or other
operator-provided arguments.

Possible directions:

- Confirm and document that BR's `command` field is a safe subcommand label.
- Rename or narrow the status field to make the intended contract explicit.
- Sanitize or omit the field until the upstream BR contract is explicit.

### 4. API reference docs do not include the new status fields

Status: fixed for `brOperations`

Severity: low

Relevant files:

- `docs/api-references/docs.md`
- `pkg/apis/pingcap/v1alpha1/types.go`
- `manifests/crd/v1/pingcap.com_backups.yaml`
- `manifests/crd/v1/pingcap.com_restores.yaml`

Current behavior:

- CRD manifests include `status.brOperations`.
- `docs/api-references/docs.md` now includes `brOperations`.

Concern:

Runtime behavior is not affected, but the published API reference will be
incomplete.

Resolution:

`./hack/update-api-references.sh` was run after simplifying the status API, so
the generated API reference now documents `brOperations` and does not introduce
`lockBlocker`.

### 5. `docs/superpowers/plans/...` appears to be an agent execution artifact

Status: undecided

Severity: low

Relevant file:

- `docs/superpowers/plans/2026-06-17-br-lock-operation-observability.md`

Current behavior:

- The PR adds a document under `docs/superpowers/plans/`.
- The document starts with instructions for "agentic workers" and references
  local execution skills.

Concern:

The base branch does not contain a `docs/superpowers` directory, and this file
looks like a personal implementation plan rather than a user-facing or
maintainer-facing project document. It may not belong in the repository.

Possible directions:

- Remove it from the PR.
- Move useful durable content into the design proposal.
- Keep it only if maintainers intentionally want agent execution plans checked
  into the repository.

## Positive Notes

- The high-level design is restrained: BR remains the authority for operation
  IDs, backup-manager observes near the child process, and status stores bounded
  recent operation history.
- Parser tests cover both default BR text logs and JSON log compatibility.
- Status updater logic caps `brOperations` at 10 and deduplicates by operation
  ID.
- Observability status update failures are logged and do not replace the main
  BR result.

## Verification Run

Commands that passed:

```sh
env GOCACHE=/tmp/go-build go test -count=1 ./cmd/backup-manager/app/brlog ./cmd/backup-manager/app/backup ./cmd/backup-manager/app/restore ./cmd/backup-manager/app/util
env GOCACHE=/tmp/go-build go test -count=1 ./pkg/controller -run 'TestUpdateBackupStatus|TestUpdateRestoreStatus|TestUpdateBROperations'
git diff --check public/release-1.x...HEAD
./hack/verify-crd.sh
./hack/verify-openapi-spec.sh
./hack/verify-api-references.sh
```

Notes:

- `./pkg/controller` full package test was not usable in the restricted sandbox
  because an existing `httptest` case could not listen on a local socket.
- `verify-openapi-spec.sh` printed existing API rule warnings but exited
  successfully and reported no generated OpenAPI diff.
