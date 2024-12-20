package hasher

import (
	"bytes"
	"fmt"
	"hash/fnv"

	"github.com/pelletier/go-toml/v2"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	hashutil "github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/util/hash"
)

// GenerateHash takes a TOML string as input, unmarshals it into a map,
// and generates a hash of the resulting configuration. The hash is then
// encoded into a safe string format and returned.
// If the order of keys in the TOML string is different, the hash will be the same.
func GenerateHash(tomlStr v1alpha1.ConfigFile) (string, error) {
	var config map[string]any
	if err := toml.NewDecoder(bytes.NewReader([]byte(tomlStr))).Decode(&config); err != nil {
		return "", fmt.Errorf("failed to unmarshal toml string %s: %w", tomlStr, err)
	}
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, config)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32())), nil
}
