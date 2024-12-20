package hasher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name                      string
		tomlStr                   v1alpha1.ConfigFile
		semanticallyEquivalentStr v1alpha1.ConfigFile
		wantHash                  string
		wantError                 bool
	}{
		{
			name: "Valid TOML string",
			tomlStr: v1alpha1.ConfigFile(`foo = 'bar'
[log]
k1 = 'v1'
k2 = 'v2'`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`foo = 'bar'
[log]
k2 = 'v2'
k1 = 'v1'`),
			wantHash:  "5dbbcf4574",
			wantError: false,
		},
		{
			name: "Different config value",
			tomlStr: v1alpha1.ConfigFile(`foo = 'foo'
[log]
k2 = 'v2'
k1 = 'v1'`),
			wantHash:  "f5bc46cb9",
			wantError: false,
		},
		{
			name: "multiple sections with blank line",
			tomlStr: v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]
k2 = 'v2'`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]

k2 = 'v2'`),
			wantHash:  "79598d5977",
			wantError: false,
		},
		{
			name:      "Empty TOML string",
			tomlStr:   v1alpha1.ConfigFile(``),
			wantHash:  "7d6fc488b7",
			wantError: false,
		},
		{
			name: "Invalid TOML string",
			tomlStr: v1alpha1.ConfigFile(`key1 = "value1"
			key2 = value2`), // Missing quotes around value2
			wantHash:  "",
			wantError: true,
		},
		{
			name: "Nested tables",
			tomlStr: v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[parent]
child2 = "value2"
child1 = "value1"
[parent.child]
grandchild2 = "value4"
grandchild1 = "value3"`),
			wantHash:  "7bf645ccb4",
			wantError: false,
		},
		{
			name: "Array of tables",
			tomlStr: v1alpha1.ConfigFile(`[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393

color = "gray"`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[[products]]
sku = 738594937
name = "Hammer"

[[products]]
sku = 284758393
name = "Nail"

color = "gray"`),
			wantHash:  "7549cf87f4",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, err := GenerateHash(tt.tomlStr)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHash, gotHash)

				if string(tt.semanticallyEquivalentStr) != "" {
					reorderedHash, err := GenerateHash(tt.semanticallyEquivalentStr)
					require.NoError(t, err)
					assert.Equal(t, tt.wantHash, reorderedHash)
				}
			}
		})
	}
}
