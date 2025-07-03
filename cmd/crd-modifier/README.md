# `crd-modifier`

## What is `crd-modifier`?

`crd-modifier` is a command-line tool designed for post-processing Custom Resource Definition (CRD) files for TiDB Operator. Its primary function is to remove certain `required` constraints within the `overlay` field's schema in the CRDs. This provides greater flexibility when customizing Pod templates, helping to avoid unnecessary validation errors.

## How it Works

The tool scans a specified directory (defaulting to `manifests/crd`) for all CRD files with `.yaml` extension. For each CRD file, it inspects its OpenAPI v3 schema to determine the CRD type and applies modifications accordingly:

- **Group-type CRDs**: If an `overlay` field is detected at the path `spec.template.spec.overlay`, the tool identifies it as a group-type CRD (e.g., `PDGroup`, `TiKVGroup`). It then proceeds to remove the `required` constraint from the `pod` sub-field within that `overlay`.

- **Instance-type CRDs**: If an `overlay` field is detected at the path `spec.overlay.pod`, the tool identifies it as an instance-type CRD (e.g., `PD`, `TiKV`). Similarly, it removes the `required` constraint from the `pod` sub-field within that `overlay`.

- **Special CRDs**: For CRDs with nested overlay structures (like `TiBRGC`), where overlays are located at `spec.overlay.pods[].overlay`, the tool automatically detects and processes these special cases.

This schema-based detection mechanism ensures robust forward compatibility. As long as new CRDs follow the established overlay structure patterns, they can be processed automatically without requiring code changes. The tool gracefully handles CRDs that don't have overlay fields by skipping them.

## How to Use

You can run the tool by executing the `main.go` file with `go run`. Use the `--dir` flag to specify the directory containing the CRDs.

```bash
go run cmd/crd-modifier/main.go --dir <your-crd-manifests-path>
```

If the `--dir` flag is omitted, it will default to the `manifests/crd` directory in the project root.

## Output

The tool provides detailed logging to show which CRDs are being processed and what type of overlay structure was detected. This helps with debugging and understanding what modifications were applied.
