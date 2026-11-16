# TiDB Operator Development Rules

## CRD Fields and Update Semantics

1. **Place fields by responsibility.** Group `spec` holds policies that coordinate
   instances, such as replicas, scheduling, and rolling updates. Group
   `spec.template.spec` holds desired configuration consumed by each instance and
   propagated to Instance `spec`. Observed state and progress belong in `status`.

2. **Separate field placement from restart behavior.** A template change does not
   necessarily require a Pod restart. Classify each new field as a controller
   parameter, an in-place configuration change, or a change requiring Pod
   replacement. Update the relevant reloadable checks accordingly.

3. **Verify the full propagation path.** Trace each new field from Group input
   through revision detection and Instance spec to its consumer. Cover both
   instance creation and updates to existing instances; adding the API field and
   copying it only during creation is insufficient.

4. **Keep lifecycle gates distinct.** Pod readiness, readiness to accept leaders,
   and availability for the next rolling-update step have different meanings.
   Use separate settings and constants for independent gates, even when their
   default durations match.
