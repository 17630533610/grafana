package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/services/ngalert/eval"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/services/ngalert/state"
	"github.com/grafana/grafana/pkg/services/ngalert/store"
	"github.com/grafana/grafana/pkg/util"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type FakeAlertingStore struct {
	orgsWithConfig map[int64]bool
}

func newFakeAlertingStore(t *testing.T) FakeAlertingStore {
	t.Helper()

	return FakeAlertingStore{
		orgsWithConfig: map[int64]bool{},
	}
}

func (f FakeAlertingStore) Setup(orgID int64) {
	f.orgsWithConfig[orgID] = true
}

func (f FakeAlertingStore) GetLatestAlertmanagerConfiguration(_ context.Context, query *models.GetLatestAlertmanagerConfigurationQuery) error {
	if _, ok := f.orgsWithConfig[query.OrgID]; ok {
		return nil
	}
	return store.ErrNoAlertmanagerConfiguration
}

type fakeAlertInstanceManager struct {
	mtx sync.Mutex
	// orgID -> RuleID -> States
	states map[int64]map[string][]*state.State
}

func NewFakeAlertInstanceManager(t *testing.T) *fakeAlertInstanceManager {
	t.Helper()

	return &fakeAlertInstanceManager{
		states: map[int64]map[string][]*state.State{},
	}
}

func (f *fakeAlertInstanceManager) GetAll(orgID int64) []*state.State {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	var s []*state.State

	for orgID := range f.states {
		for _, states := range f.states[orgID] {
			s = append(s, states...)
		}
	}

	return s
}

func (f *fakeAlertInstanceManager) GetStatesForRuleUID(orgID int64, alertRuleUID string) []*state.State {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	return f.states[orgID][alertRuleUID]
}

func (f *fakeAlertInstanceManager) GenerateAlertInstances(orgID int64, count int) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	evaluationTime := time.Now()
	evaluationDuration := 1 * time.Minute
	alertRuleUID := util.GenerateShortUID()

	for i := 0; i < count; i++ {
		_, ok := f.states[orgID]
		if !ok {
			f.states[orgID] = map[string][]*state.State{}
		}
		_, ok = f.states[orgID][alertRuleUID]
		if !ok {
			f.states[orgID][alertRuleUID] = []*state.State{}
		}

		f.states[orgID][alertRuleUID] = append(f.states[orgID][alertRuleUID], &state.State{
			AlertRuleUID: fmt.Sprintf("alert_rule_%v", i),
			OrgID:        1,
			Labels: data.Labels{
				"__alert_rule_namespace_uid__": "test_namespace_uid",
				"__alert_rule_uid__":           fmt.Sprintf("test_alert_rule_uid_%v", i),
				"alertname":                    fmt.Sprintf("test_title_%v", i),
				"label":                        "test",
				"instance_label":               "test",
			},
			State: eval.Normal,
			Results: []state.Evaluation{
				{
					EvaluationTime:  evaluationTime,
					EvaluationState: eval.Normal,
					Values:          make(map[string]*float64),
				},
				{
					EvaluationTime:  evaluationTime.Add(1 * time.Minute),
					EvaluationState: eval.Normal,
					Values:          make(map[string]*float64),
				},
			},
			LastEvaluationTime: evaluationTime.Add(1 * time.Minute),
			EvaluationDuration: evaluationDuration,
			Annotations:        map[string]string{"annotation": "test"},
		})
	}
}
