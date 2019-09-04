package apiserver

import (
	"fmt"
	"github.com/prometheus/common/log"
	auditinternal "k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

type MysqlAudit struct {
}

func (m *MysqlAudit) ProcessEvents(events ...*auditinternal.Event) bool {
	for _, e := range events {
		log.Info(fmt.Sprintf("api is called, action=%s", e.Verb))
	}
	return true
}

// Run will initialize the backend. It must not block, but may run go routines in the background. If
// stopCh is closed, it is supposed to stop them. Run will be called before the first call to ProcessEvents.
func (m *MysqlAudit) Run(stopCh <-chan struct{}) error {
	return nil
}

// Shutdown will synchronously shut down the backend while making sure that all pending
// events are delivered. It can be assumed that this method is called after
// the stopCh channel passed to the Run method has been closed.
func (m *MysqlAudit) Shutdown() {

}

// Returns the backend PluginName.
func (m *MysqlAudit) String() string {
	return "MySQL"
}

type MySQLChecker struct {
}

func (p *MySQLChecker) LevelAndStages(attrs authorizer.Attributes) (auditinternal.Level, []auditinternal.Stage) {
	//for _, rule := range p.Rules {
	//	if ruleMatches(&rule, attrs) {
	//		return rule.Level, rule.OmitStages
	//	}
	//}
	return auditinternal.LevelRequest, []auditinternal.Stage{auditinternal.StageResponseComplete}
}
