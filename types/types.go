/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

// OrgID represents organization ID
type OrgID uint64

// ClusterName represents name of cluster in format c8590f31-e97e-4b85-b506-c45ce1911a12
type ClusterName string

// ClusterReport represents cluster report
type ClusterReport string

// Timestamp represents any timestamp in a form gathered from database
// TODO: need to be improved
type Timestamp string

// Pagination represents the parameters used for paging cluster report rules
type Pagination struct {
	PageNumber uint64
	PageSize   uint64
}

// RuleOnReport represents a single (hit) rule of the string encoded report
type RuleOnReport struct {
	Module   string `json:"component"`
	ErrorKey string `json:"key"`
}

// ReportRules is a helper struct for easy JSON unmarshalling of string encoded report
type ReportRules struct {
	Rules []RuleOnReport `json:"reports"`
}

// ReportResponse represents the response of /report endpoint
type ReportResponse struct {
	Count   int
	Page    uint64
	PerPage uint64
	Report  ClusterReport
	Rules   []RuleContentResponse
}

// RuleContentResponse represents a single rule in the response of /report endpoint
type RuleContentResponse struct {
	ErrorKey     string
	RuleModule   string
	Description  string
	CreatedAt    string
	TotalRisk    int
	RiskOfChange int
}
