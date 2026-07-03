# TiKVGroup Keyspace Placement

## Summary

Add two APIs:

- `TiKVGroup.spec.template.spec.placement.exclusive`: controls whether stores in this TiKVGroup use exclusive `$`-prefixed labels.
- `PlacementPolicy`: a standalone CRD that selects keyspaces and references target groups through `spec.groupRefs`.

Only `selector.type: Keyspace` is implemented now. Database and table selectors are deferred because their key ranges depend on unstable table / partition physical IDs and DDL state.

## Goals

- Place selected keyspaces on selected `TiKVGroup`s.
- Keep store-level exclusiveness on `TiKVGroup`.
- Keep data placement rules in a reusable standalone CRD.
- Hide raw `start_key` / `end_key` from users.
- Follow the PD rule bundle model used by `pd-ctl keyspace set-placement`.

## Non-Goals

- Database or table selectors in the first version.
- User-provided raw key ranges.
- Leader / learner placement in the first version. The API keeps `role` but initially validates `voter`.

## API

### TiKVGroup

Add `placement` to `TiKVTemplateSpec`:

```go
type TiKVTemplateSpec struct {
    // Placement controls store-level placement behavior.
    // +optional
    Placement *TiKVStorePlacement `json:"placement,omitempty"`
}

type TiKVStorePlacement struct {
    // Exclusive marks stores in this TiKVGroup with an exclusive label.
    // Stores with exclusive labels are only matched by placement rules that
    // explicitly include that label. Default is false.
    // +optional
    Exclusive *bool `json:"exclusive,omitempty"`
}
```

Example:

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: TiKVGroup
metadata:
  name: hot-kv
spec:
  cluster:
    name: basic
  template:
    spec:
      placement:
        exclusive: true
```

### PlacementPolicy

Add a standalone CRD:

```go
type PlacementPolicySpec struct {
    Cluster ClusterReference `json:"cluster"`

    // GroupRefs references target groups.
    // +listType=map
    // +listMapKey=group
    // +listMapKey=kind
    // +listMapKey=name
    GroupRefs []PlacementPolicyGroupRef `json:"groupRefs"`

    // Rules are applied to every groupRef in this policy.
    // +listType=map
    // +listMapKey=name
    Rules []PlacementPolicyRule `json:"rules"`
}

type PlacementPolicyGroupRef struct {
    // Group is the API group of the referenced target.
    // First version only supports "core.pingcap.com".
    Group string `json:"group"`

    // Kind is the kind of the referenced target.
    // First version only supports "TiKVGroup".
    Kind string `json:"kind"`

    // Name is the referenced object name in the same namespace and cluster.
    Name string `json:"name"`
}

type PlacementPolicyRule struct {
    // Name is a stable identifier for this rule.
    Name string `json:"name"`

    // Role is the peer role. First version only supports "voter".
    Role string `json:"role"`

    // Count is the peer count for the groupRef set in spec.groupRefs.
    Count int32 `json:"count"`

    Selector PlacementSelector `json:"selector"`
}

type PlacementSelector struct {
    // Type is the logical selector type. First version only supports "Keyspace".
    Type string `json:"type"`

    // Keyspaces are PD keyspace IDs.
    // +listType=set
    Keyspaces []uint32 `json:"keyspaces,omitempty"`
}
```

All rules apply to all groupRefs. `count` is not multiplied by the number of groupRefs; it is the peer count across the matched target set.

Example:

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: PlacementPolicy
metadata:
  name: hot-keyspaces
spec:
  cluster:
    name: basic
  groupRefs:
  - group: core.pingcap.com
    kind: TiKVGroup
    name: hot-kv
  rules:
  - name: hot-voters
    role: voter
    count: 3
    selector:
      type: Keyspace
      keyspaces:
      - 1
      - 2
```

Placing replicas across a TiKVGroup set:

```yaml
spec:
  cluster:
    name: basic
  groupRefs:
  - group: core.pingcap.com
    kind: TiKVGroup
    name: hot-kv-a
  - group: core.pingcap.com
    kind: TiKVGroup
    name: hot-kv-b
  - group: core.pingcap.com
    kind: TiKVGroup
    name: hot-kv-c
  rules:
  - name: hot-voters
    role: voter
    count: 3
    selector:
      type: Keyspace
      keyspaces: [1]
```

## Controller

1. TiKVGroup controller adds one operator-managed PD store label to every TiKV store in the group:
   - non-exclusive key: `op/tikvgroup`
   - exclusive key: `$op/tikvgroup`
   - value: `<namespace>.<cluster>.<tikvgroup>`
   - when `exclusive` changes, remove the old key from every store and enqueue PlacementPolicies that reference this group.
2. PlacementPolicy controller resolves `spec.groupRefs[*]` to TiKVGroups in the same namespace and cluster.
3. It applies every `spec.rules[*]` to the resolved target set and buckets the generated entries by keyspace ID and store label key.
4. For each keyspace ID, generate the keyspace range with PD's `keyspace.MakeKeyRanges(id, "txn")`.
5. Build one PD placement rule bundle per keyspace ID.
6. Write bundles with `SetPlacementRuleBundles(..., partial=true)`.
7. When a policy, groupRef, or rule is removed, rebuild affected keyspace bundles. Delete a bundle only when no PlacementPolicy still owns that keyspace.

`PlacementPolicy.status.rules[*]` records selected keyspace IDs, generated PD group/rule IDs, and sync conditions.

## Placement Rule Details

The implementation mirrors `pd-ctl keyspace set-placement`:

