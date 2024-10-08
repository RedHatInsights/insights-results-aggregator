// Copyright 2020, 2021, 2022 Red Hat, Inc
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

// Package types contains declaration of various data types (usually structures)
// used elsewhere in the aggregator code.
package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	types "github.com/RedHatInsights/insights-results-types"
)

// OrgID represents organization ID
type OrgID = types.OrgID

// Account represent the account number
type Account = uint32

// ClusterName represents name of cluster in format c8590f31-e97e-4b85-b506-c45ce1911a12
type ClusterName = types.ClusterName

// UserID represents type for user id
type UserID = types.UserID

// ClusterReport represents cluster report
type ClusterReport = types.ClusterReport

// Timestamp represents any timestamp in a form gathered from database
// TODO: need to be improved
type Timestamp = types.Timestamp

// UserVote is a type for user's vote
type UserVote = types.UserVote

// InfoKey represents type for info key
type InfoKey string

const (
	// UserVoteDislike shows user's dislike
	UserVoteDislike = types.UserVoteDislike
	// UserVoteNone shows no vote from user
	UserVoteNone = types.UserVoteNone
	// UserVoteLike shows user's like
	UserVoteLike = types.UserVoteLike
	// WorkloadRecommendationSuffix is used to strip a suffix from rule ID (Component attribute) in WorkloadRecommendation
	WorkloadRecommendationSuffix = ".recommendation"
)

type (
	// RequestID is used to store the request ID supplied in input Kafka records as
	// a unique identifier of payloads. Empty string represents a missing request ID.
	RequestID = types.RequestID
	// RuleOnReport represents a single (hit) rule of the string encoded report
	RuleOnReport = types.RuleOnReport
	// ReportRules is a helper struct for easy JSON unmarshalling of string encoded report
	ReportRules = types.ReportRules
	// ReportResponse represents the response of /report endpoint
	ReportResponse = types.ReportResponse
	// ReportResponseMeta contains metadata about the report
	ReportResponseMeta = types.ReportResponseMeta
	// DisabledRuleResponse represents a single disabled rule displaying only identifying information
	DisabledRuleResponse = types.DisabledRuleResponse
	// RuleID represents type for rule id
	RuleID = types.RuleID
	// RuleSelector represents type for rule id + error key in the rule_id|error_key format
	RuleSelector = types.RuleSelector
	// ErrorKey represents type for error key
	ErrorKey = types.ErrorKey
	// Rule represents the content of rule table
	Rule = types.Rule
	// RuleErrorKey represents the content of rule_error_key table
	RuleErrorKey = types.RuleErrorKey
	// RuleWithContent represents a rule with content, basically the mix of rule and rule_error_key tables' content
	RuleWithContent = types.RuleWithContent
	// Version represents the version of the cluster
	Version = types.Version
	// DBDriver type for db driver enum
	DBDriver = types.DBDriver
	// Identity contains internal user info
	Identity = types.Identity
	// Token is x-rh-identity struct
	Token = types.Token
	// User is single-user identifying struct
	User = types.User
	// RuleFQDN is rule module, older notation
	RuleFQDN = types.RuleFQDN
)

const (
	// DBDriverPostgres shows that db driver is postgres
	DBDriverPostgres = types.DBDriverPostgres
	// DBDriverGeneral general sql(used for mock now)
	DBDriverGeneral = types.DBDriverGeneral
)

const (
	// ContextKeyUser is a constant for user authentication token in request
	ContextKeyUser = types.ContextKeyUser
)

// FeedbackRequest contains message of user feedback
type FeedbackRequest struct {
	Message string `json:"message"`
}

// ReportItem represents a single (hit) rule of the string encoded report
type ReportItem = types.ReportItem

// InfoItem represents a single (hit) info rule of the string encoded report
type InfoItem struct {
	InfoID  RuleID            `json:"info_id"`
	InfoKey InfoKey           `json:"key"`
	Details map[string]string `json:"details"`
}

