// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"fmt"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	OrgID            = types.OrgID(1)
	ClusterName      = types.ClusterName("84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc")
	UserID           = types.UserID("1")
	BadClusterName   = types.ClusterName("aaaa")
	Rule1ID          = types.RuleID("test.rule1")
	BadRuleID        = types.RuleID("rule id with spaces")
	Rule2ID          = types.RuleID("test.rule2")
	Rule3ID          = types.RuleID("test.rule3")
	Rule1Name        = "rule 1 name"
	Rule2Name        = "rule 2 name"
	Rule3Name        = "rule 3 name"
	ErrorKey1        = "ek1"
	ErrorKey2        = "ek2"
	ErrorKey3        = "ek3"
	Rule1Description = "rule 1 description"
	Rule2Description = "rule 2 description"
	Rule3Description = "rule 3 description"
	Rule1Details     = "rule 1 details"
	Rule2Details     = "rule 2 details"
	Rule3Details     = "rule 3 details"
	Rule1CreatedAt   = "1970-01-01T00:00:00Z"
	Rule2CreatedAt   = "1970-01-02T00:00:00Z"
	Rule3CreatedAt   = "1970-01-03T00:00:00Z"
	Rule1Summary     = "rule 1 summary"
	Rule2Summary     = "rule 2 summary"
	Rule3Summary     = "rule 3 summary"
	Rule1Reason      = "rule 1 reason"
	Rule2Reason      = "rule 2 reason"
	Rule3Reason      = "rule 3 reason"
	Rule1Resolution  = "rule 1 resolution"
	Rule2Resolution  = "rule 2 resolution"
	Rule3Resolution  = "rule 3 resolution"
	Rule1MoreInfo    = "rule 1 more info"
	Rule2MoreInfo    = "rule 2 more info"
	Rule3MoreInfo    = "rule 3 more info"
)

var (
	Rule1 = types.Rule{
		Module:     Rule1ID,
		Name:       Rule1Name,
		Summary:    Rule1Summary,
		Reason:     Rule1Reason,
		Resolution: Rule1Resolution,
		MoreInfo:   Rule1MoreInfo,
	}
	ConsumerReport = `{
		"fingerprints": [],
		"info": [],
		"reports": [],
		"skips": [],
		"system": {}
	}`
	ConsumerMessage = `{
		"OrgID": ` + fmt.Sprint(OrgID) + `,
		"ClusterName": "` + string(ClusterName) + `",
		"Report":` + ConsumerReport + `,
		"LastChecked": "` + LastCheckedAt.Format(time.RFC3339) + `"
	}`
	LastCheckedAt     = time.Unix(25, 0)
	RuleContent3Rules = content.RuleContentDirectory{
		Config: content.GlobalRuleConfig{Impact: map[string]int{
			"Two": 2,
			"Six": 6,
		}},
		Rules: map[string]content.RuleContent{
			"rc1": content.RuleContent{
				Summary:    []byte("rule 1 summary"),
				Reason:     []byte("rule 1 reason"),
				Resolution: []byte("rule 1 resolution"),
				MoreInfo:   []byte("rule 1 more info"),
				Plugin: content.RulePluginInfo{
					Name:         "rule 1 name",
					NodeID:       string(ClusterName),
					ProductCode:  "rule 1 product code",
					PythonModule: string(Rule1ID),
				},
				ErrorKeys: map[string]content.RuleErrorKeyContent{
					ErrorKey1: {
						Generic: []byte(Rule1Details),
						Metadata: content.ErrorKeyMetadata{
							Condition:   "condition",
							Description: Rule1Description,
							Impact:      "Two",
							Likelihood:  4,
							PublishDate: Rule1CreatedAt,
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
					NodeID:       string(ClusterName),
					ProductCode:  "rule 2 product code",
					PythonModule: string(Rule2ID),
				},
				ErrorKeys: map[string]content.RuleErrorKeyContent{
					ErrorKey2: {
						Generic: []byte(Rule2Details),
						Metadata: content.ErrorKeyMetadata{
							Condition:   "condition",
							Description: Rule2Description,
							Impact:      "Six",
							Likelihood:  2,
							PublishDate: Rule2CreatedAt,
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
					NodeID:       string(ClusterName),
					ProductCode:  "rule 3 product code",
					PythonModule: string(Rule3ID),
				},
				ErrorKeys: map[string]content.RuleErrorKeyContent{
					ErrorKey3: {
						Generic: []byte(Rule3Details),
						Metadata: content.ErrorKeyMetadata{
							Condition:   "condition",
							Description: Rule3Description,
							Impact:      "Two",
							Likelihood:  2,
							PublishDate: Rule3CreatedAt,
							Status:      "active",
						},
					},
				},
			},
		},
	}

	Report0Rules = types.ClusterReport(`
{
	"system": {
		"metadata": {},
		"hostname": null
	},
	"reports": [],
	"fingerprints": [],
	"skips": [],
	"info": []
}
`)

	Report3Rules = types.ClusterReport(`
{
	"system": {
		"metadata": {},
		"hostname": null
	},
	"reports": [
		{
			"component": "` + string(Rule1ID) + `.report",
			"key": "` + ErrorKey1 + `"
		},
		{
			"component": "` + string(Rule2ID) + `.report",
			"key": "` + ErrorKey2 + `"
		},
		{
			"component": "` + string(Rule3ID) + `.report",
			"key": "` + ErrorKey3 + `"
		}
	],
	"fingerprints": [],
	"skips": [],
	"info": []
}
`)

	Report3RulesExpectedResponse = `
{
  "report": {
    "meta": {
      "count": 3,
      "last_checked_at": "` + LastCheckedAt.Format(time.RFC3339) + `"
    },
    "data": [
      {
		"rule_id": "` + string(Rule1ID) + `",
        "description": "` + Rule1Description + `",
        "details": "` + Rule1Details + Rule1Resolution + `",
        "created_at": "` + Rule1CreatedAt + `",
        "total_risk": 3,
		"risk_of_change": 0,
		"extra_data": null
      },
      {
		"rule_id": "` + string(Rule2ID) + `",
        "description": "` + Rule2Description + `",
        "details": "` + Rule2Details + Rule2Resolution + `",
        "created_at": "` + Rule2CreatedAt + `",
        "total_risk": 4,
		"risk_of_change": 0,
		"extra_data": null
      },
      {
		"rule_id": "` + string(Rule3ID) + `",
        "description": "` + Rule3Description + `",
        "details": "` + Rule3Details + Rule3Resolution + `",
        "created_at": "` + Rule3CreatedAt + `",
        "total_risk": 2,
		"risk_of_change": 0,
		"extra_data": null
      }
    ]
  },
  "status": "ok"
}
`
)