- PD API: `POST /pd/api/v1/config/placement-rule?partial=true`
- Bundle ID: `keyspace-<keyspace-id>`
- Bundle index: `100`
- Bundle override: `true`
- Range: `keyspace.MakeKeyRanges(id, "txn")`
- Rule role/count: copied from `PlacementPolicy.spec.rules[*].role/count`
- Rule ID: derived from `<policy-name>-<rule-name>-<label-key-suffix>`
- Rule label key: based on referenced groups' `spec.template.spec.placement.exclusive`

If referenced groupRefs resolve to both exclusive and non-exclusive targets, generate one PD rule per label key:

- `op/tikvgroup`: all non-exclusive targets
- `$op/tikvgroup`: all exclusive targets

When a policy rule is split by label key, each generated PD rule keeps the configured `count`. The operator does not divide or scale `count`.

For keyspace `42` and three exclusive TiKVGroups:

```json
[
  {
    "group_id": "keyspace-42",
    "group_index": 100,
    "group_override": true,
    "rules": [
      {
        "group_id": "keyspace-42",
        "id": "hot-keyspaces-hot-voters-exclusive",
        "role": "voter",
        "count": 3,
        "start_key": "<hex txn start>",
        "end_key": "<hex txn end>",
        "label_constraints": [
          {
            "key": "$op/tikvgroup",
            "op": "in",
            "values": ["ns.basic.hot-kv-a", "ns.basic.hot-kv-b", "ns.basic.hot-kv-c"]
          }
        ]
      }
    ]
  }
]
```

If the policy references both exclusive and non-exclusive targets, split the same policy rule by label key:

```json
[
  {
    "group_id": "keyspace-42",
    "group_index": 100,
    "group_override": true,
    "rules": [
      {
        "group_id": "keyspace-42",
        "id": "hot-keyspaces-hot-voters-normal",
        "role": "voter",
        "count": 3,
        "start_key": "<hex txn start>",
        "end_key": "<hex txn end>",
        "label_constraints": [
          {
            "key": "op/tikvgroup",
            "op": "in",
            "values": ["ns.basic.warm-kv"]
          }
        ]
      },
      {
        "group_id": "keyspace-42",
        "id": "hot-keyspaces-hot-voters-exclusive",
        "role": "voter",
        "count": 3,
        "start_key": "<hex txn start>",
        "end_key": "<hex txn end>",
        "label_constraints": [
          {
            "key": "$op/tikvgroup",
            "op": "in",
            "values": ["ns.basic.hot-kv-a", "ns.basic.hot-kv-b"]
          }
        ]
      }
    ]
  }
]
```

Delete:

```http
DELETE /pd/api/v1/config/placement-rule/keyspace-42
```

Only delete the bundle when no PlacementPolicy still selects that keyspace.

## Exclusive Store Labels

PD treats store labels whose key starts with `$` as exclusive labels. A store with an exclusive label can only be selected when the placement rule explicitly contains a constraint for that label key.

TiDB Operator uses this behavior at TiKVGroup level:

| TiKVGroup mode | Store label | Generated rule constraint | Effect |
| --- | --- | --- | --- |
| non-exclusive | `op/tikvgroup=ns.basic.hot-kv` | `op/tikvgroup in [ns.basic.hot-kv]` | Selected keyspaces are placed on this TiKVGroup, but other key ranges may also be scheduled here by default PD rules. |
| exclusive | `$op/tikvgroup=ns.basic.hot-kv` | `$op/tikvgroup in [ns.basic.hot-kv]` | Only explicitly selected keyspaces can be scheduled here; default PD rules without this constraint will not match these stores. |

`exclusive` does not live in PlacementPolicy. Policies only reference groupRefs; the referenced TiKVGroup decides which label key must be used.

## Validation

- `PlacementPolicy.spec.cluster` is required.
- `groupRefs[*].group`, `groupRefs[*].kind`, and `groupRefs[*].name` must be non-empty.
- `groupRefs[*]` must be unique by `(group, kind, name)` in one PlacementPolicy.
- First version only supports `group: core.pingcap.com`, `kind: TiKVGroup`.
- Each referenced groupRef must exist in the same namespace and cluster.
- `rules[*].name` must be non-empty and unique in one PlacementPolicy.
- `selector.type` must be `Keyspace`.
- `selector.keyspaces` must be non-empty and unique in one rule.
- Keyspace IDs must be in `[0, 0xFFFFFF]`.
- `role` must be `voter` in the first version.
- `count` must be positive.
- A keyspace can be owned by only one PlacementPolicy in a cluster.
- `pd.enable-placement-rules` must be enabled.
- Each generated rule's matched target set must have at least `count` ready stores.
- User-provided store labels cannot use `op/*` or `$op/*`.

## Later: Database and Table

Reserve these selectors for future work:

```yaml
selector:
  type: Database
  databases: [...]
```

```yaml
selector:
  type: Table
  tables: [...]
```

Do not implement them now. Database ranges are a moving union of table ranges, and table ranges change with physical table / partition IDs after DDL.

## Tests

- Unit: TiKVGroup placement API validation.
- Unit: PlacementPolicy validation.
- Unit: store label generation for exclusive and non-exclusive TiKVGroups.
- Unit: PD bundle generation from policy rules and groupRef label values.
- Unit: keyspace ownership and generated rule count preservation.
- Controller: status update when TiKVGroup references or PD writes fail.
- E2E: bind one keyspace to one or more TiKVGroups, verify PD bundle and peer placement.

## Feature Gate

Add alpha feature gate `TiKVGroupKeyspacePlacement`.