/* This is how a DVO workload should look like:
{
    "system": {
        "metadata": {},
        "hostname": null
    },
    "fingerprints": [],
    "version": 1,
    "analysis_metadata": {},
    "workload_recommendations": [
        {
            "response_id": "an_issue|DVO_AN_ISSUE",
            "component": "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
            "key": "DVO_AN_ISSUE",
            "details": {},
            "tags": [],
            "links": {
                "jira": [
                    "https://issues.redhat.com/browse/AN_ISSUE"
                ],
                "product_documentation": []
            },
            "workloads": [
                {
                    "namespace": "namespace-name-A",
                    "namespace_uid": "NAMESPACE-UID-A",
                    "kind": "DaemonSet",
                    "name": "test-name-0099",
                    "uid": "UID-0099"
                }
            ]
        }
    ]
}
*/

// DVOMetrics contains all the workload recommendations for a single cluster
type DVOMetrics struct {
	WorkloadRecommendations []WorkloadRecommendation `json:"workload_recommendations"`
}

// WorkloadRecommendation contains all the information about the recommendation
// Details is generic interface{} because it contains template data used to fill rule content, i.e. we don't/should't know the structure
type WorkloadRecommendation struct {
	ResponseID string                 `json:"response_id"`
	Component  string                 `json:"component"`
	Key        string                 `json:"key"`
	Details    map[string]interface{} `json:"details"`
	Tags       []string               `json:"tags"`
	Links      DVOLinks               `json:"links"`
	Workloads  []DVOWorkload          `json:"workloads"`
}

// DVOWorkload contains the main information of the workload recommendation
type DVOWorkload struct {
	Namespace    string `json:"namespace"`
	NamespaceUID string `json:"namespace_uid"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	UID          string `json:"uid"`
}

// DVOLinks contains some URLs with relevant information about the recommendation
type DVOLinks struct {
	Jira                 []string `json:"jira"`
	ProductDocumentation []string `json:"product_documentation"`
}

// DVOReport represents a single row of the dvo.dvo_report table.
type DVOReport struct {
	OrgID           string          `json:"org_id"`
	NamespaceID     string          `json:"namespace_id"`
	NamespaceName   string          `json:"namespace_name"`
	ClusterID       string          `json:"cluster_id"`
	Recommendations uint            `json:"recommendations"`
	Report          string          `json:"report"`
	Objects         uint            `json:"objects"`
	ReportedAt      types.Timestamp `json:"reported_at"`
	LastCheckedAt   types.Timestamp `json:"last_checked_at"`
	RuleHitsCount   RuleHitsCount   `json:"rule_hits_count"`
}

// Cluster structure contains cluster UUID and cluster name
type Cluster struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
}

// Namespace structure contains basic information about namespace
type Namespace struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// DVOMetadata structure contains basic information about workload metadata
type DVOMetadata struct {
	Recommendations uint        `json:"recommendations"`
	Objects         uint        `json:"objects"`
	ReportedAt      string      `json:"reported_at"`
	LastCheckedAt   string      `json:"last_checked_at"`
	HighestSeverity int         `json:"highest_severity"`
	HitsBySeverity  map[int]int `json:"hits_by_severity"`
}

// WorkloadsForNamespace structure represents a single entry of the namespace list with some aggregations
type WorkloadsForNamespace struct {
	Cluster                 Cluster       `json:"cluster"`
	Namespace               Namespace     `json:"namespace"`
	Metadata                DVOMetadata   `json:"metadata"`
	RecommendationsHitCount RuleHitsCount `json:"recommendations_hit_count"`
}

// ClusterReports is a data structure containing list of clusters, list of
// errors and dictionary with results per cluster.
type ClusterReports = types.ClusterReports

// SchemaVersion represents the received message's version of data schema
type SchemaVersion = types.SchemaVersion

// AllowedVersions represents the currently allowed versions of data schema
type AllowedVersions = map[SchemaVersion]struct{}

// RuleRating represents a rule rating element sent by the user
type RuleRating = types.RuleRating

// Metadata represents the metadata field in the Kafka topic messages
type Metadata struct {
	GatheredAt time.Time `json:"gathering_time"`
}

// RuleHitsCount represents the number of hits for a given rule
type RuleHitsCount map[string]int

// Value convert a RuleHitsCount into a byte[]
func (in RuleHitsCount) Value() (driver.Value, error) {
	return json.Marshal(in)
}

// Scan parses a byte[] value into a RuleHitsCount
func (in *RuleHitsCount) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("not byte array")
	}

	return json.Unmarshal(b, &in)
}
