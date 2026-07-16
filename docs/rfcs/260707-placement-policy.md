# Placement Policy for TiKVGroup

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [Placement Rule Mapping](#placement-rule-mapping)
- [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Store Labels](#store-labels)
  - [PlacementPolicy Controller](#placementpolicy-controller)
  - [PD Placement Rule](#pd-placement-rule)
  - [Test Plan](#test-plan)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a release*.

- [ ] (R) This design doc has been discussed and approved
- [ ] (R) Test plan is in place
  - [ ] (R) e2e tests in kind
- [ ] (R) Graduation criteria is in place if required
- [ ] (R) User-facing documentation has been created in [pingcap/docs-tidb-operator]

## Summary

Add a `PlacementPolicy` CRD to bind selected key ranges to selected `TiKVGroup`s. The operator converts the policy to PD placement rule bundles.

Currently only keyspace is supported. Database and table selectors are reserved because their key ranges are not stable enough for an operator-managed API.

## Motivation

Users need to schedule a specific workload range to specific TiKV stores without writing raw PD key ranges.

### Goals

- Let users select ranges by keyspace ID instead of raw start/end keys.
- Let users select destination stores by `TiKVGroup`.
- Support normal and exclusive TiKVGroup placement.
- Generate PD placement rules compatible with PD keyspace placement.

### Non-Goals

- Support database/table selectors in the first version.
- Manage PD scheduling logic beyond placement rules.
- Split or rewrite the user-specified `count`.

## Proposal

### API

`TiKVGroup` gets a group-level placement switch:

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: TiKVGroup
metadata:
  name: kvg-exclusive
spec:
  template:
    spec:
      placement:
        exclusive: true
```

`exclusive: true` means stores in this TiKVGroup get an additional exclusive placement label. PD will not schedule unrelated key ranges to these stores unless a rule explicitly constrains on the exclusive label.

Placement rules are defined by a standalone CRD:

```yaml
apiVersion: core.pingcap.com/v1alpha1
kind: PlacementPolicy
metadata:
  name: ks1-to-kvg
spec:
  cluster:
    name: tc
  groupRefs:
  - group: core.pingcap.com
    kind: TiKVGroup
    name: kvg-normal
  - group: core.pingcap.com
    kind: TiKVGroup
    name: kvg-exclusive
  rules:
  - name: voter
    role: voter
    count: 3
    selector:
      keyspace:
        ids:
        - "1"
        - "2"
```

`groupRefs` and `rules` are sibling fields. Every rule applies to every referenced group.

The first version supports:

- `groupRefs[].group: core.pingcap.com`
- `groupRefs[].kind: TiKVGroup`
- `rules[].role: voter`
- `rules[].selector.keyspace.ids: []string`

### Placement Rule Mapping

The operator labels all referenced TiKVGroups with the same store label key:

- every TiKVGroup: `k/tikvgroup`
- exclusive TiKVGroup additionally: `$k/exclusive=true`

For each user rule and each selected keyspace:

- `group_id` is the shared operator-owned group: `tidb-operator`.
- each policy owns only rules whose ID starts with the policy prefix: `<policyName>:`.
- each generated rule ID is `<policyName>:<ruleName>-<keyspaceID>-<rangeType>` to avoid collisions in the shared group.
- generate one txn PD rule for each selected keyspace with `k/tikvgroup in [...]`.
- every generated rule also adds `$k/exclusive notIn ["__null__"]`. This explicitly references the `$k/exclusive` key so PD can consider exclusive stores, while `__null__` is an intentionally nonexistent value so the constraint does not filter out normal stores or stores labeled `$k/exclusive=true`.
- `count` is copied as-is to every generated PD rule.

## Risks and Mitigations

- **Conflicting policies**: two policies may select overlapping key ranges.
  - Mitigation: each policy owns only its prefixed rules. PD placement rule priorities and overlap semantics decide the effective scheduling behavior.
- **Unstable database/table ranges**: database/table ranges can change with DDL and implementation details.
  - Mitigation: do not expose these selectors in the first version.
- **Exclusive label misuse**: users may manually set operator-managed labels.
  - Mitigation: reject TiKV and TiFlash server labels with `k/` or `$k/` prefixes.

## Design Details

### Store Labels

The TiKV controller manages placement labels per store:

```text
k/tikvgroup=$namespace.$tikvgroup
$k/exclusive=true
```

All TiKV stores get `k/tikvgroup`. Exclusive stores also get `$k/exclusive=true`. When `exclusive` changes from true to false, the controller removes `$k/exclusive`.

The `$` prefix is intentional. PD treats `$`-prefixed labels as exclusive labels. A store with `$k/exclusive` will not be selected by default rules. It can still be selected by a placement rule that explicitly constrains `$k/exclusive`.

PlacementPolicy rules target both normal and exclusive TiKVGroups through the shared `k/tikvgroup` constraint. They also add `$k/exclusive notIn ["__null__"]` only to opt in to PD's exclusive-label matching. `__null__` must not be used as a real store label value; as a nonexistent sentinel, it keeps the constraint effectively non-filtering while still mentioning the exclusive label key.

### PlacementPolicy Controller

The controller reconciles `PlacementPolicy` as follows:

1. Resolve `spec.cluster.name` and all `spec.groupRefs`.
2. Skip PD rule writes while the referenced Cluster is paused or `suspendAction.suspendCompute` is true.
3. Validate every `groupRef` points to a `TiKVGroup` in the same cluster.
4. Build the txn keyspace range for every selected keyspace ID using PD's keyspace prefix encoding.
5. Build target values for the shared label key `k/tikvgroup`.
6. Put all generated PD rules in the shared `tidb-operator` rule group.
7. Post the PD rule group before writing rules. The rule group API is idempotent.
8. List current rules in the shared group, filter by policy rule ID prefix, diff with expected rules, and write the resulting batch ops.
9. Delete stale policy-prefixed rules when selected keyspaces change. The shared rule group is never deleted by the policy controller.

Status records sync state:

```yaml
status:
  conditions:
  - type: Synced
    status: "True"
```

### PD Placement Rule

All placement policies share one PD rule group bundle:

The controller posts the rule group config before writing rules. The rule group API is idempotent, so this does not require a preceding read:

```json
{
  "id": "tidb-operator",
  "index": 99,
  "override": true
}
```

The controller writes rules through:

```text
POST /pd/api/v1/config/rules/batch
```

The controller first lists current rules from:

```text
GET /pd/api/v1/config/rules/group/{groupID}
```

Then it filters by the policy rule ID prefix, diffs current rules with expected rules, and writes only necessary delete/add ops through the batch API. Deletes are exact rule-ID deletes, not prefix deletes. The controller does not delete the shared rule group.

For this policy:

```yaml
rules:
- name: voter
  role: voter
  count: 3
  selector:
    keyspace:
      ids:
      - "1"
```

If `groupRefs` contains one normal TiKVGroup and one exclusive TiKVGroup, the generated PD rules are:

```json
[
  {
    "group_id": "tidb-operator",
    "id": "ks1-to-kvg:voter-1-txn",
    "role": "voter",
    "count": 3,
    "start_key": "<keyspace-1-txn-start-hex>",
    "end_key": "<keyspace-1-txn-end-hex>",
    "label_constraints": [
      {
        "key": "k/tikvgroup",
        "op": "in",
        "values": ["ns.kvg-normal", "ns.kvg-exclusive"]
      },
      {
        "key": "$k/exclusive",
        "op": "notIn",
        "values": ["__null__"]
      }
    ]
  }
]
```

For every selected keyspace ID, the operator generates one txn key range. The generated PD rule ID ends with `-txn`. The operator only accepts keyspace IDs in PD's valid 24-bit keyspace ID range.

### Test Plan

- API validation:
  - validate `PlacementPolicy.spec.groupRefs`.
  - validate `PlacementPolicy.spec.rules`.
  - reject TiKV and TiFlash server labels with `k/` and `$k/` prefixes.
- Unit tests:
  - verify normal and exclusive targets use the shared TiKVGroup label constraint.
  - verify txn keyspace range encoding.
- E2E:
  - create normal and exclusive `TiKVGroup`s.
  - create a `PlacementPolicy` for a keyspace.
  - verify PD stores contain `k/tikvgroup` and exclusive stores additionally contain `$k/exclusive=true`.
  - verify PD placement bundle contains generated rules with the exclusive bypass constraint and unchanged `count`.

## Drawbacks

- Database/table placement must wait until stable range ownership is available.

## Alternatives

- Put rules directly under `TiKVGroup`.
  - Rejected because one policy may need to target multiple groups, and policy lifecycle should be independent from store group lifecycle.
- Let users specify raw key ranges.
  - Rejected because it exposes PD internals and is hard to operate safely.
- Use only one label key and encode exclusive in value.
  - Rejected because PD exclusive label semantics require a `$`-prefixed label key.
