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
type OrgID uint32

// ClusterName represents name of cluster in format c8590f31-e97e-4b85-b506-c45ce1911a12
type ClusterName string

// ClusterReport represents cluster report
type ClusterReport string

// Timestamp represents any timestamp in a form gathered from database
// TODO: need to be improved
type Timestamp string

// RuleOnReport represents a single (hit) rule of the string encoded report
type RuleOnReport struct {
	Module   string `json:"component"`
	ErrorKey string `json:"key"`
}

// ReportRules is a helper struct for easy JSON unmarshalling of string encoded report
type ReportRules struct {
	HitRules     []RuleOnReport `json:"reports"`
	SkippedRules []RuleOnReport `json:"skips"`
	PassedRules  []RuleOnReport `json:"pass"`
	TotalCount   int
}

// ReportResponse represents the response of /report endpoint
type ReportResponse struct {
	Meta  ReportResponseMeta    `json:"meta"`
	Rules []RuleContentResponse `json:"data"`
}

// ReportResponseMeta contains metadata about the report
type ReportResponseMeta struct {
	Count         int       `json:"count"`
	LastCheckedAt Timestamp `json:"last_checked_at"`
}

// RuleContentResponse represents a single rule in the response of /report endpoint
type RuleContentResponse struct {
	ErrorKey     string `json:"-"`
	RuleModule   string `json:"rule_id"`
	Description  string `json:"description"`
	Generic      string `json:"details"`
	CreatedAt    string `json:"created_at"`
	TotalRisk    int    `json:"total_risk"`
	RiskOfChange int    `json:"risk_of_change"`
}

// RuleID represents type for rule id
type RuleID string

// UserID represents type for user id
type UserID string

// Rule represents the content of rule table
type Rule struct {
	Module     RuleID `json:"module"`
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Reason     string `json:"reason"`
	Resolution string `json:"resolution"`
	MoreInfo   string `json:"more_info"`
}
