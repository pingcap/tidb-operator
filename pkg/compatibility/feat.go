package compatibility

// All these are features supported by different TiDB versions
// NOTE: Do not add constraints with prerelease version
// NOTE: MUST add key PRs in comments
var (
	// PDMS is supported
	// This feature is enabled after the tso primary transferring is supported
	// See https://github.com/tikv/pd/pull/8157
	PDMS = MustNewConstraints(">= v8.3.0")

	// PD ready api
	// See https://github.com/tikv/pd/pull/8749
	PDReadyAPI = MustNewConstraints(">= v9.0.0 || ^v8.5.2")

	// TiKV ready api
	// See https://github.com/tikv/tikv/pull/18237
	TiKVReadyAPI = MustNewConstraints(">= v9.0.0")
)
