package storage_test

import (
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/stretchr/testify/assert"
)

var (
	ruleContentActiveOK = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": content.RuleErrorKeyContent{
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "active",
					},
				},
			},
		},
	}
	ruleContentInactiveOK = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": content.RuleErrorKeyContent{
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "inactive",
					},
				},
			},
		},
	}
	ruleContentBadStatus = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": content.RuleErrorKeyContent{
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "bad",
					},
				},
			},
		},
	}
	ruleContentNull = content.RuleContentDirectory{
		"rc": content.RuleContent{},
	}
	ruleContentExample1 = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("summary"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			Plugin: content.RulePluginInfo{
				Name:         "test rule",
				NodeID:       string(testClusterName),
				ProductCode:  "product code",
				PythonModule: string(testRuleID),
			},
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": {
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "active",
					},
				},
			},
		},
	}
	ruleContentMultipleRules = content.RuleContentDirectory{
		"rc1": content.RuleContent{
			Summary:    []byte("rule 1 summary"),
			Reason:     []byte("rule 1 reason"),
			Resolution: []byte("rule 1 resolution"),
			MoreInfo:   []byte("rule 1 more info"),
			Plugin: content.RulePluginInfo{
				Name:         "rule 1 name",
				NodeID:       string(testClusterName),
				ProductCode:  "rule 1 product code",
				PythonModule: "test.rule1",
			},
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek1": {
					Generic: []byte("rule 1 details"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "rule 1 description",
						Impact:      2,
						Likelihood:  4,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "active",
					},
				},
			},
		},
		"rc2": content.RuleContent{
			Summary:    []byte("rule 2 summary"),
			Reason:     []byte("rule 2 reason"),
			Resolution: []byte("rule 2 resolution"),
			MoreInfo:   []byte("rule 2 more info"),
			Plugin: content.RulePluginInfo{
				Name:         "rule 2 name",
				NodeID:       string(testClusterName),
				ProductCode:  "rule 2 product code",
				PythonModule: "test.rule2",
			},
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek2": {
					Generic: []byte("rule 2 details"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "rule 2 description",
						Impact:      6,
						Likelihood:  2,
						PublishDate: "1970-01-02 00:00:00",
						Status:      "active",
					},
				},
			},
		},
		"rc3": content.RuleContent{
			Summary:    []byte("rule 3 summary"),
			Reason:     []byte("rule 3 reason"),
			Resolution: []byte("rule 3 resolution"),
			MoreInfo:   []byte("rule 3 more info"),
			Plugin: content.RulePluginInfo{
				Name:         "rule 3 name",
				NodeID:       string(testClusterName),
				ProductCode:  "rule 3 product code",
				PythonModule: "test.rule3",
			},
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek3": {
					Generic: []byte("rule 3 details"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "rule 3 description",
						Impact:      2,
						Likelihood:  2,
						PublishDate: "1970-01-03 00:00:00",
						Status:      "active",
					},
				},
			},
		},
	}
)

func TestLoadRuleContentActiveOK(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentActiveOK)
	helpers.FailOnError(t, err)
}

func TestLoadRuleContentDBError(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentActiveOK)
	if err == nil {
		t.Fatalf("error expected, got %+v", err)
	}
}

func TestLoadRuleContentInactiveOK(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentInactiveOK)
	helpers.FailOnError(t, err)
}

func TestLoadRuleContentNull(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentNull)
	if err == nil || err.Error() != "NOT NULL constraint failed: rule.summary" {
		t.Fatal(err)
	}
}

func TestLoadRuleContentBadStatus(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentBadStatus)
	if err == nil || err.Error() != "invalid rule error key status: 'bad'" {
		t.Fatal(err)
	}
}

func TestDBStorageGetContentForRulesEmpty(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules:     nil,
		SkippedRules: nil,
		PassedRules:  nil,
		TotalCount:   0,
	})
	helpers.FailOnError(t, err)

	assert.Empty(t, res)
}

func TestDBStorageGetContentForRulesDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules:     nil,
		SkippedRules: nil,
		PassedRules:  nil,
		TotalCount:   0,
	})
	if err == nil {
		t.Fatalf("error expected, got %+v", err)
	}
}

func TestDBStorageGetContentForRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentExample1)
	helpers.FailOnError(t, err)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules: []types.RuleOnReport{
			{
				Module:   string(testRuleID),
				ErrorKey: "ek",
			},
		},
		TotalCount: 1,
	})
	helpers.FailOnError(t, err)

	assert.Equal(t, []types.RuleContentResponse{
		{
			ErrorKey:     "ek",
			RuleModule:   string(testRuleID),
			Description:  "description",
			Generic:      "generic",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    1,
			RiskOfChange: 0,
		},
	}, res)
}

func TestDBStorageGetContentForMultipleRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentMultipleRules)
	helpers.FailOnError(t, err)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules: []types.RuleOnReport{
			{
				Module:   "test.rule1.report",
				ErrorKey: "ek1",
			},
			{
				Module:   "test.rule2.report",
				ErrorKey: "ek2",
			},
			{
				Module:   "test.rule3.report",
				ErrorKey: "ek3",
			},
		},
		TotalCount: 3,
	})
	helpers.FailOnError(t, err)

	assert.Len(t, res, 3)

	// TODO: check risk of change when it will be returned correctly
	// total risk is `(impact + likelihood) / 2`
	assert.Equal(t, []types.RuleContentResponse{
		{
			ErrorKey:     "ek1",
			RuleModule:   "test.rule1",
			Description:  "rule 1 description",
			Generic:      "rule 1 details",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    3,
			RiskOfChange: 0,
		},
		{
			ErrorKey:     "ek2",
			RuleModule:   "test.rule2",
			Description:  "rule 2 description",
			Generic:      "rule 2 details",
			CreatedAt:    "1970-01-02T00:00:00Z",
			TotalRisk:    4,
			RiskOfChange: 0,
		},
		{
			ErrorKey:     "ek3",
			RuleModule:   "test.rule3",
			Description:  "rule 3 description",
			Generic:      "rule 3 details",
			CreatedAt:    "1970-01-03T00:00:00Z",
			TotalRisk:    2,
			RiskOfChange: 0,
		},
	}, res)
}
