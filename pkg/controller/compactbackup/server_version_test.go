package compact

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRequireShardedJobK8sVersion(t *testing.T) {
	tests := []struct {
		name        string
		major       string
		minor       string
		expectError bool
	}{
		{name: "1.29 passes", major: "1", minor: "29"},
		{name: "1.29+ passes", major: "1", minor: "29+"},
		{name: "1.30 passes", major: "1", minor: "30"},
		{name: "1.28 fails", major: "1", minor: "28", expectError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			discovery := &fakediscovery.FakeDiscovery{
				Fake: &k8stesting.Fake{},
				FakedServerVersion: &version.Info{
					Major: test.major,
					Minor: test.minor,
				},
			}

			err := requireShardedJobK8sVersion(discovery)
			if test.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "Kubernetes >= 1.29") {
					t.Fatalf("expected version gate error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
