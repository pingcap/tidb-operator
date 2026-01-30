# TiCI support in TidbCluster (tc) – plan

This plan maps the local `tici/run-tests.sh` flow into tidb-operator CRD sync and highlights the files/areas to change.

## Stage 1 – API/CRD modeling (spec/status + defaults + validation)
- Add `TiCI` component to the `TidbClusterSpec` and `TidbClusterStatus`.
  - Files: `pkg/apis/pingcap/v1alpha1/types.go`, `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`, `pkg/apis/pingcap/v1alpha1/openapi_generated.go`.
- Define a `TiCISpec` with **meta** and **worker** sub-specs (replicas, resources, storage, image, env/args), plus **S3/MinIO settings** and **TiFlash integration settings**.
- Define `TiCIStatus` (statefulset status for meta/worker, conditions, synced flag), similar to TiCDC/Pump patterns.
- Add new `MemberType`(s) and label values:
  - `tici-meta`, `tici-worker` (or a unified `tici` if you decide single workload) in `pkg/apis/pingcap/v1alpha1/types.go` and `pkg/apis/label/label.go`.
  - Delete-slots annotations in `pkg/apis/label/label.go` if you use StatefulSet.
- Add defaulting in `pkg/apis/pingcap/v1alpha1/defaulting/tidbcluster.go` (image defaults, replicas=1, ports, config update strategy).
- Add validation in `pkg/apis/pingcap/v1alpha1/validation/validation.go` (replicas >=0, required S3 fields if enabled, incompatible flags).
- Add helper getters in `pkg/apis/pingcap/v1alpha1/tidbcluster.go` and `pkg/apis/pingcap/v1alpha1/component_spec.go` (e.g., `BaseTiCISpec()`, `TiCIImage()`, `TiCIVersion()`).

## Stage 2 – Controller wiring and member manager
- Add a TiCI member manager that reconciles **meta** and **worker** workloads.
  - New file: `pkg/manager/member/tici_member_manager.go`.
  - Reuse patterns from `ticdc_member_manager.go` / `pump_member_manager.go`.
- Create **headless services** and **statefulsets** (or deployments) for meta/worker.
  - Add naming helpers in `pkg/controller/controller_utils.go` (e.g., `TiCIMetaMemberName`, `TiCIWorkerMemberName`, peer svc names).
- Generate ConfigMaps for meta/worker config.
  - Decide between (a) templating from `tici/tici/config/*.toml.in` or (b) building config via typed wrapper similar to TiFlash.
  - For templating, render values derived from TC: PD address, TiDB service/port, S3 endpoint/credentials.
- Wire the manager into the control loop:
  - `pkg/controller/dependences.go` (dependency injection for the new manager)
  - `pkg/controller/tidbcluster/tidb_cluster_control.go` (call `ticiMemberManager.Sync(tc)` in the right order; after PD/TiKV/TiDB, before TiCDC/TiFlash if they depend).
- Update selectors/util helpers for pod ordinals and delete-slots in `pkg/manager/member/utils.go` if using StatefulSet.

## Stage 3 – TiFlash integration (tici config + reader port)
- When TiCI is enabled, ensure TiFlash config includes the **[tici]** sections from `tici/tici/config/tiflash*.toml.in`.
  - Extend `GetTiFlashConfig()` in `pkg/manager/member/tiflash_util.go` to merge tici config **only if missing**, to avoid overriding user config.
- Add the **tici reader port** to TiFlash container ports and services if required.
  - Update constants and service port lists in `pkg/manager/member/tiflash_member_manager.go` and `pkg/apis/pingcap/v1alpha1/types.go` (defaults).

## Stage 4 – TiCDC integration (newarch + changefeed)
- If TiCI is enabled, set TiCDC `--newarch=true` (or config equivalent).
  - Either add a `TiCDCSpec.NewArch` field or allow `AdditionalArgs` in `TiCDCSpec` and inject into start script (`pkg/manager/member/startscript/v2/ticdc_start_script.go`).
- Create the **TiCDC changefeed** for TiCI’s S3 sink.
  - This is *not* in operator today. Options:
    1) Implement a lightweight controller/job that runs `ticdc cli changefeed create` once per TC (idempotent check).
    2) Add/consume a ChangeFeed CRD if available in your environment.
  - Place this logic after TiCDC is ready; use a status condition to avoid repeated creation.

## Stage 5 – Tests and examples
- Unit tests for TiCI manager (creation/update, configmap, status sync).
  - Pattern: `pkg/manager/member/ticdc_member_manager_test.go`.
- Update CRD examples under `examples/` with TiCI enabled.
- Add docs describing S3/MinIO fields, required TiFlash settings, and TiCDC newarch/changefeed behavior.

## Mapping from `tici/run-tests.sh` to operator behavior
- **Start order** in script: PD → TiKV → TiDB → (MinIO) → TiCI meta/worker → TiFlash → TiCDC (newarch) → changefeed.
- In operator sync:
  - Gate TiCI until PD/TiKV/TiDB are ready.
  - Gate TiCDC changefeed creation until TiCDC is ready.
  - Gate TiFlash tici config until TiCI is enabled (don’t override user config).

## Open design decisions to settle before coding
- TiCI workloads: **StatefulSet vs Deployment**? (StatefulSet likely for stable IDs).
- TiCI config rendering: **template files** vs **typed config wrapper**.
- Changefeed creation mechanism: **new controller/job** vs **ChangeFeed CRD**.
- S3 secrets: use `SecretRef` vs inline spec fields.
