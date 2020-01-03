package monitor

import (
	"fmt"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ReleaseTargetRegex: "regex",
		ReleaseNamespaces: []string{
			"ns1",
			"ns2",
		},
		EnableTLSCluster: false,
	}
	content, err := RenderPrometheusConfig(model)
	g.Expect(err).NotTo(HaveOccurred())
	fmt.Printf(content)
}

func TestRender2(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ReleaseTargetRegex: "regex",
		AlertmanagerURL:    "alertUrl",
		ReleaseNamespaces: []string{
			"ns1",
			"ns2",
		},
		EnableTLSCluster: false,
	}
	content, err := RenderPrometheusConfig2(model)
	g.Expect(err).NotTo(HaveOccurred())
	fmt.Printf(content)
}
