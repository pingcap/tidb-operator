# Prune (Cleanup) Job Implementation Plan (Decoupled Approach)

This document outlines the plan to implement a decoupled cleanup (prune) job mechanism that triggers automatically after a restore job fails.

## 1. Guiding Principles

- **Decoupling**: The logic for handling a normal restore and a prune/cleanup action should be as separate as possible to enhance clarity and maintainability.
- **Clarity**: The state of the `Restore` custom resource should be unambiguous at all times. The purpose of each created Kubernetes Job should be easily identifiable.
- **Robustness**: The process must be resilient to controller restarts and other transient failures.

## 2. High-Level Design

The core idea is to treat the prune action as a distinct phase in the `Restore` lifecycle, managed by its own logic branch within the `restoreManager`.

1.  **State Machine**: We will introduce four new states to the `Restore` CRD to manage the prune lifecycle:
    - `RestorePruneScheduled`: The initial state after a restore fails, indicating a prune job is ready to be created.
    - `RestorePruneRunning`: The prune job has been created and is currently running.
    - `RestorePruneComplete`: The prune job has successfully completed.
    - `RestorePruneFailed`: The prune job failed to complete.

2.  **Controller Logic (`restore-controller`)**: The controller's event handler (`updateRestore`) will act as a gatekeeper. When it observes a `Restore` transitioning to the `Failed` state, it will check if `spec.prune: AfterFailed` is set. If so, it will intercept the `Failed` state, replace it with `RestorePruneScheduled`, and re-queue the object for processing.

3.  **Reconciliation Logic (`restoreManager`)**: The main `Sync` function will have a new top-level conditional branch.
    - If the `Restore` object is in the `PruneScheduled` (or `PruneRunning`) state, execution will be routed to a new, dedicated function, e.g., `syncPruneJob()`.
    - Otherwise, execution will proceed with the existing restore logic.

4.  **Prune Job**: The `syncPruneJob()` function will manage a separate Kubernetes `Job`.
    - This job can have a distinct name (e.g., `[restore-name]-prune`) to differentiate it from the original restore job.
    - The job's pod will run the `backup-manager` with an `--abort=true` argument.

5.  **Backup Manager**: The `backup-manager` binary, upon detecting the `--abort=true` flag, will execute the `br abort restore` command to perform the actual cleanup of external storage.

## 3. Implementation Steps

### Step 1: Verify CRD and State Helpers

- **File**: `pkg/apis/pingcap/v1alpha1/types.go`
- **Action**: Ensure the following `RestoreConditionType` constants are defined:
  ```go
  RestorePruneScheduled RestoreConditionType = "PruneScheduled"
  RestorePruneRunning   RestoreConditionType = "PruneRunning"
  RestorePruneComplete  RestoreConditionType = "PruneComplete"
  RestorePruneFailed    RestoreConditionType = "PruneFailed"
  ```
- **File**: `pkg/apis/pingcap/v1alpha1/restore.go`
- **Action**: Ensure helper functions like `IsRestorePruneScheduled(restore)` are implemented correctly.

### Step 2: Implement Controller State Transition Logic

- **File**: `pkg/controller/restore/restore_controller.go`
- **Function**: `updateRestore()`
- **Action**:
  - Locate the logic block that handles a failed restore job (`if jobFailed { ... }`).
  - Inside this block, if `newRestore.Spec.Prune == v1alpha1.PruneTypeAfterFailed`, instead of setting the `RestoreFailed` condition, set the `RestorePruneScheduled` condition.
  - Ensure the object is re-queued after the status update.
  - Add a check at the beginning of `updateRestore` to ignore (return early) objects already in `PruneRunning`, `PruneComplete`, or `PruneFailed` states to prevent redundant processing.

### Step 3: Implement Decoupled Logic in `restoreManager`

- **File**: `pkg/backup/restore/restore_manager.go`
- **Function**: `Sync(restore *v1alpha1.Restore)`
- **Action**:
  - At the beginning of the function, add a check:
    ```go
    if v1alpha1.IsRestorePruneScheduled(restore) || v1alpha1.IsRestorePruneRunning(restore) {
        return rm.syncPruneJob(restore)
    }
    ```
  - This will route all prune-related processing to a new, dedicated function.

### Step 4: Create the Prune Job Management Function

- **File**: `pkg/backup/restore/restore_manager.go`
- **Action**: Create a new function `syncPruneJob(restore *v1alpha1.Restore) error`.
- **Logic**: This function will mirror the structure of the existing restore job logic but will be simpler:
  1.  Define the prune job name, e.g., `pruneJobName := restore.GetRestoreJobName() + "-prune"`.
  2.  Check if the prune job already exists using this name.
  3.  If the job exists, check its status (`Succeeded`, `Failed`) and update the `Restore` CR's status to `PruneComplete` or `PruneFailed` accordingly.
  4.  If the job does not exist:
      - Call a new helper function `makePruneJob(restore)` to construct the `batchv1.Job` object.
      - Create the job using the Kubernetes client.
      - **Do not** set the `Restore` status to `Scheduled`. The `backup-manager` will be responsible for setting the `PruneRunning` status.

### Step 5: Create the Prune Job Factory Function

- **File**: `pkg/backup/restore/restore_manager.go`
- **Action**: Create a new function `makePruneJob(restore *v1alpha1.Restore) (*batchv1.Job, error)`.
- **Logic**:
  - This function can reuse much of the code from `makeRestoreJob` for setting up volumes, secrets, and pod templates. Consider refactoring shared logic into common helpers.
  - The key difference will be in the container's command/arguments.
  - The arguments must include `--abort=true`.
  - The job name in the metadata should be the new, prune-specific name.

### Step 6: Verify `backup-manager` Behavior

- **File**: `cmd/backup-manager/app/restore/restore.go`
- **Action**: Review the argument parsing logic. Confirm that if `--abort=true` is present, the manager constructs and executes a `br abort restore ...` command, and correctly updates the `Restore` CR status to `PruneRunning`, `PruneComplete`, or `PruneFailed`.
