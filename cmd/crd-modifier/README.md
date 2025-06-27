## What is `crd-modifier`?

`crd-modifier` is a command-line tool designed for post-processing Custom Resource Definition (CRD) files for TiDB Operator. Its primary function is to remove certain `required` constraints within the `overlay` field's schema in the CRDs. This provides greater flexibility when customizing Pod templates, helping to avoid unnecessary validation errors.

## How it Works

The tool scans a specified directory (defaulting to `manifests/crd`) for all CRD files that match the naming convention `core.pingcap.com_*.yaml`. For each matching file, it inspects its OpenAPI v3 schema to determine the CRD type and applies modifications accordingly:

- **Group-type CRDs**: If an `overlay` field is detected at the path `spec.template.spec.overlay`, the tool identifies it as a group-type CRD (e.g., `PDGroup`, `TiKVGroup`). It then proceeds to remove the `required` constraint from the `pod` sub-field within that `overlay`.

- **Instance-type CRDs**: If an `overlay` field is detected at the path `spec.overlay`, the tool identifies it as an instance-type CRD (e.g., `PD`, `TiKV`). Similarly, it removes the `required` constraint from the `pod` sub-field within that `overlay`.

This schema-based detection mechanism ensures robust forward compatibility. As long as new CRDs follow the established structure, they can be processed automatically without requiring code changes.
