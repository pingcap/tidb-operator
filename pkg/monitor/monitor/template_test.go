package monitor

import (
	"github.com/ghodss/yaml"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRenderPrometheusConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	model := &MonitorConfigModel{
		ReleaseTargetRegex: "regex",
	}
	c := newPrometheusConfig(model)
	_, err := yaml.Marshal(c)
	g.Expect(err).NotTo(HaveOccurred())
}
