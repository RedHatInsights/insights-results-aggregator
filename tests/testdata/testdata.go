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
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	OrgID            = types.OrgID(1)
	ClusterName      = types.ClusterName("84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc")
	UserID           = types.UserID("1")
	User2ID          = types.UserID("2")
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
	KafkaOffset      = types.KafkaOffset(1)
	TestRequestID    = types.RequestID("example12345678/requestID")
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
	Rule2 = types.Rule{
		Module:     Rule2ID,
		Name:       Rule2Name,
		Summary:    Rule2Summary,
		Reason:     Rule2Reason,
		Resolution: Rule2Resolution,
		MoreInfo:   Rule2MoreInfo,
	}
	RuleErrorKey1 = types.RuleErrorKey{
		ErrorKey:    "ek1",
		RuleModule:  Rule1ID,
		Condition:   "condition1",
		Description: "description1",
		Impact:      1,
		Likelihood:  2,
		PublishDate: LastCheckedAt,
		Active:      false,
		Generic:     "generic1",
	}
	RuleErrorKey2 = types.RuleErrorKey{
		ErrorKey:    "ek2",
		RuleModule:  Rule2ID,
		Condition:   "condition2",
		Description: "description2",
		Impact:      2,
		Likelihood:  3,
		PublishDate: LastCheckedAt,
		Active:      true,
		Generic:     "generic2",
	}
	RuleWithContent1 = types.RuleWithContent{
		Module:      Rule1.Module,
		Name:        Rule1.Name,
		Summary:     Rule1.Summary,
		Reason:      Rule1.Reason,
		Resolution:  Rule1.Resolution,
		MoreInfo:    Rule1.MoreInfo,
		ErrorKey:    RuleErrorKey1.ErrorKey,
		Condition:   RuleErrorKey1.Condition,
		Description: RuleErrorKey1.Description,
		TotalRisk:   (RuleErrorKey1.Impact + RuleErrorKey1.Likelihood) / 2,
		PublishDate: RuleErrorKey1.PublishDate,
		Active:      RuleErrorKey1.Active,
		Generic:     RuleErrorKey1.Generic,
		Tags:        []string{},
	}
	RuleWithContent2 = types.RuleWithContent{
		Module:      Rule2.Module,
		Name:        Rule2.Name,
		Summary:     Rule2.Summary,
		Reason:      Rule2.Reason,
		Resolution:  Rule2.Resolution,
		MoreInfo:    Rule2.MoreInfo,
		ErrorKey:    RuleErrorKey2.ErrorKey,
		Condition:   RuleErrorKey2.Condition,
		Description: RuleErrorKey2.Description,
		TotalRisk:   (RuleErrorKey2.Impact + RuleErrorKey2.Likelihood) / 2,
		PublishDate: RuleErrorKey2.PublishDate,
		Active:      RuleErrorKey2.Active,
		Generic:     RuleErrorKey2.Generic,
		Tags:        []string{},
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
	LastCheckedAt = time.Unix(25, 0).UTC()

	RuleContentResponses = []types.RuleContentResponse{
		types.RuleContentResponse{
			RuleModule: string(Rule1ID),
		},
		types.RuleContentResponse{
			RuleModule: string(Rule2ID),
		},
		types.RuleContentResponse{
			RuleModule: string(Rule3ID),
		},
	}

	RuleOnReportResponses = []types.RuleOnReport{
		types.RuleOnReport{
			Module: string(Rule1ID),
		},
		types.RuleOnReport{
			Module: string(Rule2ID),
		},
		types.RuleOnReport{
			Module: string(Rule3ID),
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

	Report2Rules = types.ClusterReport(`
{
	"system": {
		"metadata": {},
		"hostname": null
	},
	"reports": [
		{
			"component": "` + string(Rule1ID) + `",
			"key": "` + ErrorKey1 + `"
		},
		{
			"component": "` + string(Rule2ID) + `",
			"key": "` + ErrorKey2 + `"
		}
	],
	"fingerprints": [],
	"skips": [],
	"info": []
}
`)

	Report2RulesDisabledRule1ExpectedResponse = `
{
	"report": {
		"meta": {
			"count": 2,
			"last_checked_at": "` + LastCheckedAt.Format(time.RFC3339) + `"
		},
		"reports": [
			{
				"component": "` + string(Rule2ID) + `",
				"key": "` + ErrorKey2 + `",
				"user_vote": 0,
				"disabled": false
			},
			{
				"component": "` + string(Rule1ID) + `",
				"key": "` + ErrorKey1 + `",
				"user_vote": 0,
				"disabled": true
			}
		]
	},
	"status": "ok"
}
`

	Report2RulesDisabledExpectedResponse = `
{
	"report": {
		"meta": {
			"count": 2,
			"last_checked_at": "` + LastCheckedAt.Format(time.RFC3339) + `"
		},
		"reports": [
			{
				"component": "` + string(Rule1ID) + `",
				"key": "` + ErrorKey1 + `",
				"user_vote": 0,
				"disabled": true
			},
			{
				"component": "` + string(Rule2ID) + `",
				"key": "` + ErrorKey2 + `",
				"user_vote": 0,
				"disabled": true
			}
		]
	},
	"status": "ok"
}
`

	Report2RulesEnabledRulesExpectedResponse = `
{
	"report": {
		"meta": {
			"count": 2,
			"last_checked_at": "` + LastCheckedAt.Format(time.RFC3339) + `"
		},
		"reports": [
			{
				"component": "` + string(Rule1ID) + `",
				"key": "` + ErrorKey1 + `",
				"user_vote": 0,
				"disabled": false
			},
			{
				"component": "` + string(Rule2ID) + `",
				"key": "` + ErrorKey2 + `",
				"user_vote": 0,
				"disabled": false
			}
		]
	},
	"status": "ok"
}
`

	Report3Rules = types.ClusterReport(`
{
	"system": {
		"metadata": {},
		"hostname": null
	},
	"reports": [
		{
			"component": "` + string(Rule1ID) + `",
			"key": "` + ErrorKey1 + `"
		},
		{
			"component": "` + string(Rule2ID) + `",
			"key": "` + ErrorKey2 + `"
		},
		{
			"component": "` + string(Rule3ID) + `",
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
        "reports": [
            {
				"component": "` + string(Rule1ID) + `",
				"key": "` + ErrorKey1 + `",
				"user_vote": 0,
				"disabled": false
			},
            {
				"component": "` + string(Rule2ID) + `",
                "key": "` + ErrorKey2 + `",
                "user_vote": 0,
				"disabled": false
			},
			{
				"component": "` + string(Rule3ID) + `",
                "key": "` + ErrorKey3 + `",
				"user_vote": 0,
				"disabled": false
			}
        ]
    },
	"status": "ok"
}
`
)

func GetRandomConsumerMessage() string {
	// disable Use of weak random number generator for the whole method
	/* #nosec G404 */
	orgID := rand.Intn(999999)
	clusterName := uuid.New()
	timeRandomRange := 100000
	/* #nosec G404 */
	lastCheckedAt := time.Now().Add(-time.Duration(rand.Intn(timeRandomRange)) * time.Second)
	// TODO: generate some real looking consumer report here
	consumerReport := ConsumerReport

	consumerMessage := `{
		    "OrgID": ` + fmt.Sprint(orgID) + `,
		    "ClusterName": "` + clusterName.String() + `",
		    "Report":` + consumerReport + `,
		    "LastChecked": "` + lastCheckedAt.Format(time.RFC3339) + `"
	    }`

	return consumerMessage
}

func GetRandomRuleID(length uint) types.RuleID {
	// disable Use of weak random number generator for the whole method
	/* #nosec G404 */
	var result types.RuleID

	for i := uint(0); i < length; i++ {
		/* #nosec G404 */
		char := rune('a' + rand.Intn('z'-'a'))
		result += types.RuleID(char)
	}

	return result
}

func GetRandomUserID() types.UserID {
	// disable Use of weak random number generator for the whole method
	/* #nosec G404 */
	return types.UserID(fmt.Sprint(rand.Intn(999999)))
}

func GetRandomOrgID() types.OrgID {
	// disable Use of weak random number generator for the whole method
	/* #nosec G404 */
	return types.OrgID(rand.Intn(999999))
}

func GetRandomClusterID() types.ClusterName {
	return types.ClusterName(uuid.New().String())
}
